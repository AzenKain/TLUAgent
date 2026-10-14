package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type ModelCapability struct {
	SupportsVision    bool     `json:"supports_vision"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsReasoning bool     `json:"supports_reasoning"`
	ContextLength     int      `json:"context_length,omitempty"`
	MaxImages         int      `json:"max_images,omitempty"`
	InputModalities   []string `json:"input_modalities,omitempty"`
}

type VisionHandlingStrategy string

const (
	// VisionStrategyFallback routes the request to a vision-capable fallback model.
	VisionStrategyFallback VisionHandlingStrategy = "fallback"
	// VisionStrategyDegrade strips image parts and appends a text notice, allowing text-only completion.
	VisionStrategyDegrade VisionHandlingStrategy = "degrade"
	// VisionStrategyReject returns an explicit error explaining the model lacks vision capabilities.
	VisionStrategyReject VisionHandlingStrategy = "reject"
)

type CapabilityRegistry struct {
	mu    sync.RWMutex
	cache map[string]ModelCapability
}

// NewCapabilityRegistry creates a registry instance.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		cache: make(map[string]ModelCapability),
	}
}

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

// DetectCapabilities returns the capabilities of a given model from cache.
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

	return ModelCapability{SupportsVision: false, SupportsTools: true, MaxImages: 0}
}

// SupportsVision checks whether a model supports image input.
func (r *CapabilityRegistry) SupportsVision(ctx context.Context, model string) bool {
	return r.DetectCapabilities(ctx, model).SupportsVision
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
			return nil, fmt.Errorf("model '%s' does not support vision and no vision fallback model is configured in admin settings", targetModel)
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
