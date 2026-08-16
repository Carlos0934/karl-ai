package clients

import (
	"context"
	"fmt"
	"sort"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/clients/opencode"
)

// Registry manages the set of supported AI agent clients and their discovery sources.
type Registry struct {
	clients     map[string]domain.AgentClient
	discoveries map[string]domain.Discovery
}

func New() *Registry {
	r := &Registry{
		clients:     make(map[string]domain.AgentClient),
		discoveries: make(map[string]domain.Discovery),
	}
	// Register default supported clients
	openCodeClient := opencode.NewClient()
	openCodeDiscovery := opencode.NewDiscovery()
	r.Register(openCodeClient, openCodeDiscovery)
	return r
}

func (r *Registry) Register(client domain.AgentClient, discovery domain.Discovery) {
	r.clients[client.ID()] = client
	if discovery != nil {
		r.discoveries[client.ID()] = discovery
	}
}

func (r *Registry) Client(id string) (domain.AgentClient, error) {
	client, ok := r.clients[id]
	if !ok {
		return nil, fmt.Errorf("unsupported client %q", id)
	}
	return client, nil
}

func (r *Registry) AllClients() []domain.AgentClient {
	clients := make([]domain.AgentClient, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].ID() < clients[j].ID() })
	return clients
}

// Clients implements domain.Discovery across all registered clients.
func (r *Registry) Clients(ctx context.Context, root string) ([]domain.ClientInfo, error) {
	infos := make([]domain.ClientInfo, 0, len(r.clients))
	for _, c := range r.AllClients() {
		infos = append(infos, c.Info())
	}
	return infos, nil
}

// Catalog implements domain.Discovery delegating to the client's discovery source.
func (r *Registry) Catalog(ctx context.Context, root, clientID string) (domain.ModelCatalog, error) {
	disc, ok := r.discoveries[clientID]
	if !ok {
		return domain.ModelCatalog{}, fmt.Errorf("unsupported model discovery client %q", clientID)
	}
	return disc.Catalog(ctx, root, clientID)
}
