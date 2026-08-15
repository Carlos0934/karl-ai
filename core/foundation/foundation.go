package foundation

import (
	"embed"
	"fmt"
)

var RequiredArtifacts = []string{"docs/CONTEXT.md", "docs/DESIGN.md"}

type AssuranceLevel string

const (
	AssuranceL1 AssuranceLevel = "L1"
	AssuranceL2 AssuranceLevel = "L2"
	AssuranceL3 AssuranceLevel = "L3"
	AssuranceL4 AssuranceLevel = "L4"
)

var AssuranceLevels = []AssuranceLevel{AssuranceL1, AssuranceL2, AssuranceL3, AssuranceL4}

type FoundationTemplate struct {
	Name    string
	Path    string
	Content string
}

//go:embed templates/*.md
var templateFiles embed.FS

var templateNames = []string{"CONTEXT", "DESIGN", "JOURNEY"}

func Templates() []FoundationTemplate {
	templates := make([]FoundationTemplate, 0, len(templateNames))
	for _, name := range templateNames {
		content, err := Template(name)
		if err != nil {
			panic(err)
		}
		templates = append(templates, FoundationTemplate{
			Name:    name,
			Path:    "templates/" + name + ".template.md",
			Content: content,
		})
	}
	return templates
}

func Template(name string) (string, error) {
	content, err := templateFiles.ReadFile("templates/" + name + ".template.md")
	if err != nil {
		return "", fmt.Errorf("read embedded foundation template %s: %w", name, err)
	}
	return string(content), nil
}
