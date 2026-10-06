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
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
)

func setupRAGTestController(t *testing.T) (*controllers.RAGController, repositories.RAGRepository, *http.ServeMux, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	err = database.ApplySchema(db)
	require.NoError(t, err)

	ramCache := cache.NewRamCache()
	ragRepo := repositories.NewRAGRepository(db, ramCache)
	ragService := services.NewRAGService(ragRepo)
	ragCtrl := controllers.NewRAGController(ragService, ragRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/admin/documents", ragCtrl.HandleGetDocuments)
	mux.HandleFunc("GET /api/admin/documents/{id}", ragCtrl.HandleGetDocumentDetail)
	mux.HandleFunc("PATCH /api/admin/documents/{id}/status", ragCtrl.HandleUpdateDocumentStatus)
	mux.HandleFunc("GET /api/admin/knowledge-graph", ragCtrl.HandleGetKnowledgeGraph)
	mux.HandleFunc("POST /api/admin/knowledge-graph/edges", ragCtrl.HandleUpsertKGEdge)
	mux.HandleFunc("DELETE /api/admin/knowledge-graph/edges/{id}", ragCtrl.HandleDeleteKGEdge)
	mux.HandleFunc("GET /api/admin/rag/stats", ragCtrl.HandleGetStats)
	mux.HandleFunc("GET /api/admin/rag/audit-logs", ragCtrl.HandleGetAuditLogs)

	return ragCtrl, ragRepo, mux, db
}

func TestRAGController_DocumentsAndLifecycle(t *testing.T) {
	_, repo, mux, db := setupRAGTestController(t)
	defer db.Close()
	ctx := context.Background()

	doc1 := &models.Document{
		ID:               "doc_k64",
		DocCode:          "100/QD-TLU",
		Type:             "QUY_CHE",
		Title:            "Quy chế đào tạo tín chỉ 2021",
		Domain:           "DAO_TAO",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "P_DAO_TAO",
		PublishDate:      "2021-09-01",
		AcademicYear:     "2021-2022",
		Semester:         "1",
		ApplicableCohort: "K64",
		PriorityLevel:    75,
		FilePath:         "data/quyche2021.pdf",
		CharCount:        5000,
	}
	_, err := repo.UpsertDocument(ctx, doc1)
	require.NoError(t, err)

	doc2 := &models.Document{
		ID:               "doc_k65",
		DocCode:          "200/QD-TLU",
		Type:             "QUY_CHE",
		Title:            "Quy chế đào tạo tín chỉ sửa đổi 2024",
		Domain:           "DAO_TAO",
		ValidityTier:     "CORE_PERMANENT",
		Status:           "ACTIVE",
		DepartmentCode:   "P_DAO_TAO",
		PublishDate:      "2024-08-15",
		AcademicYear:     "2024-2025",
		Semester:         "1",
		ApplicableCohort: "K65",
		PriorityLevel:    80,
		FilePath:         "data/quyche2024.pdf",
		CharCount:        6000,
	}
	_, err = repo.UpsertDocument(ctx, doc2)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/documents?domain=DAO_TAO&page=1&limit=10", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Status bool `json:"status"`
		Data   struct {
			Documents []*models.Document `json:"documents"`
			Total     int                `json:"total"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	require.True(t, listResp.Status)
	require.Equal(t, 2, listResp.Data.Total)
	require.Len(t, listResp.Data.Documents, 2)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/documents/doc_k64", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var detailResp struct {
		Status bool                  `json:"status"`
		Data   models.DocumentDetail `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &detailResp)
	require.NoError(t, err)
	require.True(t, detailResp.Status)
	require.Equal(t, "doc_k64", detailResp.Data.Document.ID)

	patchBody, _ := json.Marshal(request.UpdateDocumentStatusRequest{
		Status:         "SUPERSEDED",
		SupersededByID: "doc_k65",
		Reason:         "Replaced by updated academic regulation 2024",
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/admin/documents/doc_k64/status", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updatedDoc, err := repo.GetDocumentByID(ctx, "doc_k64")
	require.NoError(t, err)
	require.Equal(t, "SUPERSEDED", updatedDoc.Status)
	require.Equal(t, "doc_k65", updatedDoc.SupersededByID)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/rag/audit-logs?page=1&limit=10", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var auditResp struct {
		Status bool `json:"status"`
		Data   struct {
			Logs  []*models.SupersessionAuditLog `json:"logs"`
			Total int                            `json:"total"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &auditResp)
	require.NoError(t, err)
	require.True(t, auditResp.Status)
	require.GreaterOrEqual(t, auditResp.Data.Total, 1)
}

func TestRAGController_KnowledgeGraphAndStats(t *testing.T) {
	_, repo, mux, db := setupRAGTestController(t)
	defer db.Close()
	ctx := context.Background()

	node1 := &models.KGNode{
		ID:       "node_reg_1",
		NodeType: "REGULATION",
		Name:     "Quy che tin chi",
		Status:   "ACTIVE",
	}
	_, err := repo.UpsertKGNode(ctx, node1)
	require.NoError(t, err)

	node2 := &models.KGNode{
		ID:       "node_cond_1",
		NodeType: "CONDITION",
		Name:     "So tin chi toi thieu",
		Status:   "ACTIVE",
	}
	_, err = repo.UpsertKGNode(ctx, node2)
	require.NoError(t, err)

	edgeBody, _ := json.Marshal(request.KGEdgeRequest{
		ID:           "edge_1",
		SourceNodeID: "node_reg_1",
		TargetNodeID: "node_cond_1",
		RelationType: "REQUIRES",
		Weight:       1.0,
		Status:       "ACTIVE",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/knowledge-graph/edges", bytes.NewReader(edgeBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/knowledge-graph", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var graphResp struct {
		Status bool                      `json:"status"`
		Data   models.KnowledgeGraphData `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &graphResp)
	require.NoError(t, err)
	require.True(t, graphResp.Status)
	require.Len(t, graphResp.Data.Nodes, 2)
	require.Len(t, graphResp.Data.Edges, 1)

	req = httptest.NewRequest(http.MethodGet, "/api/admin/rag/stats", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var statsResp struct {
		Status bool                  `json:"status"`
		Data   models.KnowledgeStats `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &statsResp)
	require.NoError(t, err)
	require.True(t, statsResp.Status)
	require.Equal(t, 2, statsResp.Data.TotalNodes)
	require.Equal(t, 1, statsResp.Data.TotalEdges)

	req = httptest.NewRequest(http.MethodDelete, "/api/admin/knowledge-graph/edges/edge_1", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	edges, err := repo.ListAllKGEdges(ctx)
	require.NoError(t, err)
	require.Empty(t, edges)
}
