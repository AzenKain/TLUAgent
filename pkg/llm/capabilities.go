package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ModelCapability describes the operational features of an LLM.
type ModelCapability struct {
	SupportsVision    bool     `json:"supports_vision"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsReasoning bool     `json:"supports_reasoning"`
	ContextLength     int      `json:"context_length,omitempty"`
	MaxImages         int      `json:"max_images,omitempty"`
	InputModalities   []string `json:"input_modalities,omitempty"`
}

// VisionHandlingStrategy determines how to handle requests with images sent to non-vision models.
type VisionHandlingStrategy string

const (
	// VisionStrategyFallback routes the request to a vision-capable fallback model.
	VisionStrategyFallback VisionHandlingStrategy = "fallback"
	// VisionStrategyDegrade strips image parts and appends a text notice, allowing text-only completion.
	VisionStrategyDegrade VisionHandlingStrategy = "degrade"
	// VisionStrategyReject returns an explicit error explaining the model lacks vision capabilities.
	VisionStrategyReject VisionHandlingStrategy = "reject"
)

// CapabilityRegistry caches and detects model capabilities across providers.
type CapabilityRegistry struct {
	mu            sync.RWMutex
	cache         map[string]ModelCapability
	openRouterKey string
	httpClient    *http.Client
}

// NewCapabilityRegistry creates a registry instance.
func NewCapabilityRegistry(openRouterKey string) *CapabilityRegistry {
	return &CapabilityRegistry{
		cache:         make(map[string]ModelCapability),
		openRouterKey: openRouterKey,
		httpClient:    NewSafeRetryClient(10*time.Second, 2, false),
	}
}

// RegisterCapability registers or overrides capabilities for a model.
func (r *CapabilityRegistry) RegisterCapability(model string, cap ModelCapability) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[strings.ToLower(strings.TrimSpace(model))] = cap
}

// HasImages checks whether any message in the request contains an image.
func HasImages(messages []Message) bool {
	return CountImages(messages) > 0
}

// CountImages returns the total count of images in the messages.
func CountImages(messages []Message) int {
	count := 0
	for _, m := range messages {
		for _, p := range m.Parts {
			if p.Type == ContentPartImage && p.Image != nil && (p.Image.URL != "" || p.Image.Data != "") {
				count++
			}
		}
	}
	return count
}

// SanitizeForTextOnly strips image parts from messages and replaces them with a text indicator.
func SanitizeForTextOnly(messages []Message) []Message {
	clean := make([]Message, len(messages))
	for i, m := range messages {
		cleanMsg := m
		if len(m.Parts) > 0 {
			var newParts []ContentPart
			var textParts []string
			hasImage := false

			for _, p := range m.Parts {
				if p.Type == ContentPartImage {
					hasImage = true
				} else if p.Type == ContentPartText {
					newParts = append(newParts, p)
					if p.Text != "" {
						textParts = append(textParts, p.Text)
					}
				}
			}

			if hasImage {
				notice := "[Attached image omitted: current model only supports text]"
				newParts = append(newParts, ContentPart{
					Type: ContentPartText,
					Text: notice,
				})
				textParts = append(textParts, notice)
			}

			cleanMsg.Parts = newParts
			cleanMsg.Content = strings.Join(textParts, "\n")
		}
		clean[i] = cleanMsg
	}
	return clean
}

// DetectCapabilities returns the capabilities of a given model from cache/registered models or dynamic discovery.
func (r *CapabilityRegistry) DetectCapabilities(ctx context.Context, model string) ModelCapability {
	if model == "" {
		return ModelCapability{SupportsVision: false, SupportsTools: true, MaxImages: 0}
	}

	modelLower := strings.ToLower(strings.TrimSpace(model))

	r.mu.RLock()
	cap, exists := r.cache[modelLower]
	r.mu.RUnlock()
	if exists {
		return cap
	}

	cap = ModelCapability{SupportsVision: false, SupportsTools: true, MaxImages: 0}

	if strings.Contains(modelLower, "/") && r.openRouterKey != "" {
		if discovered, ok := r.fetchOpenRouterCapability(ctx, model); ok {
			cap = discovered
		}
	}

	r.mu.Lock()
	r.cache[modelLower] = cap
	r.mu.Unlock()

	return cap
}

// SupportsVision checks whether a model supports image input.
func (r *CapabilityRegistry) SupportsVision(ctx context.Context, model string) bool {
	return r.DetectCapabilities(ctx, model).SupportsVision
}

// fetchOpenRouterCapability dynamically queries OpenRouter /models endpoint.
func (r *CapabilityRegistry) fetchOpenRouterCapability(ctx context.Context, modelID string) (ModelCapability, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return ModelCapability{}, false
	}

	req.Header.Set("Authorization", "Bearer "+r.openRouterKey)
	resp, err := r.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return ModelCapability{}, false
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelCapability{}, false
	}

	var data struct {
		Data []struct {
			ID           string `json:"id"`
			ContextLen   int    `json:"context_length"`
			Architecture struct {
				Modality        string   `json:"modality"`
				InputModalities []string `json:"input_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return ModelCapability{}, false
	}

	for _, item := range data.Data {
		if strings.EqualFold(item.ID, modelID) {
			hasImage := false
			for _, mod := range item.Architecture.InputModalities {
				if strings.EqualFold(mod, "image") {
					hasImage = true
					break
				}
			}
			if !hasImage && strings.Contains(item.Architecture.Modality, "image") {
				hasImage = true
			}

			maxImages := 0
			if hasImage {
				maxImages = 5
			}

			return ModelCapability{
				SupportsVision:  hasImage,
				SupportsTools:   true,
				ContextLength:   item.ContextLen,
				MaxImages:       maxImages,
				InputModalities: item.Architecture.InputModalities,
			}, true
		}
	}

	return ModelCapability{}, false
}

// AdaptiveChatRequest inspects if the target model supports vision and checks image limits.
func AdaptiveChatRequest(
	ctx context.Context,
	registry *CapabilityRegistry,
	req *ChatRequest,
	targetModel string,
	strategy VisionHandlingStrategy,
	visionFallbackModel string,
) (*ChatRequest, error) {
	imgCount := CountImages(req.Messages)
	if imgCount == 0 {
		return req, nil
	}

	cap := registry.DetectCapabilities(ctx, targetModel)
	if cap.SupportsVision {
		if cap.MaxImages > 0 && imgCount > cap.MaxImages {
			return nil, fmt.Errorf("request contains %d images, exceeding maximum allowed limit of %d for model '%s'", imgCount, cap.MaxImages, targetModel)
		}
		return req, nil
	}

	switch strategy {
	case VisionStrategyFallback:
		if visionFallbackModel == "" {
			visionFallbackModel = "gemini-2.5-flash"
		}
		cloned := *req
		cloned.Model = visionFallbackModel
		return &cloned, nil

	case VisionStrategyDegrade:
		cloned := *req
		cloned.Messages = SanitizeForTextOnly(req.Messages)
		return &cloned, nil

	case VisionStrategyReject:
		fallthrough
	default:
		return nil, fmt.Errorf("model '%s' is a text-only model and does not support image input. Please select a vision-capable model", targetModel)
	}
}
