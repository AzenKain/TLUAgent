package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"tluagent-web/pkg/guardrail"
)

const guardrailBodyLimit = 10 << 20

// GuardrailQuery blocks injection attempts in query strings and JSON bodies before they reach handlers.
func GuardrailQuery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if q := r.URL.Query().Get("q"); q != "" {
				if chk := guardrail.CheckQuery(q); chk.Decision == guardrail.DecisionBlockedInjection {
					http.Error(w, chk.Response, http.StatusBadRequest)
					return
				}
			}

			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/json") && r.Body != nil {
					limitedBody := http.MaxBytesReader(w, r.Body, guardrailBodyLimit)
					bodyBytes, err := io.ReadAll(limitedBody)
					if err != nil {
						http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
						return
					}
					if len(bodyBytes) > 0 {
						r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
						var payload struct {
							Query       string `json:"query"`
							Description string `json:"description"`
							Content     string `json:"content"`
						}
						if jsonErr := json.Unmarshal(bodyBytes, &payload); jsonErr == nil {
							checkTarget := payload.Query
							if checkTarget == "" {
								checkTarget = payload.Description
							}
							if checkTarget == "" {
								checkTarget = payload.Content
							}
							if checkTarget != "" {
								if chk := guardrail.CheckQuery(checkTarget); chk.Decision == guardrail.DecisionBlockedInjection {
									http.Error(w, chk.Response, http.StatusBadRequest)
									return
								}
							}
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
