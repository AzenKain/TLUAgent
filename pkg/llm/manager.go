package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tluagent-web/pkg/config"
)

// Manager manages multiple LLM providers with automatic fallback and routing.
type Manager struct {
	mu                  sync.RWMutex
	providers           map[string]Provider
	defaultProvider     string
	embedder            Embedder
	reranker            Reranker
	registry            *CapabilityRegistry
	visionStrategy      VisionHandlingStrategy
	visionFallbackModel string
	chains              map[ChainType]*LLMChain
}

// NewManager creates an empty LLM Manager.
func NewManager() *Manager {
	return &Manager{
		providers:           make(map[string]Provider),
		visionStrategy:      VisionStrategyFallback,
		visionFallbackModel: "gemini-2.5-flash",
		chains:              make(map[ChainType]*LLMChain),
	}
}

// SetRegistry sets the capability registry.
func (m *Manager) SetRegistry(reg *CapabilityRegistry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = reg
}

// Registry returns the capability registry.
func (m *Manager) Registry() *CapabilityRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry
}

// SetVisionStrategy configures how requests containing images are routed.
func (m *Manager) SetVisionStrategy(strategy VisionHandlingStrategy, fallbackModel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.visionStrategy = strategy
	if fallbackModel != "" {
		m.visionFallbackModel = fallbackModel
	}
}

// SetEmbedder sets the default Embedder.
func (m *Manager) SetEmbedder(e Embedder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedder = e
}

// Embedder returns the default Embedder.
func (m *Manager) Embedder() Embedder {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embedder
}

// SetReranker sets the default Reranker.
func (m *Manager) SetReranker(r Reranker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reranker = r
}

// Reranker returns the default Reranker.
func (m *Manager) Reranker() Reranker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reranker
}

// Register registers a provider by unique name.
func (m *Manager) Register(name string, p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	normalized := strings.ToLower(strings.TrimSpace(name))
	m.providers[normalized] = p
	if m.defaultProvider == "" {
		m.defaultProvider = normalized
	}
}

// SetDefault sets the default provider name.
func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	normalized := strings.ToLower(strings.TrimSpace(name))
	if _, ok := m.providers[normalized]; !ok {
		return fmt.Errorf("provider '%s' is not registered", name)
	}
	m.defaultProvider = normalized
	return nil
}

// Get retrieves a provider by name.
func (m *Manager) Get(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	normalized := strings.ToLower(strings.TrimSpace(name))
	p, ok := m.providers[normalized]
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not registered", name)
	}
	return p, nil
}

// Default returns the default provider.
func (m *Manager) Default() (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultProvider == "" {
		return nil, errors.New("no default provider configured")
	}
	p, ok := m.providers[m.defaultProvider]
	if !ok {
		return nil, fmt.Errorf("default provider '%s' is not found", m.defaultProvider)
	}
	return p, nil
}

// List returns a list of all registered provider names.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// RegisterChain registers an execution chain for a specific workload type.
func (m *Manager) RegisterChain(chain *LLMChain) {
	if chain == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chains[chain.Type] = chain
}

// GetChain retrieves an execution chain by workload type.
func (m *Manager) GetChain(chainType ChainType) *LLMChain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chains[chainType]
}

func (m *Manager) defaultLocked() (Provider, error) {
	if m.defaultProvider == "" {
		return nil, errors.New("no default provider configured")
	}
	p, ok := m.providers[m.defaultProvider]
	if !ok {
		return nil, fmt.Errorf("default provider '%s' is not found", m.defaultProvider)
	}
	return p, nil
}

// Chat executes chat completion, routing through the chat chain if configured or falling back to default provider.
func (m *Manager) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	targetModel := req.Model
	adaptedReq := req
	var err error

	m.mu.RLock()
	chatChain := m.chains[ChainTypeChat]
	p, defErr := m.defaultLocked()
	reg := m.registry
	strat := m.visionStrategy
	fbModel := m.visionFallbackModel
	m.mu.RUnlock()

	if targetModel == "" && p != nil {
		targetModel = p.DefaultModel()
	}

	if reg != nil {
		adaptedReq, err = AdaptiveChatRequest(ctx, reg, req, targetModel, strat, fbModel)
		if err != nil {
			return nil, err
		}
	}

	if chatChain != nil && chatChain.IsActive && len(chatChain.Nodes) > 0 {
		resp, _, chainErr := chatChain.ExecuteChat(ctx, adaptedReq)
		if chainErr == nil {
			return resp, nil
		}
	}

	if defErr != nil {
		return nil, defErr
	}
	return p.Chat(ctx, adaptedReq)
}

