package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type samplePasswordPolicy struct {
	Password string `json:"password" validate:"required,min=10,max=100,password_policy"`
}

func TestPasswordPolicyValidation(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "letters and digits accepted", password: "SecurePass123", wantErr: false},
		{name: "too short rejected", password: "Abc123", wantErr: true},
		{name: "ten chars letters only rejected", password: "abcdefghij", wantErr: true},
		{name: "ten chars digits only rejected", password: "1234567890", wantErr: true},
		{name: "ten chars mixed accepted", password: "abc1234567", wantErr: false},
		{name: "empty rejected", password: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := samplePasswordPolicy{Password: tt.password}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type sampleBaseURL struct {
	BaseURL string `json:"base_url" validate:"required,base_url"`
}

func TestBaseURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "https with API path accepted", baseURL: "https://openrouter.ai/api/v1", wantErr: false},
		{name: "https bare origin accepted", baseURL: "https://novelhub.example.com", wantErr: false},
		{name: "http localhost accepted", baseURL: "http://localhost:8080", wantErr: false},
		{name: "query string rejected", baseURL: "https://example.com/api?key=1", wantErr: true},
		{name: "fragment rejected", baseURL: "https://example.com/api#section", wantErr: true},
		{name: "ftp scheme rejected", baseURL: "ftp://example.com/api", wantErr: true},
		{name: "missing host rejected", baseURL: "https:///api/v1", wantErr: true},
		{name: "garbage rejected", baseURL: "not-a-url", wantErr: true},
		{name: "crlf rejected", baseURL: "https://example.com\r\nX-Bad: 1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sampleBaseURL{BaseURL: tt.baseURL}
			err := validate.Struct(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
