package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

var _ ports.HistoryStore = (*SQLiteHistoryStore)(nil)

// SQLiteHistoryStore implements ports.HistoryStore.
type SQLiteHistoryStore struct {
	db *sql.DB
}

// NewHistoryStore creates a new history store.
func NewHistoryStore(db *sql.DB) *SQLiteHistoryStore {
	return &SQLiteHistoryStore{db: db}
}

// SaveGeneration records a completed generation and its parameters.
func (h *SQLiteHistoryStore) SaveGeneration(ctx context.Context, spec *domain.ImageSpec, result *domain.ImageResult) error {
	query := `INSERT INTO generations (
		id, prompt_raw, prompt_enhanced, negative_prompt,
		backend, model, width, height, steps, cfg_scale, seed,
		image_path, thumb_path, duration_ms, rating, feedback, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	if !spec.CreatedAt.IsZero() {
		now = spec.CreatedAt
	}

	_, err := h.db.ExecContext(ctx, query,
		result.ID, spec.RawPrompt, spec.EnhancedPrompt, spec.NegativePrompt,
		spec.Backend, spec.Model, spec.Width, spec.Height, spec.Steps, spec.CFGScale, spec.Seed,
		result.LocalPath, "", result.Duration.Milliseconds(), 0, "", now,
	)
	if err != nil {
		return fmt.Errorf("insert generation history: %w", err)
	}
	return nil
}

// UpdateRating updates user feedback or thumbs up/down for a generation.
func (h *SQLiteHistoryStore) UpdateRating(ctx context.Context, id string, rating int, feedback string) error {
	query := `UPDATE generations SET rating = ?, feedback = ? WHERE id = ?`
	res, err := h.db.ExecContext(ctx, query, rating, feedback, id)
	if err != nil {
		return fmt.Errorf("update rating: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("generation %s not found", id)
	}
	return nil
}

// GetHistory retrieves past generation logs.
func (h *SQLiteHistoryStore) GetHistory(ctx context.Context, limit, offset int) ([]domain.GenerationRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, prompt_raw, prompt_enhanced, negative_prompt, backend, model,
		width, height, steps, cfg_scale, seed, image_path, thumb_path, duration_ms,
		rating, feedback, created_at
		FROM generations
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := h.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var records []domain.GenerationRecord
	for rows.Next() {
		var r domain.GenerationRecord
		if err := rows.Scan(
			&r.ID, &r.PromptRaw, &r.PromptEnhanced, &r.NegativePrompt, &r.Backend, &r.Model,
			&r.Width, &r.Height, &r.Steps, &r.CFGScale, &r.Seed, &r.ImagePath, &r.ThumbPath,
			&r.DurationMs, &r.Rating, &r.Feedback, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan history record: %w", err)
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// GetByID returns a single generation record by its ID.
func (h *SQLiteHistoryStore) GetByID(ctx context.Context, id string) (*domain.GenerationRecord, error) {
	query := `SELECT id, prompt_raw, prompt_enhanced, negative_prompt, backend, model,
		width, height, steps, cfg_scale, seed, image_path, thumb_path, duration_ms,
		rating, feedback, created_at
		FROM generations WHERE id = ?`

	var r domain.GenerationRecord
	err := h.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.PromptRaw, &r.PromptEnhanced, &r.NegativePrompt, &r.Backend, &r.Model,
		&r.Width, &r.Height, &r.Steps, &r.CFGScale, &r.Seed, &r.ImagePath, &r.ThumbPath,
		&r.DurationMs, &r.Rating, &r.Feedback, &r.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("generation %s not found", id)
		}
		return nil, fmt.Errorf("query generation by id: %w", err)
	}
	return &r, nil
}
