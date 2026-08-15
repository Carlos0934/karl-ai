package git

import (
	"os/exec"
	"strings"

	"github.com/carlos0934/karl-ai/core/lifecycle"
)

type Repository struct {
	root string
}

func New(root string) *Repository {
	return &Repository{root: root}
}

func (repository *Repository) IsRepository() bool {
	_, err := repository.run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

func (repository *Repository) ResolvesCommit(reference string) bool {
	_, err := repository.run("rev-parse", "--verify", reference+"^{commit}")
	return err == nil
}

func (repository *Repository) UncommittedFiles() ([]string, error) {
	output, err := repository.run("status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	files := []string{}
	for _, line := range strings.Split(output, "\n") {
		raw := ""
		if len(line) > 3 {
			raw = strings.TrimSpace(line[3:])
		}
		if parts := strings.Split(raw, " -> "); len(parts) > 1 {
			raw = parts[len(parts)-1]
		}
		files = append(files, raw)
	}
	return files, nil
}

func (repository *Repository) run(args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repository.root
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

var _ lifecycle.Git = (*Repository)(nil)
