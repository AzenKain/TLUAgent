package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
)

// RAGService provides hybrid retrieval, knowledge graph traversal, and document ingestion.
type RAGService interface {
	SetLLMManager(llmManager *llm.Manager)
	Search(ctx context.Context, params models.RAGSearchParams, queryEmbedding []float32) ([]*models.RAGSearchResultItem, error)
	IngestDocument(ctx context.Context, doc *models.Document, chunks []*models.DocumentChunk, nodes []*models.KGNode, edges []*models.KGEdge) error
	IngestFromIndexJSONL(ctx context.Context, jsonlPath string, baseDataDir string, embedder llm.Embedder) (int, error)
	GetStats(ctx context.Context) (map[string]int, error)
	GetDocumentDetail(ctx context.Context, id string) (*models.DocumentDetail, error)
	UpdateDocumentStatus(ctx context.Context, id string, req *request.UpdateDocumentStatusRequest) error
	GetKnowledgeGraph(ctx context.Context) (*models.KnowledgeGraphData, error)
	CreateOrUpdateKGEdge(ctx context.Context, edge *models.KGEdge) (*models.KGEdge, error)
	DeleteKGEdge(ctx context.Context, id string) error
	GetKnowledgeStats(ctx context.Context) (*models.KnowledgeStats, error)
	ListAuditLogs(ctx context.Context, dto *request.ListAuditLogsDto) ([]*models.SupersessionAuditLog, int, error)
	SetInquiryService(inquiryService InquiryService)
}

type ragService struct {
	ragRepo        repositories.RAGRepository
	llmManager     *llm.Manager
	inquiryService InquiryService
}

// NewRAGService creates an instance of RAGService.
func NewRAGService(ragRepo repositories.RAGRepository) RAGService {
	return &ragService{ragRepo: ragRepo}
}

func (s *ragService) SetInquiryService(inquiryService InquiryService) {
	s.inquiryService = inquiryService
}

func (s *ragService) SetLLMManager(llmManager *llm.Manager) {
	s.llmManager = llmManager
}

