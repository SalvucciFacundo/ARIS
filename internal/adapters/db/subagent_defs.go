package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

var _ ports.SubagentStore = (*SQLiteSubagentStore)(nil)

// SQLiteSubagentStore implements ports.SubagentStore.
type SQLiteSubagentStore struct {
	db *sql.DB
}

// NewSubagentStore creates a new subagent store.
func NewSubagentStore(db *sql.DB) *SQLiteSubagentStore {
	return &SQLiteSubagentStore{db: db}
}

func (s *SQLiteSubagentStore) SaveSubagent(ctx context.Context, def domain.SubagentDef) error {
	toolsJSON, err := json.Marshal(def.AllowedTools)
	if err != nil {
		toolsJSON = []byte("[]")
	}

	now := time.Now()
	if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now

	query := `INSERT INTO subagent_defs (
		name, display_name, role, description, system_prompt, personality,
		temperature, model, allowed_tools, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		display_name=excluded.display_name,
		role=excluded.role,
		description=excluded.description,
		system_prompt=excluded.system_prompt,
		personality=excluded.personality,
		temperature=excluded.temperature,
		model=excluded.model,
		allowed_tools=excluded.allowed_tools,
		updated_at=excluded.updated_at`

	_, err = s.db.ExecContext(ctx, query,
		def.Name, def.DisplayName, def.Role, def.Description, def.SystemPrompt,
		def.Personality, def.Temperature, def.Model, string(toolsJSON),
		def.CreatedAt, def.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save subagent def %q: %w", def.Name, err)
	}
	return nil
}

func (s *SQLiteSubagentStore) GetSubagent(ctx context.Context, name string) (*domain.SubagentDef, error) {
	query := `SELECT name, display_name, role, description, system_prompt, personality,
		temperature, model, allowed_tools, created_at, updated_at
		FROM subagent_defs WHERE name = ?`

	var def domain.SubagentDef
	var toolsJSON string
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&def.Name, &def.DisplayName, &def.Role, &def.Description, &def.SystemPrompt,
		&def.Personality, &def.Temperature, &def.Model, &toolsJSON,
		&def.CreatedAt, &def.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subagent @%s not found", name)
		}
		return nil, fmt.Errorf("query subagent @%s: %w", name, err)
	}

	_ = json.Unmarshal([]byte(toolsJSON), &def.AllowedTools)
	return &def, nil
}

func (s *SQLiteSubagentStore) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	query := `SELECT name, display_name, role, description, system_prompt, personality,
		temperature, model, allowed_tools, created_at, updated_at
		FROM subagent_defs ORDER BY name ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list subagents: %w", err)
	}
	defer rows.Close()

	var list []domain.SubagentDef
	for rows.Next() {
		var def domain.SubagentDef
		var toolsJSON string
		if err := rows.Scan(
			&def.Name, &def.DisplayName, &def.Role, &def.Description, &def.SystemPrompt,
			&def.Personality, &def.Temperature, &def.Model, &toolsJSON,
			&def.CreatedAt, &def.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subagent row: %w", err)
		}
		_ = json.Unmarshal([]byte(toolsJSON), &def.AllowedTools)
		list = append(list, def)
	}

	return list, rows.Err()
}

func (s *SQLiteSubagentStore) DeleteSubagent(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM subagent_defs WHERE name = ?", name)
	return err
}
