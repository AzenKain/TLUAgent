package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
)

// TLUClawRepository defines persistence operations for TLUClaw operational sessions and snapshots.
type TLUClawRepository interface {
	CreateSession(ctx context.Context, params sqlc.CreateTLUClawSessionParams) (*models.TLUClawSession, error)
	GetSessionByID(ctx context.Context, id string) (*models.TLUClawSession, error)
	ListSessionsByAdmin(ctx context.Context, adminID string, limit, offset int32) ([]*models.TLUClawSession, error)
	UpdateSessionTitle(ctx context.Context, id, title string) error
	TouchSession(ctx context.Context, id string) error
	DeleteSession(ctx context.Context, id, adminID string) error

	CreateMessage(ctx context.Context, params sqlc.CreateTLUClawMessageParams) (*models.TLUClawMessage, error)
	ListMessagesBySessionID(ctx context.Context, sessionID string) ([]*models.TLUClawMessage, error)

	CreateSnapshot(ctx context.Context, params sqlc.CreateTLUClawSnapshotParams) (*models.TLUClawActionSnapshot, error)
	GetSnapshotByID(ctx context.Context, id string) (*models.TLUClawActionSnapshot, error)
	GetSnapshotByNonce(ctx context.Context, nonce string) (*models.TLUClawActionSnapshot, error)
	GetSnapshotByBatchID(ctx context.Context, batchID string) (*models.TLUClawActionSnapshot, error)
	ListSnapshotsBySessionID(ctx context.Context, sessionID string) ([]*models.TLUClawActionSnapshot, error)
	UpdateSnapshotStatus(ctx context.Context, id, status string, appliedAt sql.NullTime) error
	ListLatestCommittedSnapshots(ctx context.Context, limit int32) ([]*models.TLUClawActionSnapshot, error)
	ListSnapshotsPaginated(ctx context.Context, status string, limit, offset int32) ([]*models.TLUClawActionSnapshot, int64, error)

	InvalidateSessionCache(ctx context.Context, adminID, sessionID string)
	InvalidateSnapshotCache(ctx context.Context, id, batchID, nonce, sessionID string)
	WithTx(tx *sql.Tx) TLUClawRepository
}

type tluclawRepository struct {
	q    *sqlc.Queries
	c    cache.Cache
	db   *sql.DB
	inTx bool
	sfg  *singleflight.Group
}

// NewTLUClawRepository initializes a new TLUClaw repository instance.
func NewTLUClawRepository(db sqlc.DBTX, c cache.Cache) TLUClawRepository {
	var sqlDB *sql.DB
	if concreteDB, ok := db.(*sql.DB); ok {
		sqlDB = concreteDB
	}
	return &tluclawRepository{
		q:   sqlc.New(db),
		c:   c,
		db:  sqlDB,
		sfg: &singleflight.Group{},
	}
}

func (r *tluclawRepository) WithTx(tx *sql.Tx) TLUClawRepository {
	return &tluclawRepository{
		q:    r.q.WithTx(tx),
		c:    r.c,
		db:   r.db,
		inTx: true,
		sfg:  &singleflight.Group{},
	}
}

func (r *tluclawRepository) InvalidateSessionCache(ctx context.Context, adminID, sessionID string) {
	if r.c == nil {
		return
	}
	_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "sess", sessionID))
	_ = r.c.DelByPattern(ctx, cache.BuildKey("tluclaw", "admin_ids", adminID)+"*")
	_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "msgs", sessionID))
}

func (r *tluclawRepository) InvalidateSnapshotCache(ctx context.Context, id, batchID, nonce, sessionID string) {
	if r.c == nil {
		return
	}
	if id != "" {
		_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "snap", "id", id))
	}
	if batchID != "" {
		_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "snap", "batch", batchID))
	}
	if nonce != "" {
		_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "snap", "nonce", nonce))
	}
	if sessionID != "" {
		_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "snaps", "session", sessionID))
	}
	_ = r.c.DelByPattern(ctx, "tluclaw:snaps_count*")
	_ = r.c.DelByPattern(ctx, "tluclaw:snaps_paged*")
	_ = r.c.DelByPattern(ctx, "tluclaw:snaps:latest*")
}

func (r *tluclawRepository) CreateSession(ctx context.Context, params sqlc.CreateTLUClawSessionParams) (*models.TLUClawSession, error) {
	row, err := r.q.CreateTLUClawSession(ctx, params)
	if err != nil {
		return nil, err
	}
	sess := mapTLUClawSession(row)
	r.InvalidateSessionCache(ctx, params.AdminUserID, params.ID)
	return sess, nil
}

