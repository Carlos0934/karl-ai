package opencode

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCommandDiscoveryReportsMissingOpenCodeBinary(t *testing.T) {
	discovery := &CommandDiscovery{Binary: "karl-ai-test-missing-opencode-binary"}
	if _, err := discovery.Providers(context.Background(), t.TempDir(), Client); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("Providers() error = %v, want missing OpenCode error", err)
	}
}

func TestCommandDiscoveryUsesProjectRootAndOpenCodeArguments(t *testing.T) {
	var calls []string
	discovery := &CommandDiscovery{
		runCommand: func(_ context.Context, root string, args ...string) ([]byte, error) {
			calls = append(calls, root+":"+strings.Join(args, " "))
			if len(args) == 1 {
				return []byte("provider/model\n"), nil
			}
			return []byte("provider/model\n{\"id\":\"model\",\"providerID\":\"provider\",\"variants\":{}}\n"), nil
		},
	}
	root := `C:\project`
	if _, err := discovery.Providers(context.Background(), root, Client); err != nil {
		t.Fatal(err)
	}
	if _, err := discovery.Models(context.Background(), root, Client, "provider"); err != nil {
		t.Fatal(err)
	}
	want := []string{root + ":models", root + ":models --verbose provider"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("command calls = %#v, want %#v", calls, want)
	}
}

func TestParseProvidersUsesOnlyDiscoveredModelReferences(t *testing.T) {
	output := []byte("provider-two/model-b\nprovider-one/model-a\nprovider-two/model-c\n")
	want := []Provider{{ID: "provider-one"}, {ID: "provider-two"}}
	if got := parseProviders(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("providers = %#v, want %#v", got, want)
	}
}

func TestParseModelsReadsVerboseMetadataAndVariants(t *testing.T) {
	output := []byte(`provider-one/model-two
{
  "id": "model-two",
  "providerID": "provider-one",
  "name": "Model two",
  "variants": {"slow": {}, "fast": {}}
}
provider-one/model-one
{
  "id": "model-one",
  "providerID": "provider-one",
  "variants": {}
}
`)
	want := []Model{
		{ID: "provider-one/model-one", Variants: []string{}},
		{ID: "provider-one/model-two", Name: "Model two", Variants: []string{"fast", "slow"}},
	}
	if got, err := parseModels(output, "provider-one"); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
}