func (s *ragService) Search(ctx context.Context, params models.RAGSearchParams, queryEmbedding []float32) ([]*models.RAGSearchResultItem, error) {
	if chk := guardrail.CheckQuery(params.Query); chk.Decision == guardrail.DecisionBlockedInjection {
		return []*models.RAGSearchResultItem{}, nil
	}

	if params.TopK <= 0 {
		params.TopK = 5
	}

	if len(queryEmbedding) == 0 && s.llmManager != nil && strings.TrimSpace(params.Query) != "" {
		if embs, err := s.llmManager.ExecuteEmbedding(ctx, []string{params.Query}); err == nil && len(embs) > 0 {
			queryEmbedding = embs[0]
		}
	}

	candidates, err := s.ragRepo.QueryActiveCandidates(ctx, params.StudentCohort, params.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to query active candidates: %w", err)
	}

	supersededDocs, err := s.ragRepo.FindSupersededDocIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse superseded graph: %w", err)
	}

	queryTokens := tokenizeSearchQuery(params.Query)
	tokenCount := float32(len(queryTokens))

	var validResults []*models.RAGSearchResultItem
	docCache := make(map[string]*models.Document)

	for _, chunk := range candidates {
		if supersededDocs[chunk.DocumentID] {
			continue
		}

		var vecSim float32
		if len(queryEmbedding) > 0 && len(chunk.Embedding) > 0 {
			vecSim = models.CosineSimilarity(queryEmbedding, chunk.Embedding)
		}

		var bm25Score float32
		if tokenCount > 0 {
			contentLower := strings.ToLower(chunk.Content)
			var matches float32
			for _, token := range queryTokens {
				if strings.Contains(contentLower, token) {
					matches++
				}
			}
			bm25Score = matches / tokenCount
		}

		doc, ok := docCache[chunk.DocumentID]
		if !ok {
			d, err := s.ragRepo.GetDocumentByID(ctx, chunk.DocumentID)
			if err == nil {
				docCache[chunk.DocumentID] = d
				doc = d
			}
		}

		var tierMult float32 = 1.0
		var recencyScore float32 = 0.1
		if doc != nil {
			switch doc.ValidityTier {
			case "CURRENT_OPERATIONAL":
				tierMult = 1.0
			case "CORE_PERMANENT":
				tierMult = 0.95
			case "HISTORICAL_ARCHIVE":
				tierMult = 0.4
			default:
				tierMult = 0.8
			}

			pubDateStr := strings.TrimSpace(doc.PublishDate)
			if len(pubDateStr) >= 10 {
				pubDateStr = pubDateStr[:10]
			}
			if t, err := time.Parse("2006-01-02", pubDateStr); err == nil {
				diffDays := float64(time.Since(t).Hours() / 24)
				if diffDays < 0 {
					diffDays = 0
				}
				decay := math.Exp(-diffDays / 365.0)
				recencyScore = float32(math.Max(0.05, math.Min(1.0, decay)))
			} else {
				if strings.HasPrefix(doc.PublishDate, "2026") {
					recencyScore = 1.0
				} else if strings.HasPrefix(doc.PublishDate, "2025") {
					recencyScore = 0.6
				} else if strings.HasPrefix(doc.PublishDate, "2024") {
					recencyScore = 0.3
				} else {
					recencyScore = 0.1
				}
			}
		}

		prioScore := float32(chunk.PriorityLevel) / 100.0
		finalScore := ((0.5 * vecSim) + (0.25 * bm25Score) + (0.15 * recencyScore) + (0.1 * prioScore)) * tierMult

		if params.MinScore > 0 && finalScore < params.MinScore {
			continue
		}

		item := &models.RAGSearchResultItem{
			ChunkID:          chunk.ID,
			DocumentID:       chunk.DocumentID,
			Domain:           chunk.Domain,
			ApplicableCohort: chunk.ApplicableCohort,
			Content:          chunk.Content,
			VectorSim:        vecSim,
			BM25Score:        bm25Score,
			PriorityLevel:    chunk.PriorityLevel,
			FinalScore:       finalScore,
		}

		if doc != nil {
			item.DocTitle = doc.Title
			item.DocType = doc.Type
		}

		validResults = append(validResults, item)
	}

	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].FinalScore > validResults[j].FinalScore
	})

	if s.llmManager != nil && len(validResults) > 0 && strings.TrimSpace(params.Query) != "" {
		poolSize := params.TopK * 3
		if poolSize > len(validResults) {
			poolSize = len(validResults)
		}
		if poolSize > 15 {
			poolSize = 15
		}

		candidatePool := validResults[:poolSize]
		var candidateTexts []string
		for _, item := range candidatePool {
			candidateTexts = append(candidateTexts, item.Content)
		}

		rerankResults, err := s.llmManager.ExecuteRerank(ctx, params.Query, candidateTexts, &params.TopK)
		if err == nil && len(rerankResults) > 0 {
			var reorderedResults []*models.RAGSearchResultItem
			for _, r := range rerankResults {
				if r.Index >= 0 && r.Index < len(candidatePool) {
					item := candidatePool[r.Index]
					item.FinalScore = float32(r.RelevanceScore)
					reorderedResults = append(reorderedResults, item)
				}
			}
			if len(reorderedResults) > 0 {
				validResults = reorderedResults
			}
		}
	}

	if len(validResults) > params.TopK {
		validResults = validResults[:params.TopK]
	}

	return validResults, nil
}

func (s *ragService) IngestDocument(ctx context.Context, doc *models.Document, chunks []*models.DocumentChunk, nodes []*models.KGNode, edges []*models.KGEdge) error {
	if _, err := s.ragRepo.UpsertDocument(ctx, doc); err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	if len(chunks) > 0 {
		if err := s.ragRepo.UpsertChunksBatch(ctx, chunks); err != nil {
			return fmt.Errorf("insert chunks batch: %w", err)
		}
	}

	for _, n := range nodes {
		if _, err := s.ragRepo.UpsertKGNode(ctx, n); err != nil {
			return fmt.Errorf("insert kg node: %w", err)
		}
	}

	for _, e := range edges {
		if _, err := s.ragRepo.UpsertKGEdge(ctx, e); err != nil {
			return fmt.Errorf("insert kg edge: %w", err)
		}
	}

	if doc.Status == "ACTIVE" && s.inquiryService != nil {
		_, _ = s.inquiryService.CheckAndExpireOutdatedQA(ctx, doc)
	}

	return nil
}

