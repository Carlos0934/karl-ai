package opencode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	filesystemadapter "github.com/carlos0934/karl-ai/adapters/filesystem"
)

const (
	ManifestVersion = 1
	Client          = "opencode"
	configPath      = ".karl-ai/config.json"
	manifestPath    = ".karl-ai/manifest.json"
	openCodePath    = ".opencode/opencode.json"
	openCodeSchema  = "https://opencode.ai/config.json"
	defaultAgent    = "karl-orchestrator"
)

var (
	ErrDrift     = errors.New("managed OpenCode files have drifted")
	ErrOutOfSync = errors.New("OpenCode projection is out of sync")
	ErrNotFound  = errors.New("OpenCode projection is not installed")
)

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type PreviousValue struct {
	Present bool            `json:"present"`
	Value   json.RawMessage `json:"value,omitempty"`
}

type SharedState struct {
	SchemaAdded        bool          `json:"schema_added,omitempty"`
	FileAdded          bool          `json:"file_added,omitempty"`
	DirectoryAdded     bool          `json:"directory_added,omitempty"`
	DefaultAgent       PreviousValue `json:"default_agent"`
	KarlTaskPermission PreviousValue `json:"karl_task_permission"`
}

type Manifest struct {
	Version     int            `json:"version"`
	KarlVersion string         `json:"karl_version"`
	Client      string         `json:"client"`
	Files       []ManifestFile `json:"files"`
	OpenCode    SharedState    `json:"opencode"`
}

type Result struct {
	Operation string   `json:"operation"`
	Client    string   `json:"client"`
	Changed   bool     `json:"changed"`
	Paths     []string `json:"paths"`
}

// ModelSelection holds the model override selected for one Karl agent.
type ModelSelection struct {
	Agent   string
	Model   string
	Variant string
}

// ModelConfigurationResult describes a saved model override and its sync.
type ModelConfigurationResult struct {
	Agent   string `json:"agent"`
	Model   string `json:"model"`
	Variant string `json:"variant,omitempty"`
	Sync    Result `json:"sync"`
}

type SyncOptions struct {
	Check bool
	Force bool
}

type Projector struct {
	root        string
	version     string
	writeAtomic func(string, []byte, os.FileMode) error
}

func New(root, version string) (*Projector, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Projector{root: absolute, version: version}, nil
}

// ConfigureModel atomically saves one selected agent override, then synchronizes
// the OpenCode projection without forcing drifted files to be replaced.
func (projector *Projector) ConfigureModel(selection ModelSelection) (ModelConfigurationResult, error) {
	if selection.Agent == "" {
		return ModelConfigurationResult{}, fmt.Errorf("selected agent cannot be empty")
	}
	if selection.Model == "" {
		return ModelConfigurationResult{}, fmt.Errorf("selected model cannot be empty")
	}
	config, err := projector.readConfig()
	if err != nil {
		return ModelConfigurationResult{}, err
	}
	config.OpenCode.Agents[selection.Agent] = AgentConfig{Model: selection.Model, Variant: selection.Variant}
	if err := validateConfig(config); err != nil {
		return ModelConfigurationResult{}, err
	}
	if err := projector.writeConfig(config); err != nil {
		return ModelConfigurationResult{}, fmt.Errorf("save model configuration: %w", err)
	}

	result := ModelConfigurationResult{
		Agent:   selection.Agent,
		Model:   selection.Model,
		Variant: selection.Variant,
	}
	result.Sync, err = projector.Sync(SyncOptions{})
	if err != nil {
		return result, fmt.Errorf("model configuration was saved, but OpenCode sync failed: %w", err)
	}
	return result, nil
}

// ModelForAgent returns the saved override for an agent, or its current default
// when the project config has no explicit override for that agent.
func (projector *Projector) ModelForAgent(agent string) (AgentConfig, error) {
	config, err := projector.readConfig()
	if err != nil {
		return AgentConfig{}, err
	}
	if saved, ok := config.OpenCode.Agents[agent]; ok && saved.Model != "" {
		return saved, nil
	}
	defaultConfig, ok := DefaultConfig().OpenCode.Agents[agent]
	if !ok {
		return AgentConfig{}, fmt.Errorf("unknown OpenCode agent %q", agent)
	}
	return defaultConfig, nil
}

func (projector *Projector) Init() (Result, error) {
	changed := []string{}
	if _, err := os.Stat(projector.absolute(configPath)); os.IsNotExist(err) {
		if writeErr := projector.writeConfig(DefaultConfig()); writeErr != nil {
			return Result{}, writeErr
		}
		changed = append(changed, configPath)
	} else if err != nil {
		return Result{}, err
	}
	result, err := projector.Sync(SyncOptions{})
	if err != nil {
		return Result{}, err
	}
	changed = append(changed, result.Paths...)
	changed = uniqueSorted(changed)
	return Result{Operation: "init", Client: Client, Changed: len(changed) > 0, Paths: changed}, nil
}

