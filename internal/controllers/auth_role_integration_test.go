package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/database"
)

func setupTestServer(t *testing.T) (*http.ServeMux, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	txManager := database.NewTxManager(db)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		t.Fatalf("Failed to reload permCache: %v", err)
	}

	roleSvc := services.NewRoleService(roleRepo, permCache, txManager)
	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)
	userSvc := services.NewUserService(userRepo, roleRepo, settingsRepo, txManager)

	authCtrl := controllers.NewAuthController(authSvc)
	roleCtrl := controllers.NewRoleController(roleSvc)
	userCtrl := controllers.NewUserController(userSvc)

	mux := http.NewServeMux()
	routes.RegisterAuthRoutes(mux, authCtrl, userRepo)
	routes.RegisterRoleRoutes(mux, roleCtrl, userRepo, permCache)
	routes.RegisterUserRoutes(mux, userCtrl, userRepo, permCache)

	return mux, db
}

func authCookies(t *testing.T, rec *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var access, refresh string
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case "access_token":
			access = c.Value
		case "refresh_token":
			refresh = c.Value
		}
	}
	return access, refresh
}

func authUserFromBody(t *testing.T, rec *httptest.ResponseRecorder) *response.AuthResponse {
	t.Helper()
	var resp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	dataBytes, _ := json.Marshal(resp.Data)
	var auth response.AuthResponse
	_ = json.Unmarshal(dataBytes, &auth)
	return &auth
}

