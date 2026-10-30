package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuardrailQueryKeepsLargeJSONBodiesIntact(t *testing.T) {
	var got int
	handler := GuardrailQuery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		b := make([]byte, 4096)
		for {
			n, err := r.Body.Read(b)
			buf.Write(b[:n])
			if err != nil {
				break
			}
		}
		got = buf.Len()
		w.WriteHeader(http.StatusOK)
	}))

	payload := `{"query":"hello","images":["` + strings.Repeat("a", 2<<20) + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || got != len(payload) {
		t.Fatalf("a body above 1 MiB must reach the handler intact: status=%d read=%d want=%d", rec.Code, got, len(payload))
	}
}
