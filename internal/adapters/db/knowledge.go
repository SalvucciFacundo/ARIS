package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.KnowledgeGraphStore = (*SQLiteKnowledgeGraph)(nil)

// SQLiteKnowledgeGraph implements ports.KnowledgeGraphStore.
type SQLiteKnowledgeGraph struct {
	db *sql.DB
}

// NewKnowledgeGraph creates a new knowledge graph store adapter.
func NewKnowledgeGraph(db *sql.DB) *SQLiteKnowledgeGraph {
	return &SQLiteKnowledgeGraph{db: db}
}

// AddFact stores a new fact in SQLite and updates the FTS5 index.
func (k *SQLiteKnowledgeGraph) AddFact(ctx context.Context, fact domain.KnowledgeFact) (string, error) {
	if fact.ID == "" {
		fact.ID = uuid.New().String()
	}
	if fact.CreatedAt.IsZero() {
		fact.CreatedAt = time.Now()
	}
	if fact.Scope == "" {
		fact.Scope = domain.ScopeUser
	}

	labelsJSON, err := json.Marshal(fact.Labels)
	if err != nil {
		return "", fmt.Errorf("marshal labels: %w", err)
	}

	tx, err := k.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	insertSQL := `INSERT INTO knowledge_facts (id, topic, concept, fact, source_agent, labels, project, scope, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, insertSQL,
		fact.ID, fact.Topic, fact.Concept, fact.Fact,
		fact.SourceAgent, string(labelsJSON), fact.Project, string(fact.Scope), fact.CreatedAt)
	if err != nil {
		return "", fmt.Errorf("insert knowledge fact: %w", err)
	}

	rowID, err := res.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("get last insert id: %w", err)
	}

	ftsSQL := `INSERT INTO knowledge_facts_fts (rowid, topic, concept, fact, labels) VALUES (?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, ftsSQL, rowID, fact.Topic, fact.Concept, fact.Fact, string(labelsJSON)); err != nil {
		return "", fmt.Errorf("insert into fts5 table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return fact.ID, nil
}

// SearchFacts performs full-text search across facts, optionally filtered by scope.
func (k *SQLiteKnowledgeGraph) SearchFacts(ctx context.Context, queryStr string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	if limit <= 0 {
		limit = 20
	}

	cleaned := sanitizeFTS(queryStr)
	if cleaned == "" {
		return k.ListAllFacts(ctx, scope, limit)
	}

	var query string
	var args []any

	if scope != "" {
		query = `SELECT k.id, k.topic, k.concept, k.fact, k.source_agent, k.labels, k.project, k.scope, k.created_at
			FROM knowledge_facts k
			JOIN knowledge_facts_fts fts ON k.rowid = fts.rowid
			WHERE knowledge_facts_fts MATCH ? AND k.scope = ?
			ORDER BY rank
			LIMIT ?`
		args = append(args, cleaned, string(scope), limit)
	} else {
		query = `SELECT k.id, k.topic, k.concept, k.fact, k.source_agent, k.labels, k.project, k.scope, k.created_at
			FROM knowledge_facts k
			JOIN knowledge_facts_fts fts ON k.rowid = fts.rowid
			WHERE knowledge_facts_fts MATCH ?
			ORDER BY rank
			LIMIT ?`
		args = append(args, cleaned, limit)
	}

	return k.queryFacts(ctx, query, args...)
}

// GetFactsByTopic retrieves all facts for a topic, ordered by newest first.
func (k *SQLiteKnowledgeGraph) GetFactsByTopic(ctx context.Context, topic string) ([]domain.KnowledgeFact, error) {
	query := `SELECT id, topic, concept, fact, source_agent, labels, project, scope, created_at
		FROM knowledge_facts WHERE topic = ? ORDER BY created_at DESC`
	return k.queryFacts(ctx, query, topic)
}

// ListAllFacts returns recent facts optionally filtered by scope.
func (k *SQLiteKnowledgeGraph) ListAllFacts(ctx context.Context, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	if limit <= 0 {
		limit = 50
	}
	var query string
	var args []any

	if scope != "" {
		query = `SELECT id, topic, concept, fact, source_agent, labels, project, scope, created_at
			FROM knowledge_facts WHERE scope = ? ORDER BY created_at DESC LIMIT ?`
		args = append(args, string(scope), limit)
	} else {
		query = `SELECT id, topic, concept, fact, source_agent, labels, project, scope, created_at
			FROM knowledge_facts ORDER BY created_at DESC LIMIT ?`
		args = append(args, limit)
	}

	return k.queryFacts(ctx, query, args...)
}

// DeleteFact deletes a fact and cleans its FTS entry.
func (k *SQLiteKnowledgeGraph) DeleteFact(ctx context.Context, id string) error {
	tx, err := k.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var rowID int64
	var topic, concept, fact, labels string
	err = tx.QueryRowContext(ctx, "SELECT rowid, topic, concept, fact, labels FROM knowledge_facts WHERE id = ?", id).
		Scan(&rowID, &topic, &concept, &fact, &labels)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("find fact for delete: %w", err)
	}

	// Delete from FTS5
	ftsDelete := `INSERT INTO knowledge_facts_fts (knowledge_facts_fts, rowid, topic, concept, fact, labels)
		VALUES ('delete', ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, ftsDelete, rowID, topic, concept, fact, labels); err != nil {
		return fmt.Errorf("delete from fts: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM knowledge_facts WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete from knowledge_facts: %w", err)
	}

	return tx.Commit()
}

func (k *SQLiteKnowledgeGraph) queryFacts(ctx context.Context, query string, args ...any) ([]domain.KnowledgeFact, error) {
	rows, err := k.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query knowledge facts: %w", err)
	}
	defer rows.Close()

	var facts []domain.KnowledgeFact
	for rows.Next() {
		var fact domain.KnowledgeFact
		var labelsJSON string
		var scopeStr string
		var createdAt time.Time

		if err := rows.Scan(&fact.ID, &fact.Topic, &fact.Concept, &fact.Fact,
			&fact.SourceAgent, &labelsJSON, &fact.Project, &scopeStr, &createdAt); err != nil {
			return nil, fmt.Errorf("scan fact row: %w", err)
		}

		if labelsJSON != "" {
			_ = json.Unmarshal([]byte(labelsJSON), &fact.Labels)
		}
		fact.Scope = domain.MemoryScope(scopeStr)
		fact.CreatedAt = createdAt
		facts = append(facts, fact)
	}

	return facts, rows.Err()
}

func sanitizeFTS(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	var quoted []string
	for _, p := range parts {
		cleaned := strings.NewReplacer(
			"\"", "", "*", "", "^", "", "(", "", ")", "",
			"NEAR", "", "NOT", "", "AND", "", "OR", "", ":", "",
		).Replace(p)
		if cleaned != "" {
			quoted = append(quoted, "\""+cleaned+"\"")
		}
	}
	return strings.Join(quoted, " ")
}
