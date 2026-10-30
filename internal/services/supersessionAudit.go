package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/constants"
)

// AuditOptions selects how much of the corpus a supersession audit re-evaluates.
type AuditOptions struct {
	Full bool
}

func documentFingerprint(doc *models.Document, text string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{doc.Title, doc.ApplicableCohort, doc.PublishDate, text}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func (e *RegulatoryDissectionEngine) fingerprintOf(ctx context.Context, doc *models.Document) (string, error) {
	chunks, err := e.ragRepo.GetChunksByDocumentID(ctx, doc.ID)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		parts = append(parts, ch.Content)
	}
	return documentFingerprint(doc, strings.Join(parts, "\n")), nil
}

func (e *RegulatoryDissectionEngine) finishFullRun(ctx context.Context, runID string, docs []*models.Document) error {
	for _, doc := range docs {
		if _, failed := e.auditFailed.LoadAndDelete(doc.ID); failed {
			continue
		}
		fp, err := e.fingerprintOf(ctx, doc)
		if err != nil {
			return fmt.Errorf("fingerprint %s: %w", doc.ID, err)
		}
		if err := e.ragRepo.SetAuditFingerprint(ctx, doc.ID, fp); err != nil {
			return fmt.Errorf("store fingerprint %s: %w", doc.ID, err)
		}
	}
	return e.completeAuditRun(ctx, runID)
}

// AuditCorpus re-evaluates the whole corpus on the first run or when requested, otherwise only documents that are new or changed since they were last audited.
func (e *RegulatoryDissectionEngine) AuditCorpus(ctx context.Context, opts AuditOptions, onProgress func(cur, total int64)) error {
	if e.ragRepo == nil {
		return fmt.Errorf("rag repo unavailable for audit")
	}
	baseline, err := e.ragRepo.CountAuditFingerprints(ctx)
	if err != nil {
		return fmt.Errorf("count audit fingerprints: %w", err)
	}
	running, err := e.ragRepo.GetRunningAuditRun(ctx)
	if err != nil {
		return fmt.Errorf("load running audit run: %w", err)
	}
	if opts.Full || baseline == 0 || running != nil {
		return e.AuditCorpusChronological(ctx, onProgress)
	}
	return e.auditChangedDocuments(ctx, onProgress)
}

func (e *RegulatoryDissectionEngine) auditChangedDocuments(ctx context.Context, onProgress func(cur, total int64)) (err error) {
	docs, err := e.ragRepo.ListAllRegulationsChronological(ctx)
	if err != nil {
		return fmt.Errorf("list regulations: %w", err)
	}

	changed := make(map[string]string)
	for _, doc := range docs {
		fp, err := e.fingerprintOf(ctx, doc)
		if err != nil {
			return fmt.Errorf("fingerprint %s: %w", doc.ID, err)
		}
		stored, err := e.ragRepo.GetAuditFingerprint(ctx, doc.ID)
		if err != nil {
			return fmt.Errorf("load fingerprint %s: %w", doc.ID, err)
		}
		if stored != fp {
			changed[doc.ID] = fp
		}
	}
	if len(changed) == 0 {
		log.Info().Int("documents", len(docs)).Msg("Supersession audit skipped: no document changed since the last audit")
		return nil
	}

	run, err := e.ragRepo.CreateAuditRun(ctx, uuid.NewString(), "", int64(len(changed)))
	if err != nil {
		return fmt.Errorf("create audit run: %w", err)
	}
	defer func() {
		if err != nil {
			_ = e.ragRepo.FailAuditRun(context.WithoutCancel(ctx), run.ID, err.Error())
		}
	}()

	memory := NewSupersessionLineageMemory()
	for _, doc := range docs {
		if doc.Status == constants.DocStatusActive {
			memory.ActiveDocIDs[doc.ID] = true
		} else if doc.Status == constants.DocStatusSuperseded && doc.SupersededByID != "" {
			memory.RecordSupersession(doc.ID, doc.SupersededByID)
		}
	}

	var majorRows []sqlc.ListAllMajorMetadataRow
	if e.curriculumRepo != nil {
		majorRows, _ = e.curriculumRepo.ListAllMajorMetadata(ctx)
	}

	var applyMu sync.Mutex
	buckets := make(map[string][]*models.Document)
	total, done := int64(len(changed)), int64(0)
	for _, doc := range docs {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := e.auditBucketKey(ctx, doc, majorRows)
		if fp, ok := changed[doc.ID]; ok {
			e.auditOneDocument(ctx, doc, buckets[key], memory, &applyMu, true)
			if _, failed := e.auditFailed.LoadAndDelete(doc.ID); !failed {
				if err := e.ragRepo.SetAuditFingerprint(ctx, doc.ID, fp); err != nil {
					return fmt.Errorf("store fingerprint %s: %w", doc.ID, err)
				}
			}
			done++
			_ = e.ragRepo.UpdateAuditRunProgress(ctx, run.ID, doc.ID, done, total)
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		buckets[key] = append(buckets[key], doc)
	}
	return e.ragRepo.CompleteAuditRun(ctx, run.ID)
}
