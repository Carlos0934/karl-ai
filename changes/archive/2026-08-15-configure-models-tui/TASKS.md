# Change Tasks

## 1. Interactive model selection for one agent through the TUI

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** None

### Outcome

A developer runs `karl-ai models configure` and steps through client, agent,
provider, model, and optional variant selection. The provider and model lists
come from live discovery, not a hard-coded catalog. The developer can cancel
at any step with no write.

### Acceptance

- The command launches a TUI and presents client, agent, provider, and model
  choices in that order.
- Provider and model choices come only from the injected discovery source.
- A chosen model that offers variants shows a variant step.
- Cancelling with Esc or Ctrl+C writes nothing and exits non-zero.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| New `models configure` command in `cli` | A registered interactive subcommand |
| Discovery interface and OpenCode implementation | Parsed provider and model lists from the client |
| TUI flow in a new package | The client-agent-provider-model-variant steps |

### Work

- [x] 1.1 **Prepare:** Fix the boundaries: the command is
  `karl-ai models configure`, the flow is one agent at a time with client,
  agent, provider, model, variant, and the variant stays a separate field.
- [x] 1.2 **Implement:** Add the command, the discovery interface, the
  OpenCode implementation, and the TUI flow.
- [x] 1.3 **Validate:** Run unit tests with an injected discovery source and
  drive the flow in a scripted run.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Command registered | `go run ./cmd/karl-ai --help` | The `models configure` command appears |
| Flow order | Unit test with fake discovery | Client, agent, provider, model, variant order |
| Live lists only | Unit test | No hard-coded provider or model value |
| Cancel is safe | Scripted TUI sends Ctrl+C | Exit non-zero, no config write |

#### Runtime Scenario

```text
Given: a temp project with a fake discovery source returning opencode and
       provider opencode-go with model opencode-go/deepseek-v4-pro.
When: the developer selects opencode, karl-planner, opencode-go, and the model.
Then: the TUI shows a summary and waits for confirm.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| Discovery returns no provider | TUI reports that no provider choices are available and writes nothing; manual entry remains Unit 3 |
| Developer presses Esc | TUI exits with no write and a non-zero exit |

### Rollback

Revert the new command, discovery interface, and TUI files. No config is
written by this unit.

### Complete When

- [x] The command runs and walks the full flow.
- [x] Provider and model lists come only from discovery.
- [x] Cancellation writes nothing and exits non-zero.
- [x] Unit tests pass.
- [x] The core catalog stays free of model identifiers.

## 2. Persist a confirmed selection and re-sync the projection

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** Unit 1

### Outcome

On confirm, the TUI writes the selected model and variant for the chosen agent
to `.karl-ai/config.json` with an atomic write, then runs the OpenCode sync so
the rendered agent file reflects the new value. Other agents and defaults stay
untouched.

### Acceptance

- The config write changes only the selected agent model and variant.
- The write is atomic; a failed write leaves the previous config intact.
- The projection sync runs after the write and the agent file shows the new
  model.
- Defaults and other agent entries are byte-identical after the change.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| Config read-modify-write path in `adapters/opencode` | Selected agent updated, others unchanged |
| Sync call after write | Rendered agent file matches the new value |
| Tests for the write and sync path | Deterministic assertions on config and render |

### Work

- [x] 2.1 **Prepare:** Fix the write-then-sync behavior: atomic save of the
  config, then an automatic drift-safe sync.
- [x] 2.2 **Implement:** Add the read-modify-write path and the sync call
  after a confirmed write.
- [x] 2.3 **Validate:** Run tests that assert the config and the rendered
  agent file after a scripted confirm.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Only selected agent changes | Unit test over a temp config | Other agents unchanged |
| Atomic write | Unit test forcing a write failure | Previous config intact |
| Rendered agent updates | `sync opencode` in a temp project | Agent file has the new model |
| Defaults unchanged | Diff of defaults before and after | Identical |

#### Runtime Scenario

```text
Given: a temp project with a config for all seven agents.
When: the developer confirms model opencode-go/gpt-5.6-luna for karl-planner.
Then: the config shows the new planner model, sync runs, and the planner agent
      file renders that model. Other agents and defaults are unchanged.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| Config file is not writable | The write fails, the old config stays, an error shows |
| Sync reports drift without force | The error is shown and no managed file is overwritten |

### Rollback

Restore the previous `.karl-ai/config.json` and re-run sync. The write path is
the only mutation this unit adds.

### Complete When

- [x] A confirmed selection persists to the config.
- [x] Sync renders the updated agent file.
- [x] Defaults and other agents stay unchanged.
- [x] Automated checks pass.
- [x] The projection journey documentation reflects write-then-sync.

