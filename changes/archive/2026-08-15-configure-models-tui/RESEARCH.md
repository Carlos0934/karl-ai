# Change Research

Evidence collected before planning. The planner cites these sections from
PLAN.md `Decision Basis` with `cites: <Section Name>`; do not copy facts into
PLAN.md.

## Research Questions

- Where are model defaults, project overrides, and agent rendering coupled
  today, and how does the config schema constrain a configuration flow?
- Which authoritative OpenCode commands or APIs list providers and available
  models, and what output, cache, and failure behavior do they have?
- Which maintained Go TUI libraries suit the configure flow, and how do at
  least two approaches compare for this change?
- Which command entry keeps existing noninteractive workflows compatible while
  adding the TUI?
- What persistence and schema-migration implications follow from saving
  per-agent model and variant assignments?
- What failure and cancellation behavior must the TUI define for discovery,
  saving, and interruption?
- Which test seams exist for driving the TUI and the discovery adapter
  deterministically?

## Repository Evidence

### Current Coupling Of Model Defaults, Overrides, And Rendering

- Model defaults are hard-coded in the OpenCode adapter:
  `adapters/opencode/render.go:34-47` — `DefaultConfig()` returns a
  `Config` with one `AgentConfig` per Karl agent, each with a fixed `Model`
  and an optional `Variant`. Example defaults: orchestrator
  `openai/gpt-5.6-sol` with `variant: high`, implementer
  `opencode-go/gpt-5.6-luna` with `variant: xhigh`, searcher
  `opencode-go/deepseek-v4-flash`.
- The config schema is a Go struct: `adapters/opencode/render.go:15-18`
  defines `AgentConfig{Model string, Variant string}`; `render.go:20-27`
  defines `OpenCodeConfig{Agents map[string]AgentConfig}` and
  `Config{Version int, OpenCode OpenCodeConfig}`. `ConfigVersion = 1`
  (`render.go:13`).
- Rendering merges project overrides over compiled defaults:
  `adapters/opencode/render.go:53-64` — `Render` reads `DefaultConfig()` as
  the base and applies non-empty `model` and `variant` values from the
  project config.
- The rendered agent file carries `model:` and optional `variant:` frontmatter
  fields: `adapters/opencode/render.go:128-131`.
- Config validation checks only the version and the set of agent identifiers:
  `adapters/opencode/render.go:85-99`. Model strings are never validated.
- The project override file is committed and mirrors the defaults:
  `.karl-ai/config.json:1-31` — all seven agents carry explicit `model`
  values; `karl-implementer` uses `opencode-go/gpt-5.6-luna` with
  `variant: xhigh` (`.karl-ai/config.json:12-13`).
- Init writes `DefaultConfig()` only when the config file is missing:
  `adapters/opencode/project.go:85-98`. An existing config is never rewritten
  by init.
- Config decoding rejects unknown fields: `adapters/opencode/project.go:262-266`
  — `readConfig` uses `DisallowUnknownFields`, so any new config key fails
  the read unless the schema is extended.
- The projection lifecycle writes config, agents, commands, skills, and
  `opencode.json`, and records hashes in the manifest:
  `adapters/opencode/project.go:108-188`.
- Drift is rejected without force and replaced with force:
  `adapters/opencode/project.go:139-141`.

### Client-Neutral Boundary

- The core catalog holds agent definitions without model identifiers:
  `core/catalog/catalog.go:85-134` — each `Agent` carries `ID`,
  `Description`, `Role`, `Prompt`, `Delegates`, and `Skills`; no model field
  exists.
- Architecture goal: the core stays client-neutral and adapters hold client
  specifics: `docs/DESIGN.md:12-22`.
- Configuration rule: client-specific model identifiers never enter the core
  catalog: `docs/DESIGN.md:183`.
- The domain context excludes concrete model identifiers and client file
  formats: `docs/CONTEXT.md:42-43`.
- The dependency direction keeps adapters on top of core:
  `docs/DESIGN.md:59-67` — `core/*` never imports Cobra, adapters, or
  OS specifics.

### CLI Wiring And Output Contract

- Command tree wiring: `cli/root.go:30-36` registers `init`, `sync`,
  `uninstall`, `change`, and `version`.
