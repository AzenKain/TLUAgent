package docparser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func parseDOCXBytes(data []byte, doc *ParsedDocument) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			return extractDOCXContent(rc, doc)
		}
	}

	return fmt.Errorf("word/document.xml not found")
}

const maxDocxTextBytes = 256 * 1024

func extractDOCXContent(r io.Reader, doc *ParsedDocument) error {
	decoder := xml.NewDecoder(r)

	var textBuilder strings.Builder
	var currentText strings.Builder
	var inT bool

	var inTable bool
	var tableCount int
	var currentTable [][]string
	var currentRow []string
	var currentCell strings.Builder

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tbl":
				inTable = true
				currentTable = [][]string{}
			case "tr":
				currentRow = []string{}
			case "tc":
				currentCell.Reset()
			case "t":
				inT = true
			case "p":
				currentText.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "p":
				pText := strings.TrimSpace(currentText.String())
				if pText != "" {
					if textBuilder.Len()+len(pText)+1 > maxDocxTextBytes {
						remaining := maxDocxTextBytes - textBuilder.Len()
						if remaining > 0 && len(pText) > remaining {
							textBuilder.WriteString(pText[:remaining])
						}
						doc.TextContent = textBuilder.String()
						return nil
					}
					textBuilder.WriteString(pText)
					textBuilder.WriteString("\n")
				}
			case "tc":
				currentRow = append(currentRow, strings.TrimSpace(currentCell.String()))
			case "tr":
				if len(currentRow) > 0 {
					currentTable = append(currentTable, currentRow)
				}
			case "tbl":
				inTable = false
				tableCount++
				if len(currentTable) > 0 {
					sheetName := fmt.Sprintf("Table%d", tableCount)
					doc.Sheets[sheetName] = currentTable
				}
			}
		case xml.CharData:
			if inT {
				str := string(t)
				if currentText.Len()+len(str) > maxDocxTextBytes {
					str = str[:maxDocxTextBytes-currentText.Len()]
				}
				currentText.WriteString(str)
				if inTable {
					currentCell.WriteString(str)
				}
			}
		}
	}

	doc.TextContent = textBuilder.String()
	return nil
}
