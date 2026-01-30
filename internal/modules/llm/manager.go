package llm

import (
	"context"
	"errors"
	"time"

	"quavixAI/internal/db"
	"quavixAI/internal/modules/vector"
)

// ================================
// Config
// ================================

type ManagerConfig struct {
	Provider  string
	APIKey    string
	Model     string
	Embedding string

	Vector   vector.Store
	Redis    *db.RedisClient
	Postgres any

	FiveWhy   bool
	RootCause bool
}

// ================================
// Manager
// ================================

type Manager struct {
	engine   *Engine
	vector   vector.Store
	redis    *db.RedisClient
	model    string
	embed    string
	provider string
}

func NewManager(cfg ManagerConfig) (*Manager, error) {
	eng := NewEngine()

	m := &Manager{
		engine:   eng,
		vector:   cfg.Vector,
		redis:    cfg.Redis,
		model:    cfg.Model,
		embed:    cfg.Embedding,
		provider: cfg.Provider,
	}

	// ================================
	// Register Providers
	// ================================

	switch cfg.Provider {
	case "openai":
		p, err := NewOpenAIProvider(cfg.APIKey)
		if err != nil {
			return nil, err
		}
		eng.RegisterProvider(p)
	case "ollama":
		p := NewOllamaProvider(cfg.Model)
		eng.RegisterProvider(p)
	case "local":
		p := NewLocalProvider()
		eng.RegisterProvider(p)
	default:
		return nil, errors.New("unsupported llm provider")
	}

	return m, nil
}

// ================================
// Core API
// ================================

func (m *Manager) Generate(ctx context.Context, req Request) (Response, error) {
	resp, err := m.engine.Generate(ctx, req)
	if err != nil {
		return Response{}, err
	}

	// ================================
	// Memory Hooks
	// ================================
	_ = m.storeMemory(ctx, req, resp)

	return resp, nil
}

// ================================
// Memory Layer
// ================================

func (m *Manager) storeMemory(ctx context.Context, req Request, resp Response) error {
	// Redis short-term memory
	if m.redis != nil {
		_ = m.redis.Set(ctx, "llm:last_response", resp.Text, 30*time.Minute)
	}

	// Vector long-term memory
	if m.vector != nil {
		doc := vector.Document{
			ID:      generateID(),
			Content: resp.Text,
			Meta: map[string]string{
				"mode":     string(req.Mode),
				"provider": resp.Provider,
				"model":    resp.Model,
			},
		}
		_ = m.vector.Store(ctx, doc)
	}

	return nil
}

// ================================
// Embeddings API
// ================================

func (m *Manager) Embed(ctx context.Context, text string) ([]float32, error) {
	// simple stub (replace with real embedding provider)
	if text == "" {
		return nil, errors.New("empty text")
	}

	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = float32(len(text)) / float32(i+1)
	}
	return vec, nil
}

// ================================
// Helpers
// ================================

func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}

// ================================
// Provider Stubs (to be moved into providers/*)
// ================================

// OpenAI Provider Stub

type OpenAIProvider struct {
	apiKey string
}

func NewOpenAIProvider(key string) (*OpenAIProvider, error) {
	if key == "" {
		return nil, errors.New("missing openai api key")
	}
	return &OpenAIProvider{apiKey: key}, nil
}

func (o *OpenAIProvider) Name() string { return "primary" }

func (o *OpenAIProvider) Generate(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	// TODO: integrate real OpenAI SDK
	return ProviderResponse{
		Text:   "[OPENAI RESPONSE PLACEHOLDER]\n" + req.Prompt,
		Tokens: 128,
		Model:  req.Model,
	}, nil
}

// Ollama Provider Stub

type OllamaProvider struct {
	model string
}

func NewOllamaProvider(model string) *OllamaProvider {
	return &OllamaProvider{model: model}
}

func (o *OllamaProvider) Name() string { return "primary" }

func (o *OllamaProvider) Generate(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	return ProviderResponse{
		Text:   "[OLLAMA RESPONSE PLACEHOLDER]\n" + req.Prompt,
		Tokens: 128,
		Model:  req.Model,
	}, nil
}

// Local Provider Stub

type LocalProvider struct{}

func NewLocalProvider() *LocalProvider { return &LocalProvider{} }

func (l *LocalProvider) Name() string { return "primary" }

func (l *LocalProvider) Generate(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	return ProviderResponse{
		Text:   "[LOCAL MODEL RESPONSE PLACEHOLDER]\n" + req.Prompt,
		Tokens: 128,
		Model:  req.Model,
	}, nil
}
