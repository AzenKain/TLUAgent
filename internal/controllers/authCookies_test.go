package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/pkg/crypto"
)

func findCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("Cookie %s not found in response", name)
	return nil
}

func TestAuthCookies_SecureFlagFollowsEnvironment(t *testing.T) {
	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(request.SetupDto{
		Email:    "securecookie@thanglong.edu.vn",
		Password: adminPass,
		FullName: "Cookie Admin",
	})

	t.Run("production defaults to Secure cookies", func(t *testing.T) {
		mux, db := setupTestServer(t)
		defer db.Close()
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SECURE", "")

		req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Setup failed: %d, body: %s", rec.Code, rec.Body.String())
		}
		for _, name := range []string{"access_token", "refresh_token", "csrf_token"} {
			if c := findCookie(t, rec, name); !c.Secure {
				t.Fatalf("Cookie %s must have the Secure flag in production", name)
			}
		}
	})

	t.Run("development defaults to non-Secure cookies", func(t *testing.T) {
		mux, db := setupTestServer(t)
		defer db.Close()
		t.Setenv("APP_ENV", "development")
		t.Setenv("COOKIE_SECURE", "")

		req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Setup failed: %d, body: %s", rec.Code, rec.Body.String())
		}
		if c := findCookie(t, rec, "access_token"); c.Secure {
			t.Fatalf("Cookie must not be Secure outside production by default")
		}
	})
}

func TestSetupStatus_Returns404AfterCompletion(t *testing.T) {
	mux, db := setupTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(request.SetupDto{
		Email:    "setup404@thanglong.edu.vn",
		Password: adminPass,
		FullName: "Setup Admin",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Setup failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/setup/status must return 404 after setup completes, got %d", rec.Code)
	}
}

func TestSetup_RequiresSetupTokenWhenConfigured(t *testing.T) {
	setupToken, err := crypto.GenerateRandomHex(16)
	if err != nil {
		t.Fatalf("Failed to generate setup token: %v", err)
	}
	t.Setenv("SETUP_TOKEN", setupToken)

	mux, db := setupTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(request.SetupDto{
		Email:    "tokenadmin@thanglong.edu.vn",
		Password: adminPass,
		FullName: "Token Admin",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Setup without token must be forbidden when SETUP_TOKEN is set, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	req.Header.Set("X-Setup-Token", "definitely-not-the-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Setup with wrong token must be forbidden, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	req.Header.Set("X-Setup-Token", setupToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Setup with correct token must succeed, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_RejectsBodyTokenFallback(t *testing.T) {
	mux, db := setupTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(request.SetupDto{
		Email:    "refreshfallback@thanglong.edu.vn",
		Password: adminPass,
		FullName: "Refresh Admin",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Setup failed: %d", rec.Code)
	}
	_, refreshToken := authCookies(t, rec)
	if refreshToken == "" {
		t.Fatalf("Expected refresh token cookie from setup")
	}

	bodyBody, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(bodyBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Refresh via JSON body must be rejected, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Refresh via cookie must succeed, got %d, body: %s", rec.Code, rec.Body.String())
	}
}
