package domain

import "context"

const (
	ClientOpenCode = "opencode"
)

// ClientInfo identifies an AI client supported by Karl.
type ClientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Result describes the outcome of a client projection operation.
type Result struct {
	Operation string   `json:"operation"`
	Client    string   `json:"client"`
	Changed   bool     `json:"changed"`
	Paths     []string `json:"paths"`
}

// SyncOptions provides configuration for projection synchronization.
type SyncOptions struct {
	Check bool
	Force bool
}

// AgentClient defines the contract for an AI client that manages its own native agent files.
type AgentClient interface {
	ID() string
	Info() ClientInfo
	Installed(root string) bool
	ReadAgentConfig(ctx context.Context, root string, agentID AgentID) (AgentConfig, error)
	Init(ctx context.Context, root string, agents []Agent) (Result, error)
	Sync(ctx context.Context, root string, agents []Agent, options SyncOptions) (Result, error)
	Uninstall(ctx context.Context, root string, agents []Agent, force bool) (Result, error)
	ConfigureModels(ctx context.Context, root string, agents []Agent, selections []ModelSelection) (ModelConfigurationResult, error)
}

// Discovery is the client-neutral seam used by the model configuration flow.
type Discovery interface {
	Clients(ctx context.Context, root string) ([]ClientInfo, error)
	Catalog(ctx context.Context, root, clientID string) (ModelCatalog, error)
}
