package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/carlos0934/karl-ai/adapters/opencode"
)

type fakeDiscovery struct {
	calls        []string
	clients      []opencode.ClientInfo
	providers    []opencode.Provider
	models       []opencode.Model
	clientsErr   error
	providersErr error
	modelsErr    error
}

type fakeSavedModelSource struct {
	calls []string
	saved opencode.AgentConfig
	err   error
}

type oneLineReader struct {
	lines []string
}

func (reader *oneLineReader) Read(buffer []byte) (int, error) {
	if len(reader.lines) == 0 {
		return 0, io.EOF
	}
	line := reader.lines[0]
	reader.lines = reader.lines[1:]
	return copy(buffer, line+"\n"), nil
}

func (fake *fakeDiscovery) Clients(context.Context, string) ([]opencode.ClientInfo, error) {
	fake.calls = append(fake.calls, "clients")
	return fake.clients, fake.clientsErr
}

func (fake *fakeDiscovery) Providers(_ context.Context, _, client string) ([]opencode.Provider, error) {
	fake.calls = append(fake.calls, "providers:"+client)
	return fake.providers, fake.providersErr
}

func (fake *fakeDiscovery) Models(_ context.Context, _, client, provider string) ([]opencode.Model, error) {
	fake.calls = append(fake.calls, "models:"+client+":"+provider)
	return fake.models, fake.modelsErr
}

func (fake *fakeSavedModelSource) ModelForAgent(agent string) (opencode.AgentConfig, error) {
	fake.calls = append(fake.calls, agent)
	return fake.saved, fake.err
}

func TestRunUsesClientAgentProviderModelVariantOrder(t *testing.T) {
	discovery := &fakeDiscovery{
		clients:   []opencode.ClientInfo{{ID: "test-client", Name: "Test client"}},
		providers: []opencode.Provider{{ID: "provider-one"}, {ID: "provider-two"}},
		models:    []opencode.Model{{ID: "provider-one/model-one", Name: "Model one", Variants: []string{"fast"}}, {ID: "provider-one/model-two"}},
	}
	input := &oneLineReader{lines: []string{"1", "1", "1", "1", "2", "y"}}
	var output strings.Builder

	selection, err := run(context.Background(), t.TempDir(), input, &output, discovery, false)
	if err != nil {
		t.Fatal(err)
	}

	want := Selection{Client: "test-client", Agent: "karl-orchestrator", Provider: "provider-one", Model: "provider-one/model-one", Variant: "fast"}
	if selection != want {
		t.Fatalf("selection = %#v, want %#v", selection, want)
	}
	if got, wantCalls := strings.Join(discovery.calls, ","), "clients,providers:test-client,models:test-client:provider-one"; got != wantCalls {
		t.Fatalf("discovery calls = %q, want %q", got, wantCalls)
	}
	for _, discovered := range []string{"provider-one", "provider-two", "provider-one/model-one", "provider-one/model-two", "fast"} {
		if !strings.Contains(output.String(), discovered) {
			t.Fatalf("TUI output does not show discovered value %q", discovered)
		}
	}
}

func TestRunEscCancelsWithoutAResult(t *testing.T) {
	discovery := &fakeDiscovery{clients: []opencode.ClientInfo{{ID: "test-client"}}}
	selection, err := Run(context.Background(), t.TempDir(), strings.NewReader("\x1b"), io.Discard, discovery)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("Run() error = %v, want ErrCancelled", err)
	}
	if selection != (Selection{}) {
		t.Fatalf("selection after cancellation = %#v", selection)
	}
	if got, want := strings.Join(discovery.calls, ","), "clients"; got != want {
		t.Fatalf("discovery calls after cancellation = %q, want %q", got, want)
	}
}

func TestRunCtrlCCancelsWithoutAResult(t *testing.T) {
	discovery := &fakeDiscovery{clients: []opencode.ClientInfo{{ID: "test-client"}}}
	selection, err := Run(context.Background(), t.TempDir(), strings.NewReader("\x03"), io.Discard, discovery)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("Run() error = %v, want ErrCancelled", err)
	}
	if selection != (Selection{}) {
		t.Fatalf("selection after cancellation = %#v", selection)
	}
}

func TestRunReturnsInputExhaustedForEmptyNonterminalInput(t *testing.T) {
	discovery := &fakeDiscovery{clients: []opencode.ClientInfo{{ID: "test-client"}}}
	selection, err := Run(context.Background(), t.TempDir(), strings.NewReader(""), io.Discard, discovery)
	if !errors.Is(err, ErrInputExhausted) {
		t.Fatalf("Run() error = %v, want ErrInputExhausted", err)
	}
	if selection != (Selection{}) {
		t.Fatalf("selection after exhausted input = %#v", selection)
	}
}

