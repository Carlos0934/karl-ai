package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
