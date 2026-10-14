package docparser

import (
	"bytes"
	"fmt"
	"strings"

	"tluagent-web/pkg/pdf"
	"tluagent-web/pkg/pipeline"
)

func parsePDFBytes(data []byte, doc *ParsedDocument) error {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open pdf: %w", err)
	}

	text, errClean := r.ExtractCleanText()
	if errClean != nil || strings.TrimSpace(text) == "" {
		plainReader, errPlain := r.GetPlainText()
		if errPlain != nil {
			return fmt.Errorf("read plain text from pdf: %w", errPlain)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(plainReader); err != nil {
			return fmt.Errorf("buffer pdf text: %w", err)
		}
		text = buf.String()
	}

	doc.TextContent = pipeline.StitchBrokenVietnameseLines(text)
	return nil
}
