package tui

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func (model *interactiveConfigureModel) Init() tea.Cmd {
	requestContext, request := model.startRequest(screenLoadingClients)
	return model.discoverClients(requestContext, request)
}

func (model *interactiveConfigureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wrapped, ok := msg.(formGenerationMsg); ok {
		if wrapped.generation != model.formGeneration {
			return model, nil
		}
		msg = wrapped.message
	}
	switch message := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(message, quitKey) {
			return model, model.handleCancel()
		}
		if key.Matches(message, backKey) {
			if model.canGoBack() {
				return model, model.back()
			}
			return model, model.handleCancel()
		}
	case tea.WindowSizeMsg:
		model.window = message
		model.hasWindow = true
		if model.form != nil {
			model.applyWindowSize(model.form)
		}
		return model, nil
	case clientsDiscoveredMsg:
		if message.request != model.activeRequest {
			return model, nil
		}
		if message.err != nil {
			return model, model.fail(message.err)
		}
		model.session.clients = message.clients
		return model, model.showClients()
	case catalogDiscoveredMsg:
		if message.request != model.activeRequest {
			return model, nil
		}
		model.session.catalog = message.catalog
		model.session.catalogErr = message.err
		if model.session.catalogErr == nil && len(message.catalog.Providers) == 0 {
			model.session.catalogErr = errors.New("model discovery returned no providers")
		}
		model.session.catalogs[message.client] = message.catalog
		model.session.catalogErrors[message.client] = model.session.catalogErr
		model.session.catalogLoaded[message.client] = true
		return model, model.showAgents()
	case savedModelDiscoveredMsg:
		if message.request != model.activeRequest {
			return model, nil
		}
		if message.err != nil {
			return model, model.fail(message.err)
		}
		model.session.originals[message.agent] = message.model
		return model, model.showProviderOrManual(model.session.originals[message.agent])
	}

	if model.form == nil {
		return model, nil
	}
	updatedForm, cmd := model.form.Update(msg)
	if f, ok := updatedForm.(*huh.Form); ok {
		model.form = f
	}
	if model.form.State == huh.StateCompleted {
		return model, model.submit()
	}
	return model, model.wrapFormCmd(cmd)
}

func (model *interactiveConfigureModel) View() tea.View {
	switch model.screen {
	case screenLoadingClients:
		return tea.NewView("Discovering AI clients...\n")
	case screenLoadingCatalog:
		return tea.NewView("Loading providers and models...\n")
	case screenLoadingSaved:
		return tea.NewView("Loading saved agent configuration...\n")
	default:
		if model.form != nil {
			if model.form.State == huh.StateCompleted && model.completedView != "" {
				return tea.NewView(model.completedView)
			}
			return tea.NewView(model.form.View())
		}
		return tea.NewView("")
	}
}

func (model *interactiveConfigureModel) submit() tea.Cmd {
	switch model.screen {
	case screenClient:
		model.session.client = model.value
		if model.session.catalogLoaded[model.session.client] {
			model.session.catalog = model.session.catalogs[model.session.client]
			model.session.catalogErr = model.session.catalogErrors[model.session.client]
			return model.showAgents()
		}
		requestContext, request := model.startRequest(screenLoadingCatalog)
		return model.discoverCatalog(requestContext, request, model.session.client)
	case screenAgent:
		if model.value == saveChoice {
			return model.showReview()
		}
		model.session.agent = model.value
		model.session.resetDraftsForAgent(model.session.agent)
		if _, loaded := model.session.originals[model.session.agent]; !loaded {
			requestContext, request := model.startRequest(screenLoadingSaved)
			return model.loadSavedModel(requestContext, request, model.session.agent)
		}
		return model.showProviderOrManual(model.session.originals[model.session.agent])
	case screenProvider:
		model.session.provider = model.value
		model.session.providerDraft = model.value
		model.session.models = model.session.catalog.ModelsByProvider[model.session.provider]
		return model.showModels()
	case screenModel:
		model.session.model = model.value
		model.session.modelDraft = model.value
		model.session.variants = model.variants[model.session.model]
		if len(model.session.variants) > 0 {
			return model.showVariants()
		}
		model.session.variant = ""
		model.session.variantDraft = ""
		model.session.variantModel = model.session.model
		model.commitDraftSelection(model.session.model, model.session.variant)
		return model.showAgents()
	case screenVariant:
		model.session.variant = model.value
		model.session.variantDraft = model.value
		model.session.variantModel = model.session.model
		model.commitDraftSelection(model.session.model, model.session.variant)
		return model.showAgents()
	case screenManualModel:
		model.session.manualModel = model.value
		model.session.modelDraft = model.value
		return model.showManualVariant()
	case screenManualVariant:
		model.session.manualVariant = model.value
		model.session.manualVariantModel = model.session.manualModel
		model.commitDraftSelection(model.session.manualModel, model.session.manualVariant)
		return model.showAgents()
	case screenReview:
		if model.confirmed {
			model.result = orderedSelections(model.session.pending)
			return tea.Quit
		}
		return model.showAgents()
	case screenDiscard:
		if model.confirmed {
			model.confirmed = false
			return tea.Quit
		}
		return model.showAgents()
	default:
		return tea.Quit
	}
}

func (model *interactiveConfigureModel) canGoBack() bool {
	switch model.screen {
	case screenClient:
		return false
	case screenAgent, screenProvider, screenModel, screenVariant, screenManualModel, screenManualVariant, screenReview, screenDiscard:
		return true
	default:
		return false
	}
}

func (model *interactiveConfigureModel) back() tea.Cmd {
	switch model.screen {
	case screenAgent:
		return model.showClients()
	case screenProvider:
		return model.showAgents()
	case screenModel:
		return model.showProviders()
	case screenVariant:
		return model.showModels()
	case screenManualModel:
		if model.manualBack == screenProvider {
			return model.showProviders()
		}
		return model.showAgents()
	case screenManualVariant:
		return model.showManualModel()
	case screenReview:
		return model.showAgents()
	case screenDiscard:
		return model.showAgents()
	default:
		return model.handleCancel()
	}
}

func (model *interactiveConfigureModel) handleCancel() tea.Cmd {
	if len(model.session.pending) == 0 || model.screen == screenDiscard {
		model.confirmed = false
		return tea.Quit
	}
	return model.showDiscard()
}

func (model *interactiveConfigureModel) fail(err error) tea.Cmd {
	model.err = err
	model.confirmed = false
	if model.cancelContext != nil {
		model.cancelContext()
	}
	return tea.Quit
}

func (model *interactiveConfigureModel) applyWindowSize(form *huh.Form) {
	if !model.hasWindow || form == nil {
		return
	}
	if model.formWindowScreen == model.screen {
		return
	}
	form.Update(model.window)
	model.formWindowScreen = model.screen
}

func (model *interactiveConfigureModel) commitDraftSelection(modelID, variant string) {
	provider, _, _ := strings.Cut(modelID, "/")
	stageSelection(model.session.pending, model.session.originals[model.session.agent], Selection{
		Client:   model.session.client,
		Agent:    model.session.agent,
		Provider: provider,
		Model:    modelID,
		Variant:  variant,
	})
}
