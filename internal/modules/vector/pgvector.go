package vector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ================================
// Data Structures
// ================================

type Document struct {
	ID      string            `json:"id"`
	Content string            `json:"content"`
	Vector  []float32         `json:"vector"`
	Meta    map[string]string `json:"meta"`
}

// ================================
// Store Interface
// ================================

type Store interface {
	Init(ctx context.Context) error
	Store(ctx context.Context, doc Document) error
	Search(ctx context.Context, vector []float32, limit int) ([]Document, error)
	Delete(ctx context.Context, id string) error
}

// ================================
// PgVector Store
// ================================

type PgVectorStore struct {
	db        *sql.DB
	dimension int
	table     string
}

func NewPgVectorStore(db *sql.DB, dimension int) *PgVectorStore {
	return &PgVectorStore{
		db:        db,
		dimension: dimension,
		table:     "vector_memory",
	}
}

// ================================
// Init
// ================================

func (p *PgVectorStore) Init(ctx context.Context) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS vector;`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			content TEXT,
			embedding VECTOR(%d),
			metadata JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`, p.table, p.dimension),

		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_embedding_idx
			ON %s USING ivfflat (embedding vector_l2_ops)
			WITH (lists = 100);`, p.table, p.table),
	}

	for _, q := range queries {
		if _, err := p.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}

	return nil
}

// ================================
// Store Document
// ================================

func (p *PgVectorStore) Store(ctx context.Context, doc Document) error {
	if doc.ID == "" {
		return errors.New("missing document id")
	}
	if len(doc.Vector) == 0 {
		return errors.New("missing embedding vector")
	}

	vecStr := vectorToSQL(doc.Vector)

	query := fmt.Sprintf(`INSERT INTO %s (id, content, embedding, metadata)
		VALUES ($1, $2, %s, $3)
		ON CONFLICT (id)
		DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata;`, p.table, vecStr)

	metaJSON := mapToJSON(doc.Meta)

	_, err := p.db.ExecContext(ctx, query, doc.ID, doc.Content, metaJSON)
	return err
}

// ================================
// Similarity Search
// ================================

func (p *PgVectorStore) Search(ctx context.Context, vector []float32, limit int) ([]Document, error) {
	if len(vector) == 0 {
		return nil, errors.New("empty query vector")
	}
	if limit <= 0 {
		limit = 5
	}

	vecStr := vectorToSQL(vector)

	query := fmt.Sprintf(`SELECT id, content, metadata
		FROM %s
		ORDER BY embedding <-> %s
		LIMIT %d;`, p.table, vecStr, limit)

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Document

	for rows.Next() {
		var id, content string
		var metaJSON []byte

		if err := rows.Scan(&id, &content, &metaJSON); err != nil {
			return nil, err
		}

		results = append(results, Document{
			ID:      id,
			Content: content,
			Meta:    jsonToMap(metaJSON),
		})
	}

	return results, nil
}

// ================================
// Delete
// ================================

func (p *PgVectorStore) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1;`, p.table)
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// ================================
// Helpers
// ================================

func vectorToSQL(vec []float32) string {
	vals := make([]string, len(vec))
	for i, v := range vec {
		vals[i] = fmt.Sprintf("%f", v)
	}
	return "ARRAY[" + strings.Join(vals, ",") + "]"
}

func mapToJSON(m map[string]string) string {
	if m == nil {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{")
	first := true
	for k, v := range m {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString(fmt.Sprintf("\"%s\":\"%s\"", k, v))
	}
	b.WriteString("}")
	return b.String()
}

func jsonToMap(b []byte) map[string]string {
	res := map[string]string{}
	if len(b) == 0 {
		return res
	}
	s := strings.Trim(string(b), "{}")
	if s == "" {
		return res
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		kv := strings.SplitN(p, ":", 2)
		if len(kv) == 2 {
			k := strings.Trim(kv[0], "\"")
			v := strings.Trim(kv[1], "\"")
			res[k] = v
		}
	}
	return res
}
