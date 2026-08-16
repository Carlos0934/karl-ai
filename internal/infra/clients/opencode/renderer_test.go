package opencode_test

import (
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/clients/opencode"
)

func TestRender(t *testing.T) {
	agent := domain.Agent{
		ID:          domain.AgentOrchestrator,
		Description: "Main orchestrator agent",
		Prompt:      "You are the orchestrator.\nLine 2.",
	}
	variantStr := "xhigh"
	settings := domain.AgentConfig{
		Model:   "openai/gpt-5.6-terra",
		Variant: &variantStr,
	}

	content := opencode.Render(agent, settings)

	if !strings.Contains(content, `description: "Main orchestrator agent"`) {
		t.Errorf("missing description: %s", content)
	}
	if !strings.Contains(content, `mode: primary`) {
		t.Errorf("missing mode primary: %s", content)
	}
	if !strings.Contains(content, `model: "openai/gpt-5.6-terra"`) {
		t.Errorf("missing model: %s", content)
	}
	if !strings.Contains(content, `variant: "xhigh"`) {
		t.Errorf("missing variant: %s", content)
	}
	if !strings.Contains(content, `permission: deny`) {
		t.Errorf("missing permission: %s", content)
	}
	if !strings.Contains(content, "You are the orchestrator.\nLine 2.\n") {
		t.Errorf("missing prompt body: %s", content)
	}
}

func TestParseAgentConfig(t *testing.T) {
	markdown := []byte(`---
description: "Test agent"
mode: primary
model: "anthropic/claude-3-5-sonnet"
variant: "high"
permission: deny
---

Prompt content goes here.
`)

	cfg, err := opencode.ParseAgentConfig(markdown)
	if err != nil {
		t.Fatalf("ParseAgentConfig failed: %v", err)
	}
	if cfg.Model != "anthropic/claude-3-5-sonnet" {
		t.Errorf("expected model 'anthropic/claude-3-5-sonnet', got %q", cfg.Model)
	}
	if cfg.Variant == nil || *cfg.Variant != "high" {
		t.Errorf("expected variant 'high', got %v", cfg.Variant)
	}
}
