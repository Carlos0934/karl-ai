# Change Plan

See [CHANGE.md](./CHANGE.md) for outcome, scope, and acceptance criteria.

## Decision Basis

- Use the OpenCode `models` CLI, `opencode models`, as the primary discovery
  source. It is simpler than running a server and reuses the client cache.
  Fall back to manual model entry when discovery fails.
  cites: Repository Evidence

- Put discovery behind an interface in `adapters/opencode`. The core catalog
  and config schema stay client-neutral and unchanged.
  cites: Repository Evidence

- Reuse the current config schema keys `model` and `variant` under
  `opencode.agents`. Add no new key, so no schema migration or version bump
  is required.
  cites: Repository Evidence

- Use Bubble Tea with Huh forms for the TUI. Huh gives selection, input,
  confirmation, and cancel signals, and filters large model lists.
  cites: External Sources

- Enter the TUI through a single new `karl-ai models configure` subcommand.
  Do not change `init` and do not add a slash command. Keep all current
  noninteractive commands and their JSON and exit-code contract unchanged.
  cites: Repository Evidence

- Define failure and cancellation behavior because live discovery changes the
  current no-network baseline. Never write silently.
  cites: Risks And Gaps

- Configure one agent at a time. The flow is client, agent, provider, model,
  then variant, matching the per-agent config schema.
  cites: Repository Evidence

- On confirmation, write the selection with an atomic write and run a
  drift-safe sync automatically. The developer does not run a second command.
  cites: Repository Evidence

- Keep the variant as the separate `variant` field. Do not use the
  `#variant` reference form, so the schema and rendered frontmatter stay
  unchanged.
  cites: Repository Evidence

- Distinguish an omitted variant override from a confirmed `No variant`
  selection. Omitted legacy values retain a compiled default; a confirmed empty
  value clears it without a schema change.
  cites: Repository Evidence

## Technical Approach

The change adds one interactive subcommand, `karl-ai models configure`, to the
existing Cobra tree. It does not modify `init` and does not add a slash
command. The command runs a small TUI built on Bubble Tea with Huh.

The flow is client, agent, provider, model, then optional variant. The client
list is discovered at run time; today it resolves to OpenCode. Provider and
model lists come from `opencode models`, grouped and filtered in the TUI.
Variants come from the `--verbose` metadata when the chosen model has them.
Discovery runs against the `--root` target directory so location-scoped model
availability is correct.

A discovery interface in `adapters/opencode` wraps the client. The TUI depends
only on that interface, so tests inject a deterministic source. The core
catalog never gains a model identifier.

On confirm, the TUI writes only the selected agent `model` and `variant`
through the existing `Config` and `AgentConfig` types with an atomic write,
then runs a drift-safe projector sync automatically so the projection matches
the config. The variant stays a separate `variant` field. The config type
preserves whether a variant was omitted: an omitted legacy override retains its
compiled default, while a confirmed `No variant` selection writes an explicit
empty value and suppresses that default in rendered frontmatter.

No new config key is added, so the existing `DisallowUnknownFields` read keeps
passing and no migration is needed.

## Work Units

| Unit | Deliverable |
|---|---|
| 1 | Interactive model selection for one agent through the TUI, with cancellation |
| 2 | Persist a confirmed selection to the config and re-sync the projection |
| 3 | Safe behavior for missing client, discovery failure, stale model, and manual entry |
| 4 | Noninteractive compatibility and end-to-end regression coverage |
| 5 | Resolve explicit-no-variant and exhausted-input review findings |
| 6 | Resolve same-model legacy no-variant persistence |

## Cross-Unit Constraints

- Discovery is injected behind an interface; no unit calls `opencode` in a way
  tests cannot fake.
- Only the selected agent model and variant are written. Defaults and other
  agents stay untouched.
- No new config key is added.
- Writes use atomic replacement; drift is never overwritten without force.
- The core catalog never gains a model identifier.

## Integrated Validation

- `go test ./...`, `go vet ./...`, and `go build ./cmd/karl-ai` pass.
- An end-to-end check initializes a temp project, runs the TUI with an
  injected discovery source, selects a model, confirms, and asserts the config
  and the rendered agent file.
- A baseline check asserts the noninteractive commands produce identical
  output before and after the change.
- A defaults check asserts the current default model values are unchanged.
- Regression checks prove that `No variant` clears a default variant, omitted
  legacy variant overrides retain their default, and exhausted nonterminal
  input returns a controlled non-success result without writing config.

## Recovery

Each unit reverts by restoring its files; config is mutated only on confirm,
and atomic writes leave the previous config intact on failure. Discovery uses
the client local cache by default, so offline behavior stays bounded. A failed
sync leaves the config written but reports the error and lets the developer
retry.
