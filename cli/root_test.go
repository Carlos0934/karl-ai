package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	opencodeadapter "github.com/carlos0934/karl-ai/adapters/opencode"
	modeltui "github.com/carlos0934/karl-ai/tui"
)

type configureDiscovery struct {
	calls        []string
	providers    []opencodeadapter.Provider
	models       []opencodeadapter.Model
	providersErr error
	modelsErr    error
}

func (discovery *configureDiscovery) Clients(context.Context, string) ([]opencodeadapter.ClientInfo, error) {
	discovery.calls = append(discovery.calls, "clients")
	return []opencodeadapter.ClientInfo{{ID: opencodeadapter.Client, Name: "OpenCode"}}, nil
}

func (discovery *configureDiscovery) Providers(_ context.Context, _, client string) ([]opencodeadapter.Provider, error) {
	discovery.calls = append(discovery.calls, "providers:"+client)
	if discovery.providersErr != nil {
		return nil, discovery.providersErr
	}
	if discovery.providers != nil {
		return discovery.providers, nil
	}
	return []opencodeadapter.Provider{{ID: "provider"}}, nil
}

func (discovery *configureDiscovery) Models(_ context.Context, _, client, provider string) ([]opencodeadapter.Model, error) {
	discovery.calls = append(discovery.calls, "models:"+client+":"+provider)
	if discovery.modelsErr != nil {
		return nil, discovery.modelsErr
	}
	if discovery.models != nil {
		return discovery.models, nil
	}
	return []opencodeadapter.Model{{ID: "provider/model", Variants: []string{"fast"}}}, nil
}

type configureInput struct {
	lines []string
}

func (input *configureInput) Read(buffer []byte) (int, error) {
	if len(input.lines) == 0 {
		return 0, io.EOF
	}
	line := input.lines[0]
	input.lines = input.lines[1:]
	return copy(buffer, line+"\n"), nil
}

func executeCLI(version string, input io.Reader, args ...string) (string, string, int, error) {
	var stdout, stderr bytes.Buffer
	command := New(version, &stdout, &stderr)
	if input != nil {
		command.SetIn(input)
	}
	command.SetArgs(args)
	err := command.Execute()
	if err != nil {
		return stdout.String(), stderr.String(), 1, err
	}
	return stdout.String(), stderr.String(), 0, nil
}

func assertCommandJSON(t *testing.T, output string, want any) {
	t.Helper()
	content, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if output != string(content) {
		t.Fatalf("command output = %q, want %q", output, content)
	}
}

func snapshotProjectFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func assertOnlyProjectFilesChanged(t *testing.T, before, after map[string][]byte, allowed map[string]bool) {
	t.Helper()
	for path, beforeContent := range before {
		afterContent, exists := after[path]
		if !exists {
			t.Fatalf("project file was removed: %s", path)
		}
		if !allowed[path] && !bytes.Equal(beforeContent, afterContent) {
			t.Fatalf("unrelated project file changed: %s", path)
		}
	}
	for path := range after {
		if _, exists := before[path]; !exists && !allowed[path] {
			t.Fatalf("unrelated project file was added: %s", path)
		}
	}
}

func baselineInitPaths() []string {
	paths := append([]string{".karl-ai/config.json", ".karl-ai/manifest.json", ".opencode/opencode.json"}, baselineProjectionPaths...)
	sort.Strings(paths)
	return paths
}

func baselineUninstallPaths() []string {
	paths := baselineInitPaths()
	return append([]string(nil), paths[1:]...)
}

