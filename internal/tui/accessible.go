package tui

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
	"github.com/carlos0934/karl-ai/internal/catalog"
	"github.com/carlos0934/karl-ai/internal/domain"
)

var errBack = errors.New("back")

type configurationStep uint8

const (
	clientStep configurationStep = iota
	agentStep
	providerStep
	modelStep
	variantStep
	saveStep
)

func runAccessibleConfigure(ctx context.Context, root string, input io.Reader, output io.Writer, discovery domain.Discovery, savedSource SavedModelSource) ([]Selection, error) {
	if discovery == nil {
		return nil, fmt.Errorf("model discovery is not configured")
	}
	session := newConfigurationSession()
	fmt.Fprintln(output, "Discovering AI clients...")
	var err error
	session.clients, err = discovery.Clients(ctx, root)
	if cancellation := contextCancellation(ctx, err); cancellation != nil {
		return nil, cancellation
	}
	if err != nil {
		return nil, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, cancellationError(err)
		}
		var stepErr error
		switch session.step {
		case clientStep:
			session.client, stepErr = selectClient(input, output, session.clients)
			if stepErr != nil {
				break
			}
			session.catalog = session.catalogs[session.client]
			session.catalogErr = session.catalogErrors[session.client]
			if !session.catalogLoaded[session.client] {
				fmt.Fprintln(output, "Loading providers and models...")
				session.catalog, session.catalogErr = discovery.Catalog(ctx, root, session.client)
				if cancellation := contextCancellation(ctx, session.catalogErr); cancellation != nil {
					return nil, cancellation
				}
				if session.catalogErr == nil && len(session.catalog.Providers) == 0 {
					session.catalogErr = errors.New("model discovery returned no providers")
				}
				session.catalogs[session.client] = session.catalog
				session.catalogErrors[session.client] = session.catalogErr
				session.catalogLoaded[session.client] = true
			}
			session.step = agentStep

		case agentStep:
			for _, agent := range catalog.Agents() {
				id := string(agent.ID)
				if _, loaded := session.originals[id]; !loaded {
					orig, _ := savedModel(ctx, savedSource, id)
					session.originals[id] = orig
				}
			}
			session.agent, stepErr = selectAgent(input, output, session.pending, session.originals)
			if stepErr != nil {
				break
			}
			if session.agent == backChoice {
				session.step = clientStep
				continue
			}
			if session.agent == saveChoice {
				session.step = saveStep
				continue
			}
			original, loaded := session.originals[session.agent]
			if !loaded {
				original, stepErr = savedModel(ctx, savedSource, session.agent)
				if stepErr != nil {
					break
				}
				session.originals[session.agent] = original
			}
			session.effective = original
			if staged, ok := session.pending[session.agent]; ok {
				session.effective = selectionConfig(staged)
			}
			if session.catalogErr != nil {
				session.model, stepErr = selectManualModel(input, output, session.agent, original, session.effective, session.catalogErr)
				if stepErr != nil {
					if errors.Is(stepErr, errBack) {
						continue
					}
					break
				}
				session.variant, stepErr = selectManualVariant(input, output, session.agent, session.model, original, session.effective, session.catalogErr)
				if stepErr != nil {
					if errors.Is(stepErr, errBack) {
						continue
					}
					break
				}
				stageSelection(session.pending, original, Selection{
					Client:   session.client,
					Agent:    session.agent,
					Provider: strings.Split(session.model, "/")[0],
					Model:    session.model,
					Variant:  session.variant,
				})
				continue
			}
			session.step = providerStep

		case providerStep:
			session.provider, stepErr = selectProvider(input, output, session.agent, session.catalog.Providers, session.effective, session.originals[session.agent])
			if stepErr != nil {
				if errors.Is(stepErr, errBack) {
					session.step = agentStep
					continue
				}
				break
			}
			session.models = session.catalog.ModelsByProvider[session.provider]
			session.step = modelStep

		case modelStep:
			var variants []string
			session.model, variants, stepErr = selectModel(input, output, session.agent, session.models, session.effective, session.originals[session.agent])
			if stepErr != nil {
				if errors.Is(stepErr, errBack) {
					session.step = providerStep
					continue
				}
				break
			}
			session.variants = variants
			if len(session.variants) > 0 {
				session.step = variantStep
			} else {
				stageSelection(session.pending, session.originals[session.agent], Selection{Client: session.client, Agent: session.agent, Provider: session.provider, Model: session.model})
				session.step = agentStep
			}

		case variantStep:
			initialVariant := ""
			if session.model == session.effective.Model {
				initialVariant = configVariant(session.effective)
			}
			session.variant, stepErr = selectVariant(input, output, session.agent, session.model, session.variants, initialVariant, session.effective, session.originals[session.agent])
			if stepErr != nil {
				if errors.Is(stepErr, errBack) {
					session.step = modelStep
					continue
				}
				break
			}
			stageSelection(session.pending, session.originals[session.agent], Selection{Client: session.client, Agent: session.agent, Provider: session.provider, Model: session.model, Variant: session.variant})
			session.step = agentStep

		case saveStep:
			confirmed, err := reviewSelections(input, output, session.pending, session.originals)
			if err != nil {
				stepErr = err
				break
			}
			if confirmed {
				return orderedSelections(session.pending), nil
			}
			session.step = agentStep
		}

		if stepErr != nil {
			if cancellation := contextCancellation(ctx, stepErr); cancellation != nil {
				return nil, cancellation
			}
			if errors.Is(stepErr, ErrCancelled) && len(session.pending) > 0 {
				discard, discardErr := confirmDiscard(input, output, len(session.pending))
				if discardErr != nil {
					return nil, discardErr
				}
				if !discard {
					continue
				}
			}
			return nil, stepErr
		}
	}
}

