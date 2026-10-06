package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type sampleQuery struct {
	Cursor string `query:"cursor" validate:"omitempty,readlist_cursor"`
}

func TestReadListCursorValidation(t *testing.T) {
	tests := []struct {
		name    string
		cursor  string
		wantErr bool
	}{
		{
			name:    "empty cursor is valid",
			cursor:  "",
			wantErr: false,
		},
		{
			name:    "RFC3339 timestamp only is valid",
			cursor:  time.Now().Format(time.RFC3339Nano),
			wantErr: false,
		},
		{
			name:    "RFC3339 timestamp with pipe and ID is valid",
			cursor:  time.Now().Format(time.RFC3339Nano) + "|readlist-123",
			wantErr: false,
		},
		{
			name:    "invalid timestamp format returns error",
			cursor:  "invalid-date|123",
			wantErr: true,
		},
		{
			name:    "random garbage string returns error",
			cursor:  "random_garbage",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sampleQuery{Cursor: tt.cursor}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type sampleServerURL struct {
	ServerURL *string `json:"server.url" validate:"omitempty,server_url,max=2048"`
}

func TestServerURLValidation(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name      string
		serverURL *string
		wantErr   bool
	}{
		{
			name:      "nil pointer is valid",
			serverURL: nil,
			wantErr:   false,
		},
		{
			name:      "empty string pointer is valid",
			serverURL: strPtr(""),
			wantErr:   false,
		},
		{
			name:      "whitespace only string pointer is valid",
			serverURL: strPtr("   "),
			wantErr:   false,
		},
		{
			name:      "valid http localhost is valid",
			serverURL: strPtr("http://localhost:8080"),
			wantErr:   false,
		},
		{
			name:      "valid https domain is valid",
			serverURL: strPtr("https://novelhub.example.com"),
			wantErr:   false,
		},
		{
			name:      "valid https with trailing slash is valid",
			serverURL: strPtr("https://novelhub.example.com/"),
			wantErr:   false,
		},
		{
			name:      "invalid URL with path returns error",
			serverURL: strPtr("https://novelhub.example.com/books"),
			wantErr:   true,
		},
		{
			name:      "invalid URL with query returns error",
			serverURL: strPtr("https://novelhub.example.com?query=1"),
			wantErr:   true,
		},
		{
			name:      "invalid scheme returns error",
			serverURL: strPtr("ftp://example.com"),
			wantErr:   true,
		},
		{
			name:      "newlines in URL returns error",
			serverURL: strPtr("https://example.com\r\nX-Bad: 1"),
			wantErr:   true,
		},
		{
			name:      "garbage text returns error",
			serverURL: strPtr("not-a-valid-url"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sampleServerURL{ServerURL: tt.serverURL}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	str := "invalid_url"
	s := sampleServerURL{ServerURL: &str}
	err := validate.Struct(s)
	assert.Error(t, err)

	errList := formatValidationError(err)
	assert.Len(t, errList, 1)
	assert.Equal(t, "server.url", errList[0].FailedField)
	assert.Equal(t, "server_url", errList[0].Tag)
	assert.Equal(t, "server.url must be a valid http or https URL", errList[0].Message)
}

type sampleDto struct {
	Email    string  `json:"email" validate:"required,email"`
	Username string  `json:"username" validate:"required,min=3,max=20"`
	SiteURL  *string `json:"site_url" validate:"omitempty,server_url"`
}

type sampleQueryDto struct {
	Page  int    `query:"page" validate:"required,min=1"`
	Limit int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Sort  string `query:"sort" validate:"omitempty,oneof=asc desc"`
}

func TestValidateBodyDto_Success(t *testing.T) {
	payload := `{"email":"test@example.com","username":"john_doe"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(payload)))

	var dto sampleDto
	errs := ValidateBodyDto(req, &dto)
	assert.Nil(t, errs)
	assert.Equal(t, "test@example.com", dto.Email)
	assert.Equal(t, "john_doe", dto.Username)
}

func TestValidateBodyDto_Failures(t *testing.T) {
	t.Run("Empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("")))
		var dto sampleDto
		errs := ValidateBodyDto(req, &dto)
		assert.NotEmpty(t, errs)
		assert.Equal(t, "Request body cannot be empty", errs[0].Message)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("{invalid-json")))
		var dto sampleDto
		errs := ValidateBodyDto(req, &dto)
		assert.NotEmpty(t, errs)
		assert.Contains(t, errs[0].Message, "Invalid JSON format")
	})

	t.Run("Invalid field validations", func(t *testing.T) {
		payload := `{"email":"invalid-email","username":"a"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(payload)))
		var dto sampleDto
		errs := ValidateBodyDto(req, &dto)
		assert.Len(t, errs, 2)

		fieldMap := make(map[string]*ErrorResponse)
		for _, e := range errs {
			fieldMap[e.FailedField] = e
		}

		assert.Contains(t, fieldMap, "email")
		assert.Equal(t, "email", fieldMap["email"].Tag)
		assert.Equal(t, "The email address is invalid", fieldMap["email"].Message)

		assert.Contains(t, fieldMap, "username")
		assert.Equal(t, "min", fieldMap["username"].Tag)
		assert.Equal(t, "username is too short (min 3)", fieldMap["username"].Message)
	})
}

func TestValidateQueryDto(t *testing.T) {
	t.Run("Valid query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=2&limit=50&sort=desc", nil)
		var q sampleQueryDto
		errs := ValidateQueryDto(req, &q)
		assert.Nil(t, errs)
		assert.Equal(t, 2, q.Page)
		assert.Equal(t, 50, q.Limit)
		assert.Equal(t, "desc", q.Sort)
	})

	t.Run("Invalid query values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=0&limit=500&sort=invalid", nil)
		var q sampleQueryDto
		errs := ValidateQueryDto(req, &q)
		assert.Len(t, errs, 3)
	})
}
