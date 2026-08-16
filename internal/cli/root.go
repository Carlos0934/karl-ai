package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/carlos0934/karl-ai/internal/catalog"
	"github.com/carlos0934/karl-ai/internal/domain"
	clients "github.com/carlos0934/karl-ai/internal/infra/clients"
	modeltui "github.com/carlos0934/karl-ai/internal/tui"
	"github.com/spf13/cobra"
)

// New constructs a complete CLI without relying on package-level state.
func New(version string, stdout, stderr io.Writer) *cobra.Command {
	reg := clients.New()
	return NewWithRegistry(version, stdout, stderr, reg)
}

// NewWithDiscovery constructs a CLI with an injected model discovery source.
func NewWithDiscovery(version string, stdout, stderr io.Writer, discovery domain.Discovery) *cobra.Command {
	reg := clients.New()
	return NewWithRegistryAndDiscovery(version, stdout, stderr, reg, discovery)
}

// NewWithRegistry constructs a CLI with an injected client clients.
func NewWithRegistry(version string, stdout, stderr io.Writer, reg *clients.Registry) *cobra.Command {
	return NewWithRegistryAndDiscovery(version, stdout, stderr, reg, reg)
}

func NewWithRegistryAndDiscovery(version string, stdout, stderr io.Writer, reg *clients.Registry, discovery domain.Discovery) *cobra.Command {
	root := &cobra.Command{
		Use:           "karl-ai",
		Short:         "Configure Karl agents and project them into AI clients",
		Example:       "  karl-ai init opencode\n  karl-ai models configure",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(
		newVersionCommand(version),
		newProjectionCommand("init", version, reg),
		newProjectionCommand("sync", version, reg),
		newProjectionCommand("uninstall", version, reg),
		newModelsCommand(version, reg, discovery),
	)
	root.InitDefaultCompletionCmd()
	return root
}

type clientModelSource struct {
	reg  *clients.Registry
	root string
}

func (s *clientModelSource) ModelForAgent(ctx context.Context, agent string) (domain.AgentConfig, error) {
	for _, client := range s.reg.AllClients() {
		cfg, err := client.ReadAgentConfig(ctx, s.root, domain.AgentID(agent))
		if err == nil {
			return cfg, nil
		}
	}
	return domain.DefaultAgentConfig(domain.AgentID(agent)), nil
}

func newModelsCommand(version string, reg *clients.Registry, discovery domain.Discovery) *cobra.Command {
	models := &cobra.Command{
		Use:   "models",
		Short: "Configure model selections",
		Args:  cobra.NoArgs,
	}
	var root string
	configure := &cobra.Command{
		Use:   "configure",
		Short: "Interactively configure agent models",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			modelSource := &clientModelSource{reg: reg, root: root}
			selections, err := modeltui.RunWithSavedModel(cmd.Context(), root, cmd.InOrStdin(), cmd.ErrOrStderr(), discovery, modelSource)
			if err != nil {
				return err
			}
			if err := cmd.Context().Err(); err != nil {
				return fmt.Errorf("%w: %v", modeltui.ErrCancelled, err)
			}

			selectionsByClient := make(map[string][]domain.ModelSelection)
			for _, selection := range selections {
				clientID := selection.Client
				if clientID == "" {
					clientID = domain.ClientOpenCode
				}
				selectionsByClient[clientID] = append(selectionsByClient[clientID], domain.ModelSelection{
					Client:  selection.Client,
					Agent:   selection.Agent,
					Model:   selection.Model,
					Variant: selection.Variant,
				})
			}

			var finalResult domain.ModelConfigurationResult
			for clientID, clientSelections := range selectionsByClient {
				client, err := reg.Client(clientID)
				if err != nil {
					return err
				}
				res, err := client.ConfigureModels(cmd.Context(), root, catalog.Agents(), clientSelections)
				if err != nil {
					return err
				}
				finalResult.Changes = append(finalResult.Changes, res.Changes...)
				finalResult.Sync = res.Sync
			}
			return printJSON(cmd, finalResult)
		},
	}
	addRootFlag(configure, &root)
	models.AddCommand(configure)
	return models
}

func newProjectionCommand(operation, version string, reg *clients.Registry) *cobra.Command {
	parent := &cobra.Command{Use: operation, Short: projectionDescription(operation), Args: cobra.NoArgs}
	var root string
	var check, force bool

	for _, client := range reg.AllClients() {
		clientID := client.ID()
		c := client
		clientCmd := &cobra.Command{
			Use:   clientID,
			Short: projectionDescription(operation) + " for " + c.Info().Name,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				var result domain.Result
				var err error
				switch operation {
				case "init":
					result, err = c.Init(cmd.Context(), root, catalog.Agents())
				case "sync":
					result, err = c.Sync(cmd.Context(), root, catalog.Agents(), domain.SyncOptions{Check: check, Force: force})
				case "uninstall":
					result, err = c.Uninstall(cmd.Context(), root, catalog.Agents(), force)
				}
				if err != nil {
					return err
				}
				return printJSON(cmd, result)
			},
		}
		addRootFlag(clientCmd, &root)
		if operation == "sync" {
			clientCmd.Flags().BoolVar(&check, "check", false, "Check whether the projection is current without writing")
		}
		if operation == "sync" || operation == "uninstall" {
			clientCmd.Flags().BoolVar(&force, "force", false, "Replace an unowned target path when synchronizing")
		}
		parent.AddCommand(clientCmd)
	}
	return parent
}

func projectionDescription(operation string) string {
	switch operation {
	case "init":
		return "Initialize a managed client projection"
	case "sync":
		return "Synchronize a managed client projection"
	default:
		return "Uninstall a managed client projection"
	}
}

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use: "version", Short: "Print the karl-ai version", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(version)
			return nil
		},
	}
}

func addRootFlag(command *cobra.Command, root *string) {
	command.Flags().StringVar(root, "root", ".", "Project root")
}

func printJSON(command *cobra.Command, value any) error {
	encoder := json.NewEncoder(command.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
