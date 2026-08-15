package opencode

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/carlos0934/karl-ai/core/catalog"
)

const ConfigVersion = 1

type AgentConfig struct {
	Model   string  `json:"model"`
	Variant *string `json:"variant,omitempty"`
}

type OpenCodeConfig struct {
	Agents map[string]AgentConfig `json:"agents"`
}

type Config struct {
	Version  int            `json:"version"`
	OpenCode OpenCodeConfig `json:"opencode"`
}

type File struct {
	Path    string `json:"path"`
	Content []byte `json:"-"`
}

func DefaultConfig() Config {
	return Config{
		Version: ConfigVersion,
		OpenCode: OpenCodeConfig{Agents: map[string]AgentConfig{
			string(catalog.AgentOrchestrator): {Model: "openai/gpt-5.6-sol", Variant: variant("high")},
			string(catalog.AgentSearcher):     {Model: "opencode-go/deepseek-v4-flash"},
			string(catalog.AgentFoundation):   {Model: "opencode-go/deepseek-v4-pro"},
			string(catalog.AgentPlanner):      {Model: "opencode-go/deepseek-v4-pro"},
			string(catalog.AgentImplementer):  {Model: "opencode-go/gpt-5.6-luna", Variant: variant("xhigh")},
			string(catalog.AgentReviewer):     {Model: "openai/gpt-5.6-sol", Variant: variant("high")},
			string(catalog.AgentArchiver):     {Model: "opencode-go/deepseek-v4-flash"},
		}},
	}
}

func Render(config Config) ([]File, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	defaults := DefaultConfig()
	files := make([]File, 0, len(catalog.Agents())+len(catalog.Commands())+2)
	for _, agent := range catalog.Agents() {
		settings := defaults.OpenCode.Agents[string(agent.ID)]
		if override, ok := config.OpenCode.Agents[string(agent.ID)]; ok {
			if override.Model != "" {
				settings.Model = override.Model
			}
			if override.Variant != nil {
				settings.Variant = override.Variant
			}
		}
		content := renderAgent(agent, settings)
		files = append(files, File{Path: path.Join(".opencode/agents", string(agent.ID)+".md"), Content: []byte(content)})
	}
	for _, command := range catalog.Commands() {
		content := frontmatter([]field{{"description", command.Description}}) +
			"Arguments: `$ARGUMENTS`\n\n" + normalizedBody(command.Body)
		files = append(files, File{Path: path.Join(".opencode/commands", string(command.ID)+".md"), Content: []byte(content)})
	}
	for _, skill := range catalog.Skills() {
		content := frontmatter([]field{{"name", string(skill.ID)}, {"description", skill.Description}}) + normalizedBody(skill.Instructions)
		base := path.Join(".opencode/skills", string(skill.ID))
		files = append(files, File{Path: path.Join(base, "SKILL.md"), Content: []byte(content)})
		for _, resource := range skill.Resources {
			files = append(files, File{Path: path.Join(base, resource.Path), Content: []byte(normalizedBody(resource.Content))})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func validateConfig(config Config) error {
	if config.Version != ConfigVersion {
		return fmt.Errorf("unsupported Karl config version %d", config.Version)
	}
	known := make(map[string]bool, len(catalog.Agents()))
	for _, agent := range catalog.Agents() {
		known[string(agent.ID)] = true
	}
	for id := range config.OpenCode.Agents {
		if !known[id] {
			return fmt.Errorf("unknown OpenCode agent override %q", id)
		}
	}
	return nil
}

type field struct {
	name  string
	value string
}

func frontmatter(fields []field) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	for _, field := range fields {
		builder.WriteString(field.name)
		builder.WriteString(": ")
		builder.WriteString(strconv.Quote(field.value))
		builder.WriteByte('\n')
	}
	builder.WriteString("---\n\n")
	return builder.String()
}

func renderAgent(agent catalog.Agent, settings AgentConfig) string {
	fields := []field{{"description", agent.Description}}
	var builder strings.Builder
	builder.WriteString(frontmatterStart(fields))
	if agent.Role == catalog.PrimaryAgent {
		builder.WriteString("mode: primary\n")
	} else {
		builder.WriteString("mode: subagent\nhidden: true\n")
	}
	builder.WriteString("model: " + strconv.Quote(settings.Model) + "\n")
	if settings.Variant != nil && *settings.Variant != "" {
		builder.WriteString("variant: " + strconv.Quote(*settings.Variant) + "\n")
	}
	builder.WriteString(agentPermissions(agent.ID))
	builder.WriteString("---\n\n")
	builder.WriteString(normalizedBody(agent.Prompt))
	return builder.String()
}

func variant(value string) *string {
	return &value
}

func frontmatterStart(fields []field) string {
	value := frontmatter(fields)
	return strings.TrimSuffix(value, "---\n\n")
}

func agentPermissions(id catalog.AgentID) string {
	switch id {
	case catalog.AgentOrchestrator:
		return `permission:
  edit: allow
  bash: allow
  question: allow
  webfetch: allow
  websearch: allow
  task:
    "*": deny
    karl-searcher: allow
    karl-foundation: allow
    karl-planner: allow
    karl-implementer: allow
    karl-reviewer: allow
    karl-archiver: allow
`
	case catalog.AgentSearcher:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit:
    "*": deny
    "changes/**/RESEARCH.md": allow
  webfetch: allow
  websearch: allow
  skill:
    "*": deny
`)
	case catalog.AgentFoundation:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit:
    "*": deny
    "docs/**": allow
  webfetch: deny
  websearch: deny
  skill:
    "*": deny
    karl-project-foundation: allow
`)
	case catalog.AgentPlanner:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit:
    "*": deny
    "changes/**": allow
  webfetch: deny
  websearch: deny
  skill:
    "*": deny
    karl-change-lifecycle: allow
`)
	case catalog.AgentImplementer:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit: allow
  webfetch: deny
  websearch: deny
  skill:
    "*": deny
    karl-change-lifecycle: allow
`)
	case catalog.AgentReviewer:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit:
    "*": deny
    "changes/**": allow
  webfetch: deny
  websearch: deny
  skill:
    "*": deny
    karl-change-lifecycle: allow
`)
	case catalog.AgentArchiver:
		return specialistPermissions(`  read: allow
  glob: allow
  grep: allow
  bash: allow
  edit: deny
  webfetch: deny
  websearch: deny
  skill:
    "*": deny
    karl-change-lifecycle: allow
`)
	default:
		panic("unknown agent permission profile: " + string(id))
	}
}

func specialistPermissions(rules string) string {
	return "permission:\n" + rules + "  task: deny\n  question: deny\n"
}

func normalizedBody(body string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n")) + "\n"
}
