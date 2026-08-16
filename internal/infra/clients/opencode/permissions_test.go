package opencode_test

import (
	"testing"

	"github.com/carlos0934/karl-ai/internal/domain"
	"github.com/carlos0934/karl-ai/internal/infra/clients/opencode"
)

func TestRenderPermissionsYAML(t *testing.T) {
	t.Run("default orchestrator permissions", func(t *testing.T) {
		perms := opencode.DefaultPermissionsForAgent(domain.AgentOrchestrator)
		yaml := opencode.RenderPermissionsYAML(perms)

		expected := "permission:\n  task:\n    general: allow\n"
		if yaml != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, yaml)
		}
	})

	t.Run("global permission", func(t *testing.T) {
		deny := opencode.PermissionDeny
		perms := opencode.Permissions{
			Global: &deny,
		}
		yaml := opencode.RenderPermissionsYAML(perms)

		expected := "permission: deny\n"
		if yaml != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, yaml)
		}
	})

	t.Run("empty permissions", func(t *testing.T) {
		perms := opencode.Permissions{}
		yaml := opencode.RenderPermissionsYAML(perms)

		if yaml != "" {
			t.Errorf("expected empty string, got: %q", yaml)
		}
	})
}
