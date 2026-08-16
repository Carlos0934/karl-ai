package catalog_test

import (
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/internal/catalog"
	"github.com/carlos0934/karl-ai/internal/domain"
)

func TestAgentsIntegrity(t *testing.T) {
	agents := catalog.Agents()
	if len(agents) == 0 {
		t.Fatal("expected at least one agent in catalog")
	}

	seen := make(map[domain.AgentID]bool, len(agents))
	for _, agent := range agents {
		if agent.ID == "" {
			t.Error("agent has empty ID")
		}
		if seen[agent.ID] {
			t.Errorf("duplicate agent ID in catalog: %s", agent.ID)
		}
		seen[agent.ID] = true

		if strings.TrimSpace(agent.Description) == "" {
			t.Errorf("agent %s has empty description", agent.ID)
		}
		if strings.TrimSpace(agent.Prompt) == "" {
			t.Errorf("agent %s has empty prompt", agent.ID)
		}
	}
}

func TestAgentByID(t *testing.T) {
	orchestrator, ok := catalog.AgentByID(domain.AgentOrchestrator)
	if !ok {
		t.Fatalf("expected agent %s to be found", domain.AgentOrchestrator)
	}
	if orchestrator.ID != domain.AgentOrchestrator {
		t.Errorf("expected ID %s, got %s", domain.AgentOrchestrator, orchestrator.ID)
	}

	_, found := catalog.AgentByID("non-existent-agent")
	if found {
		t.Error("expected non-existent agent to not be found")
	}
}
