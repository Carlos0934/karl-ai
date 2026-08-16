package domain_test

import (
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
)

func TestValidateModelReference(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{"valid openai", "openai/gpt-4o", false},
		{"valid anthropic", "anthropic/claude-3-5-sonnet", false},
		{"valid custom provider", "custom-provider/model-1", false},
		{"empty string", "", true},
		{"no provider", "gpt-4o", true},
		{"trailing slash", "openai/", true},
		{"leading slash", "/gpt-4o", true},
		{"whitespace inside", "openai/gpt 4o", true},
		{"leading whitespace", " openai/gpt-4o", true},
		{"contains hash", "openai/gpt-4o#1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateModelReference(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelReference(%q) error = %v, wantErr %v", tt.model, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVariant(t *testing.T) {
	tests := []struct {
		name    string
		variant string
		wantErr bool
	}{
		{"valid standard", "xhigh", false},
		{"valid alphanumeric", "v2-turbo", false},
		{"empty string", "", false},
		{"contains spaces", "x high", true},
		{"contains hash", "x#high", true},
		{"leading space", " xhigh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateVariant(tt.variant)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVariant(%q) error = %v, wantErr %v", tt.variant, err, tt.wantErr)
			}
		})
	}
}

func TestValidateManagedPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"valid relative path", ".opencode/agents/karl-orchestrator.md", false},
		{"valid wildcard directory", ".opencode/agents/**", false},
		{"empty pattern", "", true},
		{"dot pattern", ".", true},
		{"backslash path", ".opencode\\agents", true},
		{"path traversal double dot", ".opencode/../agents", true},
		{"leading slash absolute", "/etc/hosts", true},
		{"windows drive absolute", "C:/agents", true},
		{"invalid wildcard middle", ".opencode/*/agents/**", true},
		{"invalid question mark wildcard", ".opencode/agent?.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateManagedPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateManagedPattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestDefaultAgentConfig(t *testing.T) {
	orchestrator := domain.DefaultAgentConfig(domain.AgentOrchestrator)
	if orchestrator.Model != "openai/gpt-5.6-terra" {
		t.Fatalf("expected orchestrator model openai/gpt-5.6-terra, got %s", orchestrator.Model)
	}
	if orchestrator.Variant == nil || *orchestrator.Variant != "xhigh" {
		t.Fatalf("expected variant xhigh")
	}
}
