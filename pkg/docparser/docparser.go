package docparser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ExtractedCourse represents a structured academic course parsed from a spreadsheet or document.
type ExtractedCourse struct {
	CourseCode        string  `json:"course_code"`
	CourseName        string  `json:"course_name"`
	Credits           float64 `json:"credits"`
	Weight            float64 `json:"weight"`
	Hours             int64   `json:"hours"`
	KnowledgeBlock    string  `json:"knowledge_block"`
	PrerequisitesText string  `json:"prerequisites_text"`
	SuggestedSemester int64   `json:"suggested_semester"`
	IsElearning       bool    `json:"is_elearning"`
}

// ParsedDocument holds extracted content, tabular sheets, and detected course records.
type ParsedDocument struct {
	FileType    string              `json:"file_type"`
	FileName    string              `json:"file_name"`
	TextContent string              `json:"text_content"`
	Sheets      map[string][][]string `json:"sheets,omitempty"`
	Courses     []ExtractedCourse   `json:"courses,omitempty"`
}

// ParseFile parses a document file by its filesystem path.
func ParseFile(filePath string) (*ParsedDocument, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", filePath, err)
	}
	return ParseBytes(filepath.Base(filePath), data)
}

// ParseBytes parses a document from in-memory byte buffer.
func ParseBytes(filename string, data []byte) (*ParsedDocument, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	parsed := &ParsedDocument{
		FileType: ext,
		FileName: filename,
		Sheets:   make(map[string][][]string),
	}

	switch ext {
	case ".xlsx", ".xlsm":
		if err := parseXLSXBytes(data, parsed); err != nil {
			return nil, fmt.Errorf("parse xlsx: %w", err)
		}
	case ".csv":
		if err := parseCSVBytes(data, parsed); err != nil {
			return nil, fmt.Errorf("parse csv: %w", err)
		}
	case ".docx":
		if err := parseDOCXBytes(data, parsed); err != nil {
			return nil, fmt.Errorf("parse docx: %w", err)
		}
	case ".pdf":
		if err := parsePDFBytes(data, parsed); err != nil {
			return nil, fmt.Errorf("parse pdf: %w", err)
		}
	case ".txt", ".md":
		parsed.TextContent = string(data)
	default:
		parsed.TextContent = string(data)
	}

	detectAndPopulateCourses(parsed)
	return parsed, nil
}

func detectAndPopulateCourses(doc *ParsedDocument) {
	for _, rows := range doc.Sheets {
		courses := extractCoursesFromTable(rows)
		if len(courses) > 0 {
			doc.Courses = append(doc.Courses, courses...)
		}
	}
}

func extractCoursesFromTable(rows [][]string) []ExtractedCourse {
	if len(rows) < 2 {
		return nil
	}

	codeCol := -1
	nameCol := -1
	creditsCol := -1
	prereqCol := -1
	weightCol := -1
	hoursCol := -1
	blockCol := -1
	semCol := -1

	headerRowIdx := -1
	for rIdx, row := range rows {
		cIdx, nIdx, crIdx := -1, -1, -1
		for c, cell := range row {
			norm := strings.ToLower(strings.TrimSpace(cell))
			norm = strings.ReplaceAll(norm, " ", "")
			norm = strings.ReplaceAll(norm, "_", "")

			if strings.Contains(norm, "mãhp") || strings.Contains(norm, "mãmôn") || strings.Contains(norm, "mãhọcphần") || norm == "code" {
				cIdx = c
			} else if strings.Contains(norm, "tênhp") || strings.Contains(norm, "tênmôn") || strings.Contains(norm, "tênhọcphần") || norm == "name" || norm == "title" {
				nIdx = c
			} else if strings.Contains(norm, "sốtc") || strings.Contains(norm, "tínchỉ") || norm == "credits" || norm == "tc" {
				crIdx = c
			} else if strings.Contains(norm, "tiênquyết") || strings.Contains(norm, "prereq") {
				prereqCol = c
			} else if strings.Contains(norm, "hệsố") || strings.Contains(norm, "weight") {
				weightCol = c
			} else if strings.Contains(norm, "giờ") || strings.Contains(norm, "hours") || strings.Contains(norm, "tiết") {
				hoursCol = c
			} else if strings.Contains(norm, "khốikt") || strings.Contains(norm, "khốikiếnthức") || norm == "block" {
				blockCol = c
			} else if strings.Contains(norm, "họckỳ") || strings.Contains(norm, "semester") || norm == "hk" {
				semCol = c
			}
		}

		if cIdx != -1 && (nIdx != -1 || crIdx != -1) {
			codeCol = cIdx
			nameCol = nIdx
			creditsCol = crIdx
			headerRowIdx = rIdx
			break
		}
	}

	if headerRowIdx == -1 || codeCol == -1 {
		return nil
	}

	var results []ExtractedCourse
	for r := headerRowIdx + 1; r < len(rows); r++ {
		row := rows[r]
		if len(row) <= codeCol {
			continue
		}

		rawCode := strings.ToUpper(strings.TrimSpace(row[codeCol]))
		if rawCode == "" || len(rawCode) < 2 || strings.Contains(rawCode, "MÃ") || strings.Contains(rawCode, "TỔNG") {
			continue
		}

		var rawName string
		if nameCol != -1 && len(row) > nameCol {
			rawName = strings.TrimSpace(row[nameCol])
		}

		var credits float64 = 3.0
		if creditsCol != -1 && len(row) > creditsCol {
			credits = parseCreditValue(row[creditsCol])
		}

		var prereq string
		if prereqCol != -1 && len(row) > prereqCol {
			prereq = strings.TrimSpace(row[prereqCol])
		}

		var weight float64 = 1.0
		if weightCol != -1 && len(row) > weightCol {
			if w, err := strconv.ParseFloat(strings.TrimSpace(row[weightCol]), 64); err == nil && w > 0 {
				weight = w
			}
		}

		var hours int64 = int64(credits * 15)
		if hoursCol != -1 && len(row) > hoursCol {
			if h, err := strconv.ParseInt(strings.TrimSpace(row[hoursCol]), 10, 64); err == nil && h > 0 {
				hours = h
			}
		}

		var block string
		if blockCol != -1 && len(row) > blockCol {
			block = strings.TrimSpace(row[blockCol])
		}

		var sem int64 = 1
		if semCol != -1 && len(row) > semCol {
			if s, err := strconv.ParseInt(strings.TrimSpace(row[semCol]), 10, 64); err == nil && s > 0 {
				sem = s
			}
		}

		results = append(results, ExtractedCourse{
			CourseCode:        rawCode,
			CourseName:        rawName,
			Credits:           credits,
			Weight:            weight,
			Hours:             hours,
			KnowledgeBlock:    block,
			PrerequisitesText: prereq,
			SuggestedSemester: sem,
		})
	}

	return results
}

func parseCreditValue(raw string) float64 {
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "("); idx > 0 {
		clean = strings.TrimSpace(clean[:idx])
	}
	clean = strings.ReplaceAll(clean, ",", ".")
	if val, err := strconv.ParseFloat(clean, 64); err == nil && val > 0 {
		return val
	}
	return 3.0
}

func readAllLimit(r io.Reader, maxBytes int64) ([]byte, error) {
	lr := io.LimitReader(r, maxBytes)
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(lr)
	return buf.Bytes(), err
}
