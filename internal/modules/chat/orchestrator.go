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
// Session Model (uses types ONLY)
// ================================

type FiveWhySession struct {
	SessionID string                 `json:"session_id"`
	Steps     []types.FiveWhyStep    `json:"steps"`
	RootCause types.RootCauseResult  `json:"root_cause"`
	Solution  types.SolutionResult   `json:"solution"`
	Reframed  types.ReframedQuestion `json:"reframed"`
	CreatedAt time.Time              `json:"created_at"`
}

// ================================
// Orchestrator
// ================================

type Orchestrator struct {
	llm    *llm.Manager // ✅ POINTER
	vector vector.Store
	prompt prompt.Builder
}

// ✅ POINTER IN CONSTRUCTOR
func NewOrchestrator(llmMgr *llm.Manager, vstore vector.Store, pb prompt.Builder) *Orchestrator {
	return &Orchestrator{
		llm:    llmMgr,
		vector: vstore,
		prompt: pb,
	}
}

// ================================
// Full 5-Why Pipeline
// ================================

func (o *Orchestrator) RunFiveWhy(
	ctx context.Context,
	sessionID string,
	userQuestion string,
) (*FiveWhySession, error) {

	if userQuestion == "" {
		return nil, errors.New("empty question")
	}

	session := &FiveWhySession{
		SessionID: sessionID,
		Steps:     []types.FiveWhyStep{},
		CreatedAt: time.Now(),
	}

	currentQuestion := userQuestion

	// ================================
	// 5 WHY LOOP
	// ================================
	for i := 1; i <= 5; i++ {

		// WHY prompt
		whyPrompt := o.prompt.BuildFiveWhyPrompt(i, currentQuestion)

		resp, err := o.llm.Generate(ctx, llm.Request{
			Mode:   llm.ModeReasoning,
			Prompt: whyPrompt,
		})
		if err != nil {
			return nil, err
		}

		// Evaluation
		analysisPrompt := o.prompt.BuildEvaluationPrompt(currentQuestion, resp.Text)

		analysisResp, err := o.llm.Generate(ctx, llm.Request{
			Mode:   llm.ModeAnalysis,
			Prompt: analysisPrompt,
		})
		if err != nil {
			return nil, err
		}

		step := types.FiveWhyStep{
			Level:    i,
			Question: currentQuestion,
			Answer:   resp.Text,
			Analysis: analysisResp.Text,
		}

		session.Steps = append(session.Steps, step)

		// Generate next WHY
		nextWhyPrompt := o.prompt.BuildNextWhyPrompt(resp.Text)

		nextResp, err := o.llm.Generate(ctx, llm.Request{
			Mode:   llm.ModeReasoning,
			Prompt: nextWhyPrompt,
		})
		if err != nil {
			return nil, err
		}

		currentQuestion = nextResp.Text
	}

	// ================================
	// ROOT CAUSE EXTRACTION
	// ================================

	rcaPrompt := o.prompt.BuildRootCausePrompt(session.Steps)

	rcaResp, err := o.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeDiagnosis,
		Prompt: rcaPrompt,
	})
	if err != nil {
		return nil, err
	}

	var rootCause types.RootCauseResult
	if err := o.prompt.ParseRootCause(rcaResp.Text, &rootCause); err != nil {
		return nil, err
	}

	session.RootCause = rootCause

	// ================================
	// SOLUTION SYNTHESIS
	// ================================

	solutionPrompt := o.prompt.BuildSolutionPrompt(rootCause, session.Steps)

	solResp, err := o.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModePlanning,
		Prompt: solutionPrompt,
	})
	if err != nil {
		return nil, err
	}

	var solution types.SolutionResult
	if err := o.prompt.ParseSolution(solResp.Text, &solution); err != nil {
		return nil, err
	}

	session.Solution = solution

	// ================================
	// QUESTION REFRAMING
	// ================================

	reframePrompt := o.prompt.BuildReframePrompt(userQuestion, rootCause)

	reframeResp, err := o.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeReasoning,
		Prompt: reframePrompt,
	})
	if err != nil {
		return nil, err
	}

	var reframed types.ReframedQuestion
	if err := o.prompt.ParseReframe(reframeResp.Text, &reframed); err != nil {
		return nil, err
	}

	session.Reframed = reframed

	// ================================
	// MEMORY STORAGE (VECTOR DB)
	// ================================

	_ = o.vector.Store(ctx, vector.Document{
		ID:      sessionID,
		Content: userQuestion,
		Meta: map[string]string{
			"type": "question",
		},
	})

	_ = o.vector.Store(ctx, vector.Document{
		ID:      sessionID + "_rca",
		Content: rootCause.RootCause,
		Meta: map[string]string{
			"type": "root_cause",
		},
	})

	_ = o.vector.Store(ctx, vector.Document{
		ID:      sessionID + "_solution",
		Content: solResp.Text,
		Meta: map[string]string{
			"type": "solution",
		},
	})

	return session, nil
}

// ================================
// Partial APIs
// ================================

func (o *Orchestrator) ExtractRootCause(
	ctx context.Context,
	steps []types.FiveWhyStep,
) (*types.RootCauseResult, error) {

	rcaPrompt := o.prompt.BuildRootCausePrompt(steps)

	rcaResp, err := o.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeDiagnosis,
		Prompt: rcaPrompt,
	})
	if err != nil {
		return nil, err
	}

	var rootCause types.RootCauseResult
	if err := o.prompt.ParseRootCause(rcaResp.Text, &rootCause); err != nil {
		return nil, err
	}

	return &rootCause, nil
}

func (o *Orchestrator) ReframeQuestion(
	ctx context.Context,
	question string,
	rc types.RootCauseResult,
) (*types.ReframedQuestion, error) {

	reframePrompt := o.prompt.BuildReframePrompt(question, rc)

	reframeResp, err := o.llm.Generate(ctx, llm.Request{
		Mode:   llm.ModeReasoning,
		Prompt: reframePrompt,
	})
	if err != nil {
		return nil, err
	}

	var reframed types.ReframedQuestion
	if err := o.prompt.ParseReframe(reframeResp.Text, &reframed); err != nil {
		return nil, err
	}

	return &reframed, nil
}
