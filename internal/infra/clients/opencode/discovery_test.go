package opencode_test

import (
	"context"
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/clients/opencode"
)

func TestParseCatalog_ValidOutput(t *testing.T) {
	fixture := []byte(`
openai/gpt-4o
{
  "id": "gpt-4o",
  "providerID": "openai",
  "name": "GPT-4o",
  "release_date": "2024-05-13",
  "variants": {
    "high": {},
    "low": {}
  }
}

anthropic/claude-3-5-sonnet
{
  "id": "claude-3-5-sonnet",
  "providerID": "anthropic",
  "name": "Claude 3.5 Sonnet",
  "release_date": "2024-06-20",
  "variants": {}
}
`)

	catalog, err := opencode.ParseCatalog(fixture)
	if err != nil {
		t.Fatalf("ParseCatalog returned unexpected error: %v", err)
	}

	if len(catalog.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(catalog.Providers))
	}

	if catalog.Providers[0].ID != "anthropic" || catalog.Providers[1].ID != "openai" {
		t.Errorf("unexpected providers order: %v", catalog.Providers)
	}

	openaiModels := catalog.ModelsByProvider["openai"]
	if len(openaiModels) != 1 {
		t.Fatalf("expected 1 openai model, got %d", len(openaiModels))
	}
	model := openaiModels[0]
	if model.ID != "openai/gpt-4o" {
		t.Errorf("expected model ID 'openai/gpt-4o', got %q", model.ID)
	}
	if model.Name != "GPT-4o" {
		t.Errorf("expected name 'GPT-4o', got %q", model.Name)
	}
	if len(model.Variants) != 2 {
		t.Errorf("expected 2 variants, got %v", model.Variants)
	}
}

func TestCommandDiscovery_Clients(t *testing.T) {
	discovery := opencode.NewDiscovery()
	clients, err := discovery.Clients(context.Background(), ".")
	if err != nil {
		t.Fatalf("Clients returned error: %v", err)
	}
	if len(clients) != 1 || clients[0].ID != domain.ClientOpenCode {
		t.Errorf("unexpected clients: %v", clients)
	}
}

func TestCommandDiscovery_RunnerMock(t *testing.T) {
	customDiscovery := opencode.NewCustomDiscovery("opencode", func(ctx context.Context, root string, args ...string) ([]byte, error) {
		return []byte(`
openai/gpt-4o
{
  "id": "gpt-4o",
  "providerID": "openai",
  "name": "GPT-4o"
}
`), nil
	})

	catalog, err := customDiscovery.Catalog(context.Background(), ".", domain.ClientOpenCode)
	if err != nil {
		t.Fatalf("Catalog failed with runner mock: %v", err)
	}
	if len(catalog.Providers) != 1 || catalog.Providers[0].ID != "openai" {
		t.Errorf("unexpected providers: %v", catalog.Providers)
	}
}

func TestCommandDiscovery_UnsupportedClient(t *testing.T) {
	discovery := opencode.NewDiscovery()
	_, err := discovery.Catalog(context.Background(), ".", "unsupported-client")
	if err == nil {
		t.Error("expected error for unsupported client, got nil")
	}
}

func TestCommandDiscovery_NilDiscovery(t *testing.T) {
	var discovery *opencode.CommandDiscovery
	_, err := discovery.Catalog(context.Background(), ".", domain.ClientOpenCode)
	if err == nil {
		t.Error("expected error for nil discovery, got nil")
	}
}