func TestRunReturnsInputExhaustedAfterInvalidVariantResponse(t *testing.T) {
	discovery := &fakeDiscovery{
		clients:   []opencode.ClientInfo{{ID: "test-client"}},
		providers: []opencode.Provider{{ID: "provider"}},
		models:    []opencode.Model{{ID: "provider/model", Variants: []string{"fast"}}},
	}
	input := strings.NewReader("1\n1\n1\n1\ninvalid\n")
	selection, err := Run(context.Background(), t.TempDir(), input, io.Discard, discovery)
	if !errors.Is(err, ErrInputExhausted) {
		t.Fatalf("Run() error = %v, want ErrInputExhausted", err)
	}
	if selection != (Selection{}) {
		t.Fatalf("selection after exhausted variant input = %#v", selection)
	}
}

func TestRunFallsBackToManualEntryWhenProviderDiscoveryIsEmpty(t *testing.T) {
	discovery := &fakeDiscovery{clients: []opencode.ClientInfo{{ID: "test-client"}}}
	input := &oneLineReader{lines: []string{"1", "1", "invalid"}}
	var output strings.Builder
	selection, err := Run(context.Background(), t.TempDir(), input, &output, discovery)
	if err == nil || !strings.Contains(err.Error(), "model reference must use provider/model") {
		t.Fatalf("Run() error = %v", err)
	}
	if selection != (Selection{}) {
		t.Fatalf("selection after empty provider list = %#v", selection)
	}
	if !strings.Contains(output.String(), "provider discovery returned no choices") || !strings.Contains(output.String(), "Enter a provider/model reference manually") {
		t.Fatalf("manual fallback output = %q", output.String())
	}
}

func TestRunFallsBackToManualEntryWhenOpenCodeIsMissing(t *testing.T) {
	discovery := &fakeDiscovery{
		clients:      []opencode.ClientInfo{{ID: "opencode", Name: "OpenCode"}},
		providersErr: opencode.ErrClientNotFound,
	}
	input := &oneLineReader{lines: []string{"1", "1", "manual-provider/manual-model", "", "y"}}
	var output strings.Builder
	selection, err := Run(context.Background(), t.TempDir(), input, &output, discovery)
	if err != nil {
		t.Fatal(err)
	}
	want := Selection{Client: "opencode", Agent: "karl-orchestrator", Provider: "manual-provider", Model: "manual-provider/manual-model"}
	if selection != want {
		t.Fatalf("selection = %#v, want %#v", selection, want)
	}
	if !strings.Contains(output.String(), "OpenCode executable was not found") || !strings.Contains(output.String(), "Enter a provider/model reference manually") {
		t.Fatalf("manual fallback output = %q", output.String())
	}
}

func TestRunFallsBackToManualEntryAfterDiscoveryTimeout(t *testing.T) {
	discovery := &fakeDiscovery{
		clients:   []opencode.ClientInfo{{ID: "opencode", Name: "OpenCode"}},
		providers: []opencode.Provider{{ID: "provider"}},
		modelsErr: context.DeadlineExceeded,
	}
	input := &oneLineReader{lines: []string{"1", "1", "1", "manual-provider/manual-model", "", "y"}}
	var output strings.Builder
	selection, err := Run(context.Background(), t.TempDir(), input, &output, discovery)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Model != "manual-provider/manual-model" {
		t.Fatalf("selection = %#v", selection)
	}
	if !strings.Contains(output.String(), context.DeadlineExceeded.Error()) || !strings.Contains(output.String(), "Enter a provider/model reference manually") {
		t.Fatalf("manual fallback output = %q", output.String())
	}
}

func TestRunMarksStaleModelAndSupportsKeepingOrReplacingIt(t *testing.T) {
	discovery := &fakeDiscovery{
		clients:   []opencode.ClientInfo{{ID: "opencode", Name: "OpenCode"}},
		providers: []opencode.Provider{{ID: "provider"}},
		models:    []opencode.Model{{ID: "provider/replacement"}},
	}
	saved := &fakeSavedModelSource{saved: opencode.AgentConfig{Model: "old-provider/old-model", Variant: stringPointer("legacy")}}

	keepInput := &oneLineReader{lines: []string{"1", "1", "1", "1", "2", "y"}}
	var keepOutput strings.Builder
	kept, err := RunWithSavedModel(context.Background(), t.TempDir(), keepInput, &keepOutput, discovery, saved)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Model != "old-provider/old-model" || kept.Variant != "legacy" || !kept.Unchanged {
		t.Fatalf("kept selection = %#v", kept)
	}
	if !strings.Contains(keepOutput.String(), "old-provider/old-model (configured, stale)") {
		t.Fatalf("stale model was not marked: %q", keepOutput.String())
	}

	replaceInput := &oneLineReader{lines: []string{"1", "1", "1", "2", "y"}}
	replaced, err := RunWithSavedModel(context.Background(), t.TempDir(), replaceInput, io.Discard, discovery, saved)
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Model != "provider/replacement" || replaced.Unchanged {
		t.Fatalf("replacement selection = %#v", replaced)
	}
}

func TestManualModelReferenceRejectsInvalidVariantReference(t *testing.T) {
	if err := validateModelReference("provider/model#variant"); err == nil {
		t.Fatal("variant reference was accepted as a model reference")
	}
	if err := validateModelReference("provider-only"); err == nil {
		t.Fatal("reference without a provider separator was accepted")
	}
}

func stringPointer(value string) *string {
	return &value
}
