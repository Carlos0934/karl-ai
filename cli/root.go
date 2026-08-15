package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	filesystemadapter "github.com/carlos0934/karl-ai/adapters/filesystem"
	gitadapter "github.com/carlos0934/karl-ai/adapters/git"
	opencodeadapter "github.com/carlos0934/karl-ai/adapters/opencode"
	"github.com/carlos0934/karl-ai/core/lifecycle"
	"github.com/spf13/cobra"
)

var ErrValidationFailed = errors.New("validation failed")

// New constructs a complete CLI without relying on package-level state.
func New(version string, stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "karl-ai",
		Short:         "Manage Karl workflows and agent-client projections",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(
		newVersionCommand(version),
		newProjectionCommand("init", version),
		newProjectionCommand("sync", version),
		newProjectionCommand("uninstall", version),
		newChangeCommand(),
	)
	return root
}

func newProjectionCommand(operation, version string) *cobra.Command {
	parent := &cobra.Command{
		Use:   operation,
		Short: projectionDescription(operation),
		Args:  cobra.NoArgs,
	}
	var root string
	var check, force bool
	openCode := &cobra.Command{
		Use:   "opencode",
		Short: projectionDescription(operation) + " for OpenCode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projector, err := opencodeadapter.New(root, version)
			if err != nil {
				return err
			}
			var result opencodeadapter.Result
			switch operation {
			case "init":
				result, err = projector.Init()
			case "sync":
				result, err = projector.Sync(opencodeadapter.SyncOptions{Check: check, Force: force})
			case "uninstall":
				result, err = projector.Uninstall(force)
			}
			if err != nil {
				return err
			}
			return printJSON(cmd, result)
		},
	}
	addRootFlag(openCode, &root)
	if operation == "sync" {
		openCode.Flags().BoolVar(&check, "check", false, "Check whether the projection is current without writing")
	}
	if operation == "sync" || operation == "uninstall" {
		openCode.Flags().BoolVar(&force, "force", false, "Replace or remove drifted managed files")
	}
	parent.AddCommand(openCode)
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
		Use:   "version",
		Short: "Print the karl-ai version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(version)
			return nil
		},
	}
}

func newChangeCommand() *cobra.Command {
	change := &cobra.Command{
		Use:   "change",
		Short: "Manage change lifecycle packages",
		Args:  cobra.NoArgs,
	}
	change.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), changeUsage)
	})
	change.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		match := regexp.MustCompile(`unknown flag: (\S+)`).FindStringSubmatch(err.Error())
		if match != nil {
			return fmt.Errorf("Unknown option: %s", match[1])
		}
		return err
	})
	change.AddCommand(
		newChangeNewCommand(),
		newChangeListCommand(),
		newChangeStatusCommand(),
		newChangeValidateCommand(),
		newChangeTransitionCommand(),
		newChangeArchiveCommand(),
	)
	return change
}

func newChangeNewCommand() *cobra.Command {
	var root, level string
	command := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a change package in changes/<name>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			created, err := service.Create(args[0], level, "")
			if err != nil {
				return err
			}
			return printJSON(cmd, created)
		},
	}
	command.Flags().StringVar(&level, "level", "pending", "Assurance level: L1, L2, L3, or L4")
	addRootFlag(command, &root)
	return command
}

func newChangeListCommand() *cobra.Command {
	var root string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List active and archived changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			result, err := service.List()
			if err != nil {
				return err
			}
			_ = jsonOutput
			return printJSON(cmd, result)
		},
	}
	addRootFlag(command, &root)
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print machine-readable JSON")
	return command
}

func newChangeStatusCommand() *cobra.Command {
	var root string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "status <name>",
		Short: "Show the current state and gates of a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			result, err := service.Status(args[0])
			if err != nil {
				return err
			}
			_ = jsonOutput
			return printJSON(cmd, result)
		},
	}
	addRootFlag(command, &root)
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print machine-readable JSON")
	return command
}

func newChangeValidateCommand() *cobra.Command {
	var root string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "validate <name> <plan|implement|review|archive>",
		Short: "Run a lifecycle gate",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			result, err := service.Validate(args[0], args[1])
			if err != nil {
				return err
			}
			_ = jsonOutput
			if err := printJSON(cmd, result); err != nil {
				return err
			}
			if !result.OK {
				return ErrValidationFailed
			}
			return nil
		},
	}
	addRootFlag(command, &root)
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print machine-readable JSON")
	return command
}

func newChangeTransitionCommand() *cobra.Command {
	var root string
	command := &cobra.Command{
		Use:   "transition <name> <state>",
		Short: "Move a change to another lifecycle state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			result, err := service.Transition(args[0], args[1])
			if err != nil {
				return err
			}
			return printJSON(cmd, result)
		},
	}
	addRootFlag(command, &root)
	return command
}

func newChangeArchiveCommand() *cobra.Command {
	var root, date string
	command := &cobra.Command{
		Use:   "archive <name>",
		Short: "Move a validated change to changes/archive without committing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := lifecycleService(root)
			if err != nil {
				return err
			}
			result, err := service.Archive(args[0], date)
			if err != nil {
				return err
			}
			return printJSON(cmd, result)
		},
	}
	addRootFlag(command, &root)
	command.Flags().StringVar(&date, "date", "", "Archive date in YYYY-MM-DD format")
	return command
}

func addRootFlag(command *cobra.Command, root *string) {
	command.Flags().StringVar(root, "root", ".", "Project root")
}

func lifecycleService(root string) (*lifecycle.Service, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return lifecycle.New(filesystemadapter.New(absolute), gitadapter.New(absolute)), nil
}

func printJSON(command *cobra.Command, value any) error {
	encoder := json.NewEncoder(command.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

const changeUsage = `Karl Change Lifecycle CLI

Commands:
  new <name>                       Create a change package in changes/<name>
  list                             List active and archived changes
  status <name>                    Show the current state of a change
  validate <name> <gate>           Run a gate: plan | implement | review | archive
  transition <name> <state>        Move a change: planned | implementing | reviewing | validated
  archive <name>                   Move a validated change to changes/archive/

Options:
  --level L1|L2|L3|L4              Assurance level (with new)
  --root <path>                    Project root (default: current directory)
  --json                           Machine-readable output
  --date YYYY-MM-DD                Archive date (with archive)
  -h, --help                       Show this help

States:
  draft -> planned -> implementing -> reviewing -> validated -> archived
  Backward: implementing -> planned; reviewing -> implementing | planned

Gates:
  plan       Package complete, level selected, valid work units, no blockers
  implement  Plan gate passes, developer authorized, dependencies available
  review     Tasks complete, implementation committed, evidence collected
  archive    Validated, dependencies archived, no conflicts, nothing uncommitted

The archive command moves the package but never creates a commit.`
