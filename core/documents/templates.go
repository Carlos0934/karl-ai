package documents

import (
	"embed"
	"fmt"
)

//go:embed templates/*.md
var templates embed.FS

var RequiredChangeFiles = []string{"CHANGE.md", "PLAN.md", "TASKS.md", "RESEARCH.md", "REVIEW.md"}

func Template(name string) (string, error) {
	content, err := templates.ReadFile("templates/" + name + ".template.md")
	if err != nil {
		return "", fmt.Errorf("read embedded template %s: %w", name, err)
	}
	return string(content), nil
}
