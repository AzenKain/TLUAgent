package docparser

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/pkg/validator"
)

func redteamArchive(t *testing.T, name, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal("cannot create archive entry")
	}
	if _, err := io.WriteString(entry, content); err != nil {
		t.Fatal("cannot write archive entry")
	}
	if err := writer.Close(); err != nil {
		t.Fatal("cannot finish archive")
	}
	return buffer.Bytes()
}

func redteamValidatedAttachment(t *testing.T, filename string, data []byte) []byte {
	t.Helper()
	payload, err := json.Marshal(request.TLUClawChatRequest{
		Message: "Inspect this document",
		Attachments: []request.TLUClawAttachmentDTO{{
			FileName: filename,
			FileType: "application/octet-stream",
			Data:     base64.StdEncoding.EncodeToString(data),
		}},
	})
	if err != nil {
		t.Fatal("cannot encode attachment request")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/tluclaw/chat", bytes.NewReader(payload))
	req.Body = http.MaxBytesReader(httptest.NewRecorder(), req.Body, 10<<20)
	var dto request.TLUClawChatRequest
	if errs := validator.ValidateBodyDto(req, &dto); len(errs) != 0 {
		t.Fatal("bounded attachment failed actual DTO validation")
	}
	decoded, err := base64.StdEncoding.DecodeString(dto.Attachments[0].Data)
	if err != nil {
		t.Fatal("cannot decode validated attachment")
	}
	return decoded
}

// Tests that unchecked DOCX text expansion is bounded to prevent resource exhaustion.
func TestRedTeamObservedWeaknessDOCXInflation(t *testing.T) {
	const expandedBytes = 1 << 20
	content := "<document><body><p><t>" + strings.Repeat("A", expandedBytes) + "</t></p></body></document>"
	archive := redteamArchive(t, "word/document.xml", content)
	if len(archive) > 4096 {
		t.Fatal("fixture exceeded its compressed safety bound")
	}
	validated := redteamValidatedAttachment(t, "bounded.docx", archive)
	parsed, err := ParseBytes("bounded.docx", validated)
	if err != nil {
		t.Fatal("failed to parse bounded docx")
	}
	if len(parsed.TextContent) > maxDocxTextBytes {
		t.Fatalf("expanded document content exceeded safety limit: got %d, max %d", len(parsed.TextContent), maxDocxTextBytes)
	}
}

// Tests that sparse column amplification in XLSX is bounded to prevent unbounded memory allocation.
func TestRedTeamObservedWeaknessXLSXSparseColumnAmplification(t *testing.T) {
	content := `<worksheet><sheetData><row><c r="ZZZ1" t="inlineStr"><is><t>bounded</t></is></c></row></sheetData></worksheet>`
	archive := redteamArchive(t, "xl/worksheets/sheet1.xml", content)
	validated := redteamValidatedAttachment(t, "bounded.xlsx", archive)
	parsed, err := ParseBytes("bounded.xlsx", validated)
	if err != nil {
		t.Fatal("failed to parse bounded spreadsheet")
	}
	rows := parsed.Sheets["Sheet1"]
	if len(rows) > 0 && len(rows[0]) > maxColumnsPerSheet {
		t.Fatalf("sparse cell allocation exceeded maximum allowed columns: got %d, max %d", len(rows[0]), maxColumnsPerSheet)
	}
}
