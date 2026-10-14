package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
)

// RAGService provides hybrid retrieval, knowledge graph traversal, and document ingestion.
type RAGService interface {
	Search(ctx context.Context, params models.RAGSearchParams, queryEmbedding []float32) ([]*response.RAGSearchResultItemResponse, error)
	IngestDocument(ctx context.Context, doc *models.Document, chunks []*models.DocumentChunk, nodes []*models.KGNode, edges []*models.KGEdge) error
	IngestFromIndexJSONL(ctx context.Context, jsonlPath string, baseDataDir string, embedder llm.Embedder, onProgress func(cur, total int64)) (int, error)
	GetDocumentDetail(ctx context.Context, id string) (*response.DocumentDetailResponse, error)
	GetDocumentByURL(ctx context.Context, rawURL string) (*response.DocumentDetailResponse, error)
	DocumentExistsByURL(ctx context.Context, rawURL string) bool
	UpdateDocumentStatus(ctx context.Context, id string, req *request.UpdateDocumentStatusRequest) error
	GetKnowledgeGraph(ctx context.Context) (*response.KnowledgeGraphDataResponse, error)
	CreateOrUpdateKGEdge(ctx context.Context, edge *models.KGEdge) (*response.KGEdgeResponse, error)
	DeleteKGEdge(ctx context.Context, id string) error
	GetKnowledgeStats(ctx context.Context) (*response.KnowledgeStatsResponse, error)
	ListDocuments(ctx context.Context, dto *request.ListDocumentsDto) ([]*response.DocumentResponse, int, error)
	ListAllRegulationsChronological(ctx context.Context) ([]*response.DocumentResponse, error)
	BuildDocumentFilter(ctx context.Context) (func(id, rawURL, title string) bool, error)
	ListAuditLogs(ctx context.Context, dto *request.ListAuditLogsDto) ([]*response.SupersessionAuditLogResponse, int, error)
	WipeCorpus(ctx context.Context) error
}

type ragService struct {
	ragRepo        repositories.RAGRepository
	llmManager     *llm.Manager
	inquiryService InquiryService
	c              cache.Cache
	sfg            *singleflight.Group
	embSfg         *singleflight.Group
}

// NewRAGService creates an instance of RAGService.
func NewRAGService(
	ragRepo repositories.RAGRepository,
	c cache.Cache,
	llmManager *llm.Manager,
	inquiryService InquiryService,
) RAGService {
	return &ragService{
		ragRepo:        ragRepo,
		llmManager:     llmManager,
		inquiryService: inquiryService,
		c:              c,
		sfg:            &singleflight.Group{},
		embSfg:         &singleflight.Group{},
	}
}

func sanitizeFTS5Query(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	words := strings.Fields(q)
	var cleanWords []string
	for _, word := range words {
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, word)
		if len([]rune(clean)) >= 2 {
			cleanWords = append(cleanWords, clean)
		}
	}
	if len(cleanWords) == 0 {
		return ""
	}

	var orTerms []string
	for _, w := range cleanWords {
		if !vietnameseStopWords[strings.ToLower(w)] {
			orTerms = append(orTerms, `"`+w+`"*`)
		}
	}
	if len(orTerms) == 0 {
		for _, w := range cleanWords {
			orTerms = append(orTerms, `"`+w+`"*`)
		}
	}

	phraseQuery := `"` + strings.Join(cleanWords, " ") + `"`
	return phraseQuery + " OR " + strings.Join(orTerms, " OR ")
}

func parseFlexibleDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02",
		"02/01/2006",
		"02.01.2006",
		"2006/01/02",
		"2006-01",
		"2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, dateStr); err == nil {
			return t
		}
	}
	if len(dateStr) >= 4 {
		if yr, err := strconv.Atoi(dateStr[:4]); err == nil && yr >= 1990 && yr <= 2100 {
			return time.Date(yr, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
	}
	return time.Time{}
}

var decisionNumberRegex = regexp.MustCompile(`(?i)\b[0-9]{6,10}\b`)

var vietnameseStopWords = map[string]bool{
	"theo": true, "quy": true, "định": true, "của": true, "khi": true, "được": true,
	"cho": true, "và": true, "các": true, "những": true, "trong": true, "về": true,
	"có": true, "là": true, "bởi": true, "tại": true, "bao": true, "nhiêu": true,
	"thì": true, "sao": true, "gì": true, "nào": true, "ở": true, "với": true,
	"như": true, "đã": true, "sẽ": true, "đang": true, "bị": true, "do": true,
	"đến": true, "từ": true, "ra": true, "vào": true, "lại": true, "qua": true,
	"lên": true, "xuống": true, "phần": true, "mức": true, "tiền": true, "biết": true,
	"xin": true, "hỏi": true, "hãy": true, "giúp": true, "tôi": true, "em": true,
}

func scoreChunk(chunk *models.DocumentChunk, doc *models.Document, queryEmbedding []float32, queryTokens []string, minScore float32) *models.RAGSearchResultItem {
	var vecSim float32
	if len(queryEmbedding) > 0 && len(chunk.Embedding) > 0 {
		vecSim = models.CosineSimilarity(queryEmbedding, chunk.Embedding)
	}

	var bm25Score float32
	if len(queryTokens) > 0 {
		contentLower := strings.ToLower(chunk.Content)
		var titleLower string
		if doc != nil {
			titleLower = strings.ToLower(doc.Title)
		}

		var totalWeight float32
		var matchedWeight float32
		var hasEntityMatch bool

		for _, token := range queryTokens {
			weight := float32(1.0)
			isEntity := courseCodeRegex.MatchString(token) || decisionNumberRegex.MatchString(token)
			if isEntity {
				weight = 4.0
			} else if vietnameseStopWords[token] {
				weight = 0.05
			}

			totalWeight += weight
			matched := strings.Contains(contentLower, token) || (titleLower != "" && strings.Contains(titleLower, token))
			if matched {
				matchedWeight += weight
				if isEntity {
					hasEntityMatch = true
				}
			}
		}

		if totalWeight > 0 {
			ratio := matchedWeight / totalWeight
			if hasEntityMatch {
				bm25Score = float32(math.Min(1.0, float64(ratio*0.80+0.20)))
			} else {
				bm25Score = float32(math.Min(1.0, float64(ratio*0.80)))
			}
		}
	}

	var tierMult float32 = 1.0
	var recencyScore float32 = 0.1
	if doc != nil {
		switch doc.ValidityTier {
		case constants.ValidityTierEmergency:
			tierMult = 1.2
		case constants.ValidityTierCorePermanent:
			tierMult = 1.05
		case constants.ValidityTierHistoricalArchive:
			tierMult = 0.4
		default:
			tierMult = 1.0
		}

		if doc.ValidityTier == constants.ValidityTierCorePermanent {
			recencyScore = 0.95
		} else {
			pubDate := parseFlexibleDate(doc.PublishDate)
			if !pubDate.IsZero() {
				diffDays := float64(time.Since(pubDate).Hours() / 24)
				if diffDays < 0 {
					diffDays = 0
				}
				decay := math.Exp(-diffDays / 365.0)
				recencyScore = float32(math.Max(0.05, math.Min(1.0, decay)))
			} else {
				recencyScore = 0.5
			}
		}
	}

	prioScore := float32(chunk.PriorityLevel) / 100.0
	var finalScore float32
	if len(queryEmbedding) > 0 && len(chunk.Embedding) > 0 && vecSim > 0.01 {
		finalScore = ((0.50 * vecSim) + (0.30 * bm25Score) + (0.10 * recencyScore) + (0.10 * prioScore)) * tierMult
	} else {
		finalScore = ((0.75 * bm25Score) + (0.15 * recencyScore) + (0.10 * prioScore)) * tierMult
	}

	minThreshold := float32(0.20)
	if minScore > 0 {
		minThreshold = minScore
	}
	if finalScore < minThreshold {
		return nil
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

	return item
}


func (s *ragService) Search(ctx context.Context, params models.RAGSearchParams, queryEmbedding []float32) ([]*response.RAGSearchResultItemResponse, error) {
	if chk := guardrail.CheckQuery(params.Query); chk.Decision == guardrail.DecisionBlockedInjection {
		return []*response.RAGSearchResultItemResponse{}, nil
	}

	if params.TopK <= 0 {
		params.TopK = 5
	}

	normQuery := strings.TrimSpace(params.Query)
	searchKey := cache.BuildKey("rag_search", normQuery, params.StudentCohort, params.Domain, fmt.Sprintf("%d", params.TopK), fmt.Sprintf("%.2f", params.MinScore))
	if s.c != nil && len(queryEmbedding) == 0 {
		var cached []*response.RAGSearchResultItemResponse
		if err := s.c.Get(ctx, searchKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	searchFn := func() (any, error) {
		res, err := s.executeSearch(ctx, params, queryEmbedding)
		if err != nil {
			return nil, err
		}
		return models.RAGSearchResultsToResponse(res), nil
	}

	if len(queryEmbedding) == 0 {
		v, err, _ := s.sfg.Do(searchKey, searchFn)
		if err != nil {
			return nil, err
		}
		items := v.([]*response.RAGSearchResultItemResponse)
		if s.c != nil && len(items) > 0 {
			_ = s.c.Set(ctx, searchKey, items, 10*time.Minute)
		}
		return items, nil
	}

	res, err := s.executeSearch(ctx, params, queryEmbedding)
	if err != nil {
		return nil, err
	}
	return models.RAGSearchResultsToResponse(res), nil
}

func (s *ragService) executeSearch(ctx context.Context, params models.RAGSearchParams, queryEmbedding []float32) ([]*models.RAGSearchResultItem, error) {
	if len(queryEmbedding) == 0 && s.llmManager != nil && strings.TrimSpace(params.Query) != "" {
		embKey := cache.BuildKey("rag_emb", "q", strings.ToLower(strings.TrimSpace(params.Query)))
		if s.c != nil {
			var cachedEmb []float32
			if err := s.c.Get(ctx, embKey, &cachedEmb); err == nil && len(cachedEmb) > 0 {
				queryEmbedding = cachedEmb
			}
		}
		if len(queryEmbedding) == 0 {
			v, err, _ := s.embSfg.Do(embKey, func() (any, error) {
				embs, err := s.llmManager.ExecuteEmbedding(ctx, []string{params.Query})
				if err != nil || len(embs) == 0 {
					return nil, err
				}
				if s.c != nil {
					_ = s.c.Set(ctx, embKey, embs[0], 24*time.Hour)
				}
				return embs[0], nil
			})
			if err == nil && v != nil {
				queryEmbedding = v.([]float32)
			}
		}
	}

	candidates, err := s.ragRepo.QueryActiveCandidates(ctx, params.StudentCohort, params.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to query active candidates: %w", err)
	}

	ftsQuery := sanitizeFTS5Query(params.Query)
	if ftsQuery != "" {
		ftsLimit := max(params.TopK*15, 120)
		if chunkIDs, err := s.ragRepo.SearchActiveChunkIDsFTS(ctx, ftsQuery, params.StudentCohort, params.Domain, ftsLimit); err == nil && len(chunkIDs) > 0 {
			candidateIDs := make(map[string]bool, len(candidates))
			for _, c := range candidates {
				candidateIDs[c.ID] = true
			}
			var missingIDs []string
			for _, cid := range chunkIDs {
				if !candidateIDs[cid] {
					missingIDs = append(missingIDs, cid)
				}
			}
			if len(missingIDs) > 0 {
				if resolved, err := s.ragRepo.GetChunksByIDs(ctx, missingIDs); err == nil && len(resolved) > 0 {
					candidates = append(candidates, resolved...)
				}
			}
		}
	}

	supersededDocs, err := s.ragRepo.FindSupersededDocIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse superseded graph: %w", err)
	}

	queryTokens := tokenizeSearchQuery(params.Query)

	uniqueDocIDs := make(map[string]bool)
	for _, chunk := range candidates {
		if !supersededDocs[chunk.DocumentID] {
			uniqueDocIDs[chunk.DocumentID] = true
		}
	}

	docCache := make(map[string]*models.Document, len(uniqueDocIDs))
	if len(uniqueDocIDs) > 0 {
		docIDs := make([]string, 0, len(uniqueDocIDs))
		for id := range uniqueDocIDs {
			docIDs = append(docIDs, id)
		}
		docs, _ := s.ragRepo.GetDocumentsByIDs(ctx, docIDs)
		for _, d := range docs {
			if d != nil {
				docCache[d.ID] = d
			}
		}
	}

	var validResults []*models.RAGSearchResultItem
	if len(candidates) <= 200 {
		for _, chunk := range candidates {
			if supersededDocs[chunk.DocumentID] {
				continue
			}
			doc := docCache[chunk.DocumentID]
			if item := scoreChunk(chunk, doc, queryEmbedding, queryTokens, params.MinScore); item != nil {
				validResults = append(validResults, item)
			}
		}
	} else {
		numWorkers := 4
		chunkCount := len(candidates)
		step := (chunkCount + numWorkers - 1) / numWorkers
		workerResults := make([][]*models.RAGSearchResultItem, numWorkers)
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			start := w * step
			end := start + step
			if end > chunkCount {
				end = chunkCount
			}
			if start >= chunkCount {
				break
			}
			wg.Add(1)
			go func(workerIdx, sIdx, eIdx int) {
				defer wg.Done()
				var local []*models.RAGSearchResultItem
				for i := sIdx; i < eIdx; i++ {
					c := candidates[i]
					if supersededDocs[c.DocumentID] {
						continue
					}
					d := docCache[c.DocumentID]
					if item := scoreChunk(c, d, queryEmbedding, queryTokens, params.MinScore); item != nil {
						local = append(local, item)
					}
				}
				workerResults[workerIdx] = local
			}(w, start, end)
		}
		wg.Wait()

		for _, wr := range workerResults {
			validResults = append(validResults, wr...)
		}
	}

	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].FinalScore > validResults[j].FinalScore
	})

	if s.llmManager != nil && len(validResults) > 0 && strings.TrimSpace(params.Query) != "" {
		shouldRerank := true
		if validResults[0].VectorSim >= 0.85 || validResults[0].FinalScore >= 0.85 {
			shouldRerank = false
		}

		if shouldRerank {
			poolSize := min(params.TopK*2, 8)
			if poolSize > len(validResults) {
				poolSize = len(validResults)
			}

			candidatePool := validResults[:poolSize]
			var candidateTexts []string
			for _, item := range candidatePool {
				text := item.DocTitle + "\n" + item.Content
				runes := []rune(text)
				if len(runes) > 2500 {
					text = string(runes[:2500])
				}
				candidateTexts = append(candidateTexts, text)
			}

			rerankCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			rerankResults, err := s.llmManager.ExecuteRerank(rerankCtx, params.Query, candidateTexts, &params.TopK)
			cancel()
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

	if doc.Status == constants.DocStatusActive && s.inquiryService != nil {
		_, _ = s.inquiryService.CheckAndExpireOutdatedQA(ctx, doc)
	}

	if s.c != nil {
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGSearch)
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGCandidates)
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

func (s *ragService) IngestFromIndexJSONL(ctx context.Context, jsonlPath string, baseDataDir string, embedder llm.Embedder, onProgress func(cur, total int64)) (int, error) {
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
			Status:           constants.DocStatusActive,
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
			ID:       constants.PrefixDocNode + doc.ID,
			NodeType: constants.KGNodeTypeDocument,
			Name:     doc.Title,
			Code:     doc.DocCode,
			Status:   constants.DocStatusActive,
		}

		nodes := []*models.KGNode{docNode}
		var edges []*models.KGEdge

		if doc.ApplicableCohort != "" && doc.ApplicableCohort != constants.DocCohortAll {
			cohorts := strings.Split(doc.ApplicableCohort, ",")
			for _, c := range cohorts {
				c = strings.TrimSpace(c)
				if c == "" {
					continue
				}
				cohortNode := &models.KGNode{
					ID:       constants.PrefixCohortNode + c,
					NodeType: constants.KGNodeTypeCohort,
					Name:     "Cohort " + c,
					Code:     c,
					Status:   constants.DocStatusActive,
				}
				nodes = append(nodes, cohortNode)
				edges = append(edges, &models.KGEdge{
					ID:           fmt.Sprintf("%s%s:%s", constants.PrefixEdgeCohort, doc.ID, c),
					SourceNodeID: docNode.ID,
					TargetNodeID: cohortNode.ID,
					RelationType: constants.KGRelationAppliesToCohort,
					Weight:       1.0,
					Status:       constants.DocStatusActive,
				})
			}
		}

		if doc.Domain != "" {
			topicNode := &models.KGNode{
				ID:       constants.PrefixTopicNode + doc.Domain,
				NodeType: constants.KGNodeTypeTopic,
				Name:     doc.Domain,
				Code:     doc.Domain,
				Status:   constants.DocStatusActive,
			}
			nodes = append(nodes, topicNode)
			edges = append(edges, &models.KGEdge{
				ID:           fmt.Sprintf("%s%s:%s", constants.PrefixEdgeTopic, doc.ID, doc.Domain),
				SourceNodeID: docNode.ID,
				TargetNodeID: topicNode.ID,
				RelationType: constants.KGRelationCoversTopic,
				Weight:       1.0,
				Status:       constants.DocStatusActive,
			})
		}

		var chunks []*models.DocumentChunk
		contentToChunk := entry.OcrText
		if contentToChunk == "" && entry.MDPath != "" {
			targetMDPath := entry.MDPath
			if after, ok := strings.CutPrefix(targetMDPath, constants.DefaultProcessedDir+"/"); ok {
				targetMDPath = after
			}
			fullPath := filepath.Clean(filepath.Join(baseDataDir, targetMDPath))
			if data, err := os.ReadFile(fullPath); err == nil {
				contentToChunk = string(data)
			} else if dataDirect, errDirect := os.ReadFile(entry.MDPath); errDirect == nil {
				contentToChunk = string(dataDirect)
			}
		}

		if contentToChunk == "" {
			contentToChunk = doc.Title
		}

		chunkTexts := splitIntoChunks(contentToChunk, 1500, 200)
		var embeddings [][]float32
		if len(chunkTexts) > 0 {
			batchSize := 32
			for i := 0; i < len(chunkTexts); i += batchSize {
				end := i + batchSize
				if end > len(chunkTexts) {
					end = len(chunkTexts)
				}
				var batch [][]float32
				var err error
				if embedder != nil {
					batch, err = embedder.Embed(ctx, chunkTexts[i:end])
				} else if s.llmManager != nil {
					batch, err = s.llmManager.ExecuteEmbedding(ctx, chunkTexts[i:end])
				}
				if err != nil {
					break
				}
				embeddings = append(embeddings, batch...)
			}
		}

		for idx, text := range chunkTexts {
			var emb []float32
			if idx < len(embeddings) {
				emb = embeddings[idx]
			}
			chunk := &models.DocumentChunk{
				ID:               fmt.Sprintf("%s%s_%d", constants.PrefixChunk, doc.ID, idx),
				DocumentID:       doc.ID,
				ChunkIndex:       idx,
				ChunkType:        constants.ChunkTypeContent,
				Content:          text,
				Status:           constants.DocStatusActive,
				Domain:           doc.Domain,
				ApplicableCohort: doc.ApplicableCohort,
				AcademicYear:     doc.AcademicYear,
				PriorityLevel:    doc.PriorityLevel,
				Embedding:        emb,
			}
			chunks = append(chunks, chunk)
		}

		if err := s.IngestDocument(ctx, doc, chunks, nodes, edges); err != nil {
			continue
		}
		count++

		if onProgress != nil && count%25 == 0 {
			onProgress(int64(count), 0)
		}
	}

	if onProgress != nil {
		onProgress(int64(count), int64(count))
	}

	return count, scanner.Err()
}

