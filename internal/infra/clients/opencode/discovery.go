package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/carlos0934/karl-ai/internal/domain"
)

// CommandDiscovery discovers OpenCode models through its CLI.
type CommandDiscovery struct {
	Binary     string
	runCommand func(context.Context, string, ...string) ([]byte, error)
}

// NewDiscovery returns the default OpenCode discovery implementation.
func NewDiscovery() *CommandDiscovery {
	return &CommandDiscovery{Binary: "opencode"}
}

// NewCustomDiscovery returns a discovery implementation with a custom runner for testing.
func NewCustomDiscovery(binary string, runner func(context.Context, string, ...string) ([]byte, error)) *CommandDiscovery {
	return &CommandDiscovery{
		Binary:     binary,
		runCommand: runner,
	}
}

func (discovery *CommandDiscovery) Clients(context.Context, string) ([]domain.ClientInfo, error) {
	return []domain.ClientInfo{{ID: domain.ClientOpenCode, Name: "OpenCode"}}, nil
}

func (discovery *CommandDiscovery) Catalog(ctx context.Context, root, clientID string) (domain.ModelCatalog, error) {
	if clientID != domain.ClientOpenCode {
		return domain.ModelCatalog{}, fmt.Errorf("unsupported model discovery client %q", clientID)
	}
	output, err := discovery.run(ctx, root, "models", "--verbose")
	if err != nil {
		return domain.ModelCatalog{}, err
	}
	return ParseCatalog(output)
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
			return nil, fmt.Errorf("%w: %s", domain.ErrClientNotFound, discovery.Binary)
		}
		if exitError, ok := err.(*exec.ExitError); ok && len(exitError.Stderr) > 0 {
			return nil, fmt.Errorf("run opencode %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitError.Stderr)))
		}
		return nil, fmt.Errorf("run opencode %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func ParseCatalog(output []byte) (domain.ModelCatalog, error) {
	lines := strings.Split(string(output), "\n")
	models := map[string]map[string]domain.Model{}
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
				return domain.ModelCatalog{}, fmt.Errorf("parse opencode model metadata: %w", err)
			}
			modelID := pendingID
			if modelID == "" {
				modelID = metadata.ID
			}
			if metadata.ProviderID == "" {
				metadata.ProviderID, _, _ = strings.Cut(modelID, "/")
			}
			if modelID != "" && !strings.HasPrefix(modelID, metadata.ProviderID+"/") {
				modelID = metadata.ProviderID + "/" + modelID
			}
			if modelID != "" && metadata.ProviderID != "" {
				variants := make([]string, 0, len(metadata.Variants))
				for variant := range metadata.Variants {
					variants = append(variants, variant)
				}
				sort.Strings(variants)
				if models[metadata.ProviderID] == nil {
					models[metadata.ProviderID] = map[string]domain.Model{}
				}
				models[metadata.ProviderID][modelID] = domain.Model{
					ID:          modelID,
					Name:        metadata.Name,
					ReleaseDate: metadata.ReleaseDate,
					Variants:    variants,
				}
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

	result := domain.ModelCatalog{ModelsByProvider: make(map[string][]domain.Model, len(models))}
	for providerID, providerModels := range models {
		result.Providers = append(result.Providers, domain.Provider{ID: providerID})
		for _, model := range providerModels {
			result.ModelsByProvider[providerID] = append(result.ModelsByProvider[providerID], model)
		}
		sort.Slice(result.ModelsByProvider[providerID], func(i, j int) bool {
			return result.ModelsByProvider[providerID][i].ID < result.ModelsByProvider[providerID][j].ID
		})
	}
	sort.Slice(result.Providers, func(i, j int) bool { return result.Providers[i].ID < result.Providers[j].ID })
	return result, nil
}

type modelMetadata struct {
	ID          string                     `json:"id"`
	ProviderID  string                     `json:"providerID"`
	Name        string                     `json:"name"`
	ReleaseDate string                     `json:"release_date"`
	Variants    map[string]json.RawMessage `json:"variants"`
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
