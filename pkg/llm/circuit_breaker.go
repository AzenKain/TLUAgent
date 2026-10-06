package llm

import (
	"sync"
	"time"
)

// CircuitState represents the operational state of a circuit breaker node.
type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half_open"
)

// NodeHealthStatus summarizes the availability and cooldown metric of an LLM node.
type NodeHealthStatus struct {
	NodeID                   string       `json:"node_id"`
	State                    CircuitState `json:"state"`
	ConsecutiveFailures      int          `json:"consecutive_failures"`
	CooldownUntil            *time.Time   `json:"cooldown_until,omitempty"`
	RemainingCooldownSeconds int64        `json:"remaining_cooldown_seconds"`
}

type circuitNode struct {
	state               CircuitState
	consecutiveFailures int
	cooldownUntil       time.Time
}

// CircuitBreaker manages failure tracking and cooldown isolation for LLM chain nodes.
type CircuitBreaker struct {
	mu               sync.RWMutex
	nodes            map[string]*circuitNode
	failureThreshold int
	cooldownDuration time.Duration
}

// NewCircuitBreaker creates a circuit breaker tracker with specified threshold and cooldown.
func NewCircuitBreaker(failureThreshold int, cooldownDuration time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if cooldownDuration <= 0 {
		cooldownDuration = 5 * time.Minute
	}
	return &CircuitBreaker{
		nodes:            make(map[string]*circuitNode),
		failureThreshold: failureThreshold,
		cooldownDuration: cooldownDuration,
	}
}

// SetPolicy updates failure threshold and cooldown duration for future state evaluations.
func (cb *CircuitBreaker) SetPolicy(failureThreshold int, cooldownDuration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if failureThreshold > 0 {
		cb.failureThreshold = failureThreshold
	}
	if cooldownDuration > 0 {
		cb.cooldownDuration = cooldownDuration
	}
}

// Allow determines if execution on a target node is permitted or currently in cooldown.
func (cb *CircuitBreaker) Allow(nodeID string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	node, exists := cb.nodes[nodeID]
	if !exists {
		return true
	}

	now := time.Now()
	if node.state == StateOpen {
		if now.After(node.cooldownUntil) {
			node.state = StateHalfOpen
			return true
		}
		return false
	}

	return true
}

// RecordSuccess marks a successful invocation, resetting failure counts and closing the circuit.
func (cb *CircuitBreaker) RecordSuccess(nodeID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	node, exists := cb.nodes[nodeID]
	if !exists {
		cb.nodes[nodeID] = &circuitNode{
			state: StateClosed,
		}
		return
	}

	node.consecutiveFailures = 0
	node.state = StateClosed
	node.cooldownUntil = time.Time{}
}

// RecordFailure records a failed execution, opening the circuit if threshold is reached.
func (cb *CircuitBreaker) RecordFailure(nodeID string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	node, exists := cb.nodes[nodeID]
	if !exists {
		node = &circuitNode{
			state: StateClosed,
		}
		cb.nodes[nodeID] = node
	}

	node.consecutiveFailures++
	if node.consecutiveFailures >= cb.failureThreshold {
		node.state = StateOpen
		node.cooldownUntil = time.Now().Add(cb.cooldownDuration)
		return true
	}

	return false
}

// Reset clears failure tracking and restores a node to closed state immediately.
func (cb *CircuitBreaker) Reset(nodeID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.nodes, nodeID)
}

// Status returns current health and cooldown status for a specific node.
func (cb *CircuitBreaker) Status(nodeID string) NodeHealthStatus {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	node, exists := cb.nodes[nodeID]
	if !exists {
		return NodeHealthStatus{
			NodeID: nodeID,
			State:  StateClosed,
		}
	}

	now := time.Now()
	state := node.state
	var remainingSec int64
	var untilPtr *time.Time

	if state == StateOpen {
		if now.After(node.cooldownUntil) {
			state = StateHalfOpen
		} else {
			rem := node.cooldownUntil.Sub(now)
			remainingSec = int64(rem.Seconds())
			u := node.cooldownUntil
			untilPtr = &u
		}
	}

	return NodeHealthStatus{
		NodeID:                   nodeID,
		State:                    state,
		ConsecutiveFailures:      node.consecutiveFailures,
		CooldownUntil:            untilPtr,
		RemainingCooldownSeconds: remainingSec,
	}
}

// AllStatuses returns health snapshots for all monitored nodes.
func (cb *CircuitBreaker) AllStatuses() map[string]NodeHealthStatus {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	out := make(map[string]NodeHealthStatus, len(cb.nodes))
	now := time.Now()
	for id, node := range cb.nodes {
		state := node.state
		var remainingSec int64
		var untilPtr *time.Time

		if state == StateOpen {
			if now.After(node.cooldownUntil) {
				state = StateHalfOpen
			} else {
				rem := node.cooldownUntil.Sub(now)
				remainingSec = int64(rem.Seconds())
				u := node.cooldownUntil
				untilPtr = &u
			}
		}

		out[id] = NodeHealthStatus{
			NodeID:                   id,
			State:                    state,
			ConsecutiveFailures:      node.consecutiveFailures,
			CooldownUntil:            untilPtr,
			RemainingCooldownSeconds: remainingSec,
		}
	}
	return out
}
