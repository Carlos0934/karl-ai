package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/carlos0934/karl-ai/internal/catalog"
	"github.com/carlos0934/karl-ai/internal/domain"
)

const (
	backChoice      = "\x00karl-back"
	saveChoice      = "\x00karl-save"
	noVariantChoice = "\x00karl-no-variant"
)

func agentFormOptions(pending map[string]Selection, originals map[string]domain.AgentConfig) ([]huh.Option[string], string, string) {
	agents := catalog.Agents()
	options := make([]huh.Option[string], 0, len(agents)+1)
	for _, agent := range agents {
		id := string(agent.ID)
		var label string
		if selection, ok := pending[id]; ok {
			label = fmt.Sprintf("%s (pending: %s)", id, formatModel(selection.Model, selection.Variant))
		} else if orig, ok := originals[id]; ok && orig.Model != "" {
			label = fmt.Sprintf("%s (current: %s)", id, formatModel(orig.Model, configVariant(orig)))
		} else {
			def := domain.DefaultAgentConfig(agent.ID)
			label = fmt.Sprintf("%s (default: %s)", id, formatModel(def.Model, configVariant(def)))
		}
		options = append(options, huh.NewOption(label, id))
	}
	title := "Edit agents"
	description := "Choose an agent to edit."
	if len(pending) > 0 {
		title = fmt.Sprintf("Edit agents - %d pending", len(pending))
		description = "Choose an agent to edit, or review the pending changes."
		options = append(options, huh.NewOption(fmt.Sprintf("[ Review and save (%s) ]", changeCount(len(pending))), saveChoice))
	}
	return options, title, description
}

func (model *interactiveConfigureModel) showClients() tea.Cmd {
	options := make([]huh.Option[string], 0, len(model.session.clients))
	for _, client := range model.session.clients {
		label := client.Name
		if label == "" {
			label = client.ID
		}
		options = append(options, huh.NewOption(label, client.ID))
	}
	initial := model.session.client
	if initial == "" && len(options) > 0 {
		initial = options[0].Value
	}
	return model.setSelect(screenClient, "Client", "", options, initial, false)
}

func (model *interactiveConfigureModel) showAgents() tea.Cmd {
	for _, agent := range catalog.Agents() {
		id := string(agent.ID)
		if _, loaded := model.session.originals[id]; !loaded {
			if model.savedSource != nil {
				if saved, err := model.savedSource.ModelForAgent(model.ctx, id); err == nil {
					model.session.originals[id] = saved
				} else {
					model.session.originals[id] = domain.DefaultAgentConfig(agent.ID)
				}
			} else {
				model.session.originals[id] = domain.DefaultAgentConfig(agent.ID)
			}
		}
	}
	options, title, description := agentFormOptions(model.session.pending, model.session.originals)
	initial := model.session.agent
	if initial == "" && len(options) > 0 {
		initial = options[0].Value
	}
	return model.setSelect(screenAgent, title, description, options, initial, true)
}

func (model *interactiveConfigureModel) showProviderOrManual(original domain.AgentConfig) tea.Cmd {
	model.session.effective = original
	if staged, ok := model.session.pending[model.session.agent]; ok {
		model.session.effective = selectionConfig(staged)
	}
	if model.session.catalogErr != nil {
		model.manualErr = model.session.catalogErr
		model.manualBack = screenAgent
		return model.showManualModel()
	}
	return model.showProviders()
}

func (model *interactiveConfigureModel) showProviders() tea.Cmd {
	providers := model.session.catalog.Providers
	options := make([]huh.Option[string], 0, len(providers))
	initial, _, _ := strings.Cut(model.session.effective.Model, "/")
	initialFound := false
	for _, provider := range providers {
		options = append(options, huh.NewOption(provider.ID, provider.ID))
		initialFound = initialFound || provider.ID == model.session.providerDraft || (model.session.providerDraft == "" && provider.ID == initial)
	}
	if model.session.providerDraft != "" {
		initial = model.session.providerDraft
	} else if !initialFound && len(options) > 0 {
		initial = options[0].Value
	}
	return model.setSelect(screenProvider, "Provider - "+model.session.agent, modelContextDescription(model.session.originals[model.session.agent], model.session.effective), options, initial, true)
}

func (model *interactiveConfigureModel) showModels() tea.Cmd {
	choices := orderedModelChoices(model.session.models, model.session.effective, model.session.originals[model.session.agent])
	options := make([]huh.Option[string], 0, len(choices))
	model.variants = make(map[string][]string, len(choices))
	initial := ""
	initialFound := false
	for _, choice := range choices {
		options = append(options, huh.NewOption(modelChoiceLabel(choice, model.session.effective, model.session.originals[model.session.agent]), choice.model.ID))
		model.variants[choice.model.ID] = append(model.variants[choice.model.ID], choice.model.Variants...)
		if choice.model.ID == model.session.effective.Model {
			initial = choice.model.ID
			initialFound = true
		}
	}
	if model.session.modelDraft != "" {
		for _, choice := range choices {
			if choice.model.ID == model.session.modelDraft {
				initial = model.session.modelDraft
				initialFound = true
				break
			}
		}
	}
	if !initialFound && len(options) > 0 {
		initial = options[0].Value
	}
	for id := range model.variants {
		model.variants[id] = orderedVariants(model.variants[id])
	}
	return model.setSelect(screenModel, "Model - "+model.session.agent, modelContextDescription(model.session.originals[model.session.agent], model.session.effective), options, initial, true)
}

