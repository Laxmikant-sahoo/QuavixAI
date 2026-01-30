package prompt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"quavixAI/internal/modules/types"
)

// ================================
// Builder Interface
// ================================

type Builder interface {
	BuildFiveWhyPrompt(level int, question string) string
	BuildEvaluationPrompt(question, answer string) string
	BuildNextWhyPrompt(answer string) string
	BuildRootCausePrompt(steps []types.FiveWhyStep) string
	BuildSolutionPrompt(rc types.RootCauseResult, steps []types.FiveWhyStep) string
	BuildReframePrompt(original string, rc types.RootCauseResult) string

	ParseRootCause(raw string, out *types.RootCauseResult) error
	ParseSolution(raw string, out *types.SolutionResult) error
	ParseReframe(raw string, out *types.ReframedQuestion) error
}

// ================================
// Implementation
// ================================

type PromptBuilder struct{}

func NewBuilder() Builder {
	return &PromptBuilder{}
}

// ================================
// Template Builders
// ================================

func (b *PromptBuilder) BuildFiveWhyPrompt(level int, question string) string {
	data := map[string]interface{}{
		"Level":    level,
		"Question": question,
	}
	return render(FiveWhyTemplate, data)
}

func (b *PromptBuilder) BuildEvaluationPrompt(question, answer string) string {
	data := map[string]interface{}{
		"Question": question,
		"Answer":   answer,
	}
	return render(EvaluationTemplate, data)
}

func (b *PromptBuilder) BuildNextWhyPrompt(answer string) string {
	data := map[string]interface{}{
		"Answer": answer,
	}
	return render(NextWhyTemplate, data)
}

func (b *PromptBuilder) BuildRootCausePrompt(steps []types.FiveWhyStep) string {
	var chain strings.Builder
	for _, s := range steps {
		chain.WriteString(fmt.Sprintf(
			"WHY %d:\nQ: %s\nA: %s\nANALYSIS: %s\n\n",
			s.Level, s.Question, s.Answer, s.Analysis,
		))
	}

	data := map[string]interface{}{
		"Chain": chain.String(),
	}

	return render(RootCauseTemplate, data)
}

func (b *PromptBuilder) BuildSolutionPrompt(rc types.RootCauseResult, steps []types.FiveWhyStep) string {
	var ev strings.Builder
	for _, s := range steps {
		ev.WriteString(fmt.Sprintf("- %s\n", s.Analysis))
	}

	data := map[string]interface{}{
		"RootCause": rc.RootCause,
		"Evidence":  ev.String(),
	}

	return render(SolutionTemplate, data)
}

func (b *PromptBuilder) BuildReframePrompt(original string, rc types.RootCauseResult) string {
	data := map[string]interface{}{
		"Original":  original,
		"RootCause": rc.RootCause,
	}

	return render(ReframeTemplate, data)
}

// ================================
// Parsers
// ================================

func (b *PromptBuilder) ParseRootCause(raw string, out *types.RootCauseResult) error {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonStr), out)
}

func (b *PromptBuilder) ParseSolution(raw string, out *types.SolutionResult) error {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonStr), out)
}

func (b *PromptBuilder) ParseReframe(raw string, out *types.ReframedQuestion) error {
	jsonStr, err := extractJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonStr), out)
}

// ================================
// Utilities
// ================================

func render(tpl string, data map[string]interface{}) string {
	t, err := template.New("prompt").Parse(tpl)
	if err != nil {
		return tpl
	}
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String()
}

func extractJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")

	if start == -1 || end == -1 || end <= start {
		return "", errors.New("no json found in llm output")
	}

	jsonStr := raw[start : end+1]

	// basic validation
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &js); err != nil {
		return "", errors.New("invalid json structure")
	}

	return jsonStr, nil
}