func selectClient(input io.Reader, output io.Writer, clients []domain.ClientInfo) (string, error) {
	options := make([]huh.Option[string], 0, len(clients))
	for _, client := range clients {
		label := client.Name
		if label == "" {
			label = client.ID
		}
		options = append(options, huh.NewOption(label, client.ID))
	}
	return selectValue(input, output, "Client", options)
}

func selectAgent(input io.Reader, output io.Writer, pending map[string]Selection, originals map[string]domain.AgentConfig) (string, error) {
	options, title, description := agentFormOptions(pending, originals)
	return selectValueWithBackDescription(input, output, title, description, options)
}

func selectProvider(input io.Reader, output io.Writer, agent string, providers []domain.Provider, effective, original domain.AgentConfig) (string, error) {
	options := make([]huh.Option[string], 0, len(providers))
	initial, _, _ := strings.Cut(effective.Model, "/")
	initialFound := false
	for _, provider := range providers {
		options = append(options, huh.NewOption(provider.ID, provider.ID))
		initialFound = initialFound || provider.ID == initial
	}
	if !initialFound {
		initial = ""
	}
	return selectValueWithBackInitialDescription(input, output, "Provider - "+agent, modelContextDescription(original, effective), options, initial)
}

func selectModel(input io.Reader, output io.Writer, agent string, models []domain.Model, effective, original domain.AgentConfig) (string, []string, error) {
	choices := orderedModelChoices(models, effective, original)
	options := make([]huh.Option[string], 0, len(choices))
	variants := make(map[string][]string, len(choices))
	initial := ""
	initialFound := false
	for _, choice := range choices {
		options = append(options, huh.NewOption(modelChoiceLabel(choice, effective, original), choice.model.ID))
		variants[choice.model.ID] = append(variants[choice.model.ID], choice.model.Variants...)
		if choice.model.ID == effective.Model {
			initial = choice.model.ID
			initialFound = true
		}
	}
	if !initialFound && len(options) > 0 {
		initial = options[0].Value
	}
	for id := range variants {
		variants[id] = orderedVariants(variants[id])
	}
	selected, err := selectValueWithBackInitialDescription(input, output, "Model - "+agent, modelContextDescription(original, effective), options, initial)
	return selected, variants[selected], err
}

func selectVariant(input io.Reader, output io.Writer, agent, model string, variants []string, initial string, effective, original domain.AgentConfig) (string, error) {
	options := []huh.Option[string]{huh.NewOption("No variant", "")}
	for _, variant := range orderedVariants(variants) {
		options = append(options, huh.NewOption(variant, variant))
	}
	description := variantContextDescription(model, original, effective)
	return selectValueWithBackInitialDescription(input, output, "Variant - "+agent, description, options, initial)
}

func selectManualModel(input io.Reader, output io.Writer, agent string, original, effective domain.AgentConfig, discoveryErr error) (string, error) {
	description := modelContextDescription(original, effective) + fmt.Sprintf(" | Discovery unavailable: %v", discoveryErr)
	return promptValueWithBack(input, output, "Model - "+agent, description, effective.Model, validateModelReference)
}

