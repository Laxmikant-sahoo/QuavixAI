package types

// ================================
// 5-Why Domain Models
// ================================

type FiveWhyStep struct {
	Level    int    `json:"level"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Analysis string `json:"analysis"`
}

// ================================
// Root Cause Models
// ================================

type RootCauseResult struct {
	RootCause   string   `json:"root_cause"`
	Confidence  float64  `json:"confidence"`
	Evidence    []string `json:"evidence"`
	Category    string   `json:"category"`
	ImpactScope string   `json:"impact_scope"`
}

// ================================
// Solution Models
// ================================

type SolutionResult struct {
	Immediate   []string `json:"immediate_actions"`
	Strategic   []string `json:"strategic_actions"`
	Preventive  []string `json:"preventive_actions"`
	Automation  []string `json:"automation_opportunities"`
	Owner       string   `json:"owner"`
	Complexity  string   `json:"complexity"`
	TimeHorizon string   `json:"time_horizon"`
}

// ================================
// Reframing Models
// ================================

type ReframedQuestion struct {
	Original string `json:"original"`
	Reframed string `json:"reframed"`
	Intent   string `json:"intent"`
	Goal     string `json:"goal"`
}