## 3. Safe behavior for missing client, discovery failure, stale model, and manual entry

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** Unit 1

### Outcome

The TUI handles a missing client binary, a discovery failure, a saved model
that is no longer listed, and a manual entry fallback. Each case shows a clear
message and never writes a value the developer did not confirm.

### Acceptance

- A missing client binary shows a clear error and a manual entry path.
- A discovery failure shows an error and a manual entry path, never a crash.
- A saved model absent from discovery is marked stale and kept, not silently
  removed.
- Manual entry validates the model reference before it is offered for save.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| Error and fallback paths in the TUI | Clear messages and manual entry |
| Stale model detection and display | Saved model kept and flagged |
| Manual entry validation | The entered reference is checked before save |

### Work

- [x] 3.1 **Prepare:** Fix the stale and manual entry behavior: a stale saved
  model is kept and flagged; manual entry is validated before save.
- [x] 3.2 **Implement:** Add the error, fallback, stale, and manual entry
  paths.
- [x] 3.3 **Validate:** Run tests for each failure path with an injected
  discovery source.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Missing client binary | Test with no `opencode` on PATH | Clear error, manual entry offered |
| Discovery failure | Fake source returns an error | Clear error, manual entry offered |
| Stale saved model | Config holds a model not in discovery | Model kept and shown as stale |
| Manual entry format | Enter an invalid reference | Rejected before save |

#### Runtime Scenario

```text
Given: a config where karl-searcher holds a model no longer listed by
       discovery.
When: the developer opens the TUI for karl-searcher.
Then: the saved model is shown as stale, is kept, and the developer can pick a
      replacement or leave it unchanged.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| `opencode` is not installed | Error plus manual entry; no write |
| Discovery times out | Error plus manual entry; no write |

### Rollback

Revert the fallback and stale-handling code. No config is written on a failure
path, so rollback is source-only.

### Complete When

- [x] Each failure path shows a clear message.
- [x] Stale models are kept and flagged.
- [x] Manual entry is validated.
- [x] Automated checks pass.

## 4. Noninteractive compatibility and end-to-end regression coverage

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** Unit 1, Unit 2, Unit 3

### Outcome

The full change is validated end to end, and the current noninteractive
commands and default models are proven unchanged. The package passes the
project quality gates.

### Acceptance

- All current noninteractive commands keep their output and exit codes.
- Current default model values are byte-identical before and after the change.
- An end-to-end TUI run over a temp project updates the config and the
  rendered agent file.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/karl-ai` pass.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| Noninteractive command regression tests | Identical output and exit codes |
| End-to-end TUI test with injected discovery | Config and render update correctly |
| Project quality gate run | All tests, vet, and build pass |

### Work

- [x] 4.1 **Prepare:** Record the baseline output and exit codes of the
  noninteractive commands.
- [x] 4.2 **Implement:** Add regression and end-to-end tests.
- [x] 4.3 **Validate:** Run the full quality gates and confirm the defaults
  are unchanged.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Noninteractive commands unchanged | Run `init`, `sync`, `uninstall`, `change`, `version` before and after | Identical output and exit codes |
| Defaults unchanged | Compare `DefaultConfig` output | Identical |
| End-to-end TUI | Scripted run over a temp project | Config and render match |
| Quality gates | `go test ./...`, `go vet ./...`, `go build ./cmd/karl-ai` | All pass |

#### Runtime Scenario

