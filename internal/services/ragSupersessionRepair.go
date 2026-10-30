package services

import (
	"context"
	"fmt"
	"strings"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
)

const auditLogScanLimit = 100000

func (s *ragService) storedDocumentText(ctx context.Context, docID string) (string, error) {
	chunks, err := s.ragRepo.GetChunksByDocumentID(ctx, docID)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		parts = append(parts, ch.Content)
	}
	return strings.Join(parts, "\n"), nil
}

func (s *ragService) unwindSupersession(ctx context.Context, active, target *models.Document, status, reason string) error {
	if target.SupersededByID == active.ID {
		supersededBy := ""
		if status == constants.DocStatusPartiallyAmended {
			supersededBy = active.ID
		}
		if err := s.ragRepo.UpdateDocumentStatus(ctx, target.ID, status, supersededBy); err != nil {
			return err
		}
		if err := s.ragRepo.UpdateChunksByDocIDStatus(ctx, target.ID, constants.DocStatusActive); err != nil {
			return err
		}
		rules, err := s.ragRepo.ListRulesByDocumentID(ctx, target.ID)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if err := s.ragRepo.UpdateRuleStatus(ctx, r.ID, constants.DocStatusActive, ""); err != nil {
				return err
			}
		}
	}

	if err := s.ragRepo.DeleteKGEdge(ctx, fmt.Sprintf(constants.PrefixEdgeSuper+"%s:%s", active.ID, target.ID)); err != nil {
		return err
	}
	if err := s.ragRepo.DeleteAuditLogPair(ctx, active.ID, target.ID); err != nil {
		return err
	}
	if status == constants.DocStatusPartiallyAmended {
		if err := s.ragRepo.RecordSupersessionLog(ctx, active.ID, target.ID, constants.RuleTypePartialAmendment, reason); err != nil {
			return err
		}
		if _, err := s.ragRepo.UpsertKGEdge(ctx, &models.KGEdge{
			ID:           fmt.Sprintf(constants.PrefixEdgeAmend+"%s:%s", active.ID, target.ID),
			SourceNodeID: constants.PrefixDocNode + active.ID,
			TargetNodeID: constants.PrefixDocNode + target.ID,
			RelationType: constants.KGRelationPartiallyAmends,
			Weight:       1.0,
			Status:       constants.DocStatusActive,
		}); err != nil {
			return err
		}
	}

	s.ragRepo.InvalidateDocumentCache(ctx, active.ID)
	s.ragRepo.InvalidateDocumentCache(ctx, target.ID)
	return nil
}

// RepairInvalidSupersessions undoes full replacements made by title-only documents and downgrades those whose cohort scope is narrower than the replaced document.
func (s *ragService) RepairInvalidSupersessions(ctx context.Context) (*models.SupersessionRepairResult, error) {
	result := &models.SupersessionRepairResult{}
	logs, _, err := s.ragRepo.ListAllAuditLogs(ctx, auditLogScanLimit, 0)
	if err != nil {
		return result, fmt.Errorf("list audit logs: %w", err)
	}

	for _, entry := range logs {
		if entry.RuleType != constants.RuleTypeDocumentReplacement {
			continue
		}
		active, errActive := s.ragRepo.GetDocumentByID(ctx, entry.ActiveDocID)
		target, errTarget := s.ragRepo.GetDocumentByID(ctx, entry.SupersededDocID)
		if errActive != nil || errTarget != nil || active == nil || target == nil {
			continue
		}
		activeText, err := s.storedDocumentText(ctx, active.ID)
		if err != nil {
			return result, fmt.Errorf("load text of %s: %w", active.ID, err)
		}

		if isTitleOnlyContent(activeText, active.Title) {
			targetText, err := s.storedDocumentText(ctx, target.ID)
			if err != nil {
				return result, fmt.Errorf("load text of %s: %w", target.ID, err)
			}
			if isTitleOnlyContent(targetText, target.Title) {
				continue
			}
			if err := s.unwindSupersession(ctx, active, target, constants.DocStatusActive, ""); err != nil {
				return result, fmt.Errorf("revert %s -> %s: %w", active.ID, target.ID, err)
			}
			result.Reverted++
			continue
		}

		targetText, err := s.storedDocumentText(ctx, target.ID)
		if err != nil {
			return result, fmt.Errorf("load text of %s: %w", target.ID, err)
		}
		if !isTitleOnlyContent(targetText, target.Title) && !cohortCovers(active.ApplicableCohort, target.ApplicableCohort) {
			reason := fmt.Sprintf("Downgraded to partial amendment: %s applies to %s while %s applies to %s.", active.ID, active.ApplicableCohort, target.ID, target.ApplicableCohort)
			if err := s.unwindSupersession(ctx, active, target, constants.DocStatusPartiallyAmended, reason); err != nil {
				return result, fmt.Errorf("downgrade %s -> %s: %w", active.ID, target.ID, err)
			}
			result.Downgraded++
		}
	}
	return result, nil
}

// RevertSupersession lets an administrator undo one audited replacement, restoring the replaced document to active.
func (s *ragService) RevertSupersession(ctx context.Context, activeDocID, supersededDocID string) error {
	exists, err := s.ragRepo.AuditLogPairExists(ctx, activeDocID, supersededDocID)
	if err != nil {
		return err
	}
	if !exists {
		return apperrors.New(apperrors.ErrNotFound, "no audited supersession links these documents")
	}
	active, errActive := s.ragRepo.GetDocumentByID(ctx, activeDocID)
	target, errTarget := s.ragRepo.GetDocumentByID(ctx, supersededDocID)
	if errActive != nil || errTarget != nil || active == nil || target == nil {
		return apperrors.New(apperrors.ErrNotFound, "document not found")
	}
	return s.unwindSupersession(ctx, active, target, constants.DocStatusActive, "")
}
