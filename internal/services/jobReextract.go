package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"tluagent-web/pkg/crawler"
	"tluagent-web/pkg/pipeline"
)

const (
	titleOnlyMaxLen        = 300
	forcedMinShrinkDivisor = 4
)

type reextractPayload struct {
	DocIDs    []string `json:"doc_ids"`
	DryRun    bool     `json:"dry_run"`
	Force     bool     `json:"force"`
	All       bool     `json:"all"`
	Threshold float64  `json:"garble_threshold"`
}

func parseReextractPayload(raw string) reextractPayload {
	var p reextractPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &p); err != nil {
		return reextractPayload{}
	}
	return p
}

func countTableRows(text string) int {
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "| ") && strings.HasSuffix(line, "|") {
			seen[line] = true
		}
	}
	return len(seen)
}

// shouldReplaceContent decides whether freshly extracted text is better than the stored text.
func shouldReplaceContent(oldText, newText string, force bool) (bool, string) {
	if strings.TrimSpace(newText) == "" {
		return false, "empty extraction"
	}
	if len(newText) < len(oldText)/forcedMinShrinkDivisor {
		return false, "new text far shorter than stored text"
	}
	if force {
		return true, "forced"
	}
	if len(newText) < len(oldText)/2 {
		return false, "new text far shorter than stored text"
	}
	if len(oldText) <= titleOnlyMaxLen && len(newText) > 2*len(oldText) {
		return true, "title-only document filled with attachment text"
	}
	oldRatio, newRatio := pipeline.TextGarbleRatio(oldText), pipeline.TextGarbleRatio(newText)
	if oldRatio >= pipeline.GarbleThreshold && newRatio < oldRatio/2 {
		return true, "corrupted text layer replaced"
	}
	if countTableRows(newText) > countTableRows(oldText) {
		return true, "table structure recovered"
	}
	return false, "no improvement"
}

func cmsItemKey(title, publishDate string) string {
	return documentTitleKey(title) + "\x00" + publishDate
}

func indexCMSItems(ctx context.Context, client *crawler.CMSClient, collection string) (map[string]*crawler.CMSItem, error) {
	index := map[string]*crawler.CMSItem{}
	for skip := 0; ; skip += client.PageSize() {
		items, pagination, err := client.List(ctx, collection, skip)
		if err != nil {
			return nil, fmt.Errorf("list %s at %d: %w", collection, skip, err)
		}
		for _, item := range items {
			index[cmsItemKey(item.Title, item.PublishDate)] = item
		}
		if len(items) == 0 || !pagination.HasMore {
			return index, nil
		}
	}
}

func (s *jobService) reextractDocuments(ctx context.Context, jobID, payload string) error {
	if s.ragService == nil {
		return fmt.Errorf("rag service is not configured")
	}
	p := parseReextractPayload(payload)
	threshold := p.Threshold
	if threshold <= 0 {
		threshold = pipeline.GarbleThreshold
	}
	minGarble := threshold
	if p.All {
		minGarble = 0
	}

	candidates, err := s.ragService.ListReextractCandidates(ctx, p.DocIDs, minGarble)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		log.Info().Msg("Document re-extraction found no candidates")
		return nil
	}

	client, err := newCMSClient()
	if err != nil {
		return err
	}
	source := noticeCrawlSource(true)
	attachments, err := s.newAttachmentCrawler(client)
	if err != nil {
		return err
	}
	index, err := indexCMSItems(ctx, client, source.Collection)
	if err != nil {
		return err
	}

	var replaced, skipped int64
	total := int64(len(candidates))
	for i, cand := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, _ = s.repo.UpdateJobProgress(ctx, jobID, int64(i), total)

		event := log.With().Str("doc_id", cand.ID).Float64("garble", cand.GarbleRatio).Logger()
		if !strings.HasPrefix(cand.ID, source.IDPrefix) {
			event.Info().Msg("Skipped re-extraction: document has no CMS attachment source")
			skipped++
			continue
		}
		item := index[cmsItemKey(cand.Title, cand.PublishDate)]
		if item == nil {
			event.Warn().Msg("Skipped re-extraction: no CMS item matches title and publish date")
			skipped++
			continue
		}
		if detail, err := client.Detail(ctx, source.Collection, item.ID); err == nil && detail.Title != "" {
			item = detail
		}

		body := cmsItemBody(ctx, source, item, attachments)
		replace, reason := shouldReplaceContent(cand.Text, body, p.Force)
		if !replace {
			event.Info().Str("reason", reason).Msg("Kept stored document content")
			skipped++
			continue
		}
		if p.DryRun {
			event.Info().Str("reason", reason).Int("old_chars", len(cand.Text)).Int("new_chars", len(body)).Float64("new_garble", pipeline.TextGarbleRatio(body)).Msg("Dry run: would replace document content")
			continue
		}
		if err := s.ragService.ReplaceDocumentContent(ctx, cand.ID, body); err != nil {
			event.Error().Err(err).Msg("Failed to replace document content")
			skipped++
			continue
		}
		event.Info().Str("reason", reason).Int("old_chars", len(cand.Text)).Int("new_chars", len(body)).Float64("new_garble", pipeline.TextGarbleRatio(body)).Msg("Replaced document content")
		replaced++
	}

	_, _ = s.repo.UpdateJobProgress(ctx, jobID, total, total)
	log.Info().Int64("candidates", total).Int64("replaced", replaced).Int64("skipped", skipped).Bool("dry_run", p.DryRun).Msg("Document re-extraction finished")
	return nil
}
