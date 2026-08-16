package clients_test

import (
	"context"
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	clients "github.com/carlos0934/karl-ai/internal/infra/clients"
)

func TestRegistry_DefaultClients(t *testing.T) {
	reg := clients.New()

	client, err := reg.Client(domain.ClientOpenCode)
	if err != nil {
		t.Fatalf("expected OpenCode client to be registered: %v", err)
	}
	if client.ID() != domain.ClientOpenCode {
		t.Errorf("expected ID %s, got %s", domain.ClientOpenCode, client.ID())
	}

	_, err = reg.Client("unsupported-client")
	if err == nil {
		t.Error("expected error for unsupported client, got nil")
	}

	clientInfos, err := reg.Clients(context.Background(), ".")
	if err != nil {
		t.Fatalf("reg.Clients failed: %v", err)
	}
	if len(clientInfos) == 0 {
		t.Fatal("expected at least one client info")
	}
}
