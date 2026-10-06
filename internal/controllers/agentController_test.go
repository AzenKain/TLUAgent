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
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func setupAgentTestController(t *testing.T) (*controllers.AgentController, services.AgentService, *http.ServeMux, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	err = database.ApplySchema(db)
	require.NoError(t, err)

	ramCache := cache.NewRamCache()
	agentRepo := repositories.NewAgentRepository(db, ramCache)
	agentSvc := services.NewAgentService(agentRepo)

	err = agentSvc.SeedDefaultsIfEmpty(context.Background())
	require.NoError(t, err)

	ctrl := controllers.NewAgentController(agentSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/admin/agent/prompts", ctrl.ListPrompts)
	mux.HandleFunc("GET /api/admin/agent/prompts/{id}", ctrl.GetPrompt)
	mux.HandleFunc("PUT /api/admin/agent/prompts/{id}", ctrl.UpdatePrompt)
	mux.HandleFunc("POST /api/admin/agent/prompts/{id}/reset", ctrl.ResetPrompt)

	mux.HandleFunc("GET /api/admin/agent/skills", ctrl.ListSkills)
	mux.HandleFunc("GET /api/admin/agent/skills/{id}", ctrl.GetSkill)
	mux.HandleFunc("POST /api/admin/agent/skills", ctrl.CreateSkill)
	mux.HandleFunc("PUT /api/admin/agent/skills/{id}", ctrl.UpdateSkill)
	mux.HandleFunc("PATCH /api/admin/agent/skills/{id}/toggle", ctrl.ToggleSkill)
	mux.HandleFunc("DELETE /api/admin/agent/skills/{id}", ctrl.DeleteSkill)
	mux.HandleFunc("POST /api/admin/agent/skills/{id}/reset", ctrl.ResetSkill)

	mux.HandleFunc("POST /api/admin/agent/preview", ctrl.PreviewPrompt)
	mux.HandleFunc("GET /api/admin/agent/compactor", ctrl.GetCompactorSettings)
	mux.HandleFunc("PUT /api/admin/agent/compactor", ctrl.UpdateCompactorSettings)

	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	agentSvc.SetSettingsRepository(settingsRepo)

	return ctrl, agentSvc, mux, db
}

func TestAgentController_PromptsCRUD(t *testing.T) {
	_, _, mux, db := setupAgentTestController(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/agent/prompts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var listResp response.CommonResponse
	err := json.Unmarshal(rec.Body.Bytes(), &listResp)
	require.NoError(t, err)
	require.True(t, listResp.Status)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/agent/prompts/soul", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	updatePayload := request.UpdateAgentPromptRequest{
		Title:   "Updated Soul Title",
		Content: "Updated Soul Content",
	}
	body, _ := json.Marshal(updatePayload)
	req = httptest.NewRequest(http.MethodPut, "/api/admin/agent/prompts/soul", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/admin/agent/prompts/soul/reset", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAgentController_SkillsCRUD(t *testing.T) {
	_, _, mux, db := setupAgentTestController(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/agent/skills", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	newSkill := request.CreateAgentSkillRequest{
		ID:          "new_test_skill",
		Name:        "Test Skill",
		Description: "Skill description",
		Content:     "Skill instructions markdown",
	}
	body, _ := json.Marshal(newSkill)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/agent/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	togglePayload := request.ToggleAgentSkillRequest{
		IsEnabled: false,
	}
	body, _ = json.Marshal(togglePayload)
	req = httptest.NewRequest(http.MethodPatch, "/api/admin/agent/skills/new_test_skill/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/admin/agent/skills/new_test_skill", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	previewPayload := request.PreviewAgentPromptRequest{
		StudentCohort:    "K37",
		IncludeSampleRAG: true,
	}
	body, _ = json.Marshal(previewPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/agent/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/agent/compactor", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var compactorResp response.CommonResponse
	err := json.Unmarshal(rec.Body.Bytes(), &compactorResp)
	require.NoError(t, err)
	require.True(t, compactorResp.Status)

	updateCompactorBody, _ := json.Marshal(request.UpdateCompactorSettingsRequest{
		MaxContextTokens:      128000,
		CompactThresholdRatio: 0.85,
		KeepRecentTurns:       12,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/admin/agent/compactor", bytes.NewReader(updateCompactorBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

