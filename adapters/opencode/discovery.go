package opencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ErrClientNotFound indicates that the OpenCode executable is not available.
var ErrClientNotFound = errors.New("OpenCode executable was not found")

// ClientInfo identifies a client that can provide model discovery.
type ClientInfo struct {
	ID   string
	Name string
}

// Provider identifies a provider returned by the selected client.
type Provider struct {
	ID string
}

// Model describes a model returned by the selected provider.
type Model struct {
	ID       string
	Name     string
	Variants []string
}

// Discovery is the client-neutral seam used by the model configuration flow.
// Implementations own the client-specific discovery details.
type Discovery interface {
	Clients(context.Context, string) ([]ClientInfo, error)
	Providers(context.Context, string, string) ([]Provider, error)
	Models(context.Context, string, string, string) ([]Model, error)
}

// CommandDiscovery discovers OpenCode models through its CLI.
type CommandDiscovery struct {
	Binary     string
	runCommand func(context.Context, string, ...string) ([]byte, error)
}

// NewDiscovery returns the default OpenCode discovery implementation.
func NewDiscovery() *CommandDiscovery {
	return &CommandDiscovery{Binary: "opencode"}
}

func (discovery *CommandDiscovery) Clients(context.Context, string) ([]ClientInfo, error) {
	return []ClientInfo{{ID: Client, Name: "OpenCode"}}, nil
}

func (discovery *CommandDiscovery) Providers(ctx context.Context, root, clientID string) ([]Provider, error) {
	if clientID != Client {
		return nil, fmt.Errorf("unsupported model discovery client %q", clientID)
	}
	output, err := discovery.run(ctx, root, "models")
	if err != nil {
		return nil, err
	}
	return parseProviders(output), nil
}

func (discovery *CommandDiscovery) Models(ctx context.Context, root, clientID, providerID string) ([]Model, error) {
	if clientID != Client {
		return nil, fmt.Errorf("unsupported model discovery client %q", clientID)
	}
	if providerID == "" {
		return nil, fmt.Errorf("model provider cannot be empty")
	}
	output, err := discovery.run(ctx, root, "models", "--verbose", providerID)
	if err != nil {
		return nil, err
	}
	return parseModels(output, providerID)
}

func (discovery *CommandDiscovery) run(ctx context.Context, root string, args ...string) ([]byte, error) {
	if discovery == nil {
		return nil, fmt.Errorf("OpenCode executable is not configured")
	}
	if discovery.runCommand != nil {
		return discovery.runCommand(ctx, root, args...)
	}
	if discovery.Binary == "" {
		return nil, fmt.Errorf("OpenCode executable is not configured")
	}
	command := exec.CommandContext(ctx, discovery.Binary, args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrClientNotFound, discovery.Binary)
		}
		if exitError, ok := err.(*exec.ExitError); ok && len(exitError.Stderr) > 0 {
			return nil, fmt.Errorf("run opencode %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitError.Stderr)))
		}
		return nil, fmt.Errorf("run opencode %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func parseProviders(output []byte) []Provider {
	providers := map[string]struct{}{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		provider, _, ok := strings.Cut(line, "/")
		if ok && provider != "" {
			providers[provider] = struct{}{}
		}
	}
	result := make([]Provider, 0, len(providers))
	for provider := range providers {
		result = append(result, Provider{ID: provider})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func parseModels(output []byte, providerID string) ([]Model, error) {
	lines := strings.Split(string(output), "\n")
	models := map[string]Model{}
	pendingID := ""
	for index := 0; index < len(lines); {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			index++
			continue
		}
		if strings.HasPrefix(line, "{") {
			metadata, consumed, err := decodeModelMetadata(lines[index:])
			if err != nil {
				return nil, fmt.Errorf("parse opencode model metadata: %w", err)
			}
			modelID := pendingID
			if modelID == "" {
				modelID = metadata.ID
			}
			if metadata.ProviderID == "" {
				metadata.ProviderID, _, _ = strings.Cut(modelID, "/")
				if metadata.ProviderID == "" {
					metadata.ProviderID = providerID
				}
			}
			if modelID != "" && !strings.HasPrefix(modelID, metadata.ProviderID+"/") {
				modelID = metadata.ProviderID + "/" + modelID
			}
			if modelID != "" && metadata.ProviderID == providerID {
				variants := make([]string, 0, len(metadata.Variants))
				for variant := range metadata.Variants {
					variants = append(variants, variant)
				}
				sort.Strings(variants)
				models[modelID] = Model{ID: modelID, Name: metadata.Name, Variants: variants}
			}
			pendingID = ""
			index += consumed
			continue
		}
		if _, _, ok := strings.Cut(line, "/"); ok {
			pendingID = line
		}
		index++
	}

	result := make([]Model, 0, len(models))
	for _, model := range models {
		result = append(result, model)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

type modelMetadata struct {
	ID         string                     `json:"id"`
	ProviderID string                     `json:"providerID"`
	Name       string                     `json:"name"`
	Variants   map[string]json.RawMessage `json:"variants"`
}

func decodeModelMetadata(lines []string) (modelMetadata, int, error) {
	suffix := strings.Join(lines, "\n")
	decoder := json.NewDecoder(strings.NewReader(suffix))
	var metadata modelMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return modelMetadata{}, 0, err
	}
	if metadata.Variants == nil {
		metadata.Variants = map[string]json.RawMessage{}
	}
	consumed := strings.Count(suffix[:decoder.InputOffset()], "\n") + 1
	return metadata, consumed, nil
}
