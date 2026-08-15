---
implementation_ref: "daddf75^..7841c99"
user_validation: accepted
foundation_status: synced
blocking_findings: none
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
| Change state and implementation | `go run ./cmd/karl-ai change status configure-models-tui --root . --json`; `git show --stat --oneline 7841c99`; `git diff --name-status d861b9e 7841c99` | State was `reviewing`, assurance was `L2`, final correction commit `7841c991c2af91494a1dd7086e300b838b05be1b` resolved, and the implement gate was `ok: true`. The reviewed implementation range is `daddf75^..7841c99`. |
| Final F-1 and unchanged checks | `go test ./cli -run 'TestModelsConfigureSameLegacyModelNoVariantPersistsExplicitClear|TestModelsConfigureUnchangedSelectionDoesNotRewriteConfig|TestModelsConfigureExhaustedInputLeavesConfigUnchanged' -count=1 -v` | 3 tests passed. The legacy same-model path persists explicit empty, and a true unchanged explicit selection does not rewrite config or projection. |
| F-2 focused TUI checks | `go test ./tui -run 'TestRunReturnsInputExhaustedForEmptyNonterminalInput|TestRunReturnsInputExhaustedAfterInvalidVariantResponse' -count=1 -v` | 2 tests passed. Empty input and invalid-then-EOF return `ErrInputExhausted`. |
| Full tests | `go test ./... -count=1` | 81 tests passed in 10 packages. |
| Race check | `go test -race ./... -count=1` | Passed. |
| Static analysis and build | `go vet ./...`; `go build ./cmd/karl-ai` | Both commands passed with no output. |
| Module state | `go mod tidy -diff`; `go mod verify` | Tidy reported no diff. Module verification reported `all modules verified`. |
| Architecture boundary | `go list -f '{{join .Imports "\\n"}}' ./core/...` with adapter, CLI, TUI, Cobra, and Charm import rejection | Passed. Core has no prohibited import. `tui` imports the OpenCode adapter and catalog; OpenCode discovery remains in the adapter. |
| Default stability | `TestDefaultConfigMatchesModelBaseline`; review of `DefaultConfig` and `.karl-ai/config.json` | The serialized compiled defaults and source configuration are unchanged. Presence tracking changes only the Go representation. |
| Real OpenCode discovery | `opencode --version`; `opencode models opencode-go`; `opencode models --verbose opencode-go` | OpenCode 1.18.18 returned 19 `opencode-go` models. Verbose output included names and variant maps. The real configure flow listed four discovered providers and 19 discovered provider models plus the stale saved model. No refresh was requested. |
| Fresh explicit `No variant` runtime | Initialized `%TEMP%/opencode/karl-rereview-configure-models-20260815`, selected a discovered model with variants for `karl-orchestrator`, selected `No variant`, and confirmed | Exit was 0. Config contains `"variant": ""`; rendered orchestrator frontmatter has no `variant:` line; a following `sync --check` returned `changed: false`. The main F-1 persistence path is corrected. |
| Exact legacy same-model runtime | Started with orchestrator model `opencode-go/glm-5.1`, omitted `variant`, and inherited rendered `variant: "high"`; selected the same discovered model and confirmed `with no variant` | Exit was 0 with `sync.changed: true`. Config hash changed from `29506EC3934154FCCECADB12948878E3F0FE4B62B0B96031A1F3B428AF4F937C` to `A861E8BCD7BA4110D91EDE3E30FF7EBC19CCD7F2AEA98F3F24E7988697CB4748`; config contains `"variant": ""`; rendered frontmatter has no `variant:` line. `sync --check` then returned `changed: false`. F-1 is resolved. |
| True unchanged runtime | Repeated the same model and no-variant confirmation after config held explicit empty; compared config hashes | Exit was 0 with `sync.changed: false`. Config SHA-256 stayed `A861E8BCD7BA4110D91EDE3E30FF7EBC19CCD7F2AEA98F3F24E7988697CB4748`. No rewrite occurred. |
| Invalid variant then EOF runtime | Selected a discovered model with variants, entered `invalid`, then ended stdin; compared config SHA-256 before and after | Exit was 1 with `model configuration input exhausted`. There was no panic. Config SHA-256 stayed `A861E8BCD7BA4110D91EDE3E30FF7EBC19CCD7F2AEA98F3F24E7988697CB4748`. F-2 remains resolved. |
| Final review gate and transition | `go run ./cmd/karl-ai change validate configure-models-tui review --root . --json`; `go run ./cmd/karl-ai change transition configure-models-tui validated --root .` | The review gate returned `ok: true` with no errors or warnings. The lifecycle CLI transitioned the change from `reviewing` to `validated`. |

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
Output: Exit 0 and sync.changed true.
Observed state: Config now contains "variant": "".
Observed side effects: Rendered orchestrator no longer has a variant line.
                       sync --check reports current.

