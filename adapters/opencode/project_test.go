package opencode

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRenderCompleteDeterministicProjection(t *testing.T) {
	first, err := Render(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("Render is not deterministic")
	}

	agents, commands, skillDocuments, resources := 0, 0, 0, 0
	paths := make([]string, 0, len(first))
	for _, file := range first {
		paths = append(paths, file.Path)
		content := string(file.Content)
		if strings.Contains(content, "change.mjs") || strings.Contains(content, "node .opencode") {
			t.Fatalf("legacy Node path in %s", file.Path)
		}
		switch {
		case strings.HasPrefix(file.Path, ".opencode/agents/"):
			agents++
			assertFrontmatter(t, file.Path, content)
		case strings.HasPrefix(file.Path, ".opencode/commands/"):
			commands++
			assertFrontmatter(t, file.Path, content)
			if strings.Contains(strings.SplitN(content, "---", 3)[1], "agent:") {
				t.Fatalf("command %s has agent frontmatter", file.Path)
			}
			if !strings.Contains(content, "$ARGUMENTS") {
				t.Fatalf("command %s does not receive arguments", file.Path)
			}
		case strings.HasSuffix(file.Path, "/SKILL.md"):
			skillDocuments++
			assertFrontmatter(t, file.Path, content)
		case strings.HasPrefix(file.Path, ".opencode/skills/"):
			resources++
		}
	}
	if agents != 7 || commands != 5 || skillDocuments != 2 || resources != 15 || len(first) != 29 {
		t.Fatalf("counts agents=%d commands=%d skills=%d resources=%d total=%d", agents, commands, skillDocuments, resources, len(first))
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatal("paths are not sorted")
	}
	expected := []string{
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
	sort.Strings(expected)
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("projected paths differ\ngot:  %v\nwant: %v", paths, expected)
	}
}

func TestRenderModelOverrideAndAgentModes(t *testing.T) {
	config := DefaultConfig()
	config.OpenCode.Agents["karl-planner"] = AgentConfig{Model: "example/custom", Variant: "fast"}
	files, err := Render(config)
	if err != nil {
		t.Fatal(err)
	}
	planner := rendered(files, ".opencode/agents/karl-planner.md")
	if !strings.Contains(planner, `model: "example/custom"`) || !strings.Contains(planner, `variant: "fast"`) {
		t.Fatalf("planner override not rendered:\n%s", planner)
	}
	if !strings.Contains(planner, "mode: subagent") || !strings.Contains(planner, "task: deny") || !strings.Contains(planner, "question: deny") {
		t.Fatal("specialist mode or permissions missing")
	}
	orchestrator := rendered(files, ".opencode/agents/karl-orchestrator.md")
	if !strings.Contains(orchestrator, "mode: primary") || !strings.Contains(orchestrator, "karl-planner: allow") {
		t.Fatal("orchestrator mode or delegation permissions missing")
	}
}

func TestInitSyncCheckAndManifest(t *testing.T) {
	root := t.TempDir()
	projector, err := New(root, "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	result, err := projector.Init()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("init did not report changes")
	}
	result, err = projector.Sync(SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed || len(result.Paths) != 0 {
		t.Fatalf("idempotent sync result = %#v", result)
	}
	if _, err = projector.Sync(SyncOptions{Check: true}); err != nil {
		t.Fatal(err)
	}

	var manifest Manifest
	readJSON(t, filepath.Join(root, filepath.FromSlash(manifestPath)), &manifest)
	if manifest.KarlVersion != "v1.2.3" || manifest.Client != Client || len(manifest.Files) != 29 {
		t.Fatalf("manifest = %#v", manifest)
	}
	for i, file := range manifest.Files {
		if filepath.IsAbs(file.Path) || filepath.ToSlash(file.Path) != file.Path || len(file.SHA256) != 64 {
			t.Fatalf("invalid manifest file %#v", file)
		}
		if i > 0 && manifest.Files[i-1].Path >= file.Path {
			t.Fatal("manifest is not sorted")
		}
	}
	var config Config
	readJSON(t, filepath.Join(root, filepath.FromSlash(configPath)), &config)
	if config.Version != ConfigVersion || len(config.OpenCode.Agents) != 7 {
		t.Fatalf("config = %#v", config)
	}
}

func TestOpenCodeConfigMergeAndUninstallPreserveUserState(t *testing.T) {
	root := t.TempDir()
	existing := []byte("{\n  \"$schema\": \"custom-schema\",\n  \"default_agent\": \"user-agent\",\n  \"theme\": \"nord\",\n  \"permission\": {\"bash\": \"ask\", \"task\": {\"other-*\": \"deny\", \"karl-*\": \"ask\"}}\n}\n")
	writeFile(t, root, openCodePath, existing)
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	var installed map[string]any
	readJSON(t, filepath.Join(root, filepath.FromSlash(openCodePath)), &installed)
	if installed["theme"] != "nord" || installed["$schema"] != "custom-schema" || installed["default_agent"] != defaultAgent {
		t.Fatalf("merged config = %#v", installed)
	}
	if _, err := projector.Uninstall(false); err != nil {
		t.Fatal(err)
	}
	var restored map[string]any
	readJSON(t, filepath.Join(root, filepath.FromSlash(openCodePath)), &restored)
	if restored["theme"] != "nord" || restored["$schema"] != "custom-schema" || restored["default_agent"] != "user-agent" {
		t.Fatalf("restored config = %#v", restored)
	}
	permission := restored["permission"].(map[string]any)
	task := permission["task"].(map[string]any)
	if permission["bash"] != "ask" || task["other-*"] != "deny" || task["karl-*"] != "ask" {
		t.Fatalf("restored permissions = %#v", permission)
	}
}

func TestUninstallPreservesSharedValuesChangedAfterInstall(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, filepath.FromSlash(openCodePath))
	var document map[string]any
	readJSON(t, configPath, &document)
	document["default_agent"] = "my-agent"
	document["theme"] = "custom"
	permission := document["permission"].(map[string]any)
	task := permission["task"].(map[string]any)
	task["karl-*"] = "deny"
	content, err := marshalJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := projector.Uninstall(false); err != nil {
		t.Fatal(err)
	}
	var preserved map[string]any
	readJSON(t, configPath, &preserved)
	if preserved["default_agent"] != "my-agent" || preserved["theme"] != "custom" {
		t.Fatalf("user values were not preserved: %#v", preserved)
	}
	preservedTask := preserved["permission"].(map[string]any)["task"].(map[string]any)
	if preservedTask["karl-*"] != "deny" {
		t.Fatalf("user task permission was not preserved: %#v", preservedTask)
	}
}

