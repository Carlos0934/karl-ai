---
implementation_ref: "daddf75^..d861b9e"
user_validation: pending
foundation_status: synced
blocking_findings: open
---

# Change Review

## Target

Validate the implementation against [CHANGE.md](./CHANGE.md) and the technical
constraints in [PLAN.md](./PLAN.md).

## Change-Specific Considerations

- Provider, model, and variant choices must come from OpenCode discovery in the
  target project, except for a clearly marked stale saved model or confirmed
  manual fallback.
- `variant` remains separate from `model`. Selecting "No variant" must not
  inherit the compiled default variant for the selected agent.
- Confirmation writes the selected agent only. Cancellation, invalid manual
  input, and a failed config write must leave the config unchanged.
- Save precedes drift-safe sync. A sync failure can leave the confirmed config
  saved, but it must not overwrite drifted projection files.
- The new interactive command must not change compiled defaults, the core
  catalog, or existing noninteractive command contracts.

## Evidence

| Check | Command Or Method | Result |
|---|---|---|
| Change state and implementation | `go run ./cmd/karl-ai change status configure-models-tui --root . --json`; `git show --stat --oneline d861b9e`; `git diff --name-status daddf75 d861b9e` | State was `reviewing`, assurance was `L2`, correction commit `d861b9e5b6811177132b4bcbb6d283ab4b40d146` resolved, and the implement gate was `ok: true`. The reviewed implementation range is `daddf75^..d861b9e`. |
| F-1 focused adapter checks | `go test ./adapters/opencode -run 'TestRenderDistinguishesOmittedAndExplicitEmptyVariants|TestReadConfigWithOmittedVariantRetainsDefaultVariant|TestConfigureModelClearsDefaultVariant|TestDefaultConfigMatchesModelBaseline' -count=1 -v` | 6 tests passed. Direct rendering distinguishes omitted and explicit-empty variants, direct configuration clears a default variant, and default JSON remains stable. |
| F-2 focused TUI checks | `go test ./tui -run 'TestRunReturnsInputExhausted|TestRunEsc|TestRunCtrlC' -count=1 -v` | 4 tests passed. Empty input and invalid-then-EOF return `ErrInputExhausted`; Esc and Ctrl+C still return cancellation. |
| Correction CLI checks | `go test ./cli -run 'TestModelsConfigureNoVariantClearsDefaultVariant|TestModelsConfigureExhaustedInputLeavesConfigUnchanged|TestNoninteractiveCommandOutputAndExitCodeBaseline' -count=1 -v` | 3 tests passed for fresh explicit-empty persistence, exhausted-input no-write behavior, and noninteractive compatibility. |
| Full tests | `go test ./... -count=1` | 79 tests passed in 10 packages. |
| Race check | `go test -race ./... -count=1` | Passed. |
| Static analysis and build | `go vet ./...`; `go build ./cmd/karl-ai` | Both commands passed with no output. |
| Module state | `go mod tidy -diff`; `go mod verify` | Tidy reported no diff. Module verification reported `all modules verified`. |
| Architecture boundary | `go list -f '{{join .Imports "\\n"}}' ./core/...` with adapter, CLI, TUI, Cobra, and Charm import rejection | Passed. Core has no prohibited import. `tui` imports the OpenCode adapter and catalog; OpenCode discovery remains in the adapter. |
| Default stability | `TestDefaultConfigMatchesModelBaseline`; review of `DefaultConfig` and `.karl-ai/config.json` | The serialized compiled defaults and source configuration are unchanged. Presence tracking changes only the Go representation. |
| Real OpenCode discovery | `opencode --version`; `opencode models opencode-go`; `opencode models --verbose opencode-go` | OpenCode 1.18.18 returned 19 `opencode-go` models. Verbose output included names and variant maps. The real configure flow listed four discovered providers and 19 discovered provider models plus the stale saved model. No refresh was requested. |
| Fresh explicit `No variant` runtime | Initialized `%TEMP%/opencode/karl-rereview-configure-models-20260815`, selected a discovered model with variants for `karl-orchestrator`, selected `No variant`, and confirmed | Exit was 0. Config contains `"variant": ""`; rendered orchestrator frontmatter has no `variant:` line; a following `sync --check` returned `changed: false`. The main F-1 persistence path is corrected. |
| Legacy omitted variant runtime | Used the prior config whose orchestrator model is `opencode-go/glm-5.1` with omitted `variant`; selected that same discovered model, confirmed the summary `with no variant`, and compared config hashes | Exit was 0 with `sync.changed: false`. The config hash stayed `29506EC3934154FCCECADB12948878E3F0FE4B62B0B96031A1F3B428AF4F937C`, `variant` stayed omitted, and rendered frontmatter retained `variant: "high"`. This reproduces the remaining F-1 path. |
| Invalid variant then EOF runtime | Selected a discovered model with variants, entered `invalid`, then ended stdin; compared config SHA-256 before and after | Exit was 1 with `model configuration input exhausted`. There was no panic. The config hash was unchanged. F-2 is corrected. |
| Re-review gate and return | `go run ./cmd/karl-ai change validate configure-models-tui review --root . --json`; `go run ./cmd/karl-ai change transition configure-models-tui implementing --root .` | The gate failed only for pending user acceptance, open blocking findings, the pending User Validation table, and `Ready to archive: No`. Evidence and foundation markers passed. The legal backward transition returned the change to `implementing`. |

