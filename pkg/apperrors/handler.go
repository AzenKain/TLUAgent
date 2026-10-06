package apperrors

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"tluagent-web/internal/dtos/response"
	"tluagent-web/pkg/jsonx"
)

const internalServerErrorMessage = "Internal server error"

// HandleError converts a domain error into an HTTP status and JSON response,
// masking non-domain errors so internal details never reach the client.
func HandleError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	message := internalServerErrorMessage

	var appErr *AppError
	hasIntentionalMessage := errors.As(err, &appErr) && appErr.Message != ""

	switch {
	case errors.Is(err, ErrBadRequest):
		code = http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		code = http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, ErrTooManyRequests):
		code = http.StatusTooManyRequests
	case errors.Is(err, ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, sql.ErrNoRows):
		code = http.StatusNotFound
	}

	switch {
	case hasIntentionalMessage:
		message = appErr.Message
	case errors.Is(err, sql.ErrNoRows) && !errors.Is(err, ErrNotFound):
		message = "Not found"
	case code == http.StatusInternalServerError:
		log.Error().Err(err).Msg("Unhandled error masked as internal server error")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	data, _ := jsonx.Marshal(response.CommonResponse{
		Status:  false,
		Message: message,
	})
	_, _ = w.Write(data)
}