// ChatStream executes streaming chat, routing through the chat chain if configured or falling back to default provider.
func (m *Manager) ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	targetModel := req.Model
	adaptedReq := req
	var err error

	m.mu.RLock()
	chatChain := m.chains[ChainTypeChat]
	p, defErr := m.defaultLocked()
	reg := m.registry
	strat := m.visionStrategy
	fbModel := m.visionFallbackModel
	m.mu.RUnlock()

	if targetModel == "" && p != nil {
		targetModel = p.DefaultModel()
	}

	if reg != nil {
		adaptedReq, err = AdaptiveChatRequest(ctx, reg, req, targetModel, strat, fbModel)
		if err != nil {
			return nil, err
		}
	}

	if chatChain != nil && chatChain.IsActive && len(chatChain.Nodes) > 0 {
		stream, _, chainErr := chatChain.ExecuteChatStream(ctx, adaptedReq)
		if chainErr == nil {
			return stream, nil
		}
	}

	if defErr != nil {
		return nil, defErr
	}
	return p.ChatStream(ctx, adaptedReq)
}

// ExecuteRerank evaluates document relevance via the rerank chain or direct reranker.
func (m *Manager) ExecuteRerank(ctx context.Context, query string, documents []string, topN *int) ([]RerankResult, error) {
	m.mu.RLock()
	rerankChain := m.chains[ChainTypeRerank]
	directReranker := m.reranker
	m.mu.RUnlock()

	if rerankChain != nil && rerankChain.IsActive && len(rerankChain.Nodes) > 0 {
		results, _, err := rerankChain.ExecuteRerank(ctx, query, documents, topN)
		if err == nil {
			return results, nil
		}
	}

	if directReranker != nil {
		return directReranker.Rerank(ctx, query, documents, topN)
	}

	return nil, errors.New("no reranker or rerank chain available")
}

// ExecuteEmbedding creates vector representations via the embedding chain or direct embedder.
func (m *Manager) ExecuteEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.RLock()
	embChain := m.chains[ChainTypeEmbedding]
	directEmbedder := m.embedder
	m.mu.RUnlock()

	if embChain != nil && embChain.IsActive && len(embChain.Nodes) > 0 {
		vectors, _, err := embChain.ExecuteEmbedding(ctx, texts)
		if err == nil {
			return vectors, nil
		}
	}

	if directEmbedder != nil {
		return directEmbedder.Embed(ctx, texts)
	}

	return nil, errors.New("no embedder or embedding chain available")
}

// ChatWithFallback executes Chat across a list of providers until one succeeds.
func (m *Manager) ChatWithFallback(ctx context.Context, req *ChatRequest, providerOrder ...string) (*ChatResponse, error) {
	if len(providerOrder) == 0 {
		return m.Chat(ctx, req)
	}

	var lastErr error
	for _, name := range providerOrder {
		p, err := m.Get(name)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = fmt.Errorf("provider '%s' failed: %w", name, err)
	}

	return nil, fmt.Errorf("all fallback providers failed. Last error: %w", lastErr)
}

// ChatStreamWithFallback attempts to initiate a stream across providers until one connects.
func (m *Manager) ChatStreamWithFallback(ctx context.Context, req *ChatRequest, providerOrder ...string) (StreamReader, error) {
	if len(providerOrder) == 0 {
		return m.ChatStream(ctx, req)
	}

	var lastErr error
	for _, name := range providerOrder {
		p, err := m.Get(name)
		if err != nil {
			lastErr = err
			continue
		}

		stream, err := p.ChatStream(ctx, req)
		if err == nil {
			return stream, nil
		}
		lastErr = fmt.Errorf("provider '%s' stream failed: %w", name, err)
	}

	return nil, fmt.Errorf("all fallback stream providers failed. Last error: %w", lastErr)
}

