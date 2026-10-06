package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/jsonx"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/netx"
)

var defaultRerankTestDocs = []string{
	"Academic regulations regarding credit requirements for graduation.",
	"Procedures for issuing student ID cards and official transcripts.",
	"Scholarship policies for students with high academic performance.",
}

type LLMConfigService interface {
	ListProviders(ctx context.Context) ([]response.LLMProviderResponse, error)
	GetProvider(ctx context.Context, id string) (response.LLMProviderResponse, error)
	CreateProvider(ctx context.Context, req request.CreateLLMProviderRequest) (response.LLMProviderResponse, error)
	UpdateProvider(ctx context.Context, id string, req request.UpdateLLMProviderRequest) (response.LLMProviderResponse, error)
	DeleteProvider(ctx context.Context, id string) error
	SetDefaultProvider(ctx context.Context, id string) error

	ListModels(ctx context.Context, providerID string) ([]response.LLMModelResponse, error)
	CreateModel(ctx context.Context, req request.CreateLLMModelRequest) (response.LLMModelResponse, error)
	UpdateModel(ctx context.Context, id string, req request.UpdateLLMModelRequest) (response.LLMModelResponse, error)
	DeleteModel(ctx context.Context, id string) error

	ListChains(ctx context.Context) ([]response.LLMChainResponse, error)
	GetChain(ctx context.Context, id string) (response.LLMChainResponse, error)
	CreateChain(ctx context.Context, req request.CreateLLMChainRequest) (response.LLMChainResponse, error)
	UpdateChain(ctx context.Context, id string, req request.UpdateLLMChainRequest) (response.LLMChainResponse, error)
	DeleteChain(ctx context.Context, id string) error

	AddChainNode(ctx context.Context, chainID string, req request.AddChainNodeRequest) (response.LLMChainNodeResponse, error)
	RemoveChainNode(ctx context.Context, chainID string, nodeID string) error
	UpdateChainNodePriority(ctx context.Context, chainID string, nodeID string, req request.UpdateChainNodePriorityRequest) error
	ResetNodeCircuitBreaker(ctx context.Context, chainID string, nodeID string) error

	ProbeModelVisionCapability(ctx context.Context, providerID, modelKey string) (response.ProbeVisionResponse, error)
	GetActiveChatModels(ctx context.Context) ([]response.ChatModelClientDTO, error)
	TestModel(ctx context.Context, req request.TestLLMModelRequest) (*response.TestLLMModelResponse, error)
	SeedDefaultProvidersIfEmpty(ctx context.Context) error
	SyncLLMManager(ctx context.Context) error
	SetSettingsRepository(settings repositories.SettingsRepository)
}

type llmConfigService struct {
	repo     repositories.LLMRepository
	manager  *llm.Manager
	settings repositories.SettingsRepository
}

// NewLLMConfigService creates a new LLM configuration service.
func NewLLMConfigService(repo repositories.LLMRepository, manager *llm.Manager) LLMConfigService {
	return &llmConfigService{
		repo:    repo,
		manager: manager,
	}
}

func (s *llmConfigService) SetSettingsRepository(settings repositories.SettingsRepository) {
	s.settings = settings
}

func (s *llmConfigService) ListProviders(ctx context.Context) ([]response.LLMProviderResponse, error) {
	provs, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]response.LLMProviderResponse, len(provs))
	for i, p := range provs {
		result[i] = mapProviderResponse(p)
	}
	return result, nil
}

func (s *llmConfigService) GetProvider(ctx context.Context, id string) (response.LLMProviderResponse, error) {
	p, err := s.repo.GetProviderByID(ctx, id)
	if err != nil {
		return response.LLMProviderResponse{}, err
	}
	return mapProviderResponse(p), nil
}

func (s *llmConfigService) CreateProvider(ctx context.Context, req request.CreateLLMProviderRequest) (response.LLMProviderResponse, error) {
	cipherKey := crypto.GetLLMEncryptionKey()
	ciphertext, err := crypto.EncryptAESGCM(strings.TrimSpace(req.APIKey), cipherKey)
	if err != nil {
		return response.LLMProviderResponse{}, fmt.Errorf("encrypt api key: %w", err)
	}

	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}
	isDefault := int64(0)
	if req.IsDefault != nil && *req.IsDefault {
		isDefault = 1
	}

	headersJSON := strings.TrimSpace(req.CustomHeadersJSON)
	if headersJSON == "" {
		headersJSON = "{}"
	}

	timeoutSec := int64(60)
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		timeoutSec = int64(*req.TimeoutSeconds)
	}
	maxRetries := int64(3)
	if req.MaxRetries != nil && *req.MaxRetries >= 0 {
		maxRetries = int64(*req.MaxRetries)
	}
	initialWaitMs := int64(500)
	if req.RetryInitialWaitMs != nil && *req.RetryInitialWaitMs > 0 {
		initialWaitMs = int64(*req.RetryInitialWaitMs)
	}
	maxWaitMs := int64(5000)
	if req.RetryMaxWaitMs != nil && *req.RetryMaxWaitMs > 0 {
		maxWaitMs = int64(*req.RetryMaxWaitMs)
	}
	allowPrivate := int64(0)
	if req.AllowPrivateNetworks != nil && *req.AllowPrivateNetworks {
		allowPrivate = 1
	}

	prov, err := s.repo.CreateProvider(ctx, sqlc.CreateLLMProviderParams{
		ID:                   strings.ToLower(strings.TrimSpace(req.ID)),
		Name:                 strings.TrimSpace(req.Name),
		ProviderType:         strings.ToLower(strings.TrimSpace(req.ProviderType)),
		BaseUrl:              strings.TrimSpace(req.BaseURL),
		ApiKeyCiphertext:     ciphertext,
		IsActive:             isActive,
		IsDefault:            isDefault,
		CustomHeadersJson:    headersJSON,
		TimeoutSeconds:       timeoutSec,
		MaxRetries:           maxRetries,
		RetryInitialWaitMs:   initialWaitMs,
		RetryMaxWaitMs:       maxWaitMs,
		AllowPrivateNetworks: allowPrivate,
	})
	if err != nil {
		return response.LLMProviderResponse{}, err
	}

	if isDefault == 1 {
		_ = s.repo.SetDefaultProvider(ctx, prov.ID)
	}

	_ = s.SyncLLMManager(ctx)
	return mapProviderResponse(prov), nil
}