func TestSyncDriftRejectCheckNoWriteAndForce(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".opencode", "agents", "karl-planner.md")
	drift := []byte("user drift\n")
	if err := os.WriteFile(target, drift, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Sync(SyncOptions{}); !errors.Is(err, ErrDrift) {
		t.Fatalf("sync error = %v", err)
	}
	if _, err := projector.Sync(SyncOptions{Check: true}); !errors.Is(err, ErrDrift) {
		t.Fatalf("check error = %v", err)
	}
	current, _ := os.ReadFile(target)
	if !bytes.Equal(current, drift) {
		t.Fatal("check changed drifted file")
	}
	if _, err := projector.Sync(SyncOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	current, _ = os.ReadFile(target)
	if bytes.Equal(current, drift) {
		t.Fatal("force did not replace drift")
	}
}

func TestInitRejectsUnownedDesiredPathUnlessSyncIsForced(t *testing.T) {
	root := t.TempDir()
	targetPath := ".opencode/agents/karl-planner.md"
	target := filepath.Join(root, filepath.FromSlash(targetPath))
	conflict := []byte("unmanaged planner\n")
	writeFile(t, root, targetPath, conflict)
	projector, _ := New(root, "test")

	if _, err := projector.Init(); !errors.Is(err, ErrDrift) {
		t.Fatalf("init error = %v", err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, conflict) {
		t.Fatal("init overwrote unowned conflict")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(manifestPath))); !os.IsNotExist(err) {
		t.Fatal("init wrote a manifest after conflict")
	}

	if _, err := projector.Sync(SyncOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	desired, err := Render(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	current, err = os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, []byte(rendered(desired, targetPath))) {
		t.Fatal("forced sync did not replace and adopt conflict")
	}
}

func TestSyncRejectsDesiredPathNotRecordedInManifest(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	targetPath := ".opencode/agents/karl-planner.md"
	target := filepath.Join(root, filepath.FromSlash(targetPath))
	manifestPath := filepath.Join(root, filepath.FromSlash(manifestPath))
	var manifest Manifest
	readJSON(t, manifestPath, &manifest)
	files := manifest.Files[:0]
	for _, file := range manifest.Files {
		if file.Path != targetPath {
			files = append(files, file)
		}
	}
	manifest.Files = files
	content, err := marshalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	conflict := []byte("unrecorded conflict\n")
	if err := os.WriteFile(target, conflict, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := projector.Sync(SyncOptions{}); !errors.Is(err, ErrDrift) {
		t.Fatalf("sync error = %v", err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, conflict) {
		t.Fatal("sync overwrote unrecorded conflict")
	}
	if _, err := projector.Sync(SyncOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	var adopted Manifest
	readJSON(t, manifestPath, &adopted)
	found := false
	for _, file := range adopted.Files {
		found = found || file.Path == targetPath
	}
	if !found {
		t.Fatal("forced sync did not adopt path into manifest")
	}
}

func TestSyncDetectsSharedDriftAndForceRestoresKarlValues(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(openCodePath))
	manifest := filepath.Join(root, filepath.FromSlash(manifestPath))
	var document map[string]any
	readJSON(t, target, &document)
	document["default_agent"] = "user-agent"
	permission := document["permission"].(map[string]any)
	task := permission["task"].(map[string]any)
	task["karl-*"] = "deny"
	drift, err := marshalJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, drift, 0o644); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := projector.Sync(SyncOptions{}); !errors.Is(err, ErrDrift) {
		t.Fatalf("sync error = %v", err)
	}
	if _, err := projector.Sync(SyncOptions{Check: true}); !errors.Is(err, ErrDrift) {
		t.Fatalf("check error = %v", err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, drift) {
		t.Fatal("normal sync or check overwrote shared drift")
	}
	manifestAfter, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Fatal("shared drift changed the manifest")
	}

	if _, err := projector.Sync(SyncOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	var restored map[string]any
	readJSON(t, target, &restored)
	if restored["default_agent"] != defaultAgent {
		t.Fatalf("default agent = %#v", restored["default_agent"])
	}
	restoredTask := restored["permission"].(map[string]any)["task"].(map[string]any)
	if restoredTask["karl-*"] != "allow" {
		t.Fatalf("task permission = %#v", restoredTask["karl-*"])
	}
}

func TestCheckMissingFileDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".opencode", "commands", "karl-change-review.md")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	manifestBefore, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if _, err := projector.Sync(SyncOptions{Check: true}); !errors.Is(err, ErrOutOfSync) {
		t.Fatalf("check error = %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("check recreated missing file")
	}
	manifestAfter, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Fatal("check changed manifest")
	}
}

func TestUninstallDriftAndUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	projector, _ := New(root, "test")
	if _, err := projector.Init(); err != nil {
		t.Fatal(err)
	}
	owned := filepath.Join(root, ".opencode", "agents", "karl-reviewer.md")
	unrelated := filepath.Join(root, ".opencode", "agents", "mine.md")
	if err := os.WriteFile(owned, []byte("drift"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projector.Uninstall(false); !errors.Is(err, ErrDrift) {
		t.Fatalf("uninstall error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode", "agents", "karl-planner.md")); err != nil {
		t.Fatal("uninstall mutated before drift rejection")
	}
	if _, err := projector.Uninstall(true); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(unrelated); err != nil || string(content) != "mine" {
		t.Fatalf("unrelated file changed: %q, %v", content, err)
	}
	if _, err := os.Stat(owned); !os.IsNotExist(err) {
		t.Fatal("forced uninstall retained owned drift")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(configPath))); err != nil {
		t.Fatal("uninstall removed consumer config")
	}
}

func TestUninstallRemovesKarlCreatedEmptyOpenCodeConfig(t *testing.T) {
	for _, test := range []struct {
		name      string
		unrelated bool
	}{
		{name: "empty projection root"},
		{name: "unrelated content remains", unrelated: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			projector, _ := New(root, "test")
			if _, err := projector.Init(); err != nil {
				t.Fatal(err)
			}
			unrelated := filepath.Join(root, ".opencode", "keep.txt")
			if test.unrelated {
				if err := os.WriteFile(unrelated, []byte("keep"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if _, err := projector.Uninstall(false); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(openCodePath))); !os.IsNotExist(err) {
				t.Fatal("Karl-created empty opencode.json remains")
			}
			for _, directory := range []string{"agents", "commands", "skills"} {
				if _, err := os.Stat(filepath.Join(root, ".opencode", directory)); !os.IsNotExist(err) {
					t.Fatalf("generated directory %s remains", directory)
				}
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(configPath))); err != nil {
				t.Fatal("uninstall removed consumer config")
			}
			if test.unrelated {
				content, err := os.ReadFile(unrelated)
				if err != nil || string(content) != "keep" {
					t.Fatalf("unrelated content changed: %q, %v", content, err)
				}
			} else if _, err := os.Stat(filepath.Join(root, ".opencode")); !os.IsNotExist(err) {
				t.Fatal("empty Karl-created .opencode directory remains")
			}
		})
	}
}

func assertFrontmatter(t *testing.T, path, content string) {
	t.Helper()
	if !strings.HasPrefix(content, "---\n") {
		t.Fatalf("%s has no frontmatter", path)
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		t.Fatalf("%s has invalid frontmatter delimiters", path)
	}
	for _, line := range strings.Split(strings.TrimSpace(parts[1]), "\n") {
		if strings.TrimSpace(line) == "" || (!strings.HasPrefix(line, " ") && !strings.Contains(line, ":")) {
			t.Fatalf("%s has invalid frontmatter line %q", path, line)
		}
	}
}

func rendered(files []File, path string) string {
	for _, file := range files {
		if file.Path == path {
			return string(file.Content)
		}
	}
	return ""
}

func readJSON(t *testing.T, path string, destination any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, destination); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, root, relative string, content []byte) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
