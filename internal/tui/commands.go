package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

func (model *interactiveConfigureModel) startRequest(screen interactiveScreen) (context.Context, uint64) {
	if model.requestCancel != nil {
		model.requestCancel()
		model.requestCancel = nil
	}
	model.requestSeq++
	model.activeRequest = model.requestSeq
	requestContext, cancel := context.WithCancel(model.ctx)
	model.requestCancel = cancel
	model.screen = screen
	return requestContext, model.activeRequest
}

func (model *interactiveConfigureModel) discoverClients(ctx context.Context, request uint64) tea.Cmd {
	return func() tea.Msg {
		clients, err := model.discovery.Clients(ctx, model.root)
		return clientsDiscoveredMsg{request: request, clients: clients, err: err}
	}
}

func (model *interactiveConfigureModel) discoverCatalog(ctx context.Context, request uint64, client string) tea.Cmd {
	return func() tea.Msg {
		catalog, err := model.discovery.Catalog(ctx, model.root, client)
		return catalogDiscoveredMsg{request: request, client: client, catalog: catalog, err: err}
	}
}

func (model *interactiveConfigureModel) loadSavedModel(ctx context.Context, request uint64, agent string) tea.Cmd {
	return func() tea.Msg {
		if model.savedSource == nil {
			return savedModelDiscoveredMsg{request: request, agent: agent}
		}
		saved, err := model.savedSource.ModelForAgent(ctx, agent)
		return savedModelDiscoveredMsg{request: request, agent: agent, model: saved, err: err}
	}
}

func (model *interactiveConfigureModel) wrapFormCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	generation := model.formGeneration
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}
		return formGenerationMsg{generation: generation, message: msg}
	}
}