func (s *llmConfigService) UpdateProvider(ctx context.Context, id string, req request.UpdateLLMProviderRequest) (response.LLMProviderResponse, error) {
	existing, err := s.repo.GetProviderByID(ctx, id)
	if err != nil {
		return response.LLMProviderResponse{}, err
	}

	ciphertext := existing.ApiKeyCiphertext
	if strings.TrimSpace(req.APIKey) != "" {
		cipherKey := crypto.GetLLMEncryptionKey()
		newCipher, err := crypto.EncryptAESGCM(strings.TrimSpace(req.APIKey), cipherKey)
		if err != nil {
			return response.LLMProviderResponse{}, fmt.Errorf("encrypt api key: %w", err)
		}
		ciphertext = newCipher
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		if *req.IsActive {
			isActive = 1
		} else {
			isActive = 0
		}
	}

	isDefault := existing.IsDefault
	if req.IsDefault != nil {
		if *req.IsDefault {
			isDefault = 1
		} else {
			isDefault = 0
		}
	}

	headersJSON := existing.CustomHeadersJson
	if req.CustomHeadersJSON != "" {
		headersJSON = req.CustomHeadersJSON
	}

	timeoutSec := existing.TimeoutSeconds
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		timeoutSec = int64(*req.TimeoutSeconds)
	}
	maxRetries := existing.MaxRetries
	if req.MaxRetries != nil && *req.MaxRetries >= 0 {
		maxRetries = int64(*req.MaxRetries)
	}
	initialWaitMs := existing.RetryInitialWaitMs
	if req.RetryInitialWaitMs != nil && *req.RetryInitialWaitMs > 0 {
		initialWaitMs = int64(*req.RetryInitialWaitMs)
	}
	maxWaitMs := existing.RetryMaxWaitMs
	if req.RetryMaxWaitMs != nil && *req.RetryMaxWaitMs > 0 {
		maxWaitMs = int64(*req.RetryMaxWaitMs)
	}
	allowPrivate := existing.AllowPrivateNetworks
	if req.AllowPrivateNetworks != nil {
		if *req.AllowPrivateNetworks {
			allowPrivate = 1
		} else {
			allowPrivate = 0
		}
	}

	prov, err := s.repo.UpdateProvider(ctx, sqlc.UpdateLLMProviderParams{
		ID:                   id,
		Name:                 strings.TrimSpace(req.Name),
		ProviderType:         strings.ToLower(strings.TrimSpace(req.ProviderType)),
		BaseUrl:              strings.TrimSpace(req.BaseURL),
		ApiKeyCiphertext:     ciphertext,
		IsActive:             isActive,
		IsDefault:            isDefault,
		CustomHeadersJson:    headersJSON,
		TimeoutSeconds:       timeoutSec,
		MaxRetries:           maxRetries,
		RetryInitialWaitMs:   initialWaitMs,
		RetryMaxWaitMs:       maxWaitMs,
		AllowPrivateNetworks: allowPrivate,
	})
	if err != nil {
		return response.LLMProviderResponse{}, err
	}

	if isDefault == 1 {
		_ = s.repo.SetDefaultProvider(ctx, id)
	}

	_ = s.SyncLLMManager(ctx)
	return mapProviderResponse(prov), nil
}

func (s *llmConfigService) DeleteProvider(ctx context.Context, id string) error {
	if err := s.repo.DeleteProvider(ctx, id); err != nil {
		return err
	}
	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) SetDefaultProvider(ctx context.Context, id string) error {
	if err := s.repo.SetDefaultProvider(ctx, id); err != nil {
		return err
	}
	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) ListModels(ctx context.Context, providerID string) ([]response.LLMModelResponse, error) {
	var models []sqlc.LlmModel
	var err error
	if strings.TrimSpace(providerID) != "" {
		models, err = s.repo.ListModelsByProviderID(ctx, providerID)
	} else {
		models, err = s.repo.ListModels(ctx)
	}
	if err != nil {
		return nil, err
	}

	provMap := make(map[string]string)
	if provs, err := s.repo.ListProviders(ctx); err == nil {
		for _, p := range provs {
			provMap[p.ID] = p.Name
		}
	}

	result := make([]response.LLMModelResponse, len(models))
	for i, m := range models {
		resp := mapModelResponse(m)
		resp.ProviderName = provMap[m.ProviderID]
		result[i] = resp
	}
	return result, nil
}

func (s *llmConfigService) CreateModel(ctx context.Context, req request.CreateLLMModelRequest) (response.LLMModelResponse, error) {
	supportsVision := int64(0)
	maxImages := int64(req.MaxImages)
	visionMode := strings.ToLower(strings.TrimSpace(req.VisionMode))
	if visionMode == "" {
		visionMode = "manual"
	}

	if visionMode == "auto" {
		probe, err := s.ProbeModelVisionCapability(ctx, req.ProviderID, req.ModelKey)
		if err == nil {
			if probe.SupportsVision {
				supportsVision = 1
				if maxImages <= 0 {
					maxImages = int64(probe.SuggestedMaxImages)
				}
			}
		}
	} else if req.SupportsVision != nil && *req.SupportsVision {
		supportsVision = 1
		if maxImages <= 0 {
			maxImages = 4
		}
	}

	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}
	isDefault := int64(0)
	if req.IsDefault != nil && *req.IsDefault {
		isDefault = 1
	}

	ctxLen := int64(req.ContextLength)
	if ctxLen <= 0 {
		ctxLen = 4096
	}

	modelType := strings.ToLower(strings.TrimSpace(req.ModelType))
	if modelType == "" {
		modelType = "chat"
	}

	thinkingEnabled := int64(0)
	if req.ThinkingEnabled != nil && *req.ThinkingEnabled {
		thinkingEnabled = 1
	}
	thinkingBudget := int64(req.ThinkingBudget)
	effortLevel := strings.ToLower(strings.TrimSpace(req.EffortLevel))
	if effortLevel == "" {
		effortLevel = "medium"
	}

	created, err := s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
		ID:              strings.ToLower(strings.TrimSpace(req.ID)),
		ProviderID:      strings.TrimSpace(req.ProviderID),
		Name:            strings.TrimSpace(req.Name),
		ModelKey:        strings.TrimSpace(req.ModelKey),
		ModelType:       modelType,
		IsActive:        isActive,
		IsDefault:       isDefault,
		OrderIndex:      int64(req.OrderIndex),
		ContextLength:   ctxLen,
		VisionMode:      visionMode,
		SupportsVision:  supportsVision,
		MaxImages:       maxImages,
		ThinkingEnabled: thinkingEnabled,
		ThinkingBudget:  thinkingBudget,
		EffortLevel:     effortLevel,
	})
	if err != nil {
		return response.LLMModelResponse{}, err
	}

	if isDefault == 1 {
		_ = s.repo.SetDefaultModel(ctx, created.ID, modelType)
	}

	_ = s.SyncLLMManager(ctx)
	return mapModelResponse(created), nil
}

