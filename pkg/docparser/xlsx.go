package docparser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	maxColumnsPerSheet = 256
	maxRowsPerSheet    = 5000
)

func parseXLSXBytes(data []byte, doc *ParsedDocument) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip reader: %w", err)
	}

	sharedStrings := extractSharedStrings(r)
	sheetFiles := extractSheetFiles(r)

	var textBuilder strings.Builder
	for _, sf := range sheetFiles {
		rows, err := parseWorksheet(r, sf.Path, sharedStrings)
		if err != nil {
			continue
		}
		if len(rows) > 0 {
			doc.Sheets[sf.Name] = rows
			textBuilder.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", sf.Name))
			for _, row := range rows {
				textBuilder.WriteString(strings.Join(row, " | "))
				textBuilder.WriteString("\n")
			}
			textBuilder.WriteString("\n")
		}
	}

	doc.TextContent = textBuilder.String()
	return nil
}

type sheetRef struct {
	Name string
	Path string
}

func extractSheetFiles(r *zip.Reader) []sheetRef {
	relMap := make(map[string]string)
	for _, f := range r.File {
		if f.Name == "xl/_rels/workbook.xml.rels" {
			if rc, err := f.Open(); err == nil {
				var rels struct {
					Items []struct {
						ID     string `xml:"Id,attr"`
						Target string `xml:"Target,attr"`
					} `xml:"Relationship"`
				}
				_ = xml.NewDecoder(rc).Decode(&rels)
				rc.Close()
				for _, item := range rels.Items {
					target := item.Target
					if !strings.HasPrefix(target, "xl/") {
						target = "xl/" + strings.TrimPrefix(target, "/")
					}
					relMap[item.ID] = filepath.ToSlash(filepath.Clean(target))
				}
			}
			break
		}
	}

	var results []sheetRef
	for _, f := range r.File {
		if f.Name == "xl/workbook.xml" {
			if rc, err := f.Open(); err == nil {
				var wb struct {
					Sheets []struct {
						Name string `xml:"name,attr"`
						RID  string `xml:"id,attr"`
					} `xml:"sheets>sheet"`
				}
				_ = xml.NewDecoder(rc).Decode(&wb)
				rc.Close()
				for i, s := range wb.Sheets {
					target := relMap[s.RID]
					if target == "" {
						target = fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
					}
					results = append(results, sheetRef{Name: s.Name, Path: target})
				}
			}
			break
		}
	}

	if len(results) == 0 {
		var sheetNames []string
		for _, f := range r.File {
			if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
				sheetNames = append(sheetNames, f.Name)
			}
		}
		sort.Strings(sheetNames)
		for i, sn := range sheetNames {
			results = append(results, sheetRef{Name: fmt.Sprintf("Sheet%d", i+1), Path: sn})
		}
	}

	return results
}

func extractSharedStrings(r *zip.Reader) []string {
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			defer rc.Close()

			var shared []string
			decoder := xml.NewDecoder(rc)
			var inText bool
			var current strings.Builder

			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				switch t := tok.(type) {
				case xml.StartElement:
					if t.Name.Local == "t" {
						inText = true
					}
				case xml.EndElement:
					if t.Name.Local == "t" {
						inText = false
					} else if t.Name.Local == "si" {
						shared = append(shared, current.String())
						current.Reset()
					}
				case xml.CharData:
					if inText {
						current.WriteString(string(t))
					}
				}
			}
			return shared
		}
	}
	return nil
}

func parseWorksheet(r *zip.Reader, path string, sharedStrings []string) ([][]string, error) {
	for _, f := range r.File {
		if f.Name == path {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			var rows [][]string
			decoder := xml.NewDecoder(rc)

			var currentRow []string
			var currentCell strings.Builder
			var cellRef string
			var cellType string
			var inV, inIsT bool

			for {
				tok, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return rows, err
				}

				switch t := tok.(type) {
				case xml.StartElement:
					if t.Name.Local == "row" {
						currentRow = []string{}
					} else if t.Name.Local == "c" {
						cellRef = ""
						cellType = ""
						for _, attr := range t.Attr {
							if attr.Name.Local == "r" {
								cellRef = attr.Value
							} else if attr.Name.Local == "t" {
								cellType = attr.Value
							}
						}
						currentCell.Reset()
					} else if t.Name.Local == "v" {
						inV = true
					} else if t.Name.Local == "t" {
						inIsT = true
					}
				case xml.EndElement:
					if t.Name.Local == "v" {
						inV = false
					} else if t.Name.Local == "t" {
						inIsT = false
					} else if t.Name.Local == "c" {
						val := currentCell.String()
						if cellType == "s" {
							if idx, err := strconv.Atoi(val); err == nil && idx >= 0 && idx < len(sharedStrings) {
								val = sharedStrings[idx]
							}
						}
						colIdx := colNameToIndex(cellRef)
						if colIdx >= 0 && colIdx < maxColumnsPerSheet {
							for len(currentRow) < colIdx {
								currentRow = append(currentRow, "")
							}
							currentRow = append(currentRow, strings.TrimSpace(val))
						}
					} else if t.Name.Local == "row" {
						if len(currentRow) > 0 && len(rows) < maxRowsPerSheet {
							hasData := false
							for _, c := range currentRow {
								if strings.TrimSpace(c) != "" {
									hasData = true
									break
								}
							}
							if hasData {
								rows = append(rows, currentRow)
							}
						}
					}
				case xml.CharData:
					if inV || inIsT {
						currentCell.WriteString(string(t))
					}
				}
			}
			return rows, nil
		}
	}
	return nil, fmt.Errorf("sheet file %s not found", path)
}

func colNameToIndex(ref string) int {
	var colStr strings.Builder
	for _, r := range ref {
		if unicode.IsLetter(r) {
			colStr.WriteRune(unicode.ToUpper(r))
		} else {
			break
		}
	}
	s := colStr.String()
	if s == "" {
		return 0
	}
	idx := 0
	for i := 0; i < len(s); i++ {
		idx = idx*26 + int(s[i]-'A'+1)
	}
	return idx - 1
}
