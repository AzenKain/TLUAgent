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
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

// TestTranscriptController verifies transcript endpoints, authentication enforcement, and batch persistence.
func TestTranscriptController(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)
	defer db.Close()

	err = database.ApplySchema(db)
	require.NoError(t, err)

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	transcriptRepo := repositories.NewTranscriptRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	transcriptSvc := services.NewTranscriptService(transcriptRepo, memoryRepo, nil)
	ctrl := controllers.NewTranscriptController(transcriptSvc)

	testUser, err := userRepo.CreateUser(context.Background(), sqlc.CreateUserParams{
		ID:           "student-test-ctrl",
		Email:        "student_ctrl@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Đỗ Duy Khánh", Valid: true},
		StudentCode:  sql.NullString{String: "A44519", Valid: true},
		AuthProvider: "local",
	})
	require.NoError(t, err)
	userID := testUser.ID

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/transcript/my", ctrl.HandleGetMyTranscript)
	mux.HandleFunc("POST /api/transcript/extract-images", ctrl.HandleExtractImages)
	mux.HandleFunc("POST /api/transcript/batch-save", ctrl.HandleBatchSave)

	t.Run("UnauthorizedReturns401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/transcript/my", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("BatchSaveAndGetMyTranscript", func(t *testing.T) {
		saveReq := request.BatchSaveTranscriptRequest{
			Grades: []request.StudentCourseGradeInput{
				{
					CourseCode:  "CS100",
					CourseName:  "Tin đại cương",
					Credits:     2.0,
					Score10:     9.8,
					Score4:      4.0,
					GradeLetter: "A+",
					IsPassed:    true,
					TermTaken:   "HK01-2022",
				},
				{
					CourseCode:  "VL101",
					CourseName:  "Tiếng Việt thực hành",
					Credits:     2.0,
					Score10:     7.7,
					Score4:      3.0,
					GradeLetter: "B",
					IsPassed:    true,
					TermTaken:   "HK03-2023",
				},
			},
		}

		body, err := json.Marshal(saveReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/transcript/batch-save", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middlewares.UserIDContextKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)

		var saveResp response.CommonResponse
		err = json.Unmarshal(rr.Body.Bytes(), &saveResp)
		require.NoError(t, err)
		require.True(t, saveResp.Status)

		getReq := httptest.NewRequest(http.MethodGet, "/api/transcript/my", nil)
		getCtx := context.WithValue(getReq.Context(), middlewares.UserIDContextKey, userID)
		getReq = getReq.WithContext(getCtx)

		getRr := httptest.NewRecorder()
		mux.ServeHTTP(getRr, getReq)
		require.Equal(t, http.StatusOK, getRr.Code)

		var getResp response.CommonResponse
		err = json.Unmarshal(getRr.Body.Bytes(), &getResp)
		require.NoError(t, err)
		require.True(t, getResp.Status)
	})

	t.Run("ExtractImagesEmptyPayloadReturns400", func(t *testing.T) {
		emptyReq := request.ExtractTranscriptImagesRequest{
			Images: []string{},
		}
		body, _ := json.Marshal(emptyReq)

		req := httptest.NewRequest(http.MethodPost, "/api/transcript/extract-images", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), middlewares.UserIDContextKey, userID)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