func (s *llmConfigService) UpdateModel(ctx context.Context, id string, req request.UpdateLLMModelRequest) (response.LLMModelResponse, error) {
	existing, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return response.LLMModelResponse{}, err
	}

	visionMode := strings.ToLower(strings.TrimSpace(req.VisionMode))
	if visionMode == "" {
		visionMode = existing.VisionMode
	}

	supportsVision := existing.SupportsVision
	maxImages := int64(req.MaxImages)

	if visionMode == "auto" {
		probe, err := s.ProbeModelVisionCapability(ctx, req.ProviderID, req.ModelKey)
		if err == nil {
			if probe.SupportsVision {
				supportsVision = 1
				if maxImages <= 0 {
					maxImages = int64(probe.SuggestedMaxImages)
				}
			} else {
				supportsVision = 0
				maxImages = 0
			}
		}
	} else if req.SupportsVision != nil {
		if *req.SupportsVision {
			supportsVision = 1
			if maxImages <= 0 {
				maxImages = 4
			}
		} else {
			supportsVision = 0
			maxImages = 0
		}
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		if *req.IsActive {
			isActive = 1
		} else {
			isActive = 0
		}
	}

	isDefault := existing.IsDefault
	if req.IsDefault != nil {
		if *req.IsDefault {
			isDefault = 1
		} else {
			isDefault = 0
		}
	}

	ctxLen := int64(req.ContextLength)
	if ctxLen <= 0 {
		ctxLen = existing.ContextLength
	}

	modelType := strings.ToLower(strings.TrimSpace(req.ModelType))
	if modelType == "" {
		modelType = existing.ModelType
	}

	thinkingEnabled := existing.ThinkingEnabled
	if req.ThinkingEnabled != nil {
		if *req.ThinkingEnabled {
			thinkingEnabled = 1
		} else {
			thinkingEnabled = 0
		}
	}
	thinkingBudget := int64(req.ThinkingBudget)
	effortLevel := strings.ToLower(strings.TrimSpace(req.EffortLevel))
	if effortLevel == "" {
		effortLevel = existing.EffortLevel
	}

	updated, err := s.repo.UpdateModel(ctx, sqlc.UpdateLLMModelParams{
		ID:              id,
		ProviderID:      strings.TrimSpace(req.ProviderID),
		Name:            strings.TrimSpace(req.Name),
		ModelKey:        strings.TrimSpace(req.ModelKey),
		ModelType:       modelType,
		IsActive:        isActive,
		IsDefault:       isDefault,
		OrderIndex:      int64(req.OrderIndex),
		ContextLength:   ctxLen,
		VisionMode:      visionMode,
		SupportsVision:  supportsVision,
		MaxImages:       maxImages,
		ThinkingEnabled: thinkingEnabled,
		ThinkingBudget:  thinkingBudget,
		EffortLevel:     effortLevel,
	})
	if err != nil {
		return response.LLMModelResponse{}, err
	}

	if isDefault == 1 {
		_ = s.repo.SetDefaultModel(ctx, id, modelType)
	}

	_ = s.SyncLLMManager(ctx)
	return mapModelResponse(updated), nil
}