func TestAuthAndRBAC_Integration(t *testing.T) {
	mux, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/setup/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 for setup status, got: %d", rec.Code)
	}
	var setupStatusResp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &setupStatusResp)
	statusMap, _ := setupStatusResp.Data.(map[string]any)
	if statusMap["required"] != true {
		t.Fatalf("Expected setup required to be true, got: %v", statusMap)
	}

	req = httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader([]byte("{}")))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Public register should not be available, got status: %d", rec.Code)
	}

	adminPass, _ := crypto.GenerateRandomHex(16)
	studentPass, _ := crypto.GenerateRandomHex(16)

	setupBody, _ := json.Marshal(request.SetupDto{
		Email:       "admin@thanglong.edu.vn",
		Password:    adminPass,
		FullName:    "System Administrator",
		StudentCode: "ADMIN01",
	})
	req = httptest.NewRequest("POST", "/api/setup", bytes.NewReader(setupBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Setup wizard failed with status: %d, body: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _ := authCookies(t, rec)
	if adminAccess == "" {
		t.Fatalf("Expected access token cookie from setup")
	}
	if !strings.Contains(rec.Body.String(), "access_token") || !strings.Contains(rec.Body.String(), "refresh_token") {
		t.Log("token fields absent from JSON body as required")
	} else {
		t.Fatalf("Auth response body must not contain access_token or refresh_token fields")
	}

	adminAuth := authUserFromBody(t, rec)
	if adminAuth.User == nil {
		t.Fatalf("Expected user object in setup response body")
	}
	if !adminAuth.User.Roles[0].IsAdmin && adminAuth.User.Roles[0].Name != "ADMIN" {
		t.Fatalf("Setup user should be ADMIN, got roles: %+v", adminAuth.User.Roles)
	}

	req = httptest.NewRequest("GET", "/api/setup/status", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("Setup status must return 404 once setup is complete, got: %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/setup", bytes.NewReader(setupBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Submitting setup again should return 403 Forbidden, got: %d", rec.Code)
	}

	adminLoginBody, _ := json.Marshal(request.SignInDto{
		Email:    "admin@thanglong.edu.vn",
		Password: adminPass,
	})
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(adminLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Admin login failed with status: %d, body: %s", rec.Code, rec.Body.String())
	}
	adminAccess, _ = authCookies(t, rec)

	studentCreateBody, _ := json.Marshal(request.CreateUserDto{
		Email:       "student@thanglong.edu.vn",
		Password:    studentPass,
		FullName:    "Nguyen Van A",
		StudentCode: "B00123",
	})
	req = httptest.NewRequest("POST", "/api/users", bytes.NewReader(studentCreateBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized when creating user without auth, got: %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/users", bytes.NewReader(studentCreateBody))
	req.Header.Set("Authorization", "Bearer "+adminAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Admin creating student user failed: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var createdUserResp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &createdUserResp)
	createdUserDataBytes, _ := json.Marshal(createdUserResp.Data)
	var createdUser response.UserResponse
	_ = json.Unmarshal(createdUserDataBytes, &createdUser)
	if createdUser.Email != "student@thanglong.edu.vn" {
		t.Fatalf("Created user email mismatch: %s", createdUser.Email)
	}

	studentLoginBody, _ := json.Marshal(request.SignInDto{
		Email:    "student@thanglong.edu.vn",
		Password: studentPass,
	})
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(studentLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Student login failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	studentAccess, studentRefresh := authCookies(t, rec)

	req = httptest.NewRequest("POST", "/api/users", bytes.NewReader(studentCreateBody))
	req.Header.Set("Authorization", "Bearer "+studentAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Student creating user should return 403 Forbidden, got: %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/roles", nil)
	req.Header.Set("Authorization", "Bearer "+studentAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Student accessing /api/roles should be Forbidden 403, got: %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/roles", nil)
	req.Header.Set("Authorization", "Bearer "+adminAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Admin accessing /api/roles should be 200 OK, got: %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: studentRefresh})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Refresh token failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	_, rotatedRefresh := authCookies(t, rec)
	if rotatedRefresh == "" || rotatedRefresh == studentRefresh {
		t.Fatalf("Refresh must rotate the refresh token cookie")
	}

	req = httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: studentRefresh})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Reusing already rotated refresh token should return 401 Unauthorized, got: %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(studentLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	studentAccess, _ = authCookies(t, rec)

	req = httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+studentAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Logout failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+studentAccess)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Access token should be revoked after logout, got status: %d", rec.Code)
	}
}

func TestValidation_Integration(t *testing.T) {
	mux, db := setupTestServer(t)
	defer db.Close()

	shortPass, _ := crypto.GenerateRandomHex(2)
	invalidSetupBody, _ := json.Marshal(request.SetupDto{
		Email:    "not-an-email",
		Password: shortPass,
		FullName: "A",
	})
	req := httptest.NewRequest("POST", "/api/setup", bytes.NewReader(invalidSetupBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for invalid setup payload, got: %d", rec.Code)
	}

	var resp response.CommonResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Status {
		t.Fatalf("Expected status to be false")
	}

	errsBytes, _ := json.Marshal(resp.Errors)
	var errList []map[string]any
	_ = json.Unmarshal(errsBytes, &errList)
	if len(errList) < 3 {
		t.Fatalf("Expected at least 3 validation errors, got: %v", resp.Errors)
	}

	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("")))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for empty body, got: %d", rec.Code)
	}
}

func TestUserManagement_FullCRUD_Integration(t *testing.T) {
	mux, db := setupTestServer(t)
	defer db.Close()

	rootPass, _ := crypto.GenerateRandomHex(16)
	studentPass, _ := crypto.GenerateRandomHex(16)

	setupBody, _ := json.Marshal(request.SetupDto{
		Email:       "rootadmin@thanglong.edu.vn",
		Password:    rootPass,
		FullName:    "Root Admin",
		StudentCode: "ADMIN001",
	})
	req := httptest.NewRequest("POST", "/api/setup", bytes.NewReader(setupBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Initial setup failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	rootToken, _ := authCookies(t, rec)
	rootAuth := authUserFromBody(t, rec)
	rootAdminID := rootAuth.User.ID

	createUserBody, _ := json.Marshal(request.CreateUserDto{
		Email:       "student1@thanglong.edu.vn",
		Password:    studentPass,
		FullName:    "Student A",
		StudentCode: "A12345",
	})
	req = httptest.NewRequest("POST", "/api/users", bytes.NewReader(createUserBody))
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Create user failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	var createUserResp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &createUserResp)
	userBytes, _ := json.Marshal(createUserResp.Data)
	var createdUser response.UserResponse
	_ = json.Unmarshal(userBytes, &createdUser)
	userID := createdUser.ID
	if userID == "" || createdUser.Email != "student1@thanglong.edu.vn" {
		t.Fatalf("Unexpected created user: %+v", createdUser)
	}

	req = httptest.NewRequest("GET", "/api/users?search=student1", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Search users failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	var paginatedResp response.PaginatedResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &paginatedResp)
	if paginatedResp.Pagination == nil || paginatedResp.Pagination.TotalRecords != 1 {
		t.Fatalf("Expected 1 record found, got: %+v", paginatedResp.Pagination)
	}

	req = httptest.NewRequest("GET", "/api/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Get user by ID failed: %d", rec.Code)
	}

	updatedName := "Student A Renamed"
	updatedCode := "A12345_NEW"
	updateProfileBody, _ := json.Marshal(request.UpdateProfileDto{
		FullName:    &updatedName,
		StudentCode: &updatedCode,
	})
	req = httptest.NewRequest("PUT", "/api/users/"+userID, bytes.NewReader(updateProfileBody))
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Admin update profile failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	var updateResp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &updateResp)
	updatedBytes, _ := json.Marshal(updateResp.Data)
	var updatedUser response.UserResponse
	_ = json.Unmarshal(updatedBytes, &updatedUser)
	if updatedUser.FullName != updatedName || updatedUser.StudentCode != updatedCode {
		t.Fatalf("Profile not updated as expected: %+v", updatedUser)
	}

	req = httptest.NewRequest("GET", "/api/roles", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var rolesResp response.CommonResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &rolesResp)
	rolesBytes, _ := json.Marshal(rolesResp.Data)
	var rolesList []response.RoleResponse
	_ = json.Unmarshal(rolesBytes, &rolesList)
	var advisorRoleID string
	for _, r := range rolesList {
		if r.Name == "ADVISOR" {
			advisorRoleID = r.ID
			break
		}
	}
	if advisorRoleID == "" {
		t.Fatalf("ADVISOR role not found in system")
	}

	changeRoleBody, _ := json.Marshal(request.ChangeRoleDto{
		Roles: []string{advisorRoleID},
	})
	req = httptest.NewRequest("PUT", "/api/users/"+userID+"/roles", bytes.NewReader(changeRoleBody))
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Change role failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	newPassword, _ := crypto.GenerateRandomHex(16)
	resetPassBody, _ := json.Marshal(request.ResetPasswordDto{
		NewPassword: newPassword,
	})
	req = httptest.NewRequest("POST", "/api/users/"+userID+"/password", bytes.NewReader(resetPassBody))
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Reset password failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	userLoginBody, _ := json.Marshal(request.SignInDto{
		Email:    "student1@thanglong.edu.vn",
		Password: newPassword,
	})
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(userLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Login with new password failed: %d, body: %s", rec.Code, rec.Body.String())
	}
	userToken, _ := authCookies(t, rec)

	req = httptest.NewRequest("GET", "/api/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Get current user failed: %d", rec.Code)
	}

	myNewName := "Self Renamed"
	selfUpdateBody, _ := json.Marshal(request.UpdateProfileDto{
		FullName: &myNewName,
	})
	req = httptest.NewRequest("PUT", "/api/users/current", bytes.NewReader(selfUpdateBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Current user update profile failed: %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/users", bytes.NewReader(createUserBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Regular user should be forbidden from creating users, got: %d", rec.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/users/"+rootAdminID, nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Admin deleting themselves should be forbidden, got: %d", rec.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Delete user failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(userLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Deleted user should not be able to login, got: %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/users/"+userID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+rootToken)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Restore user failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(userLoginBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Restored user should be able to login, got: %d, body: %s", rec.Code, rec.Body.String())
	}
}
