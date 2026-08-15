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
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/carlos0934/karl-ai/adapters/opencode"
	"github.com/carlos0934/karl-ai/core/catalog"
)

// ErrCancelled indicates that the developer left the model configuration flow
// before confirming a selection.
var ErrCancelled = errors.New("model configuration cancelled")

// ErrInputExhausted indicates that a nonterminal input stream ended before the
// developer completed the configuration flow.
var ErrInputExhausted = errors.New("model configuration input exhausted")

// Selection is the unpersisted result of one model configuration flow.
type Selection struct {
	Client    string `json:"client"`
	Agent     string `json:"agent"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Variant   string `json:"variant,omitempty"`
	Unchanged bool   `json:"-"`
}

// SavedModelSource provides the current model override for an agent.
type SavedModelSource interface {
	ModelForAgent(string) (opencode.AgentConfig, error)
}

// Run presents the client, agent, provider, model, optional variant, and
// confirmation steps. It does not read or write configuration.
func Run(ctx context.Context, root string, input io.Reader, output io.Writer, discovery opencode.Discovery) (Selection, error) {
	return RunWithSavedModel(ctx, root, input, output, discovery, nil)
}

// RunWithSavedModel presents the configuration flow with the current selected
// agent model available for stale-model detection.
func RunWithSavedModel(ctx context.Context, root string, input io.Reader, output io.Writer, discovery opencode.Discovery, saved SavedModelSource) (Selection, error) {
	interactive := terminalInput(input)
	if interactive {
		return runWithSavedModel(ctx, root, input, output, discovery, saved, true)
	}
	trackedInput := &cancellationReader{reader: &lineReader{reader: bufio.NewReader(input)}}
	return runWithSavedModel(ctx, root, trackedInput, output, discovery, saved, false)
}

func run(ctx context.Context, root string, input io.Reader, output io.Writer, discovery opencode.Discovery, interactive bool) (Selection, error) {
	return runWithSavedModel(ctx, root, input, output, discovery, nil, interactive)
}

func runWithSavedModel(ctx context.Context, root string, input io.Reader, output io.Writer, discovery opencode.Discovery, savedSource SavedModelSource, interactive bool) (Selection, error) {
	if discovery == nil {
		return Selection{}, fmt.Errorf("model discovery is not configured")
	}
	if err := ctx.Err(); err != nil {
		return Selection{}, fmt.Errorf("%w: %v", ErrCancelled, err)
	}

	clients, err := discovery.Clients(ctx, root)
	if err != nil {
		return Selection{}, fmt.Errorf("discover clients: %w", err)
	}
	client, err := selectClient(input, output, clients, interactive)
	if err != nil {
		return Selection{}, err
	}

	agent, err := selectAgent(input, output, interactive)
	if err != nil {
		return Selection{}, err
	}
	saved, err := savedModel(savedSource, agent)
	if err != nil {
		return Selection{}, err
	}
	providers, err := discovery.Providers(ctx, root, client)
	if err != nil {
		return manualSelection(input, output, client, agent, saved, fmt.Errorf("discover providers: %w", err), interactive)
	}
	if len(providers) == 0 {
		return manualSelection(input, output, client, agent, saved, errors.New("provider discovery returned no choices"), interactive)
	}
	provider, err := selectProvider(input, output, providers, interactive)
	if err != nil {
		return Selection{}, err
	}

	models, err := discovery.Models(ctx, root, client, provider)
	if err != nil {
		return manualSelection(input, output, client, agent, saved, fmt.Errorf("discover models: %w", err), interactive)
	}
	if len(models) == 0 {
		return manualSelection(input, output, client, agent, saved, fmt.Errorf("model discovery returned no choices for provider %q", provider), interactive)
	}
	model, variants, err := selectModel(input, output, models, saved, interactive)
	if err != nil {
		return Selection{}, err
	}

	variant := ""
	if len(variants) > 0 {
		variant, err = selectVariant(input, output, variants, interactive)
		if err != nil {
			return Selection{}, err
		}
	}

	return confirmSelection(input, output, Selection{Client: client, Agent: agent, Provider: provider, Model: model, Variant: variant}, saved, interactive)
}

func selectClient(input io.Reader, output io.Writer, clients []opencode.ClientInfo, interactive bool) (string, error) {
	options := make([]huh.Option[string], 0, len(clients))
	for _, client := range clients {
		label := client.Name
		if label == "" {
			label = client.ID
		}
		options = append(options, huh.NewOption(label, client.ID))
	}
	return selectValue(input, output, "Client", options, interactive)
}

func selectAgent(input io.Reader, output io.Writer, interactive bool) (string, error) {
	agents := catalog.Agents()
	options := make([]huh.Option[string], 0, len(agents))
	for _, agent := range agents {
		options = append(options, huh.NewOption(string(agent.ID), string(agent.ID)))
	}
	return selectValue(input, output, "Agent", options, interactive)
}

func selectProvider(input io.Reader, output io.Writer, providers []opencode.Provider, interactive bool) (string, error) {
	options := make([]huh.Option[string], 0, len(providers))
	for _, provider := range providers {
		options = append(options, huh.NewOption(provider.ID, provider.ID))
	}
	return selectValue(input, output, "Provider", options, interactive)
}

func selectModel(input io.Reader, output io.Writer, models []opencode.Model, saved opencode.AgentConfig, interactive bool) (string, []string, error) {
	options := make([]huh.Option[string], 0, len(models))
	variants := make(map[string][]string, len(models))
	foundSaved := false
	for _, model := range models {
		if model.ID == saved.Model {
			foundSaved = true
			break
		}
	}
	if saved.Model != "" && !foundSaved {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (configured, stale)", saved.Model), saved.Model))
		if saved.Variant != nil && *saved.Variant != "" {
			variants[saved.Model] = []string{*saved.Variant}
		}
	}
	for _, model := range models {
		label := model.ID
		if model.Name != "" {
			label = fmt.Sprintf("%s (%s)", model.ID, model.Name)
		}
		options = append(options, huh.NewOption(label, model.ID))
		variants[model.ID] = append([]string(nil), model.Variants...)
	}
	selected, err := selectValue(input, output, "Model", options, interactive)
	return selected, variants[selected], err
}

func selectVariant(input io.Reader, output io.Writer, variants []string, interactive bool) (string, error) {
	options := []huh.Option[string]{huh.NewOption("No variant", "")}
	for _, variant := range variants {
		options = append(options, huh.NewOption(variant, variant))
	}
	return selectValue(input, output, "Variant (optional)", options, interactive)
}

func manualSelection(input io.Reader, output io.Writer, client, agent string, saved opencode.AgentConfig, discoveryErr error, interactive bool) (Selection, error) {
	fmt.Fprintf(output, "Model discovery is unavailable: %v. Enter a provider/model reference manually.\n", discoveryErr)
	model, err := manualModelReference(input, output, "", interactive)
	if err != nil {
		return Selection{}, err
	}
	variant, err := manualVariant(input, output, "", interactive)
	if err != nil {
		return Selection{}, err
	}
	provider, _, _ := strings.Cut(model, "/")
	return confirmSelection(input, output, Selection{Client: client, Agent: agent, Provider: provider, Model: model, Variant: variant}, saved, interactive)
}

func manualModelReference(input io.Reader, output io.Writer, initial string, interactive bool) (string, error) {
	return inputValue(input, output, "Model reference", "Enter provider/model. Enter the variant separately.", initial, validateModelReference, interactive)
}

func manualVariant(input io.Reader, output io.Writer, initial string, interactive bool) (string, error) {
	return inputValue(input, output, "Variant (optional)", "Leave blank when the model has no variant.", initial, validateVariant, interactive)
}

func inputValue(input io.Reader, output io.Writer, title, description, initial string, validate func(string) error, interactive bool) (string, error) {
	value := initial
	field := huh.NewInput().Title(title).Description(description).Value(&value).Validate(validate)
	form := newForm(input, output, interactive, huh.NewGroup(field))
	err := form.Run()
	if err != nil {
		return "", cancellationError(err)
	}
	value = strings.TrimSpace(value)
	if err := validate(value); err != nil {
		return "", err
	}
	if err := inputTermination(input); err != nil {
		return "", err
	}
	return value, nil
}

func validateModelReference(value string) error {
	value = strings.TrimSpace(value)
	provider, model, ok := strings.Cut(value, "/")
	if !ok || provider == "" || model == "" || strings.ContainsAny(value, " \t\r\n#") {
		return errors.New("model reference must use provider/model and must not include a variant")
	}
	return nil
}

func validateVariant(value string) error {
	if strings.ContainsAny(strings.TrimSpace(value), " \t\r\n#") {
		return errors.New("variant must not contain spaces or #")
	}
	return nil
}

func savedModel(source SavedModelSource, agent string) (opencode.AgentConfig, error) {
	if source == nil {
		return opencode.AgentConfig{}, nil
	}
	saved, err := source.ModelForAgent(agent)
	if err != nil {
		return opencode.AgentConfig{}, fmt.Errorf("read saved model for %s: %w", agent, err)
	}
	return saved, nil
}

func confirmSelection(input io.Reader, output io.Writer, selection Selection, saved opencode.AgentConfig, interactive bool) (Selection, error) {
	confirmed, err := confirm(input, output, selection, interactive)
	if err != nil {
		return Selection{}, err
	}
	if !confirmed {
		return Selection{}, ErrCancelled
	}
	selection.Unchanged = saved.Model != "" && selection.Model == saved.Model && saved.Variant != nil && selection.Variant == *saved.Variant
	return selection, nil
}

func confirm(input io.Reader, output io.Writer, selection Selection, interactive bool) (bool, error) {
	var confirmed bool
	variant := selection.Variant
	if variant == "" {
		variant = "no variant"
	}
	field := huh.NewConfirm().
		Title(fmt.Sprintf("Use %s from %s with %s for %s on %s?", selection.Model, selection.Provider, variant, selection.Agent, selection.Client)).
		Affirmative("Confirm").
		Negative("Cancel").
		Value(&confirmed)
	form := newForm(input, output, interactive, huh.NewGroup(field))
	err := form.Run()
	if err := inputTermination(input); err != nil {
		return false, err
	}
	if err != nil {
		return false, cancellationError(err)
	}
	return confirmed, nil
}

func selectValue(input io.Reader, output io.Writer, title string, options []huh.Option[string], interactive bool) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no choices available for %s", title)
	}
	var selected string
	field := huh.NewSelect[string]().Title(title).Options(options...).Value(&selected)
	restoreEOF := useSelectEOFDefault(input)
	defer restoreEOF()
	form := newForm(input, output, interactive, huh.NewGroup(field))
	err := form.Run()
	if err := inputTermination(input); err != nil {
		return "", err
	}
	if err != nil {
		return "", cancellationError(err)
	}
	return selected, nil
}

func newForm(input io.Reader, output io.Writer, interactive bool, group *huh.Group) *huh.Form {
	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	form := huh.NewForm(group).WithKeyMap(keymap).WithInput(input).WithOutput(output)
	if !interactive {
		form.WithAccessible(true)
	}
	return form
}

func terminalInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func cancellationError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, tea.ErrInterrupted) {
		return ErrCancelled
	}
	return err
}

type cancellationReader struct {
	reader           io.Reader
	cancelled        bool
	exhausted        bool
	selectEOFDefault bool
}

func (reader *cancellationReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	if n == 0 && errors.Is(err, io.EOF) {
		reader.exhausted = true
		if reader.selectEOFDefault {
			return copy(buffer, []byte("1\n")), nil
		}
		return n, err
	}
	if bytes.Contains(buffer[:n], []byte{'\x03'}) || bytes.Contains(buffer[:n], []byte{'\x1b'}) {
		reader.cancelled = true
		return copy(buffer, []byte("1\n")), nil
	}
	return n, err
}

func inputTermination(input io.Reader) error {
	reader, ok := input.(*cancellationReader)
	if !ok {
		return nil
	}
	if reader.exhausted {
		return ErrInputExhausted
	}
	if reader.cancelled {
		return ErrCancelled
	}
	return nil
}

func useSelectEOFDefault(input io.Reader) func() {
	reader, ok := input.(*cancellationReader)
	if !ok {
		return func() {}
	}
	reader.selectEOFDefault = true
	return func() { reader.selectEOFDefault = false }
}

type lineReader struct {
	reader  *bufio.Reader
	pending []byte
}

func (reader *lineReader) Read(buffer []byte) (int, error) {
	if len(reader.pending) > 0 {
		n := copy(buffer, reader.pending)
		reader.pending = reader.pending[n:]
		return n, nil
	}
	line, err := reader.reader.ReadString('\n')
	if len(line) == 0 {
		return 0, err
	}
	if len(line) > len(buffer) {
		copyCount := copy(buffer, line)
		reader.pending = []byte(line[copyCount:])
		return copyCount, nil
	}
	return copy(buffer, line), nil
}