func (s *llmConfigService) DeleteModel(ctx context.Context, id string) error {
	if err := s.repo.DeleteModel(ctx, id); err != nil {
		return err
	}
	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) ProbeModelVisionCapability(ctx context.Context, providerID, modelKey string) (response.ProbeVisionResponse, error) {
	prov, err := s.repo.GetProviderByID(ctx, providerID)
	if err != nil {
		return response.ProbeVisionResponse{}, err
	}

	cipherKey := crypto.GetLLMEncryptionKey()
	apiKey, err := crypto.DecryptAESGCM(prov.ApiKeyCiphertext, cipherKey)
	if err != nil {
		return response.ProbeVisionResponse{}, fmt.Errorf("decrypt provider api key: %w", err)
	}

	if prov.ProviderType == "gemini" || strings.Contains(strings.ToLower(prov.BaseUrl), "generativelanguage") {
		return response.ProbeVisionResponse{
			ModelKey:           modelKey,
			SupportsVision:     true,
			SuggestedMaxImages: 10,
		}, nil
	}

	if strings.Contains(strings.ToLower(prov.BaseUrl), "openrouter") || prov.ProviderType == "openrouter" {
		endpoint := strings.TrimRight(prov.BaseUrl, "/") + "/models"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return response.ProbeVisionResponse{}, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		probeClient := netx.NewSafeHTTPClientWithConfig(15*time.Second, prov.AllowPrivateNetworks == 1)
		resp, err := probeClient.Do(req)
		if err != nil {
			return response.ProbeVisionResponse{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			var data struct {
				Data []struct {
					ID           string `json:"id"`
					Architecture struct {
						Modality        string   `json:"modality"`
						InputModalities []string `json:"input_modalities"`
					} `json:"architecture"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &data); err == nil {
				for _, item := range data.Data {
					if strings.EqualFold(item.ID, modelKey) {
						hasImage := false
						for _, m := range item.Architecture.InputModalities {
							if strings.EqualFold(m, "image") {
								hasImage = true
								break
							}
						}
						if !hasImage && strings.Contains(item.Architecture.Modality, "image") {
							hasImage = true
						}
						suggImages := 0
						if hasImage {
							suggImages = 5
						}
						return response.ProbeVisionResponse{
							ModelKey:           modelKey,
							SupportsVision:     hasImage,
							SuggestedMaxImages: suggImages,
						}, nil
					}
				}
			}
		}
	}

	return response.ProbeVisionResponse{
		ModelKey:           modelKey,
		SupportsVision:     false,
		SuggestedMaxImages: 0,
	}, nil
}

func (s *llmConfigService) GetActiveChatModels(ctx context.Context) ([]response.ChatModelClientDTO, error) {
	rows, err := s.repo.ListActiveChatModelsWithProvider(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]response.ChatModelClientDTO, len(rows))
	for i, r := range rows {
		res[i] = response.ChatModelClientDTO{
			ID:             r.ModelID,
			Name:           r.ModelName,
			ModelKey:       r.ModelKey,
			IsDefault:      r.ModelIsDefault == 1,
			ContextLength:  int(r.ContextLength),
			SupportsVision: r.SupportsVision == 1,
			MaxImages:      int(r.MaxImages),
			ThinkingEnabled: r.ThinkingEnabled == 1,
			ThinkingBudget:  int(r.ThinkingBudget),
			EffortLevel:     r.EffortLevel,
			ProviderName:   r.ProviderName,
		}
	}
	return res, nil
}

func (s *llmConfigService) SeedDefaultProvidersIfEmpty(ctx context.Context) error {
	provs, err := s.repo.ListProviders(ctx)
	if err != nil {
		return err
	}
	if len(provs) > 0 {
		return nil
	}

	cipherKey := crypto.GetLLMEncryptionKey()

	openRouterKey := os.Getenv("OPEN_ROUTER_API")
	if openRouterKey == "" {
		openRouterKey = os.Getenv("OPENROUTER_API_KEY")
	}

	if openRouterKey != "" {
		encKey, err := crypto.EncryptAESGCM(openRouterKey, cipherKey)
		if err == nil {
			prov, err := s.repo.CreateProvider(ctx, sqlc.CreateLLMProviderParams{
				ID:                   "openrouter",
				Name:                 "OpenRouter",
				ProviderType:         "openrouter",
				BaseUrl:              "https://openrouter.ai/api/v1",
				ApiKeyCiphertext:     encKey,
				IsActive:             1,
				IsDefault:            1,
				CustomHeadersJson:    "{}",
				TimeoutSeconds:       120,
				MaxRetries:           3,
				RetryInitialWaitMs:   500,
				RetryMaxWaitMs:       5000,
				AllowPrivateNetworks: 0,
			})
			if err == nil {
				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "ling-flash",
					ProviderID:     prov.ID,
					Name:           "Ling 3.1 Flash (Text-Only)",
					ModelKey:       "inclusionai/ling-3.1-flash",
					ModelType:      "chat",
					IsActive:       1,
					IsDefault:      1,
					OrderIndex:     1,
					ContextLength:  32768,
					VisionMode:     "manual",
					SupportsVision: 0,
					MaxImages:      0,
					EffortLevel:    "medium",
				})

				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "gemini-flash-openrouter",
					ProviderID:     prov.ID,
					Name:           "Gemini 2.5 Flash (Vision)",
					ModelKey:       "google/gemini-2.5-flash",
					ModelType:      "chat",
					IsActive:       1,
					IsDefault:      0,
					OrderIndex:     2,
					ContextLength:  1048576,
					VisionMode:     "auto",
					SupportsVision: 1,
					MaxImages:      5,
					EffortLevel:    "medium",
				})

				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "qwen-embedding-8b",
					ProviderID:     prov.ID,
					Name:           "Qwen 3 Embedding 8B",
					ModelKey:       "qwen/qwen3-embedding-8b",
					ModelType:      "embedding",
					IsActive:       1,
					IsDefault:      0,
					OrderIndex:     10,
					ContextLength:  8192,
					VisionMode:     "manual",
					SupportsVision: 0,
					MaxImages:      0,
					EffortLevel:    "medium",
				})

				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "cohere-rerank-4-pro",
					ProviderID:     prov.ID,
					Name:           "Cohere Rerank 4 Pro",
					ModelKey:       "cohere/rerank-4-pro",
					ModelType:      "rerank",
					IsActive:       1,
					IsDefault:      1,
					OrderIndex:     20,
					ContextLength:  4096,
					VisionMode:     "manual",
					SupportsVision: 0,
					MaxImages:      0,
					EffortLevel:    "medium",
				})
			}
		}
	}

	geminiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if geminiKey == "" {
		geminiKey = os.Getenv("GEMINI_API_KEY")
	}
	if geminiKey != "" {
		encKey, err := crypto.EncryptAESGCM(geminiKey, cipherKey)
		if err == nil {
			prov, err := s.repo.CreateProvider(ctx, sqlc.CreateLLMProviderParams{
				ID:                   "gemini",
				Name:                 "Google Gemini",
				ProviderType:         "gemini",
				BaseUrl:              "https://generativelanguage.googleapis.com/v1beta",
				ApiKeyCiphertext:     encKey,
				IsActive:             1,
				IsDefault:            0,
				CustomHeadersJson:    "{}",
				TimeoutSeconds:       120,
				MaxRetries:           3,
				RetryInitialWaitMs:   500,
				RetryMaxWaitMs:       5000,
				AllowPrivateNetworks: 0,
			})
			if err == nil {
				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "gemini-2.5-flash-native",
					ProviderID:     prov.ID,
					Name:           "Google Gemini 2.5 Flash",
					ModelKey:       "gemini-2.5-flash",
					ModelType:      "chat",
					IsActive:       1,
					IsDefault:      0,
					OrderIndex:     3,
					ContextLength:  1048576,
					VisionMode:     "auto",
					SupportsVision: 1,
					MaxImages:      10,
					EffortLevel:    "medium",
				})
			}
		}
	}

	vilaoToken := os.Getenv("TOKEN")
	vilaoAPI := os.Getenv("API")
	vilaoModel := os.Getenv("MODEL")
	if vilaoToken != "" {
		encKey, err := crypto.EncryptAESGCM(vilaoToken, cipherKey)
		if err == nil {
			base := vilaoAPI
			if base == "" {
				base = "https://api.vilao.ai"
			}
			prov, err := s.repo.CreateProvider(ctx, sqlc.CreateLLMProviderParams{
				ID:                   "vilao",
				Name:                 "Vilao AI",
				ProviderType:         "openai",
				BaseUrl:              base,
				ApiKeyCiphertext:     encKey,
				IsActive:             1,
				IsDefault:            0,
				CustomHeadersJson:    "{}",
				TimeoutSeconds:       120,
				MaxRetries:           3,
				RetryInitialWaitMs:   500,
				RetryMaxWaitMs:       5000,
				AllowPrivateNetworks: 0,
			})
			if err == nil {
				mKey := vilaoModel
				if mKey == "" {
					mKey = "chib/deepseek-v4.1-flash"
				}
				_, _ = s.repo.CreateModel(ctx, sqlc.CreateLLMModelParams{
					ID:             "vilao-default",
					ProviderID:     prov.ID,
					Name:           "Vilao " + mKey,
					ModelKey:       mKey,
					ModelType:      "chat",
					IsActive:       1,
					IsDefault:      0,
					OrderIndex:     4,
					ContextLength:  65536,
					VisionMode:     "manual",
					SupportsVision: 0,
					MaxImages:      0,
					EffortLevel:    "medium",
				})
			}
		}
	}

	chains, err := s.repo.ListChains(ctx)
	if err == nil && len(chains) == 0 {
		chatChain, err := s.repo.CreateChain(ctx, sqlc.CreateChainParams{
			ID:               "default-chat-chain",
			ChainType:        string(llm.ChainTypeChat),
			Name:             "Default Chat Fallback Chain",
			Description:      "Sequential fallback chain across chat LLM providers",
			IsActive:         1,
			FailureThreshold: 3,
			CooldownSeconds:  300,
			RetryCount:       3,
		})
		if err == nil {
			chatModels, _ := s.repo.ListActiveModelsByType(ctx, "chat")
			for i, m := range chatModels {
				_, _ = s.repo.AddChainNode(ctx, sqlc.AddChainNodeParams{
					ID:       uuid.NewString(),
					ChainID:  chatChain.ID,
					ModelID:  m.ID,
					Priority: int64(i + 1),
					IsActive: 1,
				})
			}
		}

		rerankChain, err := s.repo.CreateChain(ctx, sqlc.CreateChainParams{
			ID:               "default-rerank-chain",
			ChainType:        string(llm.ChainTypeRerank),
			Name:             "Default Rerank Fallback Chain",
			Description:      "Sequential fallback chain across rerank providers",
			IsActive:         1,
			FailureThreshold: 3,
			CooldownSeconds:  300,
			RetryCount:       3,
		})
		if err == nil {
			rerankModels, _ := s.repo.ListActiveModelsByType(ctx, "rerank")
			for i, m := range rerankModels {
				_, _ = s.repo.AddChainNode(ctx, sqlc.AddChainNodeParams{
					ID:       uuid.NewString(),
					ChainID:  rerankChain.ID,
					ModelID:  m.ID,
					Priority: int64(i + 1),
					IsActive: 1,
				})
			}
		}

		embChain, err := s.repo.CreateChain(ctx, sqlc.CreateChainParams{
			ID:               "default-embedding-chain",
			ChainType:        string(llm.ChainTypeEmbedding),
			Name:             "Default Embedding Fallback Chain",
			Description:      "Sequential fallback chain across providers sharing identical embedding model",
			IsActive:         1,
			FailureThreshold: 3,
			CooldownSeconds:  300,
			RetryCount:       3,
		})
		if err == nil {
			embModels, _ := s.repo.ListActiveModelsByType(ctx, "embedding")
			var referenceKey string
			priority := int64(1)
			for _, m := range embModels {
				if referenceKey == "" {
					referenceKey = strings.TrimSpace(m.ModelKey)
				}
				if strings.TrimSpace(m.ModelKey) == referenceKey {
					_, _ = s.repo.AddChainNode(ctx, sqlc.AddChainNodeParams{
						ID:       uuid.NewString(),
						ChainID:  embChain.ID,
						ModelID:  m.ID,
						Priority: priority,
						IsActive: 1,
					})
					priority++
				}
			}
		}
	}

	return s.SyncLLMManager(ctx)
}

func (s *llmConfigService) SyncLLMManager(ctx context.Context) error {
	if s.manager == nil {
		return nil
	}

	provs, err := s.repo.ListProviders(ctx)
	if err != nil {
		return err
	}

	cipherKey := crypto.GetLLMEncryptionKey()

	for _, p := range provs {
		if p.IsActive == 0 {
			continue
		}
		rawKey, err := crypto.DecryptAESGCM(p.ApiKeyCiphertext, cipherKey)
		if err != nil {
			continue
		}

		timeout := time.Duration(p.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		initWait := time.Duration(p.RetryInitialWaitMs) * time.Millisecond
		maxWait := time.Duration(p.RetryMaxWaitMs) * time.Millisecond
		allowPriv := p.AllowPrivateNetworks == 1

		provClient := llm.NewCustomRetryClient(timeout, int(p.MaxRetries), initWait, maxWait, allowPriv)

		if p.ProviderType == "gemini" {
			gProv := llm.NewGeminiProvider(llm.GeminiConfig{
				BaseURL:              p.BaseUrl,
				APIKey:               rawKey,
				DefaultModel:         "gemini-2.5-flash",
				HTTPClient:           provClient,
				AllowPrivateNetworks: allowPriv,
			})
			s.manager.Register(p.ID, gProv)
		} else {
			oProv := llm.NewOpenAIProvider(llm.OpenAIConfig{
				BaseURL:              p.BaseUrl,
				APIKey:               rawKey,
				DefaultModel:         "gpt-4o-mini",
				HTTPClient:           provClient,
				AllowPrivateNetworks: allowPriv,
			})
			s.manager.Register(p.ID, oProv)
		}

		if p.IsDefault == 1 {
			_ = s.manager.SetDefault(p.ID)
		}
	}

	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return err
	}

	reg := s.manager.Registry()
	if reg == nil {
		reg = llm.NewCapabilityRegistry("")
		s.manager.SetRegistry(reg)
	}

	for _, m := range models {
		if m.IsActive == 0 {
			continue
		}

		cap := llm.ModelCapability{
			SupportsVision:  m.SupportsVision == 1,
			MaxImages:       int(m.MaxImages),
			SupportsTools:   true,
			ContextLength:   int(m.ContextLength),
			InputModalities: []string{"text"},
		}
		if m.SupportsVision == 1 {
			cap.InputModalities = append(cap.InputModalities, "image")
		}

		reg.RegisterCapability(m.ModelKey, cap)
		reg.RegisterCapability(m.ID, cap)

		if m.ModelType == "embedding" && m.IsDefault == 1 {
			skipAPIEmbedder := false
			if s.settings != nil {
				if raw, err := s.settings.GetAppSetting(ctx, SettingKeyEmbeddingProvider); err == nil && raw != "" {
					var prov string
					if err := jsonx.UnmarshalString(raw, &prov); err == nil && prov == EmbeddingProviderONNX {
						skipAPIEmbedder = true
					}
				}
			}
			if !skipAPIEmbedder {
				if prov, err := s.repo.GetProviderByID(ctx, m.ProviderID); err == nil {
					if rawKey, err := crypto.DecryptAESGCM(prov.ApiKeyCiphertext, cipherKey); err == nil {
						s.manager.SetEmbedder(llm.NewOpenAIEmbedder(prov.BaseUrl, rawKey, m.ModelKey, prov.AllowPrivateNetworks == 1))
					}
				}
			}
		}

		if m.ModelType == "rerank" && m.IsDefault == 1 {
			if prov, err := s.repo.GetProviderByID(ctx, m.ProviderID); err == nil {
				if rawKey, err := crypto.DecryptAESGCM(prov.ApiKeyCiphertext, cipherKey); err == nil {
					s.manager.SetReranker(llm.NewOpenRouterReranker(prov.BaseUrl, rawKey, m.ModelKey, prov.AllowPrivateNetworks == 1))
				}
			}
		}
	}

	activeChains, err := s.repo.ListChains(ctx)
	if err == nil {
		for _, chain := range activeChains {
			if chain.IsActive == 0 {
				continue
			}

			chainType := llm.ChainType(chain.ChainType)
			existingChain := s.manager.GetChain(chainType)
			var targetChain *llm.LLMChain
			if existingChain != nil && existingChain.ID == chain.ID {
				targetChain = existingChain
				targetChain.Breaker.SetPolicy(int(chain.FailureThreshold), time.Duration(chain.CooldownSeconds)*time.Second)
			} else {
				targetChain = llm.NewLLMChain(
					chain.ID,
					chainType,
					chain.Name,
					int(chain.FailureThreshold),
					time.Duration(chain.CooldownSeconds)*time.Second,
				)
			}

			nodeRows, err := s.repo.ListChainNodesWithDetails(ctx, chain.ID)
			if err != nil {
				continue
			}

			chainNodes := make([]llm.ChainNode, 0, len(nodeRows))
			for _, nr := range nodeRows {
				node := llm.ChainNode{
					ID:           nr.NodeID,
					Name:         nr.ModelName,
					ProviderID:   nr.ProviderID,
					ProviderName: nr.ProviderName,
					ModelKey:     nr.ModelKey,
					Priority:     int(nr.Priority),
					IsActive:     nr.NodeIsActive == 1,
				}

				if chainType == llm.ChainTypeChat {
					if p, err := s.manager.Get(nr.ProviderID); err == nil {
						node.Provider = p
					}
				} else if chainType == llm.ChainTypeRerank {
					if rawKey, err := crypto.DecryptAESGCM(nr.ApiKeyCiphertext, cipherKey); err == nil {
						node.Reranker = llm.NewOpenRouterReranker(nr.BaseUrl, rawKey, nr.ModelKey, nr.AllowPrivateNetworks == 1)
					}
				} else if chainType == llm.ChainTypeEmbedding {
					if rawKey, err := crypto.DecryptAESGCM(nr.ApiKeyCiphertext, cipherKey); err == nil {
						node.Embedder = llm.NewOpenAIEmbedder(nr.BaseUrl, rawKey, nr.ModelKey, nr.AllowPrivateNetworks == 1)
					}
				}

				chainNodes = append(chainNodes, node)
			}

			_ = targetChain.SetNodes(chainNodes)
			s.manager.RegisterChain(targetChain)
		}
	}

	return nil
}

func mapProviderResponse(p sqlc.LlmProvider) response.LLMProviderResponse {
	masked := "******"
	if cipherKey := crypto.GetLLMEncryptionKey(); cipherKey != nil {
		if decrypted, err := crypto.DecryptAESGCM(p.ApiKeyCiphertext, cipherKey); err == nil {
			masked = maskAPIKey(decrypted)
		}
	}

	return response.LLMProviderResponse{
		ID:                   p.ID,
		Name:                 p.Name,
		ProviderType:         p.ProviderType,
		BaseURL:              p.BaseUrl,
		MaskedAPIKey:         masked,
		IsActive:             p.IsActive == 1,
		IsDefault:            p.IsDefault == 1,
		CustomHeadersJSON:    p.CustomHeadersJson,
		TimeoutSeconds:       int(p.TimeoutSeconds),
		MaxRetries:           int(p.MaxRetries),
		RetryInitialWaitMs:   int(p.RetryInitialWaitMs),
		RetryMaxWaitMs:       int(p.RetryMaxWaitMs),
		AllowPrivateNetworks: p.AllowPrivateNetworks == 1,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func mapModelResponse(m sqlc.LlmModel) response.LLMModelResponse {
	return response.LLMModelResponse{
		ID:             m.ID,
		ProviderID:     m.ProviderID,
		Name:           m.Name,
		ModelKey:       m.ModelKey,
		ModelType:      m.ModelType,
		IsActive:       m.IsActive == 1,
		IsDefault:      m.IsDefault == 1,
		OrderIndex:     int(m.OrderIndex),
		ContextLength:  int(m.ContextLength),
		VisionMode:     m.VisionMode,
		SupportsVision: m.SupportsVision == 1,
		MaxImages:      int(m.MaxImages),
		ThinkingEnabled: m.ThinkingEnabled == 1,
		ThinkingBudget:  int(m.ThinkingBudget),
		EffortLevel:     m.EffortLevel,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "******"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (s *llmConfigService) ListChains(ctx context.Context) ([]response.LLMChainResponse, error) {
	chains, err := s.repo.ListChains(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]response.LLMChainResponse, len(chains))
	for i, c := range chains {
		resp, err := s.mapChainResponse(ctx, c)
		if err != nil {
			return nil, err
		}
		result[i] = resp
	}
	return result, nil
}

func (s *llmConfigService) GetChain(ctx context.Context, id string) (response.LLMChainResponse, error) {
	chain, err := s.repo.GetChain(ctx, id)
	if err != nil {
		return response.LLMChainResponse{}, err
	}
	return s.mapChainResponse(ctx, chain)
}

func (s *llmConfigService) CreateChain(ctx context.Context, req request.CreateLLMChainRequest) (response.LLMChainResponse, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = uuid.NewString()
	}

	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}

	failThresh := int64(req.FailureThreshold)
	if failThresh <= 0 {
		failThresh = 3
	}

	cooldownSec := int64(req.CooldownSeconds)
	if cooldownSec <= 0 {
		cooldownSec = 300
	}

	retryCount := int64(req.RetryCount)
	if retryCount <= 0 {
		retryCount = 3
	}

	created, err := s.repo.CreateChain(ctx, sqlc.CreateChainParams{
		ID:               id,
		ChainType:        req.ChainType,
		Name:             req.Name,
		Description:      req.Description,
		IsActive:         isActive,
		FailureThreshold: failThresh,
		CooldownSeconds:  cooldownSec,
		RetryCount:       retryCount,
	})
	if err != nil {
		return response.LLMChainResponse{}, err
	}

	_ = s.SyncLLMManager(ctx)
	return s.mapChainResponse(ctx, created)
}

func (s *llmConfigService) UpdateChain(ctx context.Context, id string, req request.UpdateLLMChainRequest) (response.LLMChainResponse, error) {
	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}

	failThresh := int64(req.FailureThreshold)
	if failThresh <= 0 {
		failThresh = 3
	}

	cooldownSec := int64(req.CooldownSeconds)
	if cooldownSec <= 0 {
		cooldownSec = 300
	}

	retryCount := int64(req.RetryCount)
	if retryCount <= 0 {
		retryCount = 3
	}

	updated, err := s.repo.UpdateChain(ctx, sqlc.UpdateChainParams{
		ID:               id,
		Name:             req.Name,
		Description:      req.Description,
		IsActive:         isActive,
		FailureThreshold: failThresh,
		CooldownSeconds:  cooldownSec,
		RetryCount:       retryCount,
	})
	if err != nil {
		return response.LLMChainResponse{}, err
	}

	_ = s.SyncLLMManager(ctx)
	return s.mapChainResponse(ctx, updated)
}

func (s *llmConfigService) DeleteChain(ctx context.Context, id string) error {
	if err := s.repo.DeleteChain(ctx, id); err != nil {
		return err
	}
	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) AddChainNode(ctx context.Context, chainID string, req request.AddChainNodeRequest) (response.LLMChainNodeResponse, error) {
	chain, err := s.repo.GetChain(ctx, chainID)
	if err != nil {
		return response.LLMChainNodeResponse{}, fmt.Errorf("chain not found: %w", err)
	}

	model, err := s.repo.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return response.LLMChainNodeResponse{}, fmt.Errorf("model not found: %w", err)
	}

	if model.ModelType != chain.ChainType {
		return response.LLMChainNodeResponse{}, fmt.Errorf("model type '%s' does not match chain type '%s'", model.ModelType, chain.ChainType)
	}

	existingNodes, err := s.repo.ListChainNodesWithDetails(ctx, chainID)
	if err != nil {
		return response.LLMChainNodeResponse{}, err
	}

	if chain.ChainType == string(llm.ChainTypeEmbedding) && len(existingNodes) > 0 {
		referenceKey := strings.TrimSpace(existingNodes[0].ModelKey)
		candidateKey := strings.TrimSpace(model.ModelKey)
		if referenceKey != candidateKey {
			return response.LLMChainNodeResponse{}, fmt.Errorf("embedding chain model mismatch: all nodes must use identical model '%s', but model '%s' uses '%s'", referenceKey, model.Name, candidateKey)
		}
	}

	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}

	priority := int64(req.Priority)
	if priority <= 0 {
		priority = int64(len(existingNodes) + 1)
	}

	nodeID := uuid.NewString()
	_, err = s.repo.AddChainNode(ctx, sqlc.AddChainNodeParams{
		ID:       nodeID,
		ChainID:  chainID,
		ModelID:  req.ModelID,
		Priority: priority,
		IsActive: isActive,
	})
	if err != nil {
		return response.LLMChainNodeResponse{}, err
	}

	_ = s.SyncLLMManager(ctx)

	updatedNodes, err := s.repo.ListChainNodesWithDetails(ctx, chainID)
	if err != nil {
		return response.LLMChainNodeResponse{}, err
	}

	for _, un := range updatedNodes {
		if un.NodeID == nodeID {
			return mapChainNodeResponse(un, llm.NodeHealthStatus{
				NodeID: nodeID,
				State:  llm.StateClosed,
			}), nil
		}
	}

	return response.LLMChainNodeResponse{}, errors.New("created node row missing")
}

func (s *llmConfigService) RemoveChainNode(ctx context.Context, chainID string, nodeID string) error {
	if err := s.repo.RemoveChainNode(ctx, nodeID); err != nil {
		return err
	}
	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) UpdateChainNodePriority(ctx context.Context, chainID string, nodeID string, req request.UpdateChainNodePriorityRequest) error {
	isActive := int64(1)
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}

	priority := int64(req.Priority)
	if priority <= 0 {
		priority = 1
	}

	if err := s.repo.UpdateChainNodePriority(ctx, sqlc.UpdateChainNodePriorityParams{
		ID:       nodeID,
		Priority: priority,
		IsActive: isActive,
	}); err != nil {
		return err
	}

	_ = s.SyncLLMManager(ctx)
	return nil
}

func (s *llmConfigService) ResetNodeCircuitBreaker(ctx context.Context, chainID string, nodeID string) error {
	chain, err := s.repo.GetChain(ctx, chainID)
	if err != nil {
		return err
	}

	if s.manager != nil {
		c := s.manager.GetChain(llm.ChainType(chain.ChainType))
		if c != nil && c.Breaker != nil {
			c.Breaker.Reset(nodeID)
		}
	}
	return nil
}

func (s *llmConfigService) mapChainResponse(ctx context.Context, chain sqlc.LlmChain) (response.LLMChainResponse, error) {
	nodeRows, err := s.repo.ListChainNodesWithDetails(ctx, chain.ID)
	if err != nil {
		return response.LLMChainResponse{}, err
	}

	var healthStatuses map[string]llm.NodeHealthStatus
	if s.manager != nil {
		if c := s.manager.GetChain(llm.ChainType(chain.ChainType)); c != nil && c.Breaker != nil {
			healthStatuses = c.Breaker.AllStatuses()
		}
	}

	nodes := make([]response.LLMChainNodeResponse, len(nodeRows))
	for i, r := range nodeRows {
		var st llm.NodeHealthStatus
		if healthStatuses != nil {
			st = healthStatuses[r.NodeID]
		}
		if st.NodeID == "" {
			st.NodeID = r.NodeID
			st.State = llm.StateClosed
		}
		nodes[i] = mapChainNodeResponse(r, st)
	}

	return response.LLMChainResponse{
		ID:               chain.ID,
		ChainType:        chain.ChainType,
		Name:             chain.Name,
		Description:      chain.Description,
		IsActive:         chain.IsActive == 1,
		FailureThreshold: int(chain.FailureThreshold),
		CooldownSeconds:  int(chain.CooldownSeconds),
		RetryCount:       int(chain.RetryCount),
		Nodes:            nodes,
		CreatedAt:        chain.CreatedAt,
		UpdatedAt:        chain.UpdatedAt,
	}, nil
}

func mapChainNodeResponse(row sqlc.ListChainNodesWithDetailsRow, status llm.NodeHealthStatus) response.LLMChainNodeResponse {
	var untilStr *string
	if status.CooldownUntil != nil {
		s := status.CooldownUntil.UTC().Format(time.RFC3339)
		untilStr = &s
	}

	state := string(status.State)
	if state == "" {
		state = string(llm.StateClosed)
	}

	return response.LLMChainNodeResponse{
		ID:                       row.NodeID,
		ChainID:                  row.ChainID,
		ModelID:                  row.ModelID,
		Priority:                 int(row.Priority),
		IsActive:                 row.NodeIsActive == 1,
		ModelName:                row.ModelName,
		ModelKey:                 row.ModelKey,
		ModelType:                row.ModelType,
		ContextLength:            int(row.ContextLength),
		SupportsVision:           row.SupportsVision == 1,
		MaxImages:                int(row.MaxImages),
		ProviderID:               row.ProviderID,
		ProviderName:             row.ProviderName,
		ProviderType:             row.ProviderType,
		BaseURL:                  row.BaseUrl,
		TimeoutSeconds:           int(row.TimeoutSeconds),
		MaxRetries:               int(row.MaxRetries),
		HealthState:              state,
		ConsecutiveFailures:      status.ConsecutiveFailures,
		CooldownUntil:            untilStr,
		RemainingCooldownSeconds: status.RemainingCooldownSeconds,
	}
}

func buildTestUserMessage(prompt string, images []string) llm.Message {
	msg := llm.UserMessage(prompt)
	if len(images) > 0 {
		parts := make([]llm.ContentPart, 0, 1+len(images))
		parts = append(parts, llm.ContentPart{
			Type: llm.ContentPartText,
			Text: prompt,
		})
		for _, img := range images {
			parts = append(parts, llm.ContentPart{
				Type: llm.ContentPartImage,
				Image: &llm.ImageSource{
					URL: img,
				},
			})
		}
		msg.Parts = parts
	}
	return msg
}

// TestModel executes a live test against an individual model or fallback chain.
func (s *llmConfigService) TestModel(ctx context.Context, req request.TestLLMModelRequest) (*response.TestLLMModelResponse, error) {
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	if strings.TrimSpace(req.ChainID) != "" {
		chain, err := s.repo.GetChain(timeoutCtx, req.ChainID)
		if err != nil {
			return nil, fmt.Errorf("chain not found: %w", err)
		}

		c := s.manager.GetChain(llm.ChainType(chain.ChainType))
		if c == nil || len(c.Nodes) == 0 {
			return nil, errors.New("chain has no active nodes registered")
		}

		if chain.ChainType == string(llm.ChainTypeChat) {
			sysPrompt := req.SystemPrompt
			if sysPrompt == "" {
				sysPrompt = "You are an AI assistant. Answer concisely."
			}
			chatReq := &llm.ChatRequest{
				Messages: []llm.Message{
					llm.SystemMessage(sysPrompt),
					buildTestUserMessage(req.Prompt, req.Images),
				},
				Temperature: req.Temperature,
				MaxTokens:   req.MaxTokens,
			}

			resp, node, err := c.ExecuteChat(timeoutCtx, chatReq)
			duration := time.Since(start).Milliseconds()
			if err != nil {
				return &response.TestLLMModelResponse{
					Success:   false,
					ChainID:   chain.ID,
					Error:     err.Error(),
					LatencyMs: duration,
				}, nil
			}

			rawMap := make(map[string]any)
			rawMap["content"] = resp.Message.Content
			rawMap["finish_reason"] = resp.FinishReason
			rawMap["usage"] = resp.Usage
			rawMap["node"] = node.Name

			return &response.TestLLMModelResponse{
				Success:          true,
				ChainID:          chain.ID,
				NodeUsed:         fmt.Sprintf("%s (%s)", node.Name, node.ProviderName),
				ModelName:        node.Name,
				ModelKey:         node.ModelKey,
				ProviderName:     node.ProviderName,
				Content:          resp.Message.Content,
				ReasoningContent: resp.Message.ReasoningContent,
				LatencyMs:        duration,
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				ReasoningTokens:  resp.Usage.ReasoningTokens,
				TotalTokens:      resp.Usage.TotalTokens,
				FinishReason:     resp.FinishReason,
				RawResponse:      rawMap,
			}, nil
		}

		if chain.ChainType == string(llm.ChainTypeEmbedding) {
			vecs, node, err := c.ExecuteEmbedding(timeoutCtx, []string{req.Prompt})
			duration := time.Since(start).Milliseconds()
			if err != nil {
				return &response.TestLLMModelResponse{
					Success:   false,
					ChainID:   chain.ID,
					Error:     err.Error(),
					LatencyMs: duration,
				}, nil
			}

			dims := 0
			if len(vecs) > 0 {
				dims = len(vecs[0])
			}
			rawMap := map[string]any{
				"vector_count": len(vecs),
				"dimensions":   dims,
			}

			return &response.TestLLMModelResponse{
				Success:      true,
				ChainID:      chain.ID,
				NodeUsed:     fmt.Sprintf("%s (%s)", node.Name, node.ProviderName),
				ModelName:    node.Name,
				ModelKey:     node.ModelKey,
				ProviderName: node.ProviderName,
				Content:      fmt.Sprintf("Embedding generated successfully with %d dimensions.", dims),
				LatencyMs:    duration,
				RawResponse:  rawMap,
			}, nil
		}

		if chain.ChainType == string(llm.ChainTypeRerank) {
			res, node, err := c.ExecuteRerank(timeoutCtx, req.Prompt, defaultRerankTestDocs, nil)
			duration := time.Since(start).Milliseconds()
			if err != nil {
				return &response.TestLLMModelResponse{
					Success:   false,
					ChainID:   chain.ID,
					Error:     err.Error(),
					LatencyMs: duration,
				}, nil
			}

			rawMap := map[string]any{
				"results": res,
			}
			var sb strings.Builder
			for i, r := range res {
				sb.WriteString(fmt.Sprintf("%d. [Score: %.4f] %s\n", i+1, r.RelevanceScore, r.Document))
			}

			return &response.TestLLMModelResponse{
				Success:      true,
				ChainID:      chain.ID,
				NodeUsed:     fmt.Sprintf("%s (%s)", node.Name, node.ProviderName),
				ModelName:    node.Name,
				ModelKey:     node.ModelKey,
				ProviderName: node.ProviderName,
				Content:      sb.String(),
				LatencyMs:    duration,
				RawResponse:  rawMap,
			}, nil
		}
	}

	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		activeModels, err := s.repo.ListActiveChatModelsWithProvider(timeoutCtx)
		if err != nil || len(activeModels) == 0 {
			return nil, errors.New("no active models available to test")
		}
		modelID = activeModels[0].ModelID
	}

	model, err := s.repo.GetModelByID(timeoutCtx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	prov, err := s.repo.GetProviderByID(timeoutCtx, model.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	cipherKey := crypto.GetLLMEncryptionKey()
	rawKey, err := crypto.DecryptAESGCM(prov.ApiKeyCiphertext, cipherKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt provider api key: %w", err)
	}

	headers := make(map[string]string)
	if prov.CustomHeadersJson != "" {
		_ = json.Unmarshal([]byte(prov.CustomHeadersJson), &headers)
	}

	if model.ModelType == "chat" {
		var provider llm.Provider
		if prov.ProviderType == "gemini" {
			provider = llm.NewGeminiProvider(llm.GeminiConfig{
				APIKey:       rawKey,
				DefaultModel: model.ModelKey,
			})
		} else {
			provider = llm.NewOpenAIProvider(llm.OpenAIConfig{
				BaseURL:              prov.BaseUrl,
				APIKey:               rawKey,
				DefaultModel:         model.ModelKey,
				CustomHeaders:        headers,
				AllowPrivateNetworks: prov.AllowPrivateNetworks == 1,
			})
		}

		sysPrompt := req.SystemPrompt
		if sysPrompt == "" {
			sysPrompt = "You are an AI assistant. Answer concisely."
		}

		chatReq := &llm.ChatRequest{
			Model: model.ModelKey,
			Messages: []llm.Message{
				llm.SystemMessage(sysPrompt),
				buildTestUserMessage(req.Prompt, req.Images),
			},
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}

		resp, err := provider.Chat(timeoutCtx, chatReq)
		duration := time.Since(start).Milliseconds()
		if err != nil {
			return &response.TestLLMModelResponse{
				Success:      false,
				ModelID:      model.ID,
				ModelName:    model.Name,
				ModelKey:     model.ModelKey,
				ProviderName: prov.Name,
				Error:        err.Error(),
				LatencyMs:    duration,
			}, nil
		}

		rawMap := make(map[string]any)
		rawMap["content"] = resp.Message.Content
		rawMap["finish_reason"] = resp.FinishReason
		rawMap["usage"] = resp.Usage

		return &response.TestLLMModelResponse{
			Success:          true,
			ModelID:          model.ID,
			ModelName:        model.Name,
			ModelKey:         model.ModelKey,
			ProviderName:     prov.Name,
			Content:          resp.Message.Content,
			ReasoningContent: resp.Message.ReasoningContent,
			LatencyMs:        duration,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			ReasoningTokens:  resp.Usage.ReasoningTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			FinishReason:     resp.FinishReason,
			RawResponse:      rawMap,
		}, nil
	}

	if model.ModelType == "embedding" {
		embedder := llm.NewOpenAIEmbedder(prov.BaseUrl, rawKey, model.ModelKey, prov.AllowPrivateNetworks == 1)
		vecs, err := embedder.Embed(timeoutCtx, []string{req.Prompt})
		duration := time.Since(start).Milliseconds()
		if err != nil {
			return &response.TestLLMModelResponse{
				Success:      false,
				ModelID:      model.ID,
				ModelName:    model.Name,
				ModelKey:     model.ModelKey,
				ProviderName: prov.Name,
				Error:        err.Error(),
				LatencyMs:    duration,
			}, nil
		}

		dims := 0
		if len(vecs) > 0 {
			dims = len(vecs[0])
		}
		rawMap := map[string]any{
			"vector_count": len(vecs),
			"dimensions":   dims,
		}

		return &response.TestLLMModelResponse{
			Success:      true,
			ModelID:      model.ID,
			ModelName:    model.Name,
			ModelKey:     model.ModelKey,
			ProviderName: prov.Name,
			Content:      fmt.Sprintf("Embedding generated successfully with %d dimensions.", dims),
			LatencyMs:    duration,
			RawResponse:  rawMap,
		}, nil
	}

	if model.ModelType == "rerank" {
		reranker := llm.NewOpenRouterReranker(prov.BaseUrl, rawKey, model.ModelKey, prov.AllowPrivateNetworks == 1)
		res, err := reranker.Rerank(timeoutCtx, req.Prompt, defaultRerankTestDocs, nil)
		duration := time.Since(start).Milliseconds()
		if err != nil {
			return &response.TestLLMModelResponse{
				Success:      false,
				ModelID:      model.ID,
				ModelName:    model.Name,
				ModelKey:     model.ModelKey,
				ProviderName: prov.Name,
				Error:        err.Error(),
				LatencyMs:    duration,
			}, nil
		}

		rawMap := map[string]any{
			"results": res,
		}
		var sb strings.Builder
		for i, r := range res {
			sb.WriteString(fmt.Sprintf("%d. [Score: %.4f] %s\n", i+1, r.RelevanceScore, r.Document))
		}

		return &response.TestLLMModelResponse{
			Success:      true,
			ModelID:      model.ID,
			ModelName:    model.Name,
			ModelKey:     model.ModelKey,
			ProviderName: prov.Name,
			Content:      sb.String(),
			LatencyMs:    duration,
			RawResponse:  rawMap,
		}, nil
	}

	return nil, fmt.Errorf("unsupported model type: %s", model.ModelType)
}