// NewManagerFromEnv initializes and registers providers from environment variables.
func NewManagerFromEnv() (*Manager, error) {
	_ = config.LoadEnv()

	manager := NewManager()

	token := os.Getenv("TOKEN")
	if token == "" {
		token = os.Getenv("OPENAI_API_KEY")
	}

	apiURL := os.Getenv("API")
	if apiURL == "" {
		apiURL = os.Getenv("OPENAI_BASE_URL")
	}

	model := os.Getenv("MODEL")
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}

	if token != "" || apiURL != "" {
		if apiURL == "" {
			apiURL = "https://api.openai.com/v1"
		}
		if model == "" {
			model = "gpt-4o-mini"
		}

		retryClient := NewRetryClient(120*time.Second, 2)
		openaiProvider := NewOpenAIProvider(OpenAIConfig{
			BaseURL:      apiURL,
			APIKey:       token,
			DefaultModel: model,
			HTTPClient:   retryClient,
		})

		manager.Register("openai", openaiProvider)
		if strings.Contains(apiURL, "vilao") {
			manager.Register("vilao", openaiProvider)
		}
	}

	geminiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if geminiKey == "" {
		geminiKey = os.Getenv("GEMINI_API_KEY")
	}

	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash"
	}

	if geminiKey != "" {
		retryClient := NewRetryClient(120*time.Second, 2)
		geminiProvider := NewGeminiProvider(GeminiConfig{
			APIKey:       geminiKey,
			DefaultModel: geminiModel,
			HTTPClient:   retryClient,
		})
		manager.Register("gemini", geminiProvider)
		manager.Register("google", geminiProvider)
	}

	openRouterKey := os.Getenv("OPEN_ROUTER_API")
	if openRouterKey == "" {
		openRouterKey = os.Getenv("OPENROUTER_API_KEY")
	}

	if openRouterKey != "" {
		retryClient := NewRetryClient(120*time.Second, 2)
		openRouterProvider := NewOpenAIProvider(OpenAIConfig{
			BaseURL:      "https://openrouter.ai/api/v1",
			APIKey:       openRouterKey,
			DefaultModel: "qwen/qwen3.8-max",
			HTTPClient:   retryClient,
		})
		manager.Register("openrouter", openRouterProvider)

		embedModel := os.Getenv("EMBEDDING_MODEL")
		if embedModel == "" {
			embedModel = "qwen/qwen3-embedding-8b"
		}
		manager.SetEmbedder(NewOpenAIEmbedder("https://openrouter.ai/api/v1", openRouterKey, embedModel))

		rerankModel := os.Getenv("RERANK_MODEL")
		if rerankModel == "" {
			rerankModel = "cohere/rerank-4-pro"
		}
		manager.SetReranker(NewOpenRouterReranker("https://openrouter.ai/api/v1", openRouterKey, rerankModel))
	}

	reg := NewCapabilityRegistry(openRouterKey)
	manager.SetRegistry(reg)

	strat := VisionHandlingStrategy(os.Getenv("LLM_VISION_STRATEGY"))
	if strat == "" {
		strat = VisionStrategyFallback
	}
	fallbackModel := os.Getenv("VISION_FALLBACK_MODEL")
	if fallbackModel == "" {
		fallbackModel = "gemini-2.5-flash"
	}
	manager.SetVisionStrategy(strat, fallbackModel)

	if len(manager.List()) == 0 {
		return nil, errors.New("no LLM providers found in environment (neither TOKEN/API, GOOGLE_AI_API_KEY nor OPEN_ROUTER_API is configured)")
	}

	if strings.Contains(strings.ToLower(model), "gemini") {
		_ = manager.SetDefault("gemini")
	} else if _, err := manager.Get("vilao"); err == nil {
		_ = manager.SetDefault("vilao")
	} else if _, err := manager.Get("openai"); err == nil {
		_ = manager.SetDefault("openai")
	} else if _, err := manager.Get("gemini"); err == nil {
		_ = manager.SetDefault("gemini")
	}

	return manager, nil
}
