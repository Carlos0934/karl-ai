package catalog

import (
	"embed"
	"fmt"

	"github.com/carlos0934/karl-ai/core/documents"
	"github.com/carlos0934/karl-ai/core/foundation"
)

type AgentID string
type CommandID string
type SkillID string

const (
	AgentOrchestrator AgentID = "karl-orchestrator"
	AgentSearcher     AgentID = "karl-searcher"
	AgentFoundation   AgentID = "karl-foundation"
	AgentPlanner      AgentID = "karl-planner"
	AgentImplementer  AgentID = "karl-implementer"
	AgentReviewer     AgentID = "karl-reviewer"
	AgentArchiver     AgentID = "karl-archiver"
)

const (
	CommandFoundation      CommandID = "karl-foundation"
	CommandChangeNew       CommandID = "karl-change-new"
	CommandChangeImplement CommandID = "karl-change-implement"
	CommandChangeReview    CommandID = "karl-change-review"
	CommandChangeArchive   CommandID = "karl-change-archive"
)

const (
	SkillChangeLifecycle   SkillID = "karl-change-lifecycle"
	SkillProjectFoundation SkillID = "karl-project-foundation"
)

type AgentRole string

const (
	PrimaryAgent    AgentRole = "primary"
	SpecialistAgent AgentRole = "specialist"
)

type Agent struct {
	ID          AgentID
	Description string
	Role        AgentRole
	Prompt      string
	Delegates   []AgentID
	Skills      []SkillID
}

type Command struct {
	ID          CommandID
	Description string
	Usage       string
	Agent       AgentID
	Body        string
}

type ResourceKind string

const (
	ReferenceResource ResourceKind = "reference"
	TemplateResource  ResourceKind = "template"
)

type Resource struct {
	Path    string
	Kind    ResourceKind
	Content string
}

type Skill struct {
	ID           SkillID
	Description  string
	Instructions string
	Resources    []Resource
}

//go:embed content
var contentFiles embed.FS

var agents = []Agent{
	{
		ID:          AgentOrchestrator,
		Description: "Routes foundation and change lifecycle work across Karl specialists and owns gates, decisions, and user contact.",
		Role:        PrimaryAgent,
		Prompt:      mustContent("content/agents/karl-orchestrator.md"),
		Delegates:   []AgentID{AgentSearcher, AgentFoundation, AgentPlanner, AgentImplementer, AgentReviewer, AgentArchiver},
	},
	{
		ID:          AgentSearcher,
		Description: "Collects compact repository, documentation, and external research evidence in RESEARCH.md.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-searcher.md"),
	},
	{
		ID:          AgentFoundation,
		Description: "Runs discovery, classification, questioning, and baseline document creation for the project foundation.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-foundation.md"),
		Skills:      []SkillID{SkillProjectFoundation},
	},
	{
		ID:          AgentPlanner,
		Description: "Builds a researched change package with vertical work units and passes the plan gate.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-planner.md"),
		Skills:      []SkillID{SkillChangeLifecycle},
	},
	{
		ID:          AgentImplementer,
		Description: "Implements one vertical work unit at a time and passes the implement gate without committing automatically.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-implementer.md"),
		Skills:      []SkillID{SkillChangeLifecycle},
	},
	{
		ID:          AgentReviewer,
		Description: "Reviews behavior, records evidence, reconciles foundation artifacts, and owns the review gate.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-reviewer.md"),
		Skills:      []SkillID{SkillChangeLifecycle},
	},
	{
		ID:          AgentArchiver,
		Description: "Archives a validated change and proposes the final integration commit without creating it.",
		Role:        SpecialistAgent,
		Prompt:      mustContent("content/agents/karl-archiver.md"),
		Skills:      []SkillID{SkillChangeLifecycle},
	},
}

