package middlewares

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/pkg/jsonx"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Interface("recover", rec).
					Msg("Panic recovered")

				bytes, _ := jsonx.Marshal(response.CommonResponse{
					Status:  false,
					Message: "Internal server error",
				})
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write(bytes)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
