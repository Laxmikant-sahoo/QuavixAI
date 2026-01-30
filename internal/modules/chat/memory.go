package chat

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"quavixAI/internal/db"
	"quavixAI/internal/modules/llm"
	"quavixAI/internal/modules/vector"
)

// ================================
// Data Models
// ================================

type MemoryMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type SessionMemory struct {
	SessionID string          `json:"session_id"`
	Messages  []MemoryMessage `json:"messages"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type RetrievedMemory struct {
	Documents []vector.Document `json:"documents"`
	Context   string            `json:"context"`
}

// ================================
// Memory Engine
// ================================

type MemoryEngine struct {
	redis  *db.RedisClient
	vector vector.Store
	llm    *llm.Manager
}

func NewMemoryEngine(redis *db.RedisClient, vstore vector.Store, llmMgr *llm.Manager) *MemoryEngine {
	return &MemoryEngine{
		redis:  redis,
		vector: vstore,
		llm:    llmMgr,
	}
}

// ================================
// Session Memory (Redis)
// ================================

func (m *MemoryEngine) AppendSession(ctx context.Context, sessionID, role, content string) error {
	if sessionID == "" {
		return errors.New("missing session id")
	}

	key := "session:" + sessionID

	msg := MemoryMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	var session SessionMemory

	data, _ := m.redis.Get(ctx, key)
	if data != "" {
		_ = json.Unmarshal([]byte(data), &session)
	}

	session.SessionID = sessionID
	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	b, _ := json.Marshal(session)
	_ = m.redis.Set(ctx, key, string(b), 24*time.Hour)

	return nil
}

func (m *MemoryEngine) GetSession(ctx context.Context, sessionID string) (*SessionMemory, error) {
	key := "session:" + sessionID
	data, err := m.redis.Get(ctx, key)
	if err != nil || data == "" {
		return nil, errors.New("session not found")
	}

	var session SessionMemory
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// ================================
// Compression (LLM Summarization)
// ================================

func (m *MemoryEngine) CompressSession(ctx context.Context, sessionID string) (string, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	b, _ := json.Marshal(session.Messages)

	summaryPrompt := "Summarize the following conversation into long-term semantic memory:\n" + string(b)

	resp, err := m.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeAnalysis,
		Prompt: summaryPrompt,
	})
	if err != nil {
		return "", err
	}

	// store compressed memory into vector DB
	emb, _ := m.llm.Embed(ctx, resp.Text)

	_ = m.vector.Store(ctx, vector.Document{
		ID:      sessionID + "_summary",
		Content: resp.Text,
		Vector:  emb,
		Meta: map[string]string{
			"type":      "session_summary",
			"sessionID": sessionID,
		},
	})

	return resp.Text, nil
}

// ================================
// Recall (Vector Search)
// ================================

func (m *MemoryEngine) Recall(ctx context.Context, query string, limit int) (*RetrievedMemory, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	emb, err := m.llm.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	docs, err := m.vector.Search(ctx, emb, limit)
	if err != nil {
		return nil, err
	}

	ctxStr := ""
	for _, d := range docs {
		ctxStr += d.Content + "\n"
	}

	return &RetrievedMemory{
		Documents: docs,
		Context:   ctxStr,
	}, nil
}

// ================================
// Hybrid Retrieval (Session + Vector)
// ================================

func (m *MemoryEngine) HybridContext(ctx context.Context, sessionID, query string, limit int) (string, error) {
	var contextStr string

	// session memory
	session, _ := m.GetSession(ctx, sessionID)
	if session != nil {
		for _, msg := range session.Messages {
			contextStr += msg.Role + ": " + msg.Content + "\n"
		}
	}

	// vector memory
	recall, err := m.Recall(ctx, query, limit)
	if err == nil {
		contextStr += "\n--- Semantic Memory ---\n"
		contextStr += recall.Context
	}

	return contextStr, nil
}

// ================================
// Long-term Memory Store
// ================================

func (m *MemoryEngine) StoreLongTerm(ctx context.Context, content string, meta map[string]string) error {
	emb, err := m.llm.Embed(ctx, content)
	if err != nil {
		return err
	}

	doc := vector.Document{
		ID:      generateMemoryID(),
		Content: content,
		Vector:  emb,
		Meta:    meta,
	}

	return m.vector.Store(ctx, doc)
}

// ================================
// Helpers
// ================================

func generateMemoryID() string {
	return time.Now().Format("20060102150405.000000000")
}