func (r *tluclawRepository) GetSessionByID(ctx context.Context, id string) (*models.TLUClawSession, error) {
	cacheKey := cache.BuildKey("tluclaw", "sess", id)
	if r.c != nil && !r.inTx {
		var cached models.TLUClawSession
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetTLUClawSessionByID(ctx, id)
		if err != nil {
			return nil, err
		}
		sess := mapTLUClawSession(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, sess, 30*time.Minute)
		}
		return sess, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.TLUClawSession), nil
}

func (r *tluclawRepository) ListSessionsByAdmin(ctx context.Context, adminID string, limit, offset int32) ([]*models.TLUClawSession, error) {
	cacheKey := cache.BuildKey("tluclaw", "admin_ids", adminID, fmt.Sprintf("%d", limit), fmt.Sprintf("%d", offset))
	var sessionIDs []string

	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, cacheKey, &sessionIDs)
	}

	if len(sessionIDs) == 0 {
		val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
			ids, err := r.q.ListTLUClawSessionIDsByAdmin(ctx, sqlc.ListTLUClawSessionIDsByAdminParams{
				AdminUserID: adminID,
				Limit:       int64(limit),
				Offset:      int64(offset),
			})
			if err != nil {
				return nil, err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, cacheKey, ids, 10*time.Minute)
			}
			return ids, nil
		})
		if err != nil {
			return nil, err
		}
		sessionIDs = val.([]string)
	}

	if len(sessionIDs) == 0 {
		return []*models.TLUClawSession{}, nil
	}

	resultMap := make(map[string]*models.TLUClawSession, len(sessionIDs))
	var missingIDs []string
	for _, id := range sessionIDs {
		key := cache.BuildKey("tluclaw", "sess", id)
		if r.c != nil && !r.inTx {
			var sess models.TLUClawSession
			if err := r.c.Get(ctx, key, &sess); err == nil {
				resultMap[id] = &sess
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	if len(missingIDs) > 0 {
		sfgKey := "tluclaw:sess_by_ids:" + strings.Join(missingIDs, ",")
		v, err, _ := r.sfg.Do(sfgKey, func() (any, error) {
			rows, err := r.q.GetTLUClawSessionsByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}
			fetched := make([]*models.TLUClawSession, 0, len(rows))
			for _, row := range rows {
				sess := mapTLUClawSession(row)
				if r.c != nil && !r.inTx {
					_ = r.c.Set(ctx, cache.BuildKey("tluclaw", "sess", sess.ID), sess, 30*time.Minute)
				}
				fetched = append(fetched, sess)
			}
			return fetched, nil
		})
		if err != nil {
			return nil, err
		}
		for _, sess := range v.([]*models.TLUClawSession) {
			resultMap[sess.ID] = sess
		}
	}

	results := make([]*models.TLUClawSession, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if sess, ok := resultMap[id]; ok {
			results = append(results, sess)
		}
	}
	return results, nil
}

func (r *tluclawRepository) UpdateSessionTitle(ctx context.Context, id, title string) error {
	sess, err := r.GetSessionByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.UpdateTLUClawSessionTitle(ctx, sqlc.UpdateTLUClawSessionTitleParams{
		Title:     title,
		UpdatedAt: time.Now(),
		ID:        id,
	}); err != nil {
		return err
	}
	r.InvalidateSessionCache(ctx, sess.AdminUserID, id)
	return nil
}

func (r *tluclawRepository) TouchSession(ctx context.Context, id string) error {
	sess, err := r.GetSessionByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.q.TouchTLUClawSession(ctx, sqlc.TouchTLUClawSessionParams{
		UpdatedAt: time.Now(),
		ID:        id,
	}); err != nil {
		return err
	}
	r.InvalidateSessionCache(ctx, sess.AdminUserID, id)
	return nil
}

func (r *tluclawRepository) DeleteSession(ctx context.Context, id, adminID string) error {
	if err := r.q.DeleteTLUClawSession(ctx, id); err != nil {
		return err
	}
	r.InvalidateSessionCache(ctx, adminID, id)
	return nil
}

func (r *tluclawRepository) CreateMessage(ctx context.Context, params sqlc.CreateTLUClawMessageParams) (*models.TLUClawMessage, error) {
	row, err := r.q.CreateTLUClawMessage(ctx, params)
	if err != nil {
		return nil, err
	}
	if r.c != nil {
		_ = r.c.Del(ctx, cache.BuildKey("tluclaw", "msgs", params.SessionID))
	}
	return mapTLUClawMessage(row), nil
}

func (r *tluclawRepository) ListMessagesBySessionID(ctx context.Context, sessionID string) ([]*models.TLUClawMessage, error) {
	cacheKey := cache.BuildKey("tluclaw", "msgs", sessionID)
	if r.c != nil && !r.inTx {
		var cached []*models.TLUClawMessage
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListTLUClawMessagesBySessionID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		results := make([]*models.TLUClawMessage, len(rows))
		for i, row := range rows {
			results[i] = mapTLUClawMessage(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, results, 15*time.Minute)
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.TLUClawMessage), nil
}

func (r *tluclawRepository) CreateSnapshot(ctx context.Context, params sqlc.CreateTLUClawSnapshotParams) (*models.TLUClawActionSnapshot, error) {
	row, err := r.q.CreateTLUClawSnapshot(ctx, params)
	if err != nil {
		return nil, err
	}
	r.InvalidateSnapshotCache(ctx, params.ID, params.BatchID, params.ConfirmationNonce, params.SessionID)
	return mapTLUClawSnapshot(row), nil
}

func (r *tluclawRepository) GetSnapshotByID(ctx context.Context, id string) (*models.TLUClawActionSnapshot, error) {
	cacheKey := cache.BuildKey("tluclaw", "snap", "id", id)
	if r.c != nil && !r.inTx {
		var cached models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetTLUClawSnapshotByID(ctx, id)
		if err != nil {
			return nil, err
		}
		snap := mapTLUClawSnapshot(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, snap, 30*time.Minute)
		}
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.TLUClawActionSnapshot), nil
}

func (r *tluclawRepository) GetSnapshotByNonce(ctx context.Context, nonce string) (*models.TLUClawActionSnapshot, error) {
	cacheKey := cache.BuildKey("tluclaw", "snap", "nonce", nonce)
	if r.c != nil && !r.inTx {
		var cached models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetTLUClawSnapshotByNonce(ctx, nonce)
		if err != nil {
			return nil, err
		}
		snap := mapTLUClawSnapshot(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, snap, 30*time.Minute)
		}
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.TLUClawActionSnapshot), nil
}