## Runtime Verification

```text
Input: Fresh initialized project; orchestrator has compiled variant high; select
       discovered opencode-go/deepseek-v4-flash, No variant, Confirm.
Output: Exit 0 and a changed sync result.
Observed state: Config contains model opencode-go/deepseek-v4-flash and
                "variant": "".
Observed side effects: Rendered orchestrator has the selected model and no
                       variant line. sync --check reports current.

Input: Legacy config has model opencode-go/glm-5.1 and omits variant; select the
       same discovered model and confirm the summary with no variant.
Output: Exit 0 and sync.changed false.
Observed state: Config hash is unchanged and variant remains omitted.
Observed side effects: Rendered orchestrator retains variant high. The confirmed
                       no-variant choice is not persisted.

Input: Select a discovered model with variants, enter an invalid variant, then
       end stdin.
Output: Exit 1 and model configuration input exhausted; no panic.
Observed state: Config SHA-256 is unchanged.
Observed side effects: None.
```

## Findings

| Finding | Class | Severity | Evidence | Resolution |
|---|---|---|---|---|
| F-1: A confirmed no-variant choice still fails when a legacy override omits `variant` and the developer selects the same model. Fresh model changes are corrected, but the legacy interaction path is not. | Blocking | High | `tui/configure.go:264` compares `selection.Variant` with `variantValue(saved.Variant)`. `variantValue(nil)` returns `""` at `tui/configure.go:268-272`, so omitted and explicit-empty values are equal. `cli/root.go:73-83` then treats the selection as unchanged and skips `ConfigureModel`. The real legacy run left config and `variant: "high"` unchanged after confirmation. | Open. Preserve variant presence in unchanged detection. A confirmed `No variant` must write an explicit empty variant even when model text is unchanged. Add a CLI regression with a legacy omitted variant and the same selected model. |
| F-2: Depleted nonterminal input can panic instead of returning an error or cancellation. | Required | Medium | Focused tests passed. The real invalid-then-EOF run returned `ErrInputExhausted`, exited 1, did not panic, and kept the config hash unchanged. | Resolved by `d861b9e`. |
| F-3: Interactive behavior remains verified only on Windows. | Advisory | Low | `RESEARCH.md:260-261` and `docs/DESIGN.md:448,456` record the platform gap. Automated and runtime review used Windows amd64. | Open, non-blocking. Add macOS and Linux checks when CI is established. |

## User Validation

| Field | Value |
|---|---|
| Status | Pending |
| Validated by |  |
| Date |  |
| Scenario | After F-1 is fully resolved, start with a legacy agent override that omits `variant` and therefore inherits a compiled default. Select the same discovered model, select or confirm `No variant`, and confirm. Verify that config now contains `"variant": ""`, rendered frontmatter has no `variant:` line, and another cancellation or exhausted-input run leaves config unchanged. |
| Notes | Do not request acceptance while blocking finding F-1 is open. Acceptance must be an explicit developer decision relayed by the orchestrator. |

## Foundation Reconciliation

| Artifact | Result |
|---|---|
| `changes/configure-models-tui/foundation/DESIGN.md` | Validated after `d861b9e`. It defines explicit-empty clearing, legacy omission compatibility, and controlled exhausted input. |
| `changes/configure-models-tui/foundation/journeys/opencode-projection/JOURNEY.md` | Validated after `d861b9e`. It contains the corrected journey and failure flow. |
| `docs/DESIGN.md` | Reconciled in `d861b9e`; config presence semantics and the 79-test baseline are current. |
| `docs/journeys/opencode-projection/JOURNEY.md` | Reconciled in `d861b9e`; explicit-empty and omitted behavior are current. |
| Links, commands, schema, and terminology | Validated. `foundation_status` remains `synced`. F-1 is a remaining implementation defect against the documented baseline, not a foundation gap. |

## Archive Decision

| Requirement | Status |
|---|---|
| Implementation committed | Yes: `daddf75^..d861b9e` |
| Blocking findings resolved | No: F-1 is open |
| User validation accepted | Pending |
| Foundation reconciled | Yes: synced |
| Ready to archive | No |
