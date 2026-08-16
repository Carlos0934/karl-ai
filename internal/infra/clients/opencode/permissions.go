package opencode

import (
	"sort"
	"strings"

	"github.com/carlos0934/karl-ai/internal/domain"
)

type PermissionAction string

const (
	PermissionAllow PermissionAction = "allow"
	PermissionAsk   PermissionAction = "ask"
	PermissionDeny  PermissionAction = "deny"
)

type ToolRule map[string]PermissionAction

type Permissions struct {
	Global *PermissionAction
	Tools  map[string]ToolRule
}

func DefaultPermissionsForAgent(id domain.AgentID) Permissions {
	switch id {
	case domain.AgentOrchestrator:
		return Permissions{
			Tools: map[string]ToolRule{
				"task": {
					"general": PermissionAllow,
				},
			},
		}
	default:
		return Permissions{}
	}
}

func RenderPermissionsYAML(p Permissions) string {
	if p.Global != nil {
		return "permission: " + string(*p.Global) + "\n"
	}
	if len(p.Tools) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("permission:\n")

	toolKeys := make([]string, 0, len(p.Tools))
	for k := range p.Tools {
		toolKeys = append(toolKeys, k)
	}
	sort.Strings(toolKeys)

	for _, tool := range toolKeys {
		rules := p.Tools[tool]
		b.WriteString("  " + tool + ":\n")

		patternKeys := make([]string, 0, len(rules))
		for pat := range rules {
			patternKeys = append(patternKeys, pat)
		}
		sort.Strings(patternKeys)

		for _, pat := range patternKeys {
			b.WriteString("    " + pat + ": " + string(rules[pat]) + "\n")
		}
	}
	return b.String()
}