func selectManualVariant(input io.Reader, output io.Writer, agent, manualModel string, original, effective domain.AgentConfig, discoveryErr error) (string, error) {
	description := variantContextDescription(manualModel, original, effective) + fmt.Sprintf(" | Discovery unavailable: %v. Leave blank for no variant.", discoveryErr)
	initial := ""
	if manualModel == effective.Model {
		initial = configVariant(effective)
	}
	return promptValueWithBack(input, output, "Variant - "+agent, description, initial, validateVariant)
}

func reviewSelections(input io.Reader, output io.Writer, pending map[string]Selection, originals map[string]domain.AgentConfig) (bool, error) {
	var summary strings.Builder
	changes := orderedSelections(pending)
	fmt.Fprintf(&summary, "%s pending", changeCount(len(changes)))
	for _, change := range changes {
		original := originals[change.Agent]
		fmt.Fprintf(&summary, "\n  * %s: %s -> %s", change.Agent, formatModel(original.Model, configVariant(original)), formatModel(change.Model, change.Variant))
	}
	var confirmed bool
	field := huh.NewConfirm().
		Title("Review and apply changes").
		Description(summary.String()).
		Affirmative("Apply").
		Negative("Back").
		Value(&confirmed)
	form := newAccessibleForm(input, output, huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return false, cancellationError(err)
	}
	return confirmed, nil
}

func confirmDiscard(input io.Reader, output io.Writer, pendingCount int) (bool, error) {
	var confirmed bool
	field := huh.NewConfirm().
		Title("Discard unsaved changes?").
		Description(fmt.Sprintf("You have %s pending.", changeCount(len(orderedSelections(make(map[string]Selection, pendingCount)))))).
		Affirmative("Discard").
		Negative("Keep editing").
		Value(&confirmed)
	form := newAccessibleForm(input, output, huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return false, cancellationError(err)
	}
	return confirmed, nil
}

func selectValue(input io.Reader, output io.Writer, title string, options []huh.Option[string]) (string, error) {
	return selectValueWithBackInitialDescription(input, output, title, "", options, "")
}

func selectValueWithBackDescription(input io.Reader, output io.Writer, title, description string, options []huh.Option[string]) (string, error) {
	return selectValueWithBackInitialDescription(input, output, title, description, options, "")
}

func selectValueWithBackInitialDescription(input io.Reader, output io.Writer, title, description string, options []huh.Option[string], initial string) (string, error) {
	var selected string
	if initial != "" {
		selected = initial
	}
	field := huh.NewSelect[string]().
		Title(title).
		Description(description).
		Options(options...).
		Value(&selected)
	form := newAccessibleForm(input, output, huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return "", cancellationError(err)
	}
	return selected, nil
}

func promptValueWithBack(input io.Reader, output io.Writer, title, description, initial string, validator func(string) error) (string, error) {
	if promptText(output, title, description, initial) {
		line, err := readPromptLine(input)
		if err != nil {
			return "", err
		}
		if line == "" {
			line = initial
		}
		if err := validator(line); err != nil {
			fmt.Fprintf(output, "Invalid value: %v\n", err)
			return promptValueWithBack(input, output, title, description, initial, validator)
		}
		return line, nil
	}
	var value string = initial
	field := huh.NewInput().
		Title(title).
		Description(description).
		Value(&value).
		Validate(validator)
	form := newAccessibleForm(input, output, huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return "", cancellationError(err)
	}
	return value, nil
}

func promptText(output io.Writer, title, description, initial string) bool {
	if output == nil {
		return false
	}
	var buffer bytes.Buffer
	fmt.Fprintln(&buffer, title)
	if description != "" {
		fmt.Fprintln(&buffer, description)
	}
	if initial != "" {
		fmt.Fprintf(&buffer, "[default: %s]\n", initial)
	}
	fmt.Fprint(output, buffer.String())
	return true
}

func readPromptLine(input io.Reader) (string, error) {
	if input == nil {
		return "", ErrInputExhausted
	}
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", ErrInputExhausted
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func newAccessibleForm(input io.Reader, output io.Writer, group *huh.Group) *huh.Form {
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+c"))
	form := huh.NewForm(group).WithKeyMap(keymap).WithInput(input).WithOutput(output)
	form.WithAccessible(true)
	return form
}

func savedModel(ctx context.Context, source SavedModelSource, agent string) (domain.AgentConfig, error) {
	if source == nil {
		return domain.DefaultAgentConfig(domain.AgentID(agent)), nil
	}
	return source.ModelForAgent(ctx, agent)
}

func cancellationError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCancelled
	}
	return err
}

func contextCancellation(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return fmt.Errorf("%w: %v", ErrCancelled, ctx.Err())
	}
	return err
}

func terminalInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
