package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ChainType defines the workload classification for an execution chain.
type ChainType string

const (
	ChainTypeChat      ChainType = "chat"
	ChainTypeRerank    ChainType = "rerank"
	ChainTypeEmbedding ChainType = "embedding"
)

// ChainNode represents an individual provider and model step within an execution chain.
type ChainNode struct {
	ID           string
	Name         string
	ProviderID   string
	ProviderName string
	ModelKey     string
	Priority     int
	IsActive     bool
	Provider     Provider
	Reranker     Reranker
	Embedder     Embedder
}

// LLMChain encapsulates an ordered fallback pipeline with circuit breaker isolation.
type LLMChain struct {
	mu               sync.RWMutex
	ID               string
	Type             ChainType
	Name             string
	Description      string
	IsActive         bool
	FailureThreshold int
	CooldownDuration time.Duration
	Nodes            []ChainNode
	Breaker          *CircuitBreaker
}

// NewLLMChain constructs an LLM chain with a dedicated circuit breaker.
func NewLLMChain(id string, chainType ChainType, name string, failureThreshold int, cooldownDuration time.Duration) *LLMChain {
	breaker := NewCircuitBreaker(failureThreshold, cooldownDuration)
	return &LLMChain{
		ID:               id,
		Type:             chainType,
		Name:             name,
		IsActive:         true,
		FailureThreshold: failureThreshold,
		CooldownDuration: cooldownDuration,
		Nodes:            make([]ChainNode, 0),
		Breaker:          breaker,
	}
}

// Validate ensures structural validity and enforces model homogeneity on embedding chains.
func (c *LLMChain) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Nodes) == 0 {
		return errors.New("chain has no configured nodes")
	}

	if c.Type == ChainTypeEmbedding {
		var referenceModelKey string
		for i, node := range c.Nodes {
			normalizedKey := strings.TrimSpace(node.ModelKey)
			if normalizedKey == "" {
				return fmt.Errorf("node '%s' is missing a model_key", node.ID)
			}
			if i == 0 {
				referenceModelKey = normalizedKey
				continue
			}
			if normalizedKey != referenceModelKey {
				return fmt.Errorf("embedding chain model mismatch: all nodes must use identical model '%s', but node '%s' (%s) uses '%s'", referenceModelKey, node.Name, node.ProviderName, normalizedKey)
			}
		}
	}

	return nil
}

// SetNodes updates and re-sorts the chain's execution nodes.
func (c *LLMChain) SetNodes(nodes []ChainNode) error {
	c.mu.Lock()
	c.Nodes = make([]ChainNode, len(nodes))
	copy(c.Nodes, nodes)
	sort.SliceStable(c.Nodes, func(i, j int) bool {
		return c.Nodes[i].Priority < c.Nodes[j].Priority
	})
	c.mu.Unlock()

	return c.Validate()
}

// ExecuteChat runs chat completion sequentially down the chain until a node succeeds.
func (c *LLMChain) ExecuteChat(ctx context.Context, req *ChatRequest) (*ChatResponse, *ChainNode, error) {
	c.mu.RLock()
	nodes := make([]ChainNode, len(c.Nodes))
	copy(nodes, c.Nodes)
	breaker := c.Breaker
	c.mu.RUnlock()

	var accumulatedErrors []string

	for i := range nodes {
		node := &nodes[i]
		if !node.IsActive || node.Provider == nil {
			continue
		}

		if !breaker.Allow(node.ID) {
			accumulatedErrors = append(accumulatedErrors, fmt.Sprintf("node '%s' (%s) skipped: in cooldown", node.Name, node.ProviderName))
			continue
		}

		nodeReq := *req
		if node.ModelKey != "" {
			nodeReq.Model = node.ModelKey
		}

		resp, err := node.Provider.Chat(ctx, &nodeReq)
		if err == nil && resp != nil && strings.TrimSpace(resp.Message.Content) != "" {
			breaker.RecordSuccess(node.ID)
			return resp, node, nil
		}

		tripped := breaker.RecordFailure(node.ID)
		errMsg := fmt.Sprintf("node '%s' failed: %v", node.Name, err)
		if tripped {
			errMsg += " [CIRCUIT TRIPPED: ENTERED COOLDOWN]"
		}
		accumulatedErrors = append(accumulatedErrors, errMsg)
	}

	return nil, nil, fmt.Errorf("all chain nodes failed. details: %s", strings.Join(accumulatedErrors, " | "))
}

