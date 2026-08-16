package opencode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/filesystem"
)

type Client struct {
	writeAtomic func(string, []byte, os.FileMode) error
}

func NewClient() *Client {
	return &Client{
		writeAtomic: filesystem.WriteAtomic,
	}
}

// NewCustomClient allows injecting custom writers for testing.
func NewCustomClient(writer func(string, []byte, os.FileMode) error) *Client {
	return &Client{
		writeAtomic: writer,
	}
}

func (c *Client) ID() string {
	return domain.ClientOpenCode
}

func (c *Client) Info() domain.ClientInfo {
	return domain.ClientInfo{
		ID:   domain.ClientOpenCode,
		Name: "OpenCode",
	}
}

func (c *Client) Installed(root string) bool {
	dir := filepath.Join(root, ".opencode", "agents")
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func (c *Client) ReadAgentConfig(ctx context.Context, root string, agentID domain.AgentID) (domain.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentConfig{}, err
	}
	targetPath := filepath.Join(root, filepath.FromSlash(AgentRelPath(agentID)))
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.DefaultAgentConfig(agentID), nil
		}
		return domain.AgentConfig{}, err
	}
	cfg, err := ParseAgentConfig(data)
	if err != nil {
		return domain.DefaultAgentConfig(agentID), nil
	}
	if cfg.Model == "" {
		cfg.Model = domain.DefaultAgentConfig(agentID).Model
	}
	if cfg.Variant == nil {
		cfg.Variant = domain.DefaultAgentConfig(agentID).Variant
	}
	return cfg, nil
}

func (c *Client) Init(ctx context.Context, root string, agents []domain.Agent) (domain.Result, error) {
	if err := ctx.Err(); err != nil {
		return domain.Result{}, err
	}
	changed := []string{}
	for _, agent := range agents {
		relPath := AgentRelPath(agent.ID)
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			content := Render(agent, domain.DefaultAgentConfig(agent.ID))
			if err := c.writeAtomic(targetPath, []byte(content), 0o644); err != nil {
				return domain.Result{}, err
			}
			changed = append(changed, relPath)
		} else if err != nil {
			return domain.Result{}, err
		}
	}
	sort.Strings(changed)
	return domain.Result{
		Operation: "init",
		Client:    domain.ClientOpenCode,
		Changed:   len(changed) > 0,
		Paths:     changed,
	}, nil
}

func (c *Client) Sync(ctx context.Context, root string, agents []domain.Agent, options domain.SyncOptions) (domain.Result, error) {
	if err := ctx.Err(); err != nil {
		return domain.Result{}, err
	}
	changed := []string{}
	drifted := []string{}

	type pendingWrite struct {
		target  string
		relPath string
		content []byte
	}
	var writes []pendingWrite

	for _, agent := range agents {
		relPath := AgentRelPath(agent.ID)
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))

		existingConfig, err := c.ReadAgentConfig(ctx, root, agent.ID)
		if err != nil {
			return domain.Result{}, err
		}
		expectedContent := []byte(Render(agent, existingConfig))

		data, err := os.ReadFile(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				changed = append(changed, relPath)
				writes = append(writes, pendingWrite{target: targetPath, relPath: relPath, content: expectedContent})
				continue
			}
			return domain.Result{}, err
		}

		if !bytes.Equal(data, expectedContent) {
			changed = append(changed, relPath)
			if !bytes.Contains(data, []byte("mode: primary\n")) {
				drifted = append(drifted, relPath)
			}
			writes = append(writes, pendingWrite{target: targetPath, relPath: relPath, content: expectedContent})
		}
	}

	if len(drifted) > 0 && !options.Force {
		sort.Strings(drifted)
		return domain.Result{}, fmt.Errorf("%w: %s", domain.ErrDrift, strings.Join(drifted, ", "))
	}

	if options.Check {
		if len(changed) > 0 {
			sort.Strings(changed)
			return domain.Result{}, fmt.Errorf("%w: %s", domain.ErrOutOfSync, strings.Join(changed, ", "))
		}
		return domain.Result{
			Operation: "check",
			Client:    domain.ClientOpenCode,
			Paths:     []string{},
		}, nil
	}

	for _, w := range writes {
		if err := c.writeAtomic(w.target, w.content, 0o644); err != nil {
			return domain.Result{}, err
		}
	}

	sort.Strings(changed)
	return domain.Result{
		Operation: "sync",
		Client:    domain.ClientOpenCode,
		Changed:   len(changed) > 0,
		Paths:     changed,
	}, nil
}

