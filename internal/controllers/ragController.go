package controllers

import (
	"net/http"
	"strings"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/validator"
)

// RAGController coordinates institutional search, knowledge graph auditing, and index synchronization.
type RAGController struct {
	ragService services.RAGService
	ragRepo    repositories.RAGRepository
}

// NewRAGController creates a new RAGController instance.
func NewRAGController(ragService services.RAGService, ragRepo repositories.RAGRepository) *RAGController {
	return &RAGController{
		ragService: ragService,
		ragRepo:    ragRepo,
	}
}

// HandleSearch executes hybrid RAG search against active institutional documents.
func (c *RAGController) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.RAGSearchRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if chk := guardrail.CheckQuery(req.Query); chk.Decision != guardrail.DecisionAllow {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: chk.Response,
		})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	params := models.RAGSearchParams{
		Query:         req.Query,
		StudentCohort: req.StudentCohort,
		Domain:        req.Domain,
		TopK:          req.TopK,
		MinScore:      req.MinScore,
	}

	results, err := c.ragService.Search(ctx, params, nil)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to perform knowledge search",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   results,
	})
}

// HandleGetDocuments lists registered institutional documents with optional filtering.
func (c *RAGController) HandleGetDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListDocumentsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	docs, total, err := c.ragRepo.ListDocuments(ctx, &dto)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to list documents",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"documents": docs,
			"total":     total,
			"page":      dto.GetPage(),
			"limit":     dto.GetLimit(20),
		},
	})
}

// HandleGetDocumentDetail returns full metadata, chunks, and supersession links for a single document.
func (c *RAGController) HandleGetDocumentDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: "Document ID is required",
		})
		return
	}

	detail, err := c.ragService.GetDocumentDetail(ctx, id)
	if err != nil {
		writeJSONResponse(w, http.StatusNotFound, response.CommonResponse{
			Status:  false,
			Message: "Document not found",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   detail,
	})
}

// HandleUpdateDocumentStatus allows administrators to adjust document lifecycle status and supersession.
func (c *RAGController) HandleUpdateDocumentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: "Document ID is required",
		})
		return
	}

	var req request.UpdateDocumentStatusRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	if err := c.ragService.UpdateDocumentStatus(ctx, id, &req); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to update document status: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Document status updated successfully",
	})
}

// HandleGetKnowledgeGraph returns full nodes and edges data for interactive visualization.
func (c *RAGController) HandleGetKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	data, err := c.ragService.GetKnowledgeGraph(ctx)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to load knowledge graph",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   data,
	})
}

// HandleUpsertKGEdge creates or updates a relationship edge between two knowledge graph nodes.
func (c *RAGController) HandleUpsertKGEdge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var req request.KGEdgeRequest
	if err := validator.ValidateBodyDto(r, &req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	edge := &models.KGEdge{
		ID:           req.ID,
		SourceNodeID: req.SourceNodeID,
		TargetNodeID: req.TargetNodeID,
		RelationType: req.RelationType,
		Weight:       req.Weight,
		Status:       req.Status,
	}

	saved, err := c.ragService.CreateOrUpdateKGEdge(ctx, edge)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to persist knowledge edge: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Knowledge edge saved successfully",
		Data:    saved,
	})
}

// HandleDeleteKGEdge removes a specified edge from the knowledge graph.
func (c *RAGController) HandleDeleteKGEdge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status:  false,
			Message: "Edge ID is required",
		})
		return
	}

	if err := c.ragService.DeleteKGEdge(ctx, id); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to delete knowledge edge: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Knowledge edge deleted successfully",
	})
}

// HandleGetStats returns entity counts across documents, chunks, and graph structures.
func (c *RAGController) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	stats, err := c.ragService.GetKnowledgeStats(ctx)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to load knowledge statistics",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data:   stats,
	})
}

// HandleGetAuditLogs returns chronological records of document supersession events.
func (c *RAGController) HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, defaultRequestTimeout)
	defer cancel()

	var dto request.ListAuditLogsDto
	if err := validator.ValidateQueryDto(r, &dto); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, response.CommonResponse{
			Status: false,
			Errors: err,
		})
		return
	}

	logs, total, err := c.ragService.ListAuditLogs(ctx, &dto)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to list audit logs",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status: true,
		Data: map[string]any{
			"logs":  logs,
			"total": total,
			"page":  dto.GetPage(),
			"limit": dto.GetLimit(20),
		},
	})
}

// HandleSyncIndex triggers ingestion from the processed index jsonl file.
func (c *RAGController) HandleSyncIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, response.CommonResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := requestContext(r, longRequestTimeout)
	defer cancel()

	indexPath := config.GetConfigWithDefault("PROCESSED_INDEX_PATH", "../data/processed/processed_index.jsonl")
	baseDataDir := config.GetConfigWithDefault("PROCESSED_DATA_DIR", "../data/processed")

	count, err := c.ragService.IngestFromIndexJSONL(ctx, indexPath, baseDataDir, nil)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, response.CommonResponse{
			Status:  false,
			Message: "Failed to sync knowledge index: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, response.CommonResponse{
		Status:  true,
		Message: "Knowledge index synchronized successfully",
		Data: map[string]any{
			"ingested_documents": count,
		},
	})
}