// ExecuteChatStream initiates chat streaming sequentially down the chain until a node connects.
func (c *LLMChain) ExecuteChatStream(ctx context.Context, req *ChatRequest) (StreamReader, *ChainNode, error) {
	c.mu.RLock()
	nodes := make([]ChainNode, len(c.Nodes))
	copy(nodes, c.Nodes)
	breaker := c.Breaker
	c.mu.RUnlock()

	var accumulatedErrors []string

	for i := range nodes {
		node := &nodes[i]
		if !node.IsActive || node.Provider == nil {
			continue
		}

		if !breaker.Allow(node.ID) {
			accumulatedErrors = append(accumulatedErrors, fmt.Sprintf("node '%s' (%s) skipped: in cooldown", node.Name, node.ProviderName))
			continue
		}

		nodeReq := *req
		if node.ModelKey != "" {
			nodeReq.Model = node.ModelKey
		}

		stream, err := node.Provider.ChatStream(ctx, &nodeReq)
		if err == nil && stream != nil {
			breaker.RecordSuccess(node.ID)
			return stream, node, nil
		}

		tripped := breaker.RecordFailure(node.ID)
		errMsg := fmt.Sprintf("node '%s' stream failed: %v", node.Name, err)
		if tripped {
			errMsg += " [CIRCUIT TRIPPED: ENTERED COOLDOWN]"
		}
		accumulatedErrors = append(accumulatedErrors, errMsg)
	}

	return nil, nil, fmt.Errorf("all streaming chain nodes failed. details: %s", strings.Join(accumulatedErrors, " | "))
}

// ExecuteRerank evaluates document relevance sequentially down the chain until a node succeeds.
func (c *LLMChain) ExecuteRerank(ctx context.Context, query string, documents []string, topN *int) ([]RerankResult, *ChainNode, error) {
	c.mu.RLock()
	nodes := make([]ChainNode, len(c.Nodes))
	copy(nodes, c.Nodes)
	breaker := c.Breaker
	c.mu.RUnlock()

	var accumulatedErrors []string

	for i := range nodes {
		node := &nodes[i]
		if !node.IsActive || node.Reranker == nil {
			continue
		}

		if !breaker.Allow(node.ID) {
			accumulatedErrors = append(accumulatedErrors, fmt.Sprintf("node '%s' (%s) skipped: in cooldown", node.Name, node.ProviderName))
			continue
		}

		results, err := node.Reranker.Rerank(ctx, query, documents, topN)
		if err == nil && len(results) > 0 {
			breaker.RecordSuccess(node.ID)
			return results, node, nil
		}

		tripped := breaker.RecordFailure(node.ID)
		errMsg := fmt.Sprintf("node '%s' rerank failed: %v", node.Name, err)
		if tripped {
			errMsg += " [CIRCUIT TRIPPED: ENTERED COOLDOWN]"
		}
		accumulatedErrors = append(accumulatedErrors, errMsg)
	}

	return nil, nil, fmt.Errorf("all rerank chain nodes failed. details: %s", strings.Join(accumulatedErrors, " | "))
}

// ExecuteEmbedding creates vector representations sequentially across providers with identical models.
func (c *LLMChain) ExecuteEmbedding(ctx context.Context, texts []string) ([][]float32, *ChainNode, error) {
	c.mu.RLock()
	nodes := make([]ChainNode, len(c.Nodes))
	copy(nodes, c.Nodes)
	breaker := c.Breaker
	c.mu.RUnlock()

	var accumulatedErrors []string

	for i := range nodes {
		node := &nodes[i]
		if !node.IsActive || node.Embedder == nil {
			continue
		}

		if !breaker.Allow(node.ID) {
			accumulatedErrors = append(accumulatedErrors, fmt.Sprintf("node '%s' (%s) skipped: in cooldown", node.Name, node.ProviderName))
			continue
		}

		vectors, err := node.Embedder.Embed(ctx, texts)
		if err == nil && len(vectors) > 0 {
			breaker.RecordSuccess(node.ID)
			return vectors, node, nil
		}

		tripped := breaker.RecordFailure(node.ID)
		errMsg := fmt.Sprintf("node '%s' embedding failed: %v", node.Name, err)
		if tripped {
			errMsg += " [CIRCUIT TRIPPED: ENTERED COOLDOWN]"
		}
		accumulatedErrors = append(accumulatedErrors, errMsg)
	}

	return nil, nil, fmt.Errorf("all embedding chain nodes failed. details: %s", strings.Join(accumulatedErrors, " | "))
}
