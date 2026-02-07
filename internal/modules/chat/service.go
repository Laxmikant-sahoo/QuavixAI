package chat

import (
	"context"
	"errors"
	"time"

	"quavixAI/internal/modules/llm"
	"quavixAI/internal/modules/prompt"
	"quavixAI/internal/modules/types"
	"quavixAI/internal/modules/vector"
)

// ================================
// Config
// ================================

type ServiceConfig struct {
	Repo   Repository
	LLM    *llm.Manager
	Vector vector.Store
	Memory *MemoryEngine

	FiveWhy   bool
	Evaluator bool
	RootCause bool
	Reframer  bool
}

// ================================
// Service
// ================================

type Service struct {
	repo         Repository
	llm          *llm.Manager
	vector       vector.Store
	memory       *MemoryEngine
	orchestrator *Orchestrator

	cfg ServiceConfig
}

func NewService(cfg ServiceConfig) *Service {
	return &Service{
		repo:         cfg.Repo,
		llm:          cfg.LLM,
		vector:       cfg.Vector,
		memory:       cfg.Memory,
		orchestrator: NewOrchestrator(cfg.LLM, cfg.Vector, prompt.NewBuilder()),
		cfg:          cfg,
	}
}

// ================================
// Core APIs
// ================================

// Standard chat (memory-augmented reasoning)
func (s *Service) Chat(ctx context.Context, sessionID, userID, message string) (*llm.Response, error) {
	if message == "" {
		return nil, errors.New("empty message")
	}

	// store in session memory
	if s.memory != nil {
		_ = s.memory.AppendSession(ctx, sessionID, "user", message)
	}

	// hybrid context
	contextStr := ""
	if s.memory != nil {
		ctxData, _ := s.memory.HybridContext(ctx, sessionID, message, 5)
		contextStr = ctxData
	}

	promptStr := "Context:\n" + contextStr + "\nUser:\n" + message

	resp, err := s.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeReasoning,
		Prompt: promptStr,
	})
	if err != nil {
		return nil, err
	}

	// store AI response
	if s.memory != nil {
		_ = s.memory.AppendSession(ctx, sessionID, "assistant", resp.Text)
	}

	// persist conversation
	if s.repo != nil {
		_ = s.repo.SaveMessage(ctx, sessionID, userID, message, resp.Text)
	}

	return &resp, nil
}

// ================================
// 5-Why Reasoning Pipeline
// ================================

func (s *Service) FiveWhy(ctx context.Context, sessionID, userID, question string) (*FiveWhySession, error) {
	if !s.cfg.FiveWhy {
		return nil, errors.New("five-why engine disabled")
	}

	// store question
	if s.memory != nil {
		_ = s.memory.AppendSession(ctx, sessionID, "user", question)
	}

	session, err := s.orchestrator.RunFiveWhy(ctx, sessionID, question)
	if err != nil {
		return nil, err
	}

	// persist root cause
	if s.vector != nil {
		_ = s.vector.Store(ctx, vector.Document{
			ID:      sessionID + "_root",
			Content: session.RootCause.RootCause,
			Meta: map[string]string{
				"type":   "root_cause",
				"userID": userID,
			},
		})
	}

	// store memory
	if s.memory != nil {
		_ = s.memory.AppendSession(ctx, sessionID, "assistant", session.RootCause.RootCause)
	}

	// persist full session
	if s.repo != nil {
		_ = s.repo.SaveFiveWhySession(ctx, userID, session)
	}

	return session, nil
}

// ================================
// Root Cause Only API
// ================================

func (s *Service) RootCause(ctx context.Context, steps []types.FiveWhyStep) (*types.RootCauseResult, error) {
	if !s.cfg.RootCause {
		return nil, errors.New("root-cause engine disabled")
	}

	return s.orchestrator.ExtractRootCause(ctx, steps)
}

// ================================
// Reframing API
// ================================

func (s *Service) Reframe(ctx context.Context, question string, rc types.RootCauseResult) (*types.ReframedQuestion, error) {
	if !s.cfg.Reframer {
		return nil, errors.New("reframing engine disabled")
	}

	return s.orchestrator.ReframeQuestion(ctx, question, rc)
}

// ================================
// Memory APIs
// ================================

func (s *Service) CompressSession(ctx context.Context, sessionID string) (string, error) {
	if s.memory == nil {
		return "", errors.New("memory engine not configured")
	}
	return s.memory.CompressSession(ctx, sessionID)
}

func (s *Service) Recall(ctx context.Context, query string, limit int) (*RetrievedMemory, error) {
	if s.memory == nil {
		return nil, errors.New("memory engine not configured")
	}
	return s.memory.Recall(ctx, query, limit)
}

// ================================
// Maintenance Jobs
// ================================

func (s *Service) BackgroundCompression(ctx context.Context, sessionID string) {
	go func() {
		_, _ = s.memory.CompressSession(ctx, sessionID)
	}()
}

func (s *Service) CleanupSession(ctx context.Context, sessionID string) {
	go func() {
		time.Sleep(1 * time.Second)
	}()
}
