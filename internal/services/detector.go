package services

import (
	"context"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/llm"
)

// SupersessionProposal encapsulates legacy supersession detection result.
type SupersessionProposal struct {
	HasConflict     bool   `json:"has_conflict"`
	SupersededDocID string `json:"superseded_doc_id"`
	Reason          string `json:"reason"`
	InquiryQuestion string `json:"inquiry_question"`
	ProposedAction  string `json:"proposed_action"`
}

// Detector wraps RegulatoryDissectionEngine for backward compatibility with crawler workers.
type Detector struct {
	engine *RegulatoryDissectionEngine
}

// NewDetector constructs a detector adapter using RegulatoryDissectionEngine.
func NewDetector(db sqlc.DBTX, mgr *llm.Manager) *Detector {
	engine := NewRegulatoryDissectionEngine(EngineParams{
		DB:         db,
		LLMManager: mgr,
	})
	return &Detector{
		engine: engine,
	}
}

// SetEngine updates the underlying regulatory dissection engine.
func (d *Detector) SetEngine(engine *RegulatoryDissectionEngine) {
	d.engine = engine
}

// Engine returns the underlying regulatory dissection engine.
func (d *Detector) Engine() *RegulatoryDissectionEngine {
	return d.engine
}

// AnalyzeAndNotify delegates regulatory dissection to the dynamic engine.
func (d *Detector) AnalyzeAndNotify(ctx context.Context, newDocID, newDocTitle, newDocContent string) (*SupersessionProposal, error) {
	if d.engine == nil {
		return nil, nil
	}

	plan, err := d.engine.AnalyzeAndNotify(ctx, newDocID, newDocTitle, newDocContent)
	if err != nil || plan == nil {
		return nil, err
	}

	if plan.ActionType == ActionNone || plan.TargetDocID == "" {
		return &SupersessionProposal{
			HasConflict: false,
		}, nil
	}

	return &SupersessionProposal{
		HasConflict:     true,
		SupersededDocID: plan.TargetDocID,
		Reason:          plan.Reason,
		InquiryQuestion: plan.InquiryQuestion,
		ProposedAction:  string(plan.ActionType),
	}, nil
}