func tokenizeSearchQuery(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	var tokens []string
	for _, f := range fields {
		f = strings.Trim(f, ",.?!;:\"'()[]{}")
		if len([]rune(f)) >= 2 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

func resolvePriorityLevel(validityTier, docType string) int {
	switch validityTier {
	case constants.ValidityTierEmergency:
		return 90
	case constants.ValidityTierCorePermanent:
		return 75
	case constants.ValidityTierSemesterDynamic:
		return 60
	}
	if docType == constants.DocTypeRegulation {
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

// BuildDocumentFilter preloads all existing document identifiers into memory and returns an ultra-fast O(1) deduplication checker.
func (s *ragService) BuildDocumentFilter(ctx context.Context) (func(id, rawURL, title string) bool, error) {
	if s.ragRepo == nil {
		return func(id, rawURL, title string) bool { return false }, nil
	}
	rows, err := s.ragRepo.ListAllDocumentIdentifiers(ctx)
	if err != nil {
		return nil, err
	}

	docIDs := make(map[string]bool, len(rows))
	nodeIDs := make(map[string]bool, len(rows)*2)
	titles := make(map[string]bool, len(rows))
	urls := make(map[string]bool, len(rows)*2)

	reNode := regexp.MustCompile(`-(\d+)\.html|/node/(\d+)|DOC_[A-Z]+_(\d+)_`)

	normalize := func(u string) string {
		u = strings.ToLower(strings.TrimSpace(u))
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.TrimPrefix(u, "thanglong.edu.vn")
		return strings.TrimRight(u, "/")
	}

	cleanTitle := func(t string) string {
		t = strings.ToLower(strings.TrimSpace(t))
		return strings.Join(strings.Fields(t), " ")
	}

	extractNum := func(s string) string {
		m := reNode.FindStringSubmatch(s)
		if len(m) > 1 && m[1] != "" {
			return m[1]
		}
		if len(m) > 2 && m[2] != "" {
			return m[2]
		}
		if len(m) > 3 && m[3] != "" {
			return m[3]
		}
		return ""
	}

	for _, r := range rows {
		if r.ID != "" {
			docIDs[r.ID] = true
			if n := extractNum(r.ID); n != "" {
				nodeIDs[n] = true
			}
		}
		if r.Title != "" {
			titles[cleanTitle(r.Title)] = true
		}
		if r.FilePath != "" {
			urls[normalize(r.FilePath)] = true
			if n := extractNum(r.FilePath); n != "" {
				nodeIDs[n] = true
			}
		}
	}

	var mu sync.RWMutex
	return func(id, rawURL, title string) bool {
		mu.RLock()
		defer mu.RUnlock()

		if id != "" && docIDs[id] {
			return true
		}
		if rawURL != "" {
			if urls[normalize(rawURL)] {
				return true
			}
			if n := extractNum(rawURL); n != "" && nodeIDs[n] {
				return true
			}
		}
		if title != "" && titles[cleanTitle(title)] {
			return true
		}
		return false
	}, nil
}

func (s *ragService) DocumentExistsByURL(ctx context.Context, rawURL string) bool {
	if s.ragRepo == nil {
		return false
	}
	exists, err := s.ragRepo.DocumentExistsByURL(ctx, rawURL)
	return err == nil && exists
}

func (s *ragService) GetDocumentByURL(ctx context.Context, rawURL string) (*response.DocumentDetailResponse, error) {
	doc, err := s.ragRepo.GetDocumentByURL(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return s.buildDocumentDetailResponse(ctx, doc)
}

func (s *ragService) GetDocumentDetail(ctx context.Context, id string) (*response.DocumentDetailResponse, error) {
	doc, err := s.ragRepo.GetDocumentByID(ctx, id)
	if err != nil {
		doc, err = s.ragRepo.FindDocumentByIdentifier(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	return s.buildDocumentDetailResponse(ctx, doc)
}

func (s *ragService) buildDocumentDetailResponse(ctx context.Context, doc *models.Document) (*response.DocumentDetailResponse, error) {
	claims := PermissionContextFrom(ctx)
	isAdvisorOrAdmin := claims != nil && (slices.Contains(claims.Roles, constants.RoleTypeAdmin) || slices.Contains(claims.Roles, constants.RoleTypeAdvisor))
	if (doc.ID == constants.DocIDTeacherQA || doc.Type == constants.DocTypeAdvisoryQA) && !isAdvisorOrAdmin {
		return nil, apperrors.New(apperrors.ErrForbidden, "Forbidden: internal advisor Q&A handbook access restricted")
	}

	chunks, err := s.ragRepo.GetChunksByDocumentID(ctx, doc.ID)
	if err != nil {
		return nil, err
	}

	var visibleChunks []*models.DocumentChunk
	for _, ch := range chunks {
		if isAdvisorOrAdmin || ch.Status == constants.DocStatusActive {
			visibleChunks = append(visibleChunks, ch)
		}
	}
	chunks = visibleChunks

	var supersededBy *models.Document
	if doc.SupersededByID != "" {
		if sb, err := s.ragRepo.GetDocumentByID(ctx, doc.SupersededByID); err == nil {
			supersededBy = sb
		}
	}

	supersedes, _ := s.ragRepo.ListSupersededDocuments(ctx, doc.ID)
	auditLogs, _ := s.ragRepo.ListAuditLogsByDocID(ctx, doc.ID)

	var fullContentBuilder strings.Builder
	for i, ch := range chunks {
		if i > 0 {
			fullContentBuilder.WriteString("\n\n")
		}
		fullContentBuilder.WriteString(ch.Content)
	}

	return (&models.DocumentDetail{
		Document:     doc,
		SupersededBy: supersededBy,
		Supersedes:   supersedes,
		Chunks:       chunks,
		AuditLogs:    auditLogs,
		FullContent:  fullContentBuilder.String(),
	}).ToResponse(), nil
}

func (s *ragService) UpdateDocumentStatus(ctx context.Context, id string, req *request.UpdateDocumentStatusRequest) error {
	doc, err := s.ragRepo.GetDocumentByID(ctx, id)
	if err != nil {
		return err
	}

	status := constants.DocStatusActive
	supersededByID := ""
	reason := ""
	if req != nil {
		status = req.Status
		supersededByID = req.SupersededByID
		reason = req.Reason
	}

	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if normalizedStatus != constants.DocStatusActive &&
		normalizedStatus != constants.DocStatusSuperseded &&
		normalizedStatus != constants.DocStatusPartiallyAmended &&
		normalizedStatus != constants.DocStatusDraft &&
		normalizedStatus != constants.DocStatusArchived {
		normalizedStatus = constants.DocStatusActive
	}

	if err := s.ragRepo.UpdateDocumentStatus(ctx, id, normalizedStatus, supersededByID); err != nil {
		return err
	}

	_ = s.ragRepo.UpdateChunksByDocIDStatus(ctx, id, normalizedStatus)

	if docNode, err := s.ragRepo.GetKGNodeByID(ctx, constants.PrefixDocNode+id); err == nil && docNode != nil {
		docNode.Status = normalizedStatus
		_, _ = s.ragRepo.UpsertKGNode(ctx, docNode)
	}

	if supersededByID != "" {
		edgeID := fmt.Sprintf("%s%s_%s", constants.PrefixEdgeSup, supersededByID, id)
		_, _ = s.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
			ID:           edgeID,
			SourceNodeID: constants.PrefixDocNode + supersededByID,
			TargetNodeID: constants.PrefixDocNode + id,
			RelationType: constants.KGRelationSupersedes,
			Weight:       1.0,
			Status:       constants.DocStatusActive,
		})
	}

	if reason == "" {
		reason = fmt.Sprintf("Document status updated to %s", normalizedStatus)
	}

	if normalizedStatus == constants.DocStatusSuperseded {
		activeID := supersededByID
		_ = s.ragRepo.RecordSupersessionLog(ctx, activeID, id, constants.RuleTypeAdminOverride, reason)
		if s.inquiryService != nil {
			_, _ = s.inquiryService.CheckAndExpireOutdatedQA(ctx, doc)
		}
	} else {
		_ = s.ragRepo.RecordSupersessionLog(ctx, id, supersededByID, constants.RuleTypeAdminOverride, reason)
	}

	if s.c != nil {
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGSearch)
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGCandidates)
	}
	return nil
}

func (s *ragService) GetKnowledgeGraph(ctx context.Context) (*response.KnowledgeGraphDataResponse, error) {
	nodes, err := s.ragRepo.ListAllKGNodes(ctx)
	if err != nil {
		return nil, err
	}
	edges, err := s.ragRepo.ListAllKGEdges(ctx)
	if err != nil {
		return nil, err
	}
	return (&models.KnowledgeGraphData{
		Nodes: nodes,
		Edges: edges,
	}).ToResponse(), nil
}

func (s *ragService) CreateOrUpdateKGEdge(ctx context.Context, edge *models.KGEdge) (*response.KGEdgeResponse, error) {
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("edge_%d", time.Now().UnixNano())
	}
	if edge.Status == "" {
		edge.Status = constants.DocStatusActive
	}
	res, err := s.ragRepo.UpsertKGEdge(ctx, edge)
	if err != nil {
		return nil, err
	}
	if s.c != nil {
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGSearch)
	}
	return res.ToResponse(), nil
}

func (s *ragService) DeleteKGEdge(ctx context.Context, id string) error {
	err := s.ragRepo.DeleteKGEdge(ctx, id)
	if err == nil && s.c != nil {
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGSearch)
	}
	return err
}