func (r *tluclawRepository) GetSnapshotByBatchID(ctx context.Context, batchID string) (*models.TLUClawActionSnapshot, error) {
	cacheKey := cache.BuildKey("tluclaw", "snap", "batch", batchID)
	if r.c != nil && !r.inTx {
		var cached models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		row, err := r.q.GetTLUClawSnapshotByBatchID(ctx, batchID)
		if err != nil {
			return nil, err
		}
		snap := mapTLUClawSnapshot(row)
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, snap, 30*time.Minute)
		}
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*models.TLUClawActionSnapshot), nil
}

func (r *tluclawRepository) ListSnapshotsBySessionID(ctx context.Context, sessionID string) ([]*models.TLUClawActionSnapshot, error) {
	cacheKey := cache.BuildKey("tluclaw", "snaps", "session", sessionID)
	if r.c != nil && !r.inTx {
		var cached []*models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListTLUClawSnapshotsBySessionID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		results := make([]*models.TLUClawActionSnapshot, len(rows))
		for i, row := range rows {
			results[i] = mapTLUClawSnapshot(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, results, 15*time.Minute)
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.TLUClawActionSnapshot), nil
}

func (r *tluclawRepository) UpdateSnapshotStatus(ctx context.Context, id, status string, appliedAt sql.NullTime) error {
	snap, _ := r.GetSnapshotByID(ctx, id)
	err := r.q.UpdateTLUClawSnapshotStatus(ctx, sqlc.UpdateTLUClawSnapshotStatusParams{
		Status:    status,
		AppliedAt: appliedAt,
		UpdatedAt: time.Now(),
		ID:        id,
	})
	if err != nil {
		return err
	}
	if snap != nil {
		r.InvalidateSnapshotCache(ctx, id, snap.BatchID, snap.ConfirmationNonce, snap.SessionID)
	} else {
		r.InvalidateSnapshotCache(ctx, id, "", "", "")
	}
	return nil
}

func (r *tluclawRepository) ListLatestCommittedSnapshots(ctx context.Context, limit int32) ([]*models.TLUClawActionSnapshot, error) {
	cacheKey := cache.BuildKey("tluclaw", "snaps", "latest", fmt.Sprintf("%d", limit))
	if r.c != nil && !r.inTx {
		var cached []*models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	val, err, _ := r.sfg.Do(cacheKey, func() (any, error) {
		rows, err := r.q.ListLatestCommittedTLUClawSnapshots(ctx, int64(limit))
		if err != nil {
			return nil, err
		}
		results := make([]*models.TLUClawActionSnapshot, len(rows))
		for i, row := range rows {
			results[i] = mapTLUClawSnapshot(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, cacheKey, results, 15*time.Minute)
		}
		return results, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]*models.TLUClawActionSnapshot), nil
}

func (r *tluclawRepository) ListSnapshotsPaginated(ctx context.Context, status string, limit, offset int32) ([]*models.TLUClawActionSnapshot, int64, error) {
	countKey := cache.BuildKey("tluclaw", "snaps_count", status)
	var total int64
	if r.c != nil && !r.inTx {
		_ = r.c.Get(ctx, countKey, &total)
	}

	if total == 0 {
		v, err, _ := r.sfg.Do(countKey, func() (any, error) {
			cnt, err := r.q.CountTLUClawSnapshots(ctx, status)
			if err != nil {
				return int64(0), err
			}
			if r.c != nil && !r.inTx {
				_ = r.c.Set(ctx, countKey, cnt, 15*time.Minute)
			}
			return cnt, nil
		})
		if err != nil {
			return nil, 0, err
		}
		total = v.(int64)
	}

	if total == 0 {
		return []*models.TLUClawActionSnapshot{}, 0, nil
	}

	pagedKey := cache.BuildKey("tluclaw", "snaps_paged", status, fmt.Sprintf("%d", limit), fmt.Sprintf("%d", offset))
	if r.c != nil && !r.inTx {
		var cached []*models.TLUClawActionSnapshot
		if err := r.c.Get(ctx, pagedKey, &cached); err == nil {
			return cached, total, nil
		}
	}

	val, err, _ := r.sfg.Do(pagedKey, func() (any, error) {
		rows, err := r.q.ListTLUClawSnapshotsPaginated(ctx, sqlc.ListTLUClawSnapshotsPaginatedParams{
			Status:    status,
			LimitVal:  int64(limit),
			OffsetVal: int64(offset),
		})
		if err != nil {
			return nil, err
		}
		results := make([]*models.TLUClawActionSnapshot, len(rows))
		for i, row := range rows {
			results[i] = mapTLUClawSnapshot(row)
		}
		if r.c != nil && !r.inTx {
			_ = r.c.Set(ctx, pagedKey, results, 15*time.Minute)
		}
		return results, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return val.([]*models.TLUClawActionSnapshot), total, nil
}

func mapTLUClawSession(s sqlc.TluclawSession) *models.TLUClawSession {
	return &models.TLUClawSession{
		ID:          s.ID,
		AdminUserID: s.AdminUserID,
		Title:       s.Title,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func mapTLUClawMessage(m sqlc.TluclawMessage) *models.TLUClawMessage {
	return &models.TLUClawMessage{
		ID:               m.ID,
		SessionID:        m.SessionID,
		Role:             m.Role,
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCallsJSON:    m.ToolCallsJson,
		ToolCallID:       m.ToolCallID,
		MetadataJSON:     m.MetadataJson,
		CreatedAt:        m.CreatedAt,
	}
}

func mapTLUClawSnapshot(s sqlc.TluclawActionSnapshot) *models.TLUClawActionSnapshot {
	var nonceExp *time.Time
	if s.NonceExpiresAt.Valid {
		t := s.NonceExpiresAt.Time
		nonceExp = &t
	}
	var applied *time.Time
	if s.AppliedAt.Valid {
		t := s.AppliedAt.Time
		applied = &t
	}
	return &models.TLUClawActionSnapshot{
		ID:                s.ID,
		BatchID:           s.BatchID,
		SessionID:         s.SessionID,
		OperatorID:        s.OperatorID,
		ActionType:        s.ActionType,
		Summary:           s.Summary,
		DiffPreviewJSON:   s.DiffPreviewJson,
		BlastRadiusJSON:   s.BlastRadiusJson,
		PreviousStateJSON: s.PreviousStateJson,
		NewStateJSON:      s.NewStateJson,
		Status:            s.Status,
		ConfirmationNonce: s.ConfirmationNonce,
		NonceExpiresAt:    nonceExp,
		AppliedAt:         applied,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}
