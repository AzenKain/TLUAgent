package validator

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateBodyDtoRejectsNonJSONContentType(t *testing.T) {
	type dto struct {
		Query string `json:"query" validate:"required"`
	}
	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{"query":"hi"}`))
	req.Header.Set("Content-Type", "text/plain")
	if errs := ValidateBodyDto(req, &dto{}); len(errs) == 0 {
		t.Fatal("a text/plain body must be rejected")
	}

	ok := httptest.NewRequest("POST", "/x", strings.NewReader(`{"query":"hi"}`))
	ok.Header.Set("Content-Type", "application/json; charset=utf-8")
	if errs := ValidateBodyDto(ok, &dto{}); len(errs) != 0 {
		t.Fatalf("application/json with parameters must pass: %v", errs)
	}
}