func (model *interactiveConfigureModel) showVariants() tea.Cmd {
	options := []huh.Option[string]{huh.NewOption("No variant", "")}
	for _, variant := range orderedVariants(model.session.variants) {
		options = append(options, huh.NewOption(variant, variant))
	}
	initial := ""
	if model.session.variantModel == model.session.model && contains(append([]string{""}, model.session.variants...), model.session.variantDraft) {
		initial = model.session.variantDraft
	} else if model.session.model == model.session.effective.Model {
		initial = configVariant(model.session.effective)
	}
	return model.setSelect(screenVariant, "Variant - "+model.session.agent, variantContextDescription(model.session.model, model.session.originals[model.session.agent], model.session.effective), options, initial, true)
}

func (model *interactiveConfigureModel) showManualModel() tea.Cmd {
	description := modelContextDescription(model.session.originals[model.session.agent], model.session.effective) + fmt.Sprintf(" | Discovery unavailable: %v", model.manualErr)
	initial := model.session.effective.Model
	if model.session.manualModel != "" {
		initial = model.session.manualModel
	}
	return model.setInput(screenManualModel, "Model - "+model.session.agent, description, initial, validateModelReference)
}

func (model *interactiveConfigureModel) showManualVariant() tea.Cmd {
	description := variantContextDescription(model.session.manualModel, model.session.originals[model.session.agent], model.session.effective) + fmt.Sprintf(" | Discovery unavailable: %v. Leave blank for no variant.", model.manualErr)
	initial := ""
	if model.session.manualVariantModel == model.session.manualModel {
		initial = model.session.manualVariant
	} else if model.session.manualModel == model.session.effective.Model {
		initial = configVariant(model.session.effective)
	}
	return model.setInput(screenManualVariant, "Variant - "+model.session.agent, description, initial, validateVariant)
}

func (model *interactiveConfigureModel) showReview() tea.Cmd {
	var summary strings.Builder
	changes := orderedSelections(model.session.pending)
	fmt.Fprintf(&summary, "%s pending", changeCount(len(changes)))
	for _, change := range changes {
		original := model.session.originals[change.Agent]
		fmt.Fprintf(&summary, "\n  * %s: %s -> %s", change.Agent, formatModel(original.Model, configVariant(original)), formatModel(change.Model, change.Variant))
	}
	return model.setConfirm(screenReview, "Review and apply changes", summary.String(), "Apply", "Back", true)
}

func (model *interactiveConfigureModel) showDiscard() tea.Cmd {
	return model.setConfirm(screenDiscard, "Discard unsaved changes?", fmt.Sprintf("You have %s pending.", changeCount(len(model.session.pending))), "Discard", "Keep editing", false)
}

func (model *interactiveConfigureModel) setSelect(screen interactiveScreen, title, description string, options []huh.Option[string], initial string, backEnabled bool) tea.Cmd {
	model.screen = screen
	model.backEnabled = backEnabled
	model.value = initial
	field := huh.NewSelect[string]().Title(title).Description(description).Options(options...).Value(&model.value)
	model.field = field
	return model.createForm(huh.NewGroup(field))
}

func (model *interactiveConfigureModel) setInput(screen interactiveScreen, title, description, initial string, validator func(string) error) tea.Cmd {
	model.screen = screen
	model.backEnabled = true
	model.value = initial
	field := huh.NewInput().Title(title).Description(description).Validate(validator).Value(&model.value)
	model.field = field
	return model.createForm(huh.NewGroup(field))
}

func (model *interactiveConfigureModel) setConfirm(screen interactiveScreen, title, description, affirmative, negative string, defaultConfirmed bool) tea.Cmd {
	model.screen = screen
	model.backEnabled = false
	model.confirmed = defaultConfirmed
	field := huh.NewConfirm().Title(title).Description(description).Affirmative(affirmative).Negative(negative).Value(&model.confirmed)
	model.field = field
	return model.createForm(huh.NewGroup(field))
}

func (model *interactiveConfigureModel) createForm(group *huh.Group) tea.Cmd {
	model.formGeneration++
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+c"))
	model.form = huh.NewForm(group).WithKeyMap(keymap)
	model.applyWindowSize(model.form)
	return model.wrapFormCmd(model.form.Init())
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
