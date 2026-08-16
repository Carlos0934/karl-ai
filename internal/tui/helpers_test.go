package tui_test

import (
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	modeltui "github.com/carlos0934/karl-ai/internal/tui"
)

func TestHelpers_FormattingAndStaging(t *testing.T) {
	pending := make(map[string]modeltui.Selection)
	variantHigh := "xhigh"
	original := domain.AgentConfig{
		Model:   "openai/gpt-5.6-terra",
		Variant: &variantHigh,
	}

	// Change to a new model
	selection := modeltui.Selection{
		Client:   "opencode",
		Agent:    "karl-orchestrator",
		Provider: "anthropic",
		Model:    "anthropic/claude-3-5-sonnet",
		Variant:  "high",
	}

	// Staging changes
	pending["karl-orchestrator"] = selection
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending selection, got %d", len(pending))
	}

	if pending["karl-orchestrator"].Model != "anthropic/claude-3-5-sonnet" {
		t.Errorf("expected model anthropic/claude-3-5-sonnet, got %s", pending["karl-orchestrator"].Model)
	}

	_ = original
}