- Projection command factory: `cli/root.go:40-81` builds the `opencode`
  subcommand, the `--check` and `--force` flags for `sync` and `uninstall`,
  and calls the projector.
- Root flag: `cli/root.go:282-284` — `--root` defaults to `.`.
- Output contract: commands write JSON to stdout, errors to stderr, exit 0 on
  success, 1 on error: `docs/DESIGN.md:70-75`; `cli/root.go:294-298`;
  `cmd/karl-ai/main.go:13-19`.
- The `--json` flag on list, status, and validate is a no-op because all
  output is JSON: `docs/DESIGN.md:267-268`; `cli/root.go:172,197,222`.
- The change lifecycle command surface is fixed and tested:
  `cli/root_test.go:26-58` (init/sync/uninstall flags) and
  `cli/root_test.go:60-96` (init, check, uninstall over temp dirs).

### Persistence, Migration, And Compatibility Baseline

- Persistence model: config overrides in `.karl-ai/config.json`, ownership in
  `.karl-ai/manifest.json`, shared client settings in
  `.opencode/opencode.json`: `docs/DESIGN.md:204-211`.
- Migration rules: none; there is no database and no schema migration:
  `docs/DESIGN.md:225-227`.
- Config and manifest carry a `version` field; unsupported versions are
  rejected: `docs/DESIGN.md:329-330`; `adapters/opencode/project.go:285-287`.
- The OpenCode client format is not pinned to a released OpenCode version;
  compatibility relies on the current schema URL and frontmatter shape:
  `docs/DESIGN.md:331-333`.
- Runtime has no network calls: `docs/DESIGN.md:35`.
- Verification is Windows-only: `docs/DESIGN.md:421`. No CI, tags, or releases
  exist: `docs/DESIGN.md:426-429`.
- Current baseline: 44 tests pass in 9 packages; `go vet` and `go build`
  pass: `docs/CONTEXT.md:8`; `docs/DESIGN.md:367`.
- Deterministic projection tests fix the file set at 29 files and assert the
  rendered model and variant values: `adapters/opencode/project_test.go:15-97`
  and `adapters/opencode/project_test.go:99-117`.
- The only production dependency is Cobra; no TUI library exists in the
  module: `go.mod:5`.
- Atomic writes exist for safe config replacement:
  `adapters/filesystem/managed.go:9-31` (`WriteAtomic`).
- The research document requires exactly the sections named here:
  `core/documents/validators.go:379-381`.
- This change is a draft at assurance level L2:
  `changes/configure-models-tui/CHANGE.md:2-8`.

### Dynamic Discovery Primary Evidence

Verified in this environment with the installed OpenCode binary version
1.18.18 (`opencode --version`). These are runtime observations, not file
references:

- `opencode --help` lists `opencode models [provider]` with the description
  "list all available models".