```text
Given: a clean temp project initialized by karl-ai.
When: the TUI selects and confirms a new model for one agent.
Then: the config and the rendered agent file change, and every other file and
      command output stays identical to the baseline.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| A noninteractive command output differs | The regression test fails and blocks the unit |

### Rollback

Revert this unit only (tests and any baseline recording). The behavior units
are reverted by their own boundaries.

### Complete When

- [x] Noninteractive compatibility is proven.
- [x] Defaults are proven unchanged.
- [x] The end-to-end run passes.
- [x] All quality gates pass.

## 5. Resolve explicit-no-variant and exhausted-input review findings

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** Unit 4

### Outcome

A confirmed `No variant` selection clears a compiled default variant for the
selected agent, while a legacy override that omits `variant` keeps the default.
Exhausted accessible input returns a controlled non-success result before any
config write.

### Acceptance

- A confirmed `No variant` selection stores an explicit empty variant and
  renders no `variant:` frontmatter for an agent with a default variant.
- Existing config that omits `variant` retains the compiled default variant.
- Empty or exhausted nonterminal input, including after an invalid variant
  response, returns `ErrInputExhausted` and does not panic or write config.
- Compiled defaults and noninteractive commands remain unchanged.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| Presence-aware OpenCode variant override | Omitted and explicit-empty variants have distinct render behavior |
| Controlled nonterminal-input termination | Accessible TUI input cannot drive Huh into an EOF panic |
| Adapter, TUI, and CLI regressions | Tests cover persistence, rendering, legacy config, and no-write termination |
| Foundation reconciliation | Design and journey state the explicit-empty variant rule |

### Work

- [x] 5.1 **Prepare:** Define the compatibility rule from F-1 and the
  controlled exhausted-input result from F-2 without adding a config key.
- [x] 5.2 **Implement:** Preserve variant-field presence through config merge
  and persistence, and convert exhausted accessible input to
  `ErrInputExhausted`.
- [x] 5.3 **Validate:** Run focused, full, race, static, build, and module
  checks; verify the scripted F-1 and F-2 regressions.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Explicit empty variant | Adapter and CLI scripted configuration tests | Config contains `"variant": ""`; rendered selected agent has no `variant:` line |
| Legacy omitted variant | Adapter config-read and render test | Omitted variant retains the compiled default |
| Exhausted input | TUI and CLI tests with empty input and invalid variant followed by EOF | `ErrInputExhausted`, no panic, and config bytes unchanged |
| Quality gates | `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/karl-ai`, `go mod tidy`, `go mod verify` | All commands pass; tidy leaves module files unchanged |

#### Runtime Scenario

```text
Given: a temporary initialized project and an orchestrator with a compiled
       default variant.
When: the developer selects a discovered model, No variant, and confirms.
Then: config records an explicit empty variant and the rendered orchestrator
      has no variant frontmatter.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| Config omits `variant` from an existing override | Render retains the compiled default variant |
| Variant input is invalid and then EOF occurs | TUI returns `ErrInputExhausted`; no panic or config write |

### Rollback

Restore the former config merge and TUI reader behavior together with this
unit's tests and foundation wording. No persisted project config is changed by
the code change itself.

### Complete When

- [x] Explicit empty and omitted variants have demonstrated distinct behavior.
- [x] Exhausted nonterminal input has demonstrated controlled termination.
- [x] Regression tests and all quality gates pass.
- [x] Design and journey artifacts describe the compatibility rule.

## 6. Resolve same-model legacy no-variant persistence

**Status:** Complete
**Type:** Vertical Slice
**Depends on:** Unit 5

### Outcome

When a legacy override has the selected model but omits `variant`, confirming
`No variant` writes an explicit empty variant and removes inherited rendered
frontmatter. A selected model and explicit variant remain unchanged only when
their values and variant presence match.

### Acceptance

- A same-model legacy override with omitted `variant` is not unchanged after a
  confirmed `No variant` selection.
- The command saves `"variant": ""` and synchronizes an agent without
  `variant:` frontmatter.
- A same model and explicit matching variant remain unchanged and do not
  rewrite config.
- Fresh `No variant`, cancellation, and exhausted-input behavior remain valid.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| Presence-aware unchanged comparison | Omitted and explicit-empty variants are not equal |
| CLI same-model legacy regression | Config persistence and rendered frontmatter prove the correction |
| CLI unchanged regression | Matching explicit model and variant avoid a config rewrite |

### Work

- [x] 6.1 **Prepare:** Confirm that F-1 is caused by the unchanged comparison,
  not rendering or persistence.
- [x] 6.2 **Implement:** Include variant presence in unchanged detection and
  add same-model legacy and true-unchanged CLI regressions.
- [x] 6.3 **Validate:** Run focused, full, race, static, build, and module
  checks without weakening prior F-1 or F-2 coverage.

### Validation

| Check | Method | Expected Result |
|---|---|---|
| Same-model legacy clear | Scripted CLI test | Explicit empty config variant and no rendered `variant:` line |
| True unchanged selection | Scripted CLI test | Config bytes and projection remain unchanged |
| Regression suite | Focused F-1/F-2 tests and project quality commands | All pass |

#### Runtime Scenario

```text
Given: a legacy orchestrator override with model provider/model and no variant.
When: the developer selects provider/model, No variant, and confirms.
Then: config writes an explicit empty variant and the rendered orchestrator has
      no variant frontmatter.
```

#### Failure Scenarios

| Condition | Expected Result |
|---|---|
| Same model with an explicit matching variant | The command does not rewrite config |
| Input ends before confirmation | The command returns `ErrInputExhausted` and writes nothing |

### Rollback

Restore the prior unchanged comparison and remove this unit's tests. The
correction has no direct mutation outside a developer-confirmed command run.

### Complete When

- [x] Same-model legacy clearing is demonstrated through the CLI.
- [x] True unchanged behavior is demonstrated without a config write.
- [x] Focused and project quality checks pass.