func (projector *Projector) Sync(options SyncOptions) (Result, error) {
	config, err := projector.readConfig()
	if err != nil {
		return Result{}, err
	}
	files, err := Render(config)
	if err != nil {
		return Result{}, err
	}
	oldManifest, installed, err := projector.readManifest()
	if err != nil {
		return Result{}, err
	}
	sharedDocument, sharedState, sharedChanged, sharedDrift, err := projector.desiredOpenCodeConfig(oldManifest, installed)
	if err != nil {
		return Result{}, err
	}

	desired := make(map[string]File, len(files))
	newManifest := Manifest{Version: ManifestVersion, KarlVersion: projector.version, Client: Client, OpenCode: sharedState}
	for _, file := range files {
		desired[file.Path] = file
		newManifest.Files = append(newManifest.Files, ManifestFile{Path: file.Path, SHA256: hash(file.Content)})
	}
	sort.Slice(newManifest.Files, func(i, j int) bool { return newManifest.Files[i].Path < newManifest.Files[j].Path })

	drifted, changed, err := projector.compareProjection(desired, oldManifest, installed)
	if err != nil {
		return Result{}, err
	}
	drifted = uniqueSorted(append(drifted, sharedDrift...))
	if len(drifted) > 0 && !options.Force {
		return Result{}, fmt.Errorf("%w: %s", ErrDrift, strings.Join(drifted, ", "))
	}
	if sharedChanged {
		changed = append(changed, openCodePath)
	}
	manifestContent, err := marshalJSON(newManifest)
	if err != nil {
		return Result{}, err
	}
	currentManifest, readErr := os.ReadFile(projector.absolute(manifestPath))
	if readErr != nil && !os.IsNotExist(readErr) {
		return Result{}, readErr
	}
	if !bytes.Equal(currentManifest, manifestContent) {
		changed = append(changed, manifestPath)
	}
	changed = uniqueSorted(changed)
	if options.Check {
		if len(changed) > 0 {
			return Result{}, fmt.Errorf("%w: %s", ErrOutOfSync, strings.Join(changed, ", "))
		}
		return Result{Operation: "check", Client: Client, Paths: []string{}}, nil
	}

	for _, path := range stalePaths(oldManifest, desired) {
		if err := os.Remove(projector.absolute(path)); err != nil && !os.IsNotExist(err) {
			return Result{}, err
		}
		projector.removeEmptyGeneratedParents(path)
	}
	for _, file := range files {
		current, _ := os.ReadFile(projector.absolute(file.Path))
		if !bytes.Equal(current, file.Content) {
			if err := filesystemadapter.WriteAtomic(projector.absolute(file.Path), file.Content, 0o644); err != nil {
				return Result{}, err
			}
		}
	}
	if sharedChanged {
		if err := filesystemadapter.WriteAtomic(projector.absolute(openCodePath), sharedDocument, 0o644); err != nil {
			return Result{}, err
		}
	}
	if !bytes.Equal(currentManifest, manifestContent) {
		if err := filesystemadapter.WriteAtomic(projector.absolute(manifestPath), manifestContent, 0o644); err != nil {
			return Result{}, err
		}
	}
	return Result{Operation: "sync", Client: Client, Changed: len(changed) > 0, Paths: changed}, nil
}

func (projector *Projector) Uninstall(force bool) (Result, error) {
	manifest, installed, err := projector.readManifest()
	if err != nil {
		return Result{}, err
	}
	if !installed {
		return Result{}, ErrNotFound
	}
	drifted := []string{}
	for _, owned := range manifest.Files {
		content, readErr := os.ReadFile(projector.absolute(owned.Path))
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return Result{}, readErr
		}
		if hash(content) != owned.SHA256 {
			drifted = append(drifted, owned.Path)
		}
	}
	if len(drifted) > 0 && !force {
		sort.Strings(drifted)
		return Result{}, fmt.Errorf("%w: %s", ErrDrift, strings.Join(drifted, ", "))
	}
	sharedDocument, configChanged, removeSharedFile, err := projector.prepareUninstallOpenCodeConfig(manifest.OpenCode)
	if err != nil {
		return Result{}, err
	}

	changed := []string{}
	for _, owned := range manifest.Files {
		if err := os.Remove(projector.absolute(owned.Path)); err == nil {
			changed = append(changed, owned.Path)
			projector.removeEmptyGeneratedParents(owned.Path)
		} else if !os.IsNotExist(err) {
			return Result{}, err
		}
	}
	if configChanged {
		if removeSharedFile {
			if err := os.Remove(projector.absolute(openCodePath)); err != nil && !os.IsNotExist(err) {
				return Result{}, err
			}
			if manifest.OpenCode.DirectoryAdded {
				_ = os.Remove(projector.absolute(".opencode"))
			}
		} else {
			if err := filesystemadapter.WriteAtomic(projector.absolute(openCodePath), sharedDocument, 0o644); err != nil {
				return Result{}, err
			}
		}
		changed = append(changed, openCodePath)
	}
	if err := os.Remove(projector.absolute(manifestPath)); err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	changed = append(changed, manifestPath)
	changed = uniqueSorted(changed)
	return Result{Operation: "uninstall", Client: Client, Changed: len(changed) > 0, Paths: changed}, nil
}

