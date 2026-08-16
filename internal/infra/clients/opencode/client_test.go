package opencode_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/clients/opencode"
)

var testAgents = []domain.Agent{
	{
		ID:          domain.AgentOrchestrator,
		Description: "Orchestrator Agent",
		Prompt:      "Orchestrator prompt",
	},
}

func TestClient_InitAndIdempotentSync(t *testing.T) {
	tempDir := t.TempDir()
	client := opencode.NewClient()

	if client.Installed(tempDir) {
		t.Error("expected Installed to be false initially")
	}

	// 1. Init
	initRes, err := client.Init(context.Background(), tempDir, testAgents)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !initRes.Changed {
		t.Error("expected init Changed = true")
	}

	agentFile := filepath.Join(tempDir, ".opencode", "agents", "karl-orchestrator.md")
	if _, err := os.Stat(agentFile); err != nil {
		t.Fatalf("expected agent file to exist: %v", err)
	}

	if !client.Installed(tempDir) {
		t.Error("expected Installed to be true after init")
	}

	// 2. ReadAgentConfig directly from native file
	cfg, err := client.ReadAgentConfig(context.Background(), tempDir, domain.AgentOrchestrator)
	if err != nil {
		t.Fatalf("ReadAgentConfig failed: %v", err)
	}
	if cfg.Model != "openai/gpt-5.6-terra" {
		t.Errorf("expected default model openai/gpt-5.6-terra, got %s", cfg.Model)
	}

	// 3. Sync immediately -> Idempotent (Changed = false)
	syncRes, err := client.Sync(context.Background(), tempDir, testAgents, domain.SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if syncRes.Changed {
		t.Errorf("expected idempotent sync Changed = false, got true with paths: %v", syncRes.Paths)
	}
}

func TestClient_DriftDetectionAndForce(t *testing.T) {
	tempDir := t.TempDir()
	client := opencode.NewClient()

	if _, err := client.Init(context.Background(), tempDir, testAgents); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Tamper with file
	agentFile := filepath.Join(tempDir, ".opencode", "agents", "karl-orchestrator.md")
	if err := os.WriteFile(agentFile, []byte("unmanaged manual edits"), 0o644); err != nil {
		t.Fatalf("failed to tamper file: %v", err)
	}

	// Sync should detect drift
	_, err := client.Sync(context.Background(), tempDir, testAgents, domain.SyncOptions{})
	if err == nil || !errors.Is(err, domain.ErrDrift) {
		t.Fatalf("expected ErrDrift, got: %v", err)
	}

	// Force sync should overwrite
	forceRes, err := client.Sync(context.Background(), tempDir, testAgents, domain.SyncOptions{Force: true})
	if err != nil {
		t.Fatalf("Force sync failed: %v", err)
	}
	if !forceRes.Changed {
		t.Error("expected Force sync Changed = true")
	}
}

func TestClient_SyncCheckMode(t *testing.T) {
	tempDir := t.TempDir()
	client := opencode.NewClient()

	if _, err := client.Init(context.Background(), tempDir, testAgents); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Clean check
	checkRes, err := client.Sync(context.Background(), tempDir, testAgents, domain.SyncOptions{Check: true})
	if err != nil {
		t.Fatalf("Check on clean failed: %v", err)
	}
	if checkRes.Operation != "check" {
		t.Errorf("expected check operation, got %s", checkRes.Operation)
	}

	// Delete agent file -> out of sync
	agentFile := filepath.Join(tempDir, ".opencode", "agents", "karl-orchestrator.md")
	_ = os.Remove(agentFile)

	_, err = client.Sync(context.Background(), tempDir, testAgents, domain.SyncOptions{Check: true})
	if err == nil || !errors.Is(err, domain.ErrOutOfSync) {
		t.Fatalf("expected ErrOutOfSync, got: %v", err)
	}
}

func TestClient_ConfigureModelsDirectlyInNativeFile(t *testing.T) {
	tempDir := t.TempDir()
	client := opencode.NewClient()

	if _, err := client.Init(context.Background(), tempDir, testAgents); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Configure new model
	result, err := client.ConfigureModels(context.Background(), tempDir, testAgents, []domain.ModelSelection{
		{
			Agent:   string(domain.AgentOrchestrator),
			Model:   "anthropic/claude-3-5-sonnet",
			Variant: "high",
		},
	})
	if err != nil {
		t.Fatalf("ConfigureModels failed: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}

	// Read directly from file
	cfg, err := client.ReadAgentConfig(context.Background(), tempDir, domain.AgentOrchestrator)
	if err != nil {
		t.Fatalf("ReadAgentConfig failed: %v", err)
	}
	if cfg.Model != "anthropic/claude-3-5-sonnet" {
		t.Errorf("expected updated model 'anthropic/claude-3-5-sonnet', got %s", cfg.Model)
	}
	if cfg.Variant == nil || *cfg.Variant != "high" {
		t.Errorf("expected updated variant 'high', got %v", cfg.Variant)
	}

	// Check file content
	agentFile := filepath.Join(tempDir, ".opencode", "agents", "karl-orchestrator.md")
	content, _ := os.ReadFile(agentFile)
	if !strings.Contains(string(content), `model: "anthropic/claude-3-5-sonnet"`) {
		t.Errorf("file missing new model: %s", string(content))
	}
}

func TestClient_Uninstall(t *testing.T) {
	tempDir := t.TempDir()
	client := opencode.NewClient()

	if _, err := client.Init(context.Background(), tempDir, testAgents); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	uninstRes, err := client.Uninstall(context.Background(), tempDir, testAgents, false)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !uninstRes.Changed {
		t.Error("expected Uninstall Changed = true")
	}

	agentFile := filepath.Join(tempDir, ".opencode", "agents", "karl-orchestrator.md")
	if _, err := os.Stat(agentFile); !os.IsNotExist(err) {
		t.Errorf("expected agent file to be deleted, got %v", err)
	}
}
