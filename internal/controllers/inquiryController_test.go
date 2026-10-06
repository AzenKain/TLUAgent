package controllers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func setupInquiryTest(t *testing.T) (*controllers.InquiryController, *controllers.NotificationController, *http.ServeMux, string, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	err = database.ApplySchema(db)
	require.NoError(t, err)

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	inquiryRepo := repositories.NewInquiryRepository(db, ramCache)
	notificationRepo := repositories.NewNotificationRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)

	ctx := context.Background()

	studentID := "ctrl-student-1"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           studentID,
		Email:        "sv_ctrl@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Van Ctrl", Valid: true},
		StudentCode:  sql.NullString{String: "A36777", Valid: true},
		AuthProvider: "local",
	})
	require.NoError(t, err)

	teacherID := "ctrl-teacher-1"
	_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           teacherID,
		Email:        "teacher_ctrl@thanglong.edu.vn",
		FullName:     sql.NullString{String: "TS. Pham Advisor", Valid: true},
		AuthProvider: "local",
	})
	require.NoError(t, err)

	inquirySvc := services.NewInquiryService(inquiryRepo, ragRepo, userRepo, notificationRepo)
	notifSvc := services.NewNotificationService(notificationRepo)

	inquiryCtrl := controllers.NewInquiryController(inquirySvc)
	notifCtrl := controllers.NewNotificationController(notifSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inquiries", inquiryCtrl.CreateInquiry)
	mux.HandleFunc("GET /api/student/inquiries", inquiryCtrl.GetStudentInquiries)
	mux.HandleFunc("GET /api/inquiries/{id}", inquiryCtrl.GetInquiryDetail)
	mux.HandleFunc("GET /api/advisor/inquiries", inquiryCtrl.GetTeacherInquiries)
	mux.HandleFunc("POST /api/advisor/inquiries/{id}/answer", inquiryCtrl.AnswerInquiry)
	mux.HandleFunc("POST /api/advisor/inquiries/{id}/expire", inquiryCtrl.ExpireInquiry)

	mux.HandleFunc("GET /api/student/notifications", notifCtrl.ListNotifications)
	mux.HandleFunc("PATCH /api/student/notifications/{id}/read", notifCtrl.MarkAsRead)
	mux.HandleFunc("GET /api/student/notifications/unread-count", notifCtrl.GetUnreadCount)

	return inquiryCtrl, notifCtrl, mux, studentID, teacherID
}

func withUserContext(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middlewares.UserIDContextKey, userID)
	return r.WithContext(ctx)
}

func TestInquiryController_EndToEnd(t *testing.T) {
	_, _, mux, studentID, teacherID := setupInquiryTest(t)

	createBody, _ := json.Marshal(request.CreateInquiryRequest{
		StudentName:  "Test Student",
		StudentCode:  "A35999",
		StudentClass: "CNTT01",
		Question:     "Cac buoc dang ky bao luu hoc tap duoc quy dinh nhu the nao?",
		Context:      "Sinh vien can nghi 1 ky vi ly do suc khoe",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/inquiries", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, studentID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp struct {
		Status bool                    `json:"status"`
		Data   response.InquiryResponse `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &createResp)
	require.NoError(t, err)
	require.True(t, createResp.Status)
	inquiryID := createResp.Data.ID
	require.NotEmpty(t, inquiryID)

	req = httptest.NewRequest(http.MethodGet, "/api/student/inquiries", nil)
	req = withUserContext(req, studentID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	answerBody, _ := json.Marshal(request.AnswerInquiryRequest{
		Reply: "Sinh vien nop don xin bao luu tai Phong Dao tao Nha T kem theo giay xac nhan y te.",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/advisor/inquiries/"+inquiryID+"/answer", bytes.NewReader(answerBody))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, teacherID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var answerResp struct {
		Status bool                    `json:"status"`
		Data   response.InquiryResponse `json:"data"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &answerResp)
	require.NoError(t, err)
	require.Equal(t, string(models.InquiryStatusAnswered), answerResp.Data.Status)
	require.NotEmpty(t, answerResp.Data.KnowledgeChunkID)

	req = httptest.NewRequest(http.MethodGet, "/api/student/notifications/unread-count", nil)
	req = withUserContext(req, studentID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	expireBody, _ := json.Marshal(request.ExpireInquiryRequest{
		Reason: "Quy dinh bao luu da cap nhat sang he thong online",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/advisor/inquiries/"+inquiryID+"/expire", bytes.NewReader(expireBody))
	req.Header.Set("Content-Type", "application/json")
	req = withUserContext(req, teacherID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/inquiries/"+inquiryID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var getResp struct {
		Status bool                    `json:"status"`
		Data   response.InquiryResponse `json:"data"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &getResp)
	require.NoError(t, err)
	require.Equal(t, string(models.InquiryStatusExpired), getResp.Data.Status)
	require.True(t, getResp.Data.IsExpired)
}