func (projector *Projector) readConfig() (Config, error) {
	content, err := os.ReadFile(projector.absolute(configPath))
	if os.IsNotExist(err) {
		return Config{}, fmt.Errorf("Karl config not found; run karl-ai init opencode")
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", configPath, err)
	}
	if config.OpenCode.Agents == nil {
		config.OpenCode.Agents = map[string]AgentConfig{}
	}
	return config, validateConfig(config)
}

func (projector *Projector) writeConfig(config Config) error {
	content, err := marshalJSON(config)
	if err != nil {
		return err
	}
	writeAtomic := projector.writeAtomic
	if writeAtomic == nil {
		writeAtomic = filesystemadapter.WriteAtomic
	}
	return writeAtomic(projector.absolute(configPath), content, 0o644)
}

func (projector *Projector) readManifest() (Manifest, bool, error) {
	content, err := os.ReadFile(projector.absolute(manifestPath))
	if os.IsNotExist(err) {
		return Manifest{}, false, nil
	}
	if err != nil {
		return Manifest{}, false, err
	}
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, false, fmt.Errorf("read %s: %w", manifestPath, err)
	}
	if manifest.Version != ManifestVersion || manifest.Client != Client {
		return Manifest{}, false, fmt.Errorf("unsupported manifest version or client")
	}
	for i := 1; i < len(manifest.Files); i++ {
		if manifest.Files[i-1].Path >= manifest.Files[i].Path {
			return Manifest{}, false, fmt.Errorf("manifest file paths are not sorted and unique")
		}
	}
	for _, file := range manifest.Files {
		if !validManagedPath(file.Path) {
			return Manifest{}, false, fmt.Errorf("manifest contains invalid managed path %q", file.Path)
		}
		if len(file.SHA256) != sha256.Size*2 {
			return Manifest{}, false, fmt.Errorf("manifest contains invalid hash for %q", file.Path)
		}
		if _, err := hex.DecodeString(file.SHA256); err != nil {
			return Manifest{}, false, fmt.Errorf("manifest contains invalid hash for %q", file.Path)
		}
	}
	return manifest, true, nil
}

