package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/constants"
)

const (
	scopeBodyMaxRunes = 2500
	scopeMaxCohorts   = 6
	noticeDocIDPrefix = "DOC_THONG_BAO_"
)

var scopeLineRegex = regexp.MustCompile(`(?im)^[\s\-*\x{2022}]*\x{0111}\x{1ed1}i t\x{01b0}\x{1ee3}ng[^\n]*`)

// extractScopeCohortFromBody reads the cohorts a short notice declares on its own scope line.
func extractScopeCohortFromBody(content string) string {
	if utf8.RuneCountInString(content) > scopeBodyMaxRunes {
		return ""
	}
	for _, line := range scopeLineRegex.FindAllString(content, -1) {
		cohorts := extractNormalizedCohortList(line)
		if cohorts == "" {
			continue
		}
		if len(strings.Split(cohorts, ",")) > scopeMaxCohorts {
			return ""
		}
		return cohorts
	}
	return ""
}

// RepairDocumentCohorts narrows notices stored as applying to every cohort when their body declares a specific scope.
func (s *ragService) RepairDocumentCohorts(ctx context.Context) (int, error) {
	rows, err := s.ragRepo.ListAllDocumentIdentifiers(ctx)
	if err != nil {
		return 0, fmt.Errorf("list document identifiers: %w", err)
	}
	updated := 0
	for _, row := range rows {
		if !strings.HasPrefix(row.ID, noticeDocIDPrefix) {
			continue
		}
		doc, err := s.ragRepo.GetDocumentByID(ctx, row.ID)
		if err != nil || doc == nil || !strings.EqualFold(doc.ApplicableCohort, constants.DocCohortAll) {
			continue
		}
		text, err := s.storedDocumentText(ctx, doc.ID)
		if err != nil {
			return updated, fmt.Errorf("load text of %s: %w", doc.ID, err)
		}
		cohorts := extractScopeCohortFromBody(text)
		if cohorts == "" || extractNormalizedCohortList(doc.Title) != "" {
			continue
		}
		doc.ApplicableCohort = cohorts
		if err := s.saveDocumentCohort(ctx, doc, cohorts); err != nil {
			return updated, fmt.Errorf("update cohort of %s: %w", doc.ID, err)
		}
		updated++
	}
	return updated, nil
}

var cohortTokenRegex = regexp.MustCompile(`^K\d{2}$`)

func normalizeCohortInput(raw string) (string, bool) {
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	if cleaned == strings.ToUpper(constants.DocCohortAll) {
		return constants.DocCohortAll, true
	}
	parts := strings.Split(cleaned, ",")
	for _, p := range parts {
		if !cohortTokenRegex.MatchString(p) {
			return "", false
		}
	}
	return strings.Join(parts, ","), true
}

// UpdateDocumentCohort lets an administrator set the cohort scope of a document and its chunks.
func (s *ragService) UpdateDocumentCohort(ctx context.Context, docID, cohort string) error {
	normalized, ok := normalizeCohortInput(cohort)
	if !ok {
		return apperrors.New(apperrors.ErrBadRequest, "cohort must be ALL or a comma separated list such as K36,K37")
	}
	doc, err := s.ragRepo.GetDocumentByID(ctx, docID)
	if err != nil || doc == nil {
		return apperrors.New(apperrors.ErrNotFound, "document not found")
	}
	doc.ApplicableCohort = normalized
	return s.saveDocumentCohort(ctx, doc, normalized)
}

func (s *ragService) saveDocumentCohort(ctx context.Context, doc *models.Document, cohorts string) error {
	if _, err := s.ragRepo.UpsertDocument(ctx, doc); err != nil {
		return err
	}
	return s.ragRepo.UpdateChunksCohortByDocumentID(ctx, doc.ID, cohorts)
}
