package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/carlos0934/karl-ai/internal/domain"
)

type clientsDiscoveredMsg struct {
	request uint64
	clients []domain.ClientInfo
	err     error
}

type catalogDiscoveredMsg struct {
	request uint64
	client  string
	catalog domain.ModelCatalog
	err     error
}

type savedModelDiscoveredMsg struct {
	request uint64
	agent   string
	model   domain.AgentConfig
	err     error
}

type formGenerationMsg struct {
	generation uint64
	message    tea.Msg
}
