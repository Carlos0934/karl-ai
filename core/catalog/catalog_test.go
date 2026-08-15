package catalog

import (
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/core/foundation"
)

func TestCatalogCompletenessAndUniqueIDs(t *testing.T) {
	if got := len(Agents()); got != 7 {
		t.Fatalf("agents = %d, want 7", got)
	}
	if got := len(Commands()); got != 5 {
		t.Fatalf("commands = %d, want 5", got)
	}
	if got := len(Skills()); got != 2 {
		t.Fatalf("skills = %d, want 2", got)
	}

	agentIDs := map[AgentID]bool{}
	for _, agent := range Agents() {
		if agent.ID == "" || agentIDs[agent.ID] {
			t.Fatalf("agent ID %q is empty or duplicated", agent.ID)
		}
		agentIDs[agent.ID] = true
	}
	commandIDs := map[CommandID]bool{}
	for _, command := range Commands() {
		if command.ID == "" || commandIDs[command.ID] {
			t.Fatalf("command ID %q is empty or duplicated", command.ID)
		}
		commandIDs[command.ID] = true
	}
	skillIDs := map[SkillID]bool{}
	for _, skill := range Skills() {
		if skill.ID == "" || skillIDs[skill.ID] {
			t.Fatalf("skill ID %q is empty or duplicated", skill.ID)
		}
		skillIDs[skill.ID] = true
	}
}

func TestCatalogRelationshipsResolve(t *testing.T) {
	for _, agent := range Agents() {
		for _, delegate := range agent.Delegates {
			if _, ok := AgentByID(delegate); !ok {
				t.Errorf("agent %s delegates to missing agent %s", agent.ID, delegate)
			}
		}
		for _, skill := range agent.Skills {
			if _, ok := SkillByID(skill); !ok {
				t.Errorf("agent %s uses missing skill %s", agent.ID, skill)
			}
		}
	}
	for _, command := range Commands() {
		if _, ok := AgentByID(command.Agent); !ok {
			t.Errorf("command %s routes to missing agent %s", command.ID, command.Agent)
		}
	}
	for _, skill := range Skills() {
		paths := map[string]bool{}
		for _, resource := range skill.Resources {
			if resource.Path == "" || paths[resource.Path] {
				t.Errorf("skill %s resource path %q is empty or duplicated", skill.ID, resource.Path)
			}
			paths[resource.Path] = true
		}
	}
}

func TestCatalogResourcesAreComplete(t *testing.T) {
	required := map[SkillID][]string{
		SkillChangeLifecycle: {
			"references/workflow.md",
			"references/questioning-guide.md",
			"references/work-units.md",
			"references/foundation-integration.md",
			"templates/CHANGE.template.md",
			"templates/PLAN.template.md",
			"templates/TASKS.template.md",
			"templates/RESEARCH.template.md",
			"templates/REVIEW.template.md",
		},
		SkillProjectFoundation: {
			"references/documentation-model.md",
			"references/complexity-levels.md",
			"references/question-matrix.md",
			"templates/CONTEXT.template.md",
			"templates/DESIGN.template.md",
			"templates/JOURNEY.template.md",
		},
	}

	for skillID, paths := range required {
		for _, path := range paths {
			resource, ok := ResourceByPath(skillID, path)
			if !ok {
				t.Errorf("skill %s is missing resource %s", skillID, path)
				continue
			}
			if strings.TrimSpace(resource.Content) == "" {
				t.Errorf("skill %s resource %s is empty", skillID, path)
			}
		}
	}
}

func TestFoundationTemplatesAreEmbedded(t *testing.T) {
	templates := foundation.Templates()
	if got := len(templates); got != 3 {
		t.Fatalf("foundation templates = %d, want 3", got)
	}
	for _, name := range []string{"CONTEXT", "DESIGN", "JOURNEY"} {
		content, err := foundation.Template(name)
		if err != nil {
			t.Fatalf("foundation template %s: %v", name, err)
		}
		if strings.TrimSpace(content) == "" {
			t.Errorf("foundation template %s is empty", name)
		}
	}
}

func TestPortableContentExcludesLegacyClientDetails(t *testing.T) {
	content := make([]string, 0)
	for _, agent := range Agents() {
		content = append(content, agent.Prompt)
	}
	for _, command := range Commands() {
		content = append(content, command.Body)
	}
	for _, skill := range Skills() {
		content = append(content, skill.Instructions)
		for _, resource := range skill.Resources {
			content = append(content, resource.Content)
		}
	}

	for _, forbidden := range []string{
		".opencode",
		"change.mjs",
		"opencode-go/",
		"openai/",
		"anthropic/",
		"google/",
	} {
		for i, item := range content {
			if strings.Contains(strings.ToLower(item), forbidden) {
				t.Errorf("portable content %d contains forbidden string %q", i, forbidden)
			}
		}
	}
}

func TestPromptsAndBodiesHaveNoClientFrontmatter(t *testing.T) {
	assertNoFrontmatter := func(label, body string) {
		t.Helper()
		if strings.HasPrefix(strings.TrimSpace(body), "---") {
			t.Errorf("%s starts with YAML frontmatter", label)
		}
		for _, syntax := range []string{"\npermission:", "\nmodel:", "\nvariant:", "\nmode:", "\nhidden:"} {
			if strings.Contains(strings.ToLower(body), syntax) {
				t.Errorf("%s contains client configuration syntax %q", label, syntax)
			}
		}
	}

	for _, agent := range Agents() {
		assertNoFrontmatter("agent "+string(agent.ID), agent.Prompt)
	}
	for _, command := range Commands() {
		assertNoFrontmatter("command "+string(command.ID), command.Body)
	}
	for _, skill := range Skills() {
		assertNoFrontmatter("skill "+string(skill.ID), skill.Instructions)
	}
}
