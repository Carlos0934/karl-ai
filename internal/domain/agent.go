package domain

type AgentID string

const AgentOrchestrator AgentID = "karl-orchestrator"

type Agent struct {
	ID          AgentID
	Description string
	Prompt      string
}

type AgentConfig struct {
	Model   string  `json:"model"`
	Variant *string `json:"variant,omitempty"`
}

func DefaultAgentConfig(id AgentID) AgentConfig {
	variantHigh := "xhigh"
	switch id {
	case AgentOrchestrator:
		return AgentConfig{
			Model:   "openai/gpt-5.6-terra",
			Variant: &variantHigh,
		}
	default:
		return AgentConfig{
			Model: "openai/gpt-5.6-terra",
		}
	}
}
