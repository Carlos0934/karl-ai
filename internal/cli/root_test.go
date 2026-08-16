package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/internal/cli"
)

func TestCLI_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := cli.New("v1.2.3", &stdout, &stderr)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %q", output)
	}
}

func TestCLI_InitOpenCode(t *testing.T) {
	tempDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd := cli.New("dev", &stdout, &stderr)
	cmd.SetArgs([]string{"init", "opencode", "--root", tempDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init opencode command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"operation": "init"`) {
		t.Errorf("output missing init operation: %s", output)
	}
	if !strings.Contains(output, `"changed": true`) {
		t.Errorf("output missing changed true: %s", output)
	}
}

func TestCLI_SyncOpenCode_Check(t *testing.T) {
	tempDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	// First init
	initCmd := cli.New("dev", &stdout, &stderr)
	initCmd.SetArgs([]string{"init", "opencode", "--root", tempDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Then sync with --check
	stdout.Reset()
	stderr.Reset()
	syncCmd := cli.New("dev", &stdout, &stderr)
	syncCmd.SetArgs([]string{"sync", "opencode", "--root", tempDir, "--check"})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("sync check failed on clean project: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"operation": "check"`) {
		t.Errorf("output missing check operation: %s", output)
	}
}
