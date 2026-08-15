---
name: configure-models-tui
state: archived
created: 2026-08-15
assurance_level: L2
depends_on: []
conflicts_with: []
blocked_by: []
---

# Configure Models Tui

## Outcome

A developer runs one interactive command to view and change the model and
variant that a Karl agent uses. The command discovers the client, the
providers, and the available models from the installed client at run time. It
writes the chosen value to `.karl-ai/config.json` for the selected agent and
re-syncs the projection so the change takes effect. The developer does not
edit JSON by hand. All current noninteractive commands keep their behavior,
output, and exit codes.

## Scope

### Included

- A new `karl-ai models configure` command that walks the developer through
  client, agent, provider, model, and optional variant selection for one agent
  at a time.
- Live discovery of providers and models from the selected client. No
  provider or model name is hard-coded in the TUI.
- Writing the selection to `.karl-ai/config.json` for the selected agent with
  an atomic write and re-syncing the OpenCode projection in a drift-safe way.
- Keeping the variant as the separate `variant` field; the `#variant`
  reference form is not used.
- Defined behavior for a missing client binary, a discovery failure, a user
  cancellation, an invalid or stale saved model, an offline or stale cache,
  and a persistence failure.

### Excluded

- Changes to current default model values.
- Changes to noninteractive command behavior, output, or exit codes.
- Support for clients other than OpenCode in this change. The boundary stays
  client-neutral, but only OpenCode is wired.
- A hard-coded provider or model catalog.
- Edits to RESEARCH.md or to other active changes.

## Acceptance Criteria

- The TUI lets a developer choose a client, an agent, a provider, a model, and
  an optional variant, then saves the result without hand-editing JSON.
- Every provider and model choice comes from live discovery from the selected
  client.
- After a confirmed selection, `.karl-ai/config.json` holds the new value and
  `sync opencode` renders the updated agent file.
- Current noninteractive commands keep their output and exit codes.
- Current default model values do not change.
- A cancellation writes nothing and returns a distinct non-success exit.
- A discovery failure or a missing client binary shows a clear error and a
  manual entry fallback; it never writes silently.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/karl-ai` pass.

## Foundation Areas

- CLI experience: the new interactive command and the unchanged
  noninteractive contract.
- Config contract: read and write of the existing model and variant fields.
  No new schema key.
- OpenCode projection journey: write then sync.
- Client-neutral boundary: discovery stays in the OpenCode adapter; the core
  catalog stays unchanged.
- Testing: deterministic TUI tests through an injected discovery source.
