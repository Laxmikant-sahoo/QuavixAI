package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PostgresClient struct {
	DB *sql.DB
}

func NewPostgres(dsn string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	client := &PostgresClient{DB: db}

	if err := client.InitSchema(context.Background()); err != nil {
		return nil, err
	}

	return client, nil
}

func (p *PostgresClient) Close() error {
	return p.DB.Close()
}

func (p *PostgresClient) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
		`CREATE EXTENSION IF NOT EXISTS vector;`,

		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			name TEXT,
			role TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			user_id TEXT,
			user_msg TEXT,
			ai_msg TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS fivewhy_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			session_id TEXT,
			steps JSONB,
			root_cause JSONB,
			solution JSONB,
			reframed JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS vector_memory (
			id TEXT PRIMARY KEY,
			content TEXT,
			embedding VECTOR(384),
			metadata JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,

		`CREATE INDEX IF NOT EXISTS vector_memory_embedding_idx
		 ON vector_memory USING ivfflat (embedding vector_l2_ops)
		 WITH (lists = 100);`,
	}

	for _, q := range queries {
		if _, err := p.DB.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("schema init failed: %w", err)
		}
	}

	return nil
}