type processedIndexEntry struct {
	DocID            string `json:"doc_id"`
	NodeID           string `json:"node_id"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	PublishDate      string `json:"publish_date"`
	Department       string `json:"department"`
	URL              string `json:"url"`
	MDPath           string `json:"md_path"`
	CharCount        int    `json:"char_count"`
	Domain           string `json:"domain"`
	ValidityTier     string `json:"validity_tier"`
	AcademicYear     string `json:"academic_year"`
	Semester         string `json:"semester"`
	ApplicableCohort string `json:"applicable_cohort"`
	DepartmentCode   string `json:"department_code"`
	OcrText          string `json:"ocr_text,omitempty"`
}

func (s *ragService) IngestFromIndexJSONL(ctx context.Context, jsonlPath string, baseDataDir string, embedder llm.Embedder) (int, error) {
	file, err := os.Open(jsonlPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	count := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry processedIndexEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		doc := &models.Document{
			ID:               entry.DocID,
			DocCode:          entry.NodeID,
			Type:             entry.Type,
			Title:            entry.Title,
			Domain:           entry.Domain,
			ValidityTier:     entry.ValidityTier,
			Status:           "ACTIVE",
			DepartmentCode:   entry.DepartmentCode,
			PublishDate:      entry.PublishDate,
			AcademicYear:     entry.AcademicYear,
			Semester:         entry.Semester,
			ApplicableCohort: entry.ApplicableCohort,
			PriorityLevel:    resolvePriorityLevel(entry.ValidityTier, entry.Type),
			FilePath:         entry.MDPath,
			CharCount:        entry.CharCount,
		}

		docNode := &models.KGNode{
			ID:       "doc:" + doc.ID,
			NodeType: "DOCUMENT",
			Name:     doc.Title,
			Code:     doc.DocCode,
			Status:   "ACTIVE",
		}

		nodes := []*models.KGNode{docNode}
		var edges []*models.KGEdge

		if doc.ApplicableCohort != "" && doc.ApplicableCohort != "ALL" {
			cohorts := strings.Split(doc.ApplicableCohort, ",")
			for _, c := range cohorts {
				c = strings.TrimSpace(c)
				if c == "" {
					continue
				}
				cohortNode := &models.KGNode{
					ID:       "cohort:" + c,
					NodeType: "COHORT",
					Name:     "Cohort " + c,
					Code:     c,
					Status:   "ACTIVE",
				}
				nodes = append(nodes, cohortNode)
				edges = append(edges, &models.KGEdge{
					ID:           fmt.Sprintf("edge:cohort:%s:%s", doc.ID, c),
					SourceNodeID: docNode.ID,
					TargetNodeID: cohortNode.ID,
					RelationType: "APPLIES_TO_COHORT",
					Weight:       1.0,
					Status:       "ACTIVE",
				})
			}
		}

		if doc.Domain != "" {
			topicNode := &models.KGNode{
				ID:       "topic:" + doc.Domain,
				NodeType: "TOPIC",
				Name:     doc.Domain,
				Code:     doc.Domain,
				Status:   "ACTIVE",
			}
			nodes = append(nodes, topicNode)
			edges = append(edges, &models.KGEdge{
				ID:           fmt.Sprintf("edge:topic:%s:%s", doc.ID, doc.Domain),
				SourceNodeID: docNode.ID,
				TargetNodeID: topicNode.ID,
				RelationType: "COVERS_TOPIC",
				Weight:       1.0,
				Status:       "ACTIVE",
			})
		}

		var chunks []*models.DocumentChunk
		contentToChunk := entry.OcrText
		if contentToChunk == "" && baseDataDir != "" && entry.MDPath != "" {
			fullPath := filepath.Join(baseDataDir, entry.MDPath)
			if data, err := os.ReadFile(fullPath); err == nil {
				contentToChunk = string(data)
			}
		}

		if contentToChunk != "" {
			chunkTexts := splitIntoChunks(contentToChunk, 1500, 200)
			var embeddings [][]float32
			if embedder != nil && len(chunkTexts) > 0 {
				embeddings, _ = embedder.Embed(ctx, chunkTexts)
			}

			for idx, text := range chunkTexts {
				var emb []float32
				if idx < len(embeddings) {
					emb = embeddings[idx]
				}
				chunk := &models.DocumentChunk{
					ID:               fmt.Sprintf("chk_%s_%d", doc.ID, idx),
					DocumentID:       doc.ID,
					ChunkIndex:       idx,
					ChunkType:        "CONTENT_CHUNK",
					Content:          text,
					Status:           "ACTIVE",
					Domain:           doc.Domain,
					ApplicableCohort: doc.ApplicableCohort,
					AcademicYear:     doc.AcademicYear,
					PriorityLevel:    doc.PriorityLevel,
					Embedding:        emb,
				}
				chunks = append(chunks, chunk)
			}
		}

		if err := s.IngestDocument(ctx, doc, chunks, nodes, edges); err != nil {
			continue
		}
		count++
	}

	return count, scanner.Err()
}

func (s *ragService) GetStats(ctx context.Context) (map[string]int, error) {
	docs, chunks, nodes, edges, err := s.ragRepo.CountStats(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]int{
		"documents": docs,
		"chunks":    chunks,
		"kg_nodes":  nodes,
		"kg_edges":  edges,
	}, nil
}

func tokenizeSearchQuery(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	var tokens []string
	for _, f := range fields {
		f = strings.Trim(f, ",.?!;:\"'()[]{}")
		if len(f) > 2 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

func resolvePriorityLevel(validityTier, docType string) int {
	switch validityTier {
	case "EMERGENCY_OVERRIDE":
		return 90
	case "CORE_PERMANENT":
		return 75
	case "SEMESTER_DYNAMIC":
		return 60
	}
	if docType == "QUY_CHE" {
		return 80
	}
	return 50
}

func splitIntoChunks(text string, maxChunkLen, overlap int) []string {
	if len(text) <= maxChunkLen {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	var current strings.Builder

	for _, line := range lines {
		if current.Len()+len(line)+1 > maxChunkLen && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func (s *ragService) GetDocumentDetail(ctx context.Context, id string) (*models.DocumentDetail, error) {
	doc, err := s.ragRepo.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	chunks, err := s.ragRepo.GetChunksByDocumentID(ctx, id)
	if err != nil {
		return nil, err
	}

	var supersededBy *models.Document
	if doc.SupersededByID != "" {
		if sb, err := s.ragRepo.GetDocumentByID(ctx, doc.SupersededByID); err == nil {
			supersededBy = sb
		}
	}

	supersedes, _ := s.ragRepo.ListSupersededDocuments(ctx, id)
	auditLogs, _ := s.ragRepo.ListAuditLogsByDocID(ctx, id)

	return &models.DocumentDetail{
		Document:     doc,
		SupersededBy: supersededBy,
		Supersedes:   supersedes,
		Chunks:       chunks,
		AuditLogs:    auditLogs,
	}, nil
}

func (s *ragService) UpdateDocumentStatus(ctx context.Context, id string, req *request.UpdateDocumentStatusRequest) error {
	doc, err := s.ragRepo.GetDocumentByID(ctx, id)
	if err != nil {
		return err
	}

	status := "ACTIVE"
	supersededByID := ""
	reason := ""
	if req != nil {
		status = req.Status
		supersededByID = req.SupersededByID
		reason = req.Reason
	}

	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if normalizedStatus != "ACTIVE" && normalizedStatus != "SUPERSEDED" && normalizedStatus != "DRAFT" && normalizedStatus != "ARCHIVED" {
		normalizedStatus = "ACTIVE"
	}

	if err := s.ragRepo.UpdateDocumentStatus(ctx, id, normalizedStatus, supersededByID); err != nil {
		return err
	}

	if supersededByID != "" {
		edgeID := fmt.Sprintf("edge_sup_%s_%s", supersededByID, id)
		_, _ = s.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
			ID:           edgeID,
			SourceNodeID: "doc:" + supersededByID,
			TargetNodeID: "doc:" + id,
			RelationType: "SUPERSEDES",
			Weight:       1.0,
			Status:       "ACTIVE",
		})
	}

	if reason == "" {
		reason = fmt.Sprintf("Document status updated to %s", normalizedStatus)
	}

	activeID := id
	supersededID := supersededByID
	if normalizedStatus == "SUPERSEDED" && supersededByID != "" {
		activeID = supersededByID
		supersededID = id
	}
	_ = s.ragRepo.RecordSupersessionLog(ctx, activeID, supersededID, "ADMIN_OVERRIDE", reason)
	if normalizedStatus == "ACTIVE" && s.inquiryService != nil {
		_, _ = s.inquiryService.CheckAndExpireOutdatedQA(ctx, doc)
	}
	return nil
}

func (s *ragService) GetKnowledgeGraph(ctx context.Context) (*models.KnowledgeGraphData, error) {
	nodes, err := s.ragRepo.ListAllKGNodes(ctx)
	if err != nil {
		return nil, err
	}
	edges, err := s.ragRepo.ListAllKGEdges(ctx)
	if err != nil {
		return nil, err
	}
	return &models.KnowledgeGraphData{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (s *ragService) CreateOrUpdateKGEdge(ctx context.Context, edge *models.KGEdge) (*models.KGEdge, error) {
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("edge_%d", time.Now().UnixNano())
	}
	if edge.Status == "" {
		edge.Status = "ACTIVE"
	}
	return s.ragRepo.UpsertKGEdge(ctx, edge)
}

func (s *ragService) DeleteKGEdge(ctx context.Context, id string) error {
	return s.ragRepo.DeleteKGEdge(ctx, id)
}

func (s *ragService) GetKnowledgeStats(ctx context.Context) (*models.KnowledgeStats, error) {
	return s.ragRepo.GetKnowledgeStats(ctx)
}

func (s *ragService) ListAuditLogs(ctx context.Context, dto *request.ListAuditLogsDto) ([]*models.SupersessionAuditLog, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}
	return s.ragRepo.ListAllAuditLogs(ctx, limit, offset)
}
