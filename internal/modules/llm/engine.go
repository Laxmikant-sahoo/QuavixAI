package llm

import (
	"context"
	"errors"
	"time"
)

// ================================
// Modes
// ================================

type Mode string

const (
	ModeReasoning Mode = "reasoning"
	ModeAnalysis  Mode = "analysis"
	ModeDiagnosis Mode = "diagnosis"
	ModePlanning  Mode = "planning"
	ModeDefault   Mode = "default"
)

// ================================
// Request / Response
// ================================

type Request struct {
	Mode        Mode              `json:"mode"`
	Prompt      string            `json:"prompt"`
	Temperature float32           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	Metadata    map[string]string `json:"metadata"`
}

type Response struct {
	Text       string        `json:"text"`
	Tokens     int           `json:"tokens"`
	Latency    time.Duration `json:"latency"`
	Provider   string        `json:"provider"`
	Model      string        `json:"model"`
	Confidence float64       `json:"confidence"`
}

// ================================
// Provider Interface
// ================================

type Provider interface {
	Name() string
	Generate(ctx context.Context, req ProviderRequest) (ProviderResponse, error)
}

// ================================
// Provider DTOs
// ================================

type ProviderRequest struct {
	Prompt      string
	Temperature float32
	MaxTokens   int
	Model       string
}

type ProviderResponse struct {
	Text     string
	Tokens   int
	Model    string
	Metadata map[string]string
}

// ================================
// Engine
// ================================

type Engine struct {
	providers map[string]Provider
	policy    *PolicyEngine
	modelMap  map[Mode]string
}

func NewEngine() *Engine {
	return &Engine{
		providers: make(map[string]Provider),
		policy:    NewPolicyEngine(),
		modelMap: map[Mode]string{
			ModeReasoning: "reasoning-model",
			ModeAnalysis:  "analysis-model",
			ModeDiagnosis: "diagnosis-model",
			ModePlanning:  "planning-model",
			ModeDefault:   "default-model",
		},
	}
}

// ================================
// Provider Registration
// ================================

func (e *Engine) RegisterProvider(p Provider) {
	e.providers[p.Name()] = p
}

// ================================
// Core Execution
// ================================

func (e *Engine) Generate(ctx context.Context, req Request) (Response, error) {
	start := time.Now()

	// Validate
	if req.Prompt == "" {
		return Response{}, errors.New("empty prompt")
	}

	// Mode defaults
	if req.Mode == "" {
		req.Mode = ModeDefault
	}

	// Policy routing
	providerName, err := e.policy.SelectProvider(req.Mode)
	if err != nil {
		return Response{}, err
	}

	provider, ok := e.providers[providerName]
	if !ok {
		return Response{}, errors.New("llm provider not registered: " + providerName)
	}

	// Model routing
	model := e.modelMap[req.Mode]
	if model == "" {
		model = e.modelMap[ModeDefault]
	}

	// Defaults
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}

	// Build provider request
	pReq := ProviderRequest{
		Prompt:      req.Prompt,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Model:       model,
	}

	// Execute
	pResp, err := provider.Generate(ctx, pReq)
	if err != nil {
		return Response{}, err
	}

	latency := time.Since(start)

	// Build response
	resp := Response{
		Text:     pResp.Text,
		Tokens:   pResp.Tokens,
		Latency:  latency,
		Provider: provider.Name(),
		Model:    pResp.Model,
	}

	// Confidence estimation (simple heuristic)
	resp.Confidence = e.policy.EstimateConfidence(req.Mode, pResp.Text)

	return resp, nil
}

// ================================
// Policy Engine
// ================================

type PolicyEngine struct{}

func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{}
}

// Select provider based on reasoning mode
func (p *PolicyEngine) SelectProvider(mode Mode) (string, error) {
	switch mode {
	case ModeReasoning:
		return "primary", nil
	case ModeAnalysis:
		return "primary", nil
	case ModeDiagnosis:
		return "primary", nil
	case ModePlanning:
		return "primary", nil
	default:
		return "primary", nil
	}
}

// Simple confidence estimator (can be replaced with model-based scoring)
func (p *PolicyEngine) EstimateConfidence(mode Mode, text string) float64 {
	l := len(text)
	switch {
	case l > 1500:
		return 0.95
	case l > 800:
		return 0.85
	case l > 300:
		return 0.7
	default:
		return 0.5
	}
}