- `opencode models --help` shows the positional `provider` ("provider ID to
  filter models by"), `--refresh` ("refresh the models cache from
  models.dev"), and `--verbose` ("use more verbose model output (includes
  metadata like costs)").
- `opencode models` prints one plain `provider/model` line per model to
  stdout. Example observed output: `opencode-go/deepseek-v4-flash`,
  `opencode-go/deepseek-v4-pro`, `opencode-go/gpt-5.6-luna`.
- `opencode models opencode-go` returned 19 models for the `opencode-go`
  provider in the current project directory.
- `opencode models --refresh openai` printed "Models cache refreshed" and
  then listed the models; the process exited 0.
- `opencode models --verbose PROVIDER` prints a plain `provider/model`
  line followed by one JSON object per model. The JSON object includes
  `id`, `providerID`, `name`, `api`, `status`, `cost`, `limit`,
  `capabilities`, `release_date`, and a `variants` map (for
  `deepseek-v4-flash`, the observed variants were `high` and `max`).
- The HTTP server API is available: `opencode serve --port PORT` then
  `GET /api/model` returns `{ "location": {...}, "data": [...] }`. A local
  call returned 6564 model entries; each entry includes `providerID`,
  `enabled`, `status`, `variants`, `cost`, and `limit`.

### Previous Project-ID Failure

- No repository evidence exists for the OpenCode project-ID foreign-key
  failure that blocked subagent sessions during the restart; it was an
  environment event. Inferred: the failure shows that OpenCode session
  integrity depends on consistent project and model references, so a
  configuration value that OpenCode rejects can break client-side sessions.
  Record this only as an integration risk (see Risks And Gaps).

## External Sources

- OpenCode CLI reference, `models` command: `opencode models [provider]`
  lists all available models in `provider/model` format; `--refresh`
  refreshes the models cache from Models.dev; `--verbose` adds metadata
  such as costs. URL: https://opencode.ai/docs/cli/
- OpenCode CLI environment variables: `OPENCODE_DISABLE_MODELS_FETCH`
  disables fetching models from remote sources, and `OPENCODE_MODELS_URL`
  overrides the models configuration URL. URL:
  https://opencode.ai/docs/cli/
- OpenCode models documentation: the catalog is built from Models.dev,
  provider integrations, and user configuration; model references use the
  form `provider/model#variant`; variants are model-specific and derived
  from catalog metadata. URL: https://opencode.ai/docs/models/
- OpenCode agents documentation: the `model` option on an agent overrides
  the model for that agent in `provider/model` format; `opencode agent
  create --model` assigns a model interactively; subagents use the model
  of the invoking primary agent when no model is set. URL:
  https://opencode.ai/docs/agents/
- OpenCode server API: `opencode serve` exposes an HTTP API; `GET
  /api/model` returns the available models, and `GET /config/providers`
  lists providers and default models. URL:
  https://opencode.ai/docs/server/
- OpenCode V2 models documentation: only enabled models whose provider is
  available for the current project appear in the model picker; model
  availability is location-scoped; the `/variants` command shows valid
  variant names. URL: https://opencode.ai/v2/docs/models
- OpenCode V2 API documentation: `GET /api/model` returns the current
  snapshot of available models ordered by release date. URL:
  https://opencode.ai/v2/docs/api/model/v2-model-list
- Bubble Tea: maintained Go TUI framework based on the Elm Architecture;
  version v1.3.10 on pkg.go.dev (MIT license, over 11,000 importers);
  `Program` supports `WithInput`, `WithOutput`, and `WithContext` options;
  `ErrInterrupted` is returned when the program receives SIGINT or an
  `InterruptMsg`. URL: https://pkg.go.dev/github.com/charmbracelet/bubbletea
- Huh: maintained Charm form library built on Bubble Tea, version v2.0.0;
  provides `Select`, `Input`, `Confirm`, and `Validate`; `Run` returns
  `ErrUserAborted` when the user cancels and `ErrTimeout` on timeout;
  the quit key sets the aborted flag and returns `tea.Interrupt`. URL:
  https://github.com/charmbracelet/huh
- Huh form API reference: `Run` blocks until the user submits or aborts;
  error values include `ErrUserAborted` and `ErrTimeout`. URL:
  https://github.com/charmbracelet/huh/blob/main/_autodocs/api-reference/form.md
- PTerm: maintained Go terminal presentation library, version v0.12.83
  (MIT license, over 2,500 importers); provides `DefaultInteractiveSelect`,
  `DefaultInteractiveConfirm`, and `DefaultSpinner`. URL:
  https://pkg.go.dev/github.com/pterm/pterm
- PTerm interactive select example: `pterm.DefaultInteractiveSelect.
  WithOptions(options).Show()` presents a fuzzy-searchable selection menu.
  URL:
  https://github.com/pterm/pterm/blob/master/_examples/interactive_select/demo/README.md
- PTerm interactive confirm example: `pterm.DefaultInteractiveConfirm.Show()`
  returns a boolean result. URL:
  https://github.com/pterm/pterm/blob/master/_examples/interactive_confirm/README.md

## Risks And Gaps

- The plain `opencode models` list format is simple but is not documented as
  a stable machine contract. The `--verbose` format prints one JSON object
  per model but was verified only partially; the exact stream structure must
  be confirmed during implementation. Gap: no official statement that the
  output format is stable.
- Model availability is location-scoped. A model available in one project
  may be unavailable in another (https://opencode.ai/v2/docs/models). The
  TUI must run discovery against the target project root, not against the
  current working directory. Inferred: the `--root` semantics of the CLI
  must be preserved by setting the discovery working directory.
- `opencode models` can make a network call when the cache is stale, and
  `--refresh` always contacts Models.dev. The project baseline states "no
  network calls at runtime" (`docs/DESIGN.md:35`). Dynamic discovery is a
  deliberate change to that baseline; offline behavior must be defined.
- If `opencode` is not installed or not on PATH, discovery fails. The TUI
  needs a defined fallback (manual model entry) and a clear error path.
- The `opencode models` command lists models from configured providers, but
  the filter semantics for authentication status are not documented. A model
  that OpenCode lists may still fail at session time if its provider has no
  credentials. The TUI cannot guarantee that a selected model will work.
- The current implementer default is `opencode-go/gpt-5.6-luna`
  (`adapters/opencode/render.go:42` and `.karl-ai/config.json:12-13`), and
  the model exists in the OpenCode list. The decision "do not reintroduce
  Luna" is ambiguous: it may mean the model or the discarded OpenCode
  changes. This decision is out of scope for research.
- OpenCode client compatibility is unpinned to a released version
  (`docs/DESIGN.md:331-333`, `docs/DESIGN.md:428`). The discovery surface may
  change with new OpenCode releases.
- Integration risk (from the previous project-ID failure): a config value
  that OpenCode rejects can break subagent sessions. The TUI must write only
  values observed from discovery or explicitly entered by the user, and must
  not silently remove existing overrides.
- Any new config field is rejected today by `DisallowUnknownFields`
  (`adapters/opencode/project.go:262-266`), and no migration mechanism
  exists (`docs/DESIGN.md:225-227`). A schema change requires version
  handling and a migration decision.
- The change lifecycle gates require that every RESEARCH section contain
  meaningful content (`core/documents/validators.go:288-297,328-333`). This
  document keeps all five sections populated.
- Tests are Windows-only and there is no CI (`docs/DESIGN.md:421-429`).
  TUI behavior on macOS and Linux is unproven.

## Open Questions

- Which TUI approach is selected: Bubble Tea with Huh forms, PTerm, or a
  standard-library prompt loop? Evidence compares the first two (see
  Repository Evidence and External Sources); the choice is a planning
  decision.
- Which discovery source is selected: the `opencode models` CLI, the
  `opencode serve` `/api/model` endpoint, or both with fallback? The CLI is
  simpler and offline-cacheable; the API returns richer metadata including
  per-model variants.
- How is the TUI entered: a new `karl-ai` subcommand (for example under a
  `models` group), an OpenCode slash command that shells out to the binary,
  or both? The current command surface and JSON/exit-code contract
  (`docs/DESIGN.md:70-75`) must remain valid for noninteractive use.
- Does the TUI write the config and then run `sync`, or write the config and
  instruct the user to run `sync`? Evidence shows sync re-renders the
  projection from config (`adapters/opencode/project.go:108-121`).
- What are the cancellation and exit-code semantics when the user cancels
  (Esc or Ctrl+C)? The evidence records `ErrUserAborted` and `ErrInterrupted`
  as library signals; the CLI exit code for cancellation is not defined in
  the foundation.
- How does the TUI handle a model that exists in the current config but is
  no longer listed by discovery (for example after an OpenCode upgrade)?
  Preserving the value is the backward-compatible baseline; flagging it as
  stale is an open behavior decision.
- Does the OpenCode agent `variant` frontmatter field remain supported, or
  must variants move to the `provider/model#variant` reference form? The
  repository emits a separate `variant:` field
  (`adapters/opencode/render.go:129-131`), while the V2 documentation
  describes the `#variant` reference form
  (https://opencode.ai/v2/docs/models). The OpenCode 1.18.18 schema was not
  inspected during this research.
- What is the exact `--verbose` output stream structure for parsing, and is
  the non-verbose list written only to stdout? Partially verified; a full
  stream capture is missing.
- Which provider filter does the TUI present when the provider list is large
  (the `/api/model` call returned 6564 entries)? Filtering and pagination
  behavior is a planning decision.
- How is the project-ID foreign-key failure recorded for regression
  verification, given that no repository artifact documents it?