var baselineProjectionPaths = []string{
	".opencode/agents/karl-archiver.md",
	".opencode/agents/karl-foundation.md",
	".opencode/agents/karl-implementer.md",
	".opencode/agents/karl-orchestrator.md",
	".opencode/agents/karl-planner.md",
	".opencode/agents/karl-reviewer.md",
	".opencode/agents/karl-searcher.md",
	".opencode/commands/karl-change-archive.md",
	".opencode/commands/karl-change-implement.md",
	".opencode/commands/karl-change-new.md",
	".opencode/commands/karl-change-review.md",
	".opencode/commands/karl-foundation.md",
	".opencode/skills/karl-change-lifecycle/SKILL.md",
	".opencode/skills/karl-change-lifecycle/references/foundation-integration.md",
	".opencode/skills/karl-change-lifecycle/references/questioning-guide.md",
	".opencode/skills/karl-change-lifecycle/references/work-units.md",
	".opencode/skills/karl-change-lifecycle/references/workflow.md",
	".opencode/skills/karl-change-lifecycle/templates/CHANGE.template.md",
	".opencode/skills/karl-change-lifecycle/templates/PLAN.template.md",
	".opencode/skills/karl-change-lifecycle/templates/RESEARCH.template.md",
	".opencode/skills/karl-change-lifecycle/templates/REVIEW.template.md",
	".opencode/skills/karl-change-lifecycle/templates/TASKS.template.md",
	".opencode/skills/karl-project-foundation/SKILL.md",
	".opencode/skills/karl-project-foundation/references/complexity-levels.md",
	".opencode/skills/karl-project-foundation/references/documentation-model.md",
	".opencode/skills/karl-project-foundation/references/question-matrix.md",
	".opencode/skills/karl-project-foundation/templates/CONTEXT.template.md",
	".opencode/skills/karl-project-foundation/templates/DESIGN.template.md",
	".opencode/skills/karl-project-foundation/templates/JOURNEY.template.md",
}

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	command := New("v0.1.0", &stdout, &bytes.Buffer{})
	command.SetArgs([]string{"version"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := stdout.String(); got != "v0.1.0\n" {
		t.Fatalf("version output = %q", got)
	}
}

func TestOpenCodeCommandSurface(t *testing.T) {
	command := New("test-version", &bytes.Buffer{}, &bytes.Buffer{})
	for _, test := range []struct {
		path  []string
		flags []string
	}{
		{[]string{"init", "opencode"}, []string{"root"}},
		{[]string{"sync", "opencode"}, []string{"root", "check", "force"}},
		{[]string{"uninstall", "opencode"}, []string{"root", "force"}},
	} {
		current := command
		for _, name := range test.path {
			next, _, err := current.Find([]string{name})
			if err != nil || next.Name() != name {
				t.Fatalf("command %v not found: %v", test.path, err)
			}
			current = next
		}
		for _, flag := range test.flags {
			if current.Flags().Lookup(flag) == nil {
				t.Fatalf("command %v missing --%s", test.path, flag)
			}
		}
	}
	initCommand, _, _ := command.Find([]string{"init", "opencode"})
	if initCommand.Flags().Lookup("force") != nil || initCommand.Flags().Lookup("check") != nil {
		t.Fatal("init exposes unsupported flags")
	}
	uninstallCommand, _, _ := command.Find([]string{"uninstall", "opencode"})
	if uninstallCommand.Flags().Lookup("check") != nil {
		t.Fatal("uninstall exposes --check")
	}
}

func TestModelsConfigureCommandSurface(t *testing.T) {
	command := New("test-version", &bytes.Buffer{}, &bytes.Buffer{})
	configure, _, err := command.Find([]string{"models", "configure"})
	if err != nil || configure.Name() != "configure" {
		t.Fatalf("models configure command not found: %v", err)
	}
	if configure.Flags().Lookup("root") == nil {
		t.Fatal("models configure command missing --root")
	}

	var help bytes.Buffer
	command = New("test-version", &help, &bytes.Buffer{})
	command.SetArgs([]string{"models", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "configure") {
		t.Fatalf("models help = %q", help.String())
	}

	help.Reset()
	command = New("test-version", &help, &bytes.Buffer{})
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "karl-ai models configure") {
		t.Fatalf("root help does not show the configure command: %q", help.String())
	}
}

func TestModelsConfigureEndToEndUpdatesOnlySelectedAgent(t *testing.T) {
	root := t.TempDir()
	projector, err := opencodeadapter.New(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ".karl-ai", "config.json")
	var before opencodeadapter.Config
	configBefore, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(configBefore, &before); err != nil {
		t.Fatal(err)
	}
	filesBefore := snapshotProjectFiles(t, root)
	defaults := opencodeadapter.DefaultConfig()

	discovery := &configureDiscovery{}
	var output bytes.Buffer
	command := NewWithDiscovery("test", &output, &bytes.Buffer{}, discovery)
	command.SetIn(&configureInput{lines: []string{"1", "4", "1", "2", "2", "y"}})
	command.SetArgs([]string{"models", "configure", "--root", root})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var configured struct {
		Client   string `json:"client"`
		Agent    string `json:"agent"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Variant  string `json:"variant"`
		Sync     struct {
			Operation string `json:"operation"`
			Changed   bool   `json:"changed"`
		} `json:"sync"`
	}
	if err := json.Unmarshal(output.Bytes(), &configured); err != nil {
		t.Fatal(err)
	}
	if configured.Agent != "karl-planner" || configured.Model != "provider/model" || configured.Variant != "fast" || configured.Sync.Operation != "sync" || !configured.Sync.Changed {
		t.Fatalf("configured result = %#v", configured)
	}
	if got, want := strings.Join(discovery.calls, ","), "clients,providers:opencode,models:opencode:provider"; got != want {
		t.Fatalf("discovery calls = %q, want %q", got, want)
	}
	var after opencodeadapter.Config
	if got, err := os.ReadFile(configPath); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(got, &after); err != nil {
		t.Fatal(err)
	}
	if got := after.OpenCode.Agents["karl-planner"]; got != (opencodeadapter.AgentConfig{Model: "provider/model", Variant: "fast"}) {
		t.Fatalf("planner config = %#v", got)
	}
	for agent, expected := range before.OpenCode.Agents {
		if agent != "karl-planner" && after.OpenCode.Agents[agent] != expected {
			t.Fatalf("config for %s changed from %#v to %#v", agent, expected, after.OpenCode.Agents[agent])
		}
	}
	if !reflect.DeepEqual(defaults, opencodeadapter.DefaultConfig()) {
		t.Fatal("default config changed")
	}
	rendered, err := os.ReadFile(filepath.Join(root, ".opencode", "agents", "karl-planner.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rendered), `model: "provider/model"`) || !strings.Contains(string(rendered), `variant: "fast"`) {
		t.Fatalf("planner projection was not synchronized:\n%s", rendered)
	}
	assertOnlyProjectFilesChanged(t, filesBefore, snapshotProjectFiles(t, root), map[string]bool{
		".karl-ai/config.json":             true,
		".karl-ai/manifest.json":           true,
		".opencode/agents/karl-planner.md": true,
	})
}

func TestNoninteractiveCommandOutputAndExitCodeBaseline(t *testing.T) {
	versionOutput, _, versionExit, versionErr := executeCLI("baseline", nil, "version")
	if versionErr != nil || versionExit != 0 || versionOutput != "baseline\n" {
		t.Fatalf("version result output=%q exit=%d error=%v", versionOutput, versionExit, versionErr)
	}

	root := t.TempDir()
	initOutput, _, initExit, initErr := executeCLI("baseline", nil, "init", "opencode", "--root", root)
	if initErr != nil || initExit != 0 {
		t.Fatalf("init exit=%d error=%v", initExit, initErr)
	}
	assertCommandJSON(t, initOutput, opencodeadapter.Result{
		Operation: "init",
		Client:    opencodeadapter.Client,
		Changed:   true,
		Paths:     baselineInitPaths(),
	})

	syncOutput, _, syncExit, syncErr := executeCLI("baseline", nil, "sync", "opencode", "--root", root)
	if syncErr != nil || syncExit != 0 {
		t.Fatalf("sync exit=%d error=%v", syncExit, syncErr)
	}
	assertCommandJSON(t, syncOutput, opencodeadapter.Result{Operation: "sync", Client: opencodeadapter.Client, Paths: []string{}})

	checkOutput, _, checkExit, checkErr := executeCLI("baseline", nil, "sync", "opencode", "--root", root, "--check")
	if checkErr != nil || checkExit != 0 {
		t.Fatalf("sync --check exit=%d error=%v", checkExit, checkErr)
	}
	assertCommandJSON(t, checkOutput, opencodeadapter.Result{Operation: "check", Client: opencodeadapter.Client, Paths: []string{}})

	uninstallOutput, _, uninstallExit, uninstallErr := executeCLI("baseline", nil, "uninstall", "opencode", "--root", root)
	if uninstallErr != nil || uninstallExit != 0 {
		t.Fatalf("uninstall exit=%d error=%v", uninstallExit, uninstallErr)
	}
	assertCommandJSON(t, uninstallOutput, opencodeadapter.Result{
		Operation: "uninstall",
		Client:    opencodeadapter.Client,
		Changed:   true,
		Paths:     baselineUninstallPaths(),
	})

	changeOutput, _, changeExit, changeErr := executeCLI("baseline", nil, "change", "list", "--root", t.TempDir())
	if changeErr != nil || changeExit != 0 {
		t.Fatalf("change list exit=%d error=%v", changeExit, changeErr)
	}
	if want := "{\n  \"active\": [],\n  \"archived\": []\n}\n"; changeOutput != want {
		t.Fatalf("change list output = %q, want %q", changeOutput, want)
	}

	checkRoot := t.TempDir()
	if _, _, exit, err := executeCLI("baseline", nil, "init", "opencode", "--root", checkRoot); err != nil || exit != 0 {
		t.Fatalf("check fixture init exit=%d error=%v", exit, err)
	}
	if err := os.Remove(filepath.Join(checkRoot, ".opencode", "commands", "karl-change-review.md")); err != nil {
		t.Fatal(err)
	}
	failedOutput, _, failedExit, failedErr := executeCLI("baseline", nil, "sync", "opencode", "--root", checkRoot, "--check")
	if !errors.Is(failedErr, opencodeadapter.ErrOutOfSync) || failedExit != 1 || failedOutput != "" {
		t.Fatalf("failing sync --check output=%q exit=%d error=%v", failedOutput, failedExit, failedErr)
	}
}

func TestModelsConfigureCtrlCCancelsWithoutWritingConfig(t *testing.T) {
	root := t.TempDir()
	configDirectory := filepath.Join(root, ".karl-ai")
	if err := os.MkdirAll(configDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDirectory, "config.json")
	original := []byte("unchanged\n")
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	command := NewWithDiscovery("test", &bytes.Buffer{}, &bytes.Buffer{}, &configureDiscovery{})
	command.SetIn(strings.NewReader("\x03"))
	command.SetArgs([]string{"models", "configure", "--root", root})
	if err := command.Execute(); !errors.Is(err, modeltui.ErrCancelled) {
		t.Fatalf("Execute() error = %v, want model cancellation", err)
	}
	if got, err := os.ReadFile(configPath); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(got, original) {
		t.Fatalf("config changed after cancellation: %q", got)
	}
}

func TestModelsConfigureReportsSyncFailureAfterSavingSelection(t *testing.T) {
	root := t.TempDir()
	projector, err := opencodeadapter.New(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	plannerPath := filepath.Join(root, ".opencode", "agents", "karl-planner.md")
	drift := []byte("user drift\n")
	if err := os.WriteFile(plannerPath, drift, 0o644); err != nil {
		t.Fatal(err)
	}

	command := NewWithDiscovery("test", &bytes.Buffer{}, &bytes.Buffer{}, &configureDiscovery{})
	command.SetIn(&configureInput{lines: []string{"1", "4", "1", "2", "2", "y"}})
	command.SetArgs([]string{"models", "configure", "--root", root})
	err = command.Execute()
	if !errors.Is(err, opencodeadapter.ErrDrift) || !strings.Contains(err.Error(), "model configuration was saved, but OpenCode sync failed") {
		t.Fatalf("Execute() error = %v", err)
	}
	current, err := os.ReadFile(plannerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, drift) {
		t.Fatal("models configure overwrote a drifted agent file")
	}
	var config opencodeadapter.Config
	content, err := os.ReadFile(filepath.Join(root, ".karl-ai", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	if got := config.OpenCode.Agents["karl-planner"]; got != (opencodeadapter.AgentConfig{Model: "provider/model", Variant: "fast"}) {
		t.Fatalf("saved config = %#v", got)
	}
}

func TestModelsConfigureManualFallbackFailuresDoNotWriteConfig(t *testing.T) {
	for _, test := range []struct {
		name      string
		discovery *configureDiscovery
		input     []string
	}{
		{
			name:      "missing OpenCode binary",
			discovery: &configureDiscovery{providersErr: opencodeadapter.ErrClientNotFound},
			input:     []string{"1", "1", "not-a-reference"},
		},
		{
			name:      "provider discovery timeout",
			discovery: &configureDiscovery{providersErr: context.DeadlineExceeded},
			input:     []string{"1", "1", "not-a-reference"},
		},
		{
			name:      "empty model discovery",
			discovery: &configureDiscovery{models: []opencodeadapter.Model{}},
			input:     []string{"1", "1", "1", "not-a-reference"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			projector, err := opencodeadapter.New(root, "test")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := projector.Init(); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(root, ".karl-ai", "config.json")
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			command := NewWithDiscovery("test", &bytes.Buffer{}, &stderr, test.discovery)
			command.SetIn(&configureInput{lines: test.input})
			command.SetArgs([]string{"models", "configure", "--root", root})
			err = command.Execute()
			if err == nil || !strings.Contains(err.Error(), "model reference must use provider/model") {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(stderr.String(), "Model discovery is unavailable") || !strings.Contains(stderr.String(), "Enter a provider/model reference manually") {
				t.Fatalf("manual fallback output = %q", stderr.String())
			}
			after, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("manual fallback failure changed config")
			}
		})
	}
}

func TestModelsConfigureKeepsStaleSavedModelWithoutRewritingConfig(t *testing.T) {
	root := t.TempDir()
	projector, err := opencodeadapter.New(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := projector.ConfigureModel(opencodeadapter.ModelSelection{Agent: "karl-planner", Model: "old-provider/old-model", Variant: "legacy"}); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ".karl-ai", "config.json")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	command := NewWithDiscovery("test", &stdout, &stderr, &configureDiscovery{models: []opencodeadapter.Model{{ID: "provider/replacement"}}})
	command.SetIn(&configureInput{lines: []string{"1", "4", "1", "1", "2", "y"}})
	command.SetArgs([]string{"models", "configure", "--root", root})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "old-provider/old-model (configured, stale)") {
		t.Fatalf("stale model was not marked: %q", stderr.String())
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("keeping stale selection rewrote config")
	}
	var result opencodeadapter.ModelConfigurationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Model != "old-provider/old-model" || result.Variant != "legacy" || result.Sync.Operation != "sync" || result.Sync.Changed {
		t.Fatalf("unchanged stale result = %#v", result)
	}
}

func TestOpenCodeCLIUsesBinaryVersionAndLifecycle(t *testing.T) {
	root := t.TempDir()
	var output bytes.Buffer
	command := New("v9.8.7", &output, &bytes.Buffer{})
	command.SetArgs([]string{"init", "opencode", "--root", root})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Operation string `json:"operation"`
		Changed   bool   `json:"changed"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Operation != "init" || !result.Changed {
		t.Fatalf("init result = %#v", result)
	}
	manifest, err := os.ReadFile(filepath.Join(root, ".karl-ai", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"karl_version": "v9.8.7"`) {
		t.Fatalf("manifest does not contain binary version: %s", manifest)
	}

	check := New("v9.8.7", &bytes.Buffer{}, &bytes.Buffer{})
	check.SetArgs([]string{"sync", "opencode", "--root", root, "--check"})
	if err := check.Execute(); err != nil {
		t.Fatal(err)
	}
	uninstall := New("v9.8.7", &bytes.Buffer{}, &bytes.Buffer{})
	uninstall.SetArgs([]string{"uninstall", "opencode", "--root", root})
	if err := uninstall.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestChangeHelpPrintsUsageWithoutError(t *testing.T) {
	var stdout bytes.Buffer
	command := New("test", &stdout, &bytes.Buffer{})
	command.SetArgs([]string{"change", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.HasPrefix(output, "Karl Change Lifecycle CLI") || !strings.Contains(output, "archive <name>") || !strings.Contains(output, "never creates a commit") {
		t.Fatalf("help output = %q", output)
	}
}

func TestChangeHelpWorksAfterSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	command := New("test", &stdout, &bytes.Buffer{})
	command.SetArgs([]string{"change", "status", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if output := stdout.String(); !strings.HasPrefix(output, "Karl Change Lifecycle CLI") || !strings.Contains(output, "status <name>") {
		t.Fatalf("help output = %q", output)
	}
}

func TestChangeUnknownOptionsAreRejected(t *testing.T) {
	command := New("test", &bytes.Buffer{}, &bytes.Buffer{})
	command.SetArgs([]string{"change", "--bogus"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "Unknown option: --bogus") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestChangeCommandsWireRootLevelAndJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CONTEXT.md", "DESIGN.md"} {
		if err := os.WriteFile(filepath.Join(root, "docs", name), []byte("# Foundation\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var createOutput bytes.Buffer
	create := New("test", &createOutput, &bytes.Buffer{})
	create.SetArgs([]string{"change", "new", "add-refunds", "--level", "L2", "--root", root})
	if err := create.Execute(); err != nil {
		t.Fatal(err)
	}
	var created struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(createOutput.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "add-refunds" || created.State != "draft" {
		t.Fatalf("created output = %#v", created)
	}

	var validateOutput bytes.Buffer
	validate := New("test", &validateOutput, &bytes.Buffer{})
	validate.SetArgs([]string{"change", "validate", "add-refunds", "plan", "--root", root, "--json"})
	if err := validate.Execute(); !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("Execute() error = %v", err)
	}
	var gate struct {
		OK   bool     `json:"ok"`
		Gate string   `json:"gate"`
		Errs []string `json:"errors"`
	}
	if err := json.Unmarshal(validateOutput.Bytes(), &gate); err != nil {
		t.Fatal(err)
	}
	if gate.OK || gate.Gate != "plan" || len(gate.Errs) == 0 {
		t.Fatalf("gate output = %#v", gate)
	}
}