Input: Repeat the same model and no-variant confirmation with explicit empty
       already saved.
Output: Exit 0 and sync.changed false.
Observed state: Config SHA-256 is unchanged.
Observed side effects: None.

Input: Select a discovered model with variants, enter an invalid variant, then
       end stdin.
Output: Exit 1 and model configuration input exhausted; no panic.
Observed state: Config SHA-256 is unchanged.
Observed side effects: None.
```

## Findings

| Finding | Class | Severity | Evidence | Resolution |
|---|---|---|---|---|
| F-1: A confirmed no-variant choice failed when a legacy override omitted `variant` and the developer selected the same model. | Blocking | High | `tui/configure.go:264` now requires matching model, variant value, and non-nil variant presence before it marks a selection unchanged. The exact runtime wrote explicit empty, removed rendered variant frontmatter, and left the following true unchanged run byte-stable. | Resolved by `7841c99`. |
| F-2: Depleted nonterminal input can panic instead of returning an error or cancellation. | Required | Medium | Focused tests passed. The real invalid-then-EOF run returned `ErrInputExhausted`, exited 1, did not panic, and kept the config hash unchanged. | Resolved by `d861b9e`. |
| F-3: Interactive behavior remains verified only on Windows. | Advisory | Low | `RESEARCH.md:260-261` and `docs/DESIGN.md:465` record the platform gap. Automated and runtime review used Windows amd64. | Open, non-blocking. Add macOS and Linux checks when CI is established. |

## User Validation

| Field | Value |
|---|---|
| Status | Accepted |
| Validated by | Developer |
| Date | 2026-08-15 |
| Scenario | Start with a legacy agent override that omits `variant` and therefore inherits a compiled default. Select the same discovered model, select or confirm `No variant`, and confirm. Verify that config contains `"variant": ""` and rendered frontmatter has no `variant:` line. Repeat the same confirmed selection and verify that config is not rewritten. End one invalid variant flow with EOF and verify controlled failure with no write. |
| Notes | The developer selected `Acepto la validación` after review of the recorded evidence. This is explicit validation acceptance for `configure-models-tui`. |

## Foundation Reconciliation

| Artifact | Result |
|---|---|
| `changes/configure-models-tui/foundation/DESIGN.md` | Validated after `7841c99`. It defines explicit-empty clearing, legacy omission compatibility, presence-aware unchanged detection, and controlled exhausted input. |
| `changes/configure-models-tui/foundation/journeys/opencode-projection/JOURNEY.md` | Validated after `d861b9e`. It contains the corrected journey and failure flow. |
| `docs/DESIGN.md` | Reconciled in `7841c99`; unchanged detection semantics and the 81-test baseline are current. |
| `docs/journeys/opencode-projection/JOURNEY.md` | Reconciled in `d861b9e`; explicit-empty and omitted behavior are current. |
| Links, commands, schema, and terminology | Validated. `foundation_status` remains `synced`; no foundation gap is open. |

## Archive Decision

| Requirement | Status |
|---|---|
| Implementation committed | Yes: `daddf75^..7841c99` |
| Blocking findings resolved | Yes: none open |
| User validation accepted | Yes: 2026-08-15 |
| Foundation reconciled | Yes: synced |
| Ready to archive | Yes |
