package docparser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

func parseCSVBytes(data []byte, doc *ParsedDocument) error {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	doc.Sheets["Sheet1"] = rows
	var textBuilder strings.Builder
	for _, row := range rows {
		textBuilder.WriteString(strings.Join(row, " | "))
		textBuilder.WriteString("\n")
	}
	doc.TextContent = textBuilder.String()
	return nil
}
