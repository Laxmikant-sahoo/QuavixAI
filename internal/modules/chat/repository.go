package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ================================
// Repository Interface
// ================================

type Repository interface {
	SaveMessage(ctx context.Context, sessionID, userID, userMsg, aiMsg string) error
	SaveFiveWhySession(ctx context.Context, userID string, session *FiveWhySession) error
	GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]ChatRecord, error)
}

// ================================
// Data Models
// ================================

type ChatRecord struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	UserMsg   string    `json:"user_msg"`
	AIMsg     string    `json:"ai_msg"`
	CreatedAt time.Time `json:"created_at"`
}

// ================================
// Postgres Repository
// ================================

type PostgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

// ================================
// Schema Init
// ================================

func (r *PostgresRepository) Init(ctx context.Context) error {
	queries := []string{
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
	}

	for _, q := range queries {
		if _, err := r.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// ================================
// Save Chat Message
// ================================

func (r *PostgresRepository) SaveMessage(ctx context.Context, sessionID, userID, userMsg, aiMsg string) error {
	if sessionID == "" {
		return errors.New("missing session id")
	}

	id := generateRepoID()

	query := `INSERT INTO chat_messages (id, session_id, user_id, user_msg, ai_msg)
		VALUES ($1,$2,$3,$4,$5);`

	_, err := r.db.ExecContext(ctx, query, id, sessionID, userID, userMsg, aiMsg)
	return err
}

// ================================
// Save Five-Why Session
// ================================

func (r *PostgresRepository) SaveFiveWhySession(ctx context.Context, userID string, session *FiveWhySession) error {
	if session == nil {
		return errors.New("nil session")
	}

	id := generateRepoID()

	stepsJSON, _ := json.Marshal(session.Steps)
	rcJSON, _ := json.Marshal(session.RootCause)
	solJSON, _ := json.Marshal(session.Solution)
	refJSON, _ := json.Marshal(session.Reframed)

	query := `INSERT INTO fivewhy_sessions
		(id, user_id, session_id, steps, root_cause, solution, reframed)
		VALUES ($1,$2,$3,$4,$5,$6,$7);`

	_, err := r.db.ExecContext(ctx, query,
		id,
		userID,
		session.SessionID,
		stepsJSON,
		rcJSON,
		solJSON,
		refJSON,
	)

	return err
}

// ================================
// Get Session History
// ================================

func (r *PostgresRepository) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]ChatRecord, error) {
	if sessionID == "" {
		return nil, errors.New("missing session id")
	}

	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, session_id, user_id, user_msg, ai_msg, created_at
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2;`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ChatRecord

	for rows.Next() {
		var rec ChatRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.SessionID,
			&rec.UserID,
			&rec.UserMsg,
			&rec.AIMsg,
			&rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, nil
}

// ================================
// Helpers
// ================================

func generateRepoID() string {
	return time.Now().Format("20060102150405.000000000")
}