func (s *ragService) GetKnowledgeStats(ctx context.Context) (*response.KnowledgeStatsResponse, error) {
	stats, err := s.ragRepo.GetKnowledgeStats(ctx)
	if err != nil {
		return nil, err
	}
	return stats.ToResponse(), nil
}

func (s *ragService) ListDocuments(ctx context.Context, dto *request.ListDocumentsDto) ([]*response.DocumentResponse, int, error) {
	docs, total, err := s.ragRepo.ListDocuments(ctx, dto)
	if err != nil {
		return nil, 0, err
	}
	return models.DocumentsToResponse(docs), total, nil
}

// ListAllRegulationsChronological returns all regulations ordered chronologically for audits.
func (s *ragService) ListAllRegulationsChronological(ctx context.Context) ([]*response.DocumentResponse, error) {
	docs, err := s.ragRepo.ListAllRegulationsChronological(ctx)
	if err != nil {
		return nil, err
	}
	return models.DocumentsToResponse(docs), nil
}

func (s *ragService) ListAuditLogs(ctx context.Context, dto *request.ListAuditLogsDto) ([]*response.SupersessionAuditLogResponse, int, error) {
	limit := 20
	offset := 0
	if dto != nil {
		limit = dto.GetLimit(20)
		offset = dto.GetOffset(20)
	}
	logs, total, err := s.ragRepo.ListAllAuditLogs(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return models.SupersessionAuditLogsToResponse(logs), total, nil
}

func (s *ragService) WipeCorpus(ctx context.Context) error {
	if err := s.ragRepo.WipeCorpus(ctx); err != nil {
		return err
	}
	if s.c != nil {
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGSearch)
		_ = s.c.DelByPattern(ctx, constants.CachePrefixRAGCandidates)
	}
	return nil
}