var commands = []Command{
	{CommandFoundation, "Analyze project state and establish or refresh the Karl project foundation.", "karl-foundation [scope]", AgentOrchestrator, mustContent("content/commands/karl-foundation.md")},
	{CommandChangeNew, "Plan a tracked Karl change and pass the plan gate.", "karl-change-new <name> [--level L1|L2|L3|L4]", AgentOrchestrator, mustContent("content/commands/karl-change-new.md")},
	{CommandChangeImplement, "Implement a planned Karl change with vertical work units through the implement gate.", "karl-change-implement [<name>]", AgentOrchestrator, mustContent("content/commands/karl-change-implement.md")},
	{CommandChangeReview, "Review a Karl change, record evidence, reconcile foundation artifacts, and validate it.", "karl-change-review <name>", AgentOrchestrator, mustContent("content/commands/karl-change-review.md")},
	{CommandChangeArchive, "Archive a validated Karl change and prepare the final integration commit.", "karl-change-archive <name>", AgentOrchestrator, mustContent("content/commands/karl-change-archive.md")},
}

var skills = []Skill{
	{
		ID:           SkillChangeLifecycle,
		Description:  "Manage behavior-changing work through researched plans, vertical work units, evidence, and enforced lifecycle gates.",
		Instructions: mustContent("content/skills/karl-change-lifecycle/SKILL.md"),
		Resources: append(
			contentResources("content/skills/karl-change-lifecycle/references", []string{"workflow.md", "questioning-guide.md", "work-units.md", "foundation-integration.md"}),
			documentResources()...,
		),
	},
	{
		ID:           SkillProjectFoundation,
		Description:  "Establish project context, journeys, and an engineering baseline through adaptive discovery.",
		Instructions: mustContent("content/skills/karl-project-foundation/SKILL.md"),
		Resources: append(
			contentResources("content/skills/karl-project-foundation/references", []string{"documentation-model.md", "complexity-levels.md", "question-matrix.md"}),
			foundationResources()...,
		),
	},
}

func Agents() []Agent {
	result := make([]Agent, len(agents))
	for i, agent := range agents {
		result[i] = agent
		result[i].Delegates = append([]AgentID(nil), agent.Delegates...)
		result[i].Skills = append([]SkillID(nil), agent.Skills...)
	}
	return result
}

func AgentByID(id AgentID) (Agent, bool) {
	for _, agent := range Agents() {
		if agent.ID == id {
			return agent, true
		}
	}
	return Agent{}, false
}

func Commands() []Command {
	return append([]Command(nil), commands...)
}

func CommandByID(id CommandID) (Command, bool) {
	for _, command := range commands {
		if command.ID == id {
			return command, true
		}
	}
	return Command{}, false
}

func Skills() []Skill {
	result := make([]Skill, len(skills))
	for i, skill := range skills {
		result[i] = skill
		result[i].Resources = append([]Resource(nil), skill.Resources...)
	}
	return result
}

func SkillByID(id SkillID) (Skill, bool) {
	for _, skill := range Skills() {
		if skill.ID == id {
			return skill, true
		}
	}
	return Skill{}, false
}

func ResourceByPath(skillID SkillID, path string) (Resource, bool) {
	skill, ok := SkillByID(skillID)
	if !ok {
		return Resource{}, false
	}
	for _, resource := range skill.Resources {
		if resource.Path == path {
			return resource, true
		}
	}
	return Resource{}, false
}

func mustContent(path string) string {
	content, err := contentFiles.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read embedded catalog content %s: %v", path, err))
	}
	return string(content)
}

func contentResources(base string, names []string) []Resource {
	resources := make([]Resource, 0, len(names))
	for _, name := range names {
		resources = append(resources, Resource{
			Path:    "references/" + name,
			Kind:    ReferenceResource,
			Content: mustContent(base + "/" + name),
		})
	}
	return resources
}

func documentResources() []Resource {
	resources := make([]Resource, 0, len(documents.RequiredChangeFiles))
	for _, file := range documents.RequiredChangeFiles {
		name := file[:len(file)-len(".md")]
		content, err := documents.Template(name)
		if err != nil {
			panic(err)
		}
		resources = append(resources, Resource{
			Path:    "templates/" + name + ".template.md",
			Kind:    TemplateResource,
			Content: content,
		})
	}
	return resources
}

func foundationResources() []Resource {
	templates := foundation.Templates()
	resources := make([]Resource, 0, len(templates))
	for _, template := range templates {
		resources = append(resources, Resource{
			Path:    template.Path,
			Kind:    TemplateResource,
			Content: template.Content,
		})
	}
	return resources
}
