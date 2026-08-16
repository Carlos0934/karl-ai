package tui

import (
	"context"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/carlos0934/karl-ai/internal/domain"
)

// ErrCancelled indicates that the developer left the model configuration flow
// before confirming a selection.
var ErrCancelled = domain.ErrCancelled

// ErrInputExhausted indicates that a nonterminal input stream ended before the
// developer completed the configuration flow.
var ErrInputExhausted = domain.ErrInputExhausted

// Run launches the model configuration flow using default model values.
func Run(ctx context.Context, root string, input io.Reader, output io.Writer, discovery domain.Discovery) ([]Selection, error) {
	return RunWithSavedModel(ctx, root, input, output, discovery, nil)
}

// RunWithSavedModel launches the model configuration flow using existing model selections.
func RunWithSavedModel(ctx context.Context, root string, input io.Reader, output io.Writer, discovery domain.Discovery, saved SavedModelSource) ([]Selection, error) {
	if terminalInput(input) {
		return runInteractiveConfigure(ctx, root, input, output, discovery, saved)
	}
	return runAccessibleConfigure(ctx, root, input, output, discovery, saved)
}

func runInteractiveConfigure(ctx context.Context, root string, input io.Reader, output io.Writer, discovery domain.Discovery, saved SavedModelSource) ([]Selection, error) {
	if discovery == nil {
		return nil, fmt.Errorf("model discovery is not configured")
	}
	model := newInteractiveConfigureModel(ctx, root, discovery, saved)
	options := []tea.ProgramOption{tea.WithContext(model.ctx)}
	if input != nil {
		options = append(options, tea.WithInput(input))
	}
	if output != nil {
		options = append(options, tea.WithOutput(output))
	}
	program := tea.NewProgram(model, options...)
	finalModel, err := program.Run()
	if err != nil {
		return nil, cancellationError(err)
	}
	interactiveModel, ok := finalModel.(*interactiveConfigureModel)
	if !ok {
		return nil, ErrCancelled
	}
	if interactiveModel.err != nil {
		return nil, interactiveModel.err
	}
	if interactiveModel.confirmed {
		return interactiveModel.result, nil
	}
	return nil, ErrCancelled
}
