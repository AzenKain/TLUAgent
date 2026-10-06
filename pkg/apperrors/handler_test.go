package apperrors

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tluagent-web/internal/dtos/response"
)

func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) response.CommonResponse {
	t.Helper()
	var resp response.CommonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	return resp
}

func TestHandleError_MasksUnknownErrors(t *testing.T) {
	leaky := fmt.Errorf("sqlite error: no such column: secret_internal_column near users table")
	rec := httptest.NewRecorder()
	HandleError(rec, leaky)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500 for unknown error, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec)
	if resp.Message != "Internal server error" {
		t.Fatalf("Expected masked message, got %q", resp.Message)
	}
	if strings.Contains(rec.Body.String(), "secret_internal_column") {
		t.Fatalf("Internal error details must not leak to the client")
	}
}

func TestHandleError_AppErrorKeepsIntentionalMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	HandleError(rec, New(ErrForbidden, "You do not have access to this session"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 for forbidden, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec)
	if resp.Message != "You do not have access to this session" {
		t.Fatalf("Expected intentional message, got %q", resp.Message)
	}
}

func TestHandleError_RawNoRowsMapsToNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	HandleError(rec, sql.ErrNoRows)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for sql.ErrNoRows, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec)
	if resp.Message != "Not found" {
		t.Fatalf("Expected generic not found message, got %q", resp.Message)
	}
}

func TestHandleError_WrappedSentinels(t *testing.T) {
	wrapped := fmt.Errorf("handler: %w", New(ErrUnauthorized, "Invalid email or password"))
	rec := httptest.NewRecorder()
	HandleError(rec, wrapped)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for wrapped unauthorized error, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec)
	if resp.Message != "Invalid email or password" {
		t.Fatalf("Expected intentional message through wrapping, got %q", resp.Message)
	}
	if errors.Is(wrapped, ErrUnauthorized) != true {
		t.Fatalf("Wrapped AppError must remain detectable via errors.Is")
	}
}
