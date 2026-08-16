package catalog

import (
	"embed"
	"fmt"

	"github.com/carlos0934/karl-ai/internal/domain"
)

type AgentID = domain.AgentID
type Agent = domain.Agent

const AgentOrchestrator = domain.AgentOrchestrator

//go:embed content
var contentFiles embed.FS

var agents = []domain.Agent{
	{
		ID:          domain.AgentOrchestrator,
		Description: "Reserved primary agent with no assigned responsibilities.",
		Prompt:      mustContent("content/agents/karl-orchestrator.md"),
	},
}

func Agents() []domain.Agent {
	return append([]domain.Agent(nil), agents...)
}

func AgentByID(id domain.AgentID) (domain.Agent, bool) {
	for _, agent := range agents {
		if agent.ID == id {
			return agent, true
		}
	}
	return domain.Agent{}, false
}

func mustContent(path string) string {
	content, err := contentFiles.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read embedded catalog content %s: %v", path, err))
	}
	return string(content)
}
