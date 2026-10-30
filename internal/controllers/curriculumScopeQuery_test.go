package controllers

import (
	"net/http/httptest"
	"testing"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/pkg/validator"
)

func TestRequiredCurriculumScopeQueryDto(t *testing.T) {
	var ok request.RequiredCurriculumScopeQueryDto
	if errs := validator.ValidateQueryDto(httptest.NewRequest("GET", "/x?cohort=K35&major=CNTT", nil), &ok); len(errs) != 0 {
		t.Fatalf("both parameters present must validate: %v", errs)
	}
	if ok.Cohort != "K35" || ok.Major != "CNTT" {
		t.Fatalf("query values must be decoded, got %+v", ok)
	}

	var missing request.RequiredCurriculumScopeQueryDto
	if errs := validator.ValidateQueryDto(httptest.NewRequest("GET", "/x?cohort=K35", nil), &missing); len(errs) == 0 {
		t.Fatal("a missing major must be rejected")
	}

	var optional request.CurriculumScopeQueryDto
	if errs := validator.ValidateQueryDto(httptest.NewRequest("GET", "/x", nil), &optional); len(errs) != 0 {
		t.Fatalf("the optional scope must accept an empty query: %v", errs)
	}
}
