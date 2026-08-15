package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/carlos0934/karl-ai/core/documents"
	"github.com/carlos0934/karl-ai/core/lifecycle"
)

type Store struct {
	root string
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func New(root string) *Store {
	return &Store{root: root}
}

func (store *Store) Missing(paths []string) ([]string, error) {
	missing := []string{}
	for _, relative := range paths {
		if _, err := os.Stat(filepath.Join(store.root, filepath.FromSlash(relative))); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, relative)
				continue
			}
			return nil, err
		}
	}
	return missing, nil
}

func (store *Store) CreateChange(slug string, files map[string]string) (string, error) {
	destination := store.activeChangePath(slug)
	if _, err := os.Stat(destination); err == nil {
		return "", fmt.Errorf("Active change already exists: %s", slug)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	for _, name := range documents.RequiredChangeFiles {
		if err := writeAtomic(filepath.Join(destination, name), files[name]); err != nil {
			_ = os.RemoveAll(destination)
			return "", err
		}
	}
	return destination, nil
}

func (store *Store) ReadChange(slug string) (lifecycle.Package, error) {
	if !slugPattern.MatchString(slug) {
		return lifecycle.Package{}, fmt.Errorf("Change name must be lowercase kebab-case")
	}
	directory := store.activeChangePath(slug)
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		if err == nil || os.IsNotExist(err) {
			return lifecycle.Package{}, fmt.Errorf("Active change not found: %s", slug)
		}
		return lifecycle.Package{}, err
	}

	files := make(map[string]string, len(documents.RequiredChangeFiles))
	for _, name := range documents.RequiredChangeFiles {
		content, readErr := os.ReadFile(filepath.Join(directory, name))
		if readErr != nil {
			if os.IsNotExist(readErr) {
				return lifecycle.Package{}, fmt.Errorf("Missing required change file: %s", name)
			}
			return lifecycle.Package{}, readErr
		}
		files[name] = string(content)
	}
	parsed, err := documents.ParseFrontmatter(files["CHANGE.md"])
	if err != nil {
		return lifecycle.Package{}, fmt.Errorf("Invalid CHANGE.md: %s", err)
	}
	name := documents.StringValue(parsed.Data["name"])
	if name != slug {
		return lifecycle.Package{}, fmt.Errorf("CHANGE.md name '%s' does not match folder '%s'", name, slug)
	}
	return lifecycle.Package{Directory: directory, Files: files, Metadata: parsed.Data}, nil
}

func (store *Store) WriteChange(slug, content string) error {
	return writeAtomic(filepath.Join(store.activeChangePath(slug), "CHANGE.md"), content)
}

func (store *Store) ArchivedDependencyExists(dependency string) (bool, error) {
	entries, err := os.ReadDir(store.archiveRoot())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	pattern, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}-` + dependency + `$`)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() && pattern.MatchString(entry.Name()) {
			return true, nil
		}
	}
	return false, nil
}

func (store *Store) FoundationFileCount(slug string) (int, error) {
	directory := filepath.Join(store.activeChangePath(slug), "foundation")
	count := 0
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func (store *Store) ArchiveChange(slug, date, updatedChange, originalChange string) (string, error) {
	destination := filepath.Join(store.archiveRoot(), date+"-"+slug)
	if _, err := os.Stat(destination); err == nil {
		return "", fmt.Errorf("Archive destination already exists: %s", destination)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	source := store.activeChangePath(slug)
	changePath := filepath.Join(source, "CHANGE.md")
	if err := writeAtomic(changePath, updatedChange); err != nil {
		return "", err
	}
	if err := os.MkdirAll(store.archiveRoot(), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(source, destination); err != nil {
		_ = writeAtomic(changePath, originalChange)
		return "", err
	}
	return destination, nil
}

func (store *Store) ListChanges() (lifecycle.ChangeList, error) {
	result := lifecycle.ChangeList{Active: []lifecycle.ChangeItem{}, Archived: []string{}}
	entries, err := os.ReadDir(filepath.Join(store.root, "changes"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archive" {
			continue
		}
		var state any = "invalid"
		content, readErr := os.ReadFile(filepath.Join(store.root, "changes", entry.Name(), "CHANGE.md"))
		if readErr == nil {
			parsed, parseErr := documents.ParseFrontmatter(string(content))
			if parseErr == nil {
				state = parsed.Data["state"]
				if state == nil {
					state = "unknown"
				}
			}
		}
		result.Active = append(result.Active, lifecycle.ChangeItem{Name: entry.Name(), State: state})
	}
	archived, archiveErr := os.ReadDir(store.archiveRoot())
	if archiveErr != nil && !os.IsNotExist(archiveErr) {
		return result, archiveErr
	}
	for _, entry := range archived {
		if entry.IsDir() {
			result.Archived = append(result.Archived, entry.Name())
		}
	}
	sort.Slice(result.Active, func(i, j int) bool { return result.Active[i].Name < result.Active[j].Name })
	sort.Strings(result.Archived)
	return result, nil
}

func (store *Store) activeChangePath(slug string) string {
	return filepath.Join(store.root, "changes", slug)
}

func (store *Store) archiveRoot() string {
	return filepath.Join(store.root, "changes", "archive")
}

func writeAtomic(target, content string) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".karl-ai-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err = temporary.WriteString(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporaryName, target); err != nil {
		return err
	}
	return nil
}

var _ lifecycle.Store = (*Store)(nil)

// Keep filepath normalization local to the adapter.
func normalizePath(path string) string { return strings.ReplaceAll(path, "\\", "/") }