func (projector *Projector) compareProjection(desired map[string]File, old Manifest, installed bool) ([]string, []string, error) {
	drifted := []string{}
	changed := []string{}
	owned := map[string]string{}
	if installed {
		for _, file := range old.Files {
			owned[file.Path] = file.SHA256
		}
	}
	for path, file := range desired {
		content, err := os.ReadFile(projector.absolute(path))
		if os.IsNotExist(err) {
			changed = append(changed, path)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if bytes.Equal(content, file.Content) {
			continue
		}
		changed = append(changed, path)
		priorHash, ownedPath := owned[path]
		if !ownedPath || hash(content) != priorHash {
			drifted = append(drifted, path)
		}
	}
	for _, path := range stalePaths(old, desired) {
		content, err := os.ReadFile(projector.absolute(path))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		changed = append(changed, path)
		for _, oldFile := range old.Files {
			if oldFile.Path == path && hash(content) != oldFile.SHA256 {
				drifted = append(drifted, path)
			}
		}
	}
	return uniqueSorted(drifted), uniqueSorted(changed), nil
}

func stalePaths(old Manifest, desired map[string]File) []string {
	paths := []string{}
	for _, file := range old.Files {
		if _, ok := desired[file.Path]; !ok {
			paths = append(paths, file.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

func (projector *Projector) desiredOpenCodeConfig(old Manifest, installed bool) ([]byte, SharedState, bool, []string, error) {
	document, exists, err := projector.readOpenCodeDocument()
	if err != nil {
		return nil, SharedState{}, false, nil, err
	}
	state := old.OpenCode
	if !installed {
		state.SchemaAdded = !hasKey(document, "$schema")
		state.FileAdded = !exists
		if _, err := os.Stat(projector.absolute(".opencode")); os.IsNotExist(err) {
			state.DirectoryAdded = true
		} else if err != nil {
			return nil, state, false, nil, err
		}
		state.DefaultAgent = previous(document, "default_agent")
		permission, err := object(document, "permission", true)
		if err != nil {
			return nil, state, false, nil, err
		}
		if permission != nil {
			task, err := object(permission, "task", true)
			if err != nil {
				return nil, state, false, nil, err
			}
			if task != nil {
				state.KarlTaskPermission = previous(task, "karl-*")
			}
		}
	}
	sharedDrift := []string{}
	if installed && sharedValuesDrifted(document) {
		sharedDrift = append(sharedDrift, openCodePath)
	}
	changed := !exists
	if !hasKey(document, "$schema") {
		document["$schema"] = openCodeSchema
		changed = true
	}
	if value, ok := document["default_agent"]; !ok || value != defaultAgent {
		document["default_agent"] = defaultAgent
		changed = true
	}
	permission, err := object(document, "permission", false)
	if err != nil {
		return nil, state, false, nil, err
	}
	task, err := object(permission, "task", false)
	if err != nil {
		return nil, state, false, nil, err
	}
	if value, ok := task["karl-*"]; !ok || value != "allow" {
		task["karl-*"] = "allow"
		changed = true
	}
	content, err := marshalJSON(document)
	return content, state, changed, sharedDrift, err
}

func (projector *Projector) prepareUninstallOpenCodeConfig(state SharedState) ([]byte, bool, bool, error) {
	document, exists, err := projector.readOpenCodeDocument()
	if err != nil || !exists {
		return nil, false, false, err
	}
	changed := false
	if state.SchemaAdded && document["$schema"] == openCodeSchema {
		delete(document, "$schema")
		changed = true
	}
	if document["default_agent"] == defaultAgent {
		restore(document, "default_agent", state.DefaultAgent)
		changed = true
	}
	permission, err := object(document, "permission", true)
	if err != nil {
		return nil, false, false, err
	}
	if permission != nil {
		task, err := object(permission, "task", true)
		if err != nil {
			return nil, false, false, err
		}
		if task != nil && task["karl-*"] == "allow" {
			restore(task, "karl-*", state.KarlTaskPermission)
			changed = true
			if len(task) == 0 {
				delete(permission, "task")
			}
			if len(permission) == 0 {
				delete(document, "permission")
			}
		}
	}
	if changed && state.FileAdded && len(document) == 0 {
		return nil, true, true, nil
	}
	if !changed {
		return nil, false, false, nil
	}
	content, err := marshalJSON(document)
	if err != nil {
		return nil, false, false, err
	}
	return content, true, false, nil
}

func sharedValuesDrifted(document map[string]any) bool {
	if document["default_agent"] != defaultAgent {
		return true
	}
	permission, ok := document["permission"].(map[string]any)
	if !ok {
		return true
	}
	task, ok := permission["task"].(map[string]any)
	return !ok || task["karl-*"] != "allow"
}

func (projector *Projector) readOpenCodeDocument() (map[string]any, bool, error) {
	content, err := os.ReadFile(projector.absolute(openCodePath))
	if os.IsNotExist(err) {
		return map[string]any{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	document := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, false, fmt.Errorf("read %s: %w", openCodePath, err)
	}
	return document, true, nil
}

func object(parent map[string]any, key string, optional bool) (map[string]any, error) {
	value, exists := parent[key]
	if !exists {
		if optional {
			return nil, nil
		}
		child := map[string]any{}
		parent[key] = child
		return child, nil
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s in %s must be a JSON object", key, openCodePath)
	}
	return child, nil
}

func previous(parent map[string]any, key string) PreviousValue {
	value, present := parent[key]
	if !present {
		return PreviousValue{}
	}
	encoded, _ := json.Marshal(value)
	return PreviousValue{Present: true, Value: encoded}
}

func restore(parent map[string]any, key string, previous PreviousValue) {
	if !previous.Present {
		delete(parent, key)
		return
	}
	var value any
	if json.Unmarshal(previous.Value, &value) == nil {
		parent[key] = value
	}
}

func hasKey(parent map[string]any, key string) bool {
	_, ok := parent[key]
	return ok
}

func validManagedPath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.ToSlash(value) != value {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	return clean == value && strings.HasPrefix(value, ".opencode/") && value != openCodePath
}

func (projector *Projector) removeEmptyGeneratedParents(relative string) {
	directory := filepath.Dir(projector.absolute(relative))
	stop := projector.absolute(".opencode")
	for directory != stop && strings.HasPrefix(directory, stop+string(filepath.Separator)) {
		if err := os.Remove(directory); err != nil {
			return
		}
		directory = filepath.Dir(directory)
	}
}

func (projector *Projector) absolute(relative string) string {
	return filepath.Join(projector.root, filepath.FromSlash(relative))
}

func hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func marshalJSON(value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(value)
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