func (c *Client) ConfigureModels(ctx context.Context, root string, agents []domain.Agent, selections []domain.ModelSelection) (domain.ModelConfigurationResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.ModelConfigurationResult{}, err
	}
	if len(selections) == 0 {
		return domain.ModelConfigurationResult{}, fmt.Errorf("model configuration batch cannot be empty")
	}

	agentMap := make(map[domain.AgentID]domain.Agent, len(agents))
	for _, agent := range agents {
		agentMap[agent.ID] = agent
	}

	for _, sel := range selections {
		if sel.Agent == "" {
			return domain.ModelConfigurationResult{}, fmt.Errorf("selected agent cannot be empty")
		}
		if _, ok := agentMap[domain.AgentID(sel.Agent)]; !ok {
			return domain.ModelConfigurationResult{}, fmt.Errorf("unknown agent %q", sel.Agent)
		}
		if err := domain.ValidateModelReference(sel.Model); err != nil {
			return domain.ModelConfigurationResult{}, fmt.Errorf("invalid model for agent %q: %w", sel.Agent, err)
		}
		if err := domain.ValidateVariant(sel.Variant); err != nil {
			return domain.ModelConfigurationResult{}, fmt.Errorf("invalid variant for agent %q: %w", sel.Agent, err)
		}
	}

	changedPaths := []string{}
	for _, sel := range selections {
		agent := agentMap[domain.AgentID(sel.Agent)]
		relPath := AgentRelPath(agent.ID)
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))

		variantStr := sel.Variant
		cfg := domain.AgentConfig{
			Model:   sel.Model,
			Variant: &variantStr,
		}
		content := Render(agent, cfg)
		if err := c.writeAtomic(targetPath, []byte(content), 0o644); err != nil {
			return domain.ModelConfigurationResult{}, fmt.Errorf("write agent %s: %w", agent.ID, err)
		}
		changedPaths = append(changedPaths, relPath)
	}

	sort.Strings(changedPaths)
	return domain.ModelConfigurationResult{
		Changes: selections,
		Sync: domain.Result{
			Operation: "sync",
			Client:    domain.ClientOpenCode,
			Changed:   len(changedPaths) > 0,
			Paths:     changedPaths,
		},
	}, nil
}

func (c *Client) Uninstall(ctx context.Context, root string, agents []domain.Agent, force bool) (domain.Result, error) {
	if err := ctx.Err(); err != nil {
		return domain.Result{}, err
	}
	removed := []string{}
	drifted := []string{}

	for _, agent := range agents {
		relPath := AgentRelPath(agent.ID)
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))
		data, err := os.ReadFile(targetPath)
		if err != nil {
			continue
		}
		if !bytes.Contains(data, []byte("mode: primary\n")) {
			drifted = append(drifted, relPath)
		}
	}

	if len(drifted) > 0 && !force {
		sort.Strings(drifted)
		return domain.Result{}, fmt.Errorf("%w: %s", domain.ErrDrift, strings.Join(drifted, ", "))
	}

	for _, agent := range agents {
		relPath := AgentRelPath(agent.ID)
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.Remove(targetPath); err == nil {
			removed = append(removed, relPath)
		}
	}

	if len(removed) == 0 {
		return domain.Result{}, domain.ErrNotFound
	}

	// Clean directory if empty
	agentsDir := filepath.Join(root, ".opencode", "agents")
	_ = os.Remove(agentsDir)
	opencodeDir := filepath.Join(root, ".opencode")
	_ = os.Remove(opencodeDir)

	sort.Strings(removed)
	return domain.Result{
		Operation: "uninstall",
		Client:    domain.ClientOpenCode,
		Changed:   len(removed) > 0,
		Paths:     removed,
	}, nil
}
