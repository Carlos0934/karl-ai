package opencode

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/carlos0934/karl-ai/internal/domain"
)

func AgentRelPath(id domain.AgentID) string {
	return path.Join(".opencode/agents", string(id)+".md")
}

func DesiredManagedPatterns(agents []domain.Agent) []string {
	patterns := make([]string, 0, len(agents))
	for _, agent := range agents {
		patterns = append(patterns, AgentRelPath(agent.ID))
	}
	return patterns
}

func Render(agent domain.Agent, settings domain.AgentConfig) string {
	fields := []struct{ name, value string }{
		{"description", agent.Description},
	}
	var builder strings.Builder
	builder.WriteString("---\n")
	for _, f := range fields {
		builder.WriteString(f.name)
		builder.WriteString(": ")
		builder.WriteString(strconv.Quote(f.value))
		builder.WriteByte('\n')
	}
	builder.WriteString("mode: primary\n")
	builder.WriteString("model: " + strconv.Quote(settings.Model) + "\n")
	if settings.Variant != nil && *settings.Variant != "" {
		builder.WriteString("variant: " + strconv.Quote(*settings.Variant) + "\n")
	}
	builder.WriteString(RenderPermissionsYAML(DefaultPermissionsForAgent(agent.ID)))
	builder.WriteString("---\n\n")
	builder.WriteString(normalizedBody(agent.Prompt))
	return builder.String()
}

func ParseAgentConfig(content []byte) (domain.AgentConfig, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return domain.AgentConfig{}, fmt.Errorf("missing frontmatter delimiter '---'")
	}

	var model string
	var variant *string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		unquoted, err := strconv.Unquote(val)
		if err == nil {
			val = unquoted
		}

		switch key {
		case "model":
			model = val
		case "variant":
			v := val
			variant = &v
		}
	}

	if err := scanner.Err(); err != nil {
		return domain.AgentConfig{}, err
	}

	return domain.AgentConfig{
		Model:   model,
		Variant: variant,
	}, nil
}

func normalizedBody(body string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n")) + "\n"
}

