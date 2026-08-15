---
implementation_ref: "daddf75"
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
| Change state and implementation | `go run ./cmd/karl-ai change status configure-models-tui --root . --json`; `git show --stat --oneline daddf75` | State was `reviewing`, assurance was `L2`, commit `daddf755253c752a24ae54eb06e40477d770d3ed` resolved, and the implement gate was `ok: true`. |
| Focused behavior | `go test ./adapters/opencode ./tui ./cli -count=1` | 47 tests passed in 3 packages. The packages cover discovery parsing, cancellation, write failure, drift refusal, stale models, manual validation, end-to-end save and sync, and noninteractive output and exit codes. |
| Named failure paths | `go test ./adapters/opencode -run 'TestDefaultConfigMatchesModelBaseline|TestConfigureModelUpdatesOneAgentAndSynchronizes|TestConfigureModelWriteFailureLeavesConfigUnchanged|TestConfigureModelReportsDriftAfterSavingConfig' -count=1 -v`; equivalent focused runs for `./tui` and `./cli` | 4 adapter tests, 7 TUI tests, and 10 CLI tests passed. |
| Full tests | `go test ./... -count=1` | 70 tests passed in 10 packages. |
| Race check | `go test -race ./... -count=1` | Passed. |
| Static analysis and build | `go vet ./...`; `go build ./cmd/karl-ai` | Both commands passed with no output. |
| Dependency integrity | `go mod verify` | `all modules verified`. Bubble Tea v2.0.2, Huh v2.0.3, and Bubbles v2.0.0 are pinned in `go.mod`. |
| Architecture boundary | `go list -f '{{join .Imports "\\n"}}' ./core/...` with adapter, CLI, TUI, Cobra, and Charm import rejection | Passed. Core has no prohibited import. `tui` imports the OpenCode adapter and catalog; OpenCode discovery remains in the adapter. |
| Default stability | `git diff --quiet daddf75^ daddf75 -- adapters/opencode/render.go .karl-ai/config.json core/catalog` | Passed. The compiled defaults, source config, and core catalog are byte-stable across the implementation commit. |
| Real OpenCode discovery | `opencode --version`; `opencode models opencode-go`; `opencode models --verbose opencode-go` | OpenCode 1.18.18 returned 19 `opencode-go` models. Verbose output included names and variant maps. The real configure flow listed four discovered providers and 19 discovered provider models plus the stale saved model. No refresh was requested. |
| Real cancellation | Piped Esc to `go run ./cmd/karl-ai models configure --root %TEMP%/opencode/karl-review-configure-models-20260815` and compared SHA-256 before and after | Exit was 1, output was `model configuration cancelled`, and config SHA-256 stayed `29506EC3934154FCCECADB12948878E3F0FE4B62B0B96031A1F3B428AF4F937C`. |
| Real missing-client fallback | Built the binary, removed OpenCode from `PATH`, entered `manual-provider/manual-model`, then declined confirmation | The command reported `OpenCode executable was not found: opencode`, offered manual entry, exited 1 on cancellation, and left the config hash unchanged. |
| Real save and projection | Selected `karl-orchestrator`, discovered `opencode-go/glm-5.1`, and `No variant` in a temporary initialized project | Save and sync returned exit 0. Config stored the model with no variant, but the rendered agent retained `variant: "high"`. This reproduces F-1. |
| Projection check | `go run ./cmd/karl-ai sync opencode --root %TEMP%/opencode/karl-review-configure-models-20260815 --check` after the real save | Returned `changed: false`, which confirms that the incorrect inherited variant is the projector's desired state, not residual drift. |
| Review gate and return | `go run ./cmd/karl-ai change validate configure-models-tui review --root . --json`; `go run ./cmd/karl-ai change transition configure-models-tui implementing --root .` | The review gate failed only for pending user acceptance, open blocking findings, the pending User Validation table, and `Ready to archive: No`. The legal backward transition returned the change from `reviewing` to `implementing`. |

## Runtime Verification

```text
Input: OpenCode 1.18.18; temporary initialized project; choose OpenCode,
       karl-orchestrator, opencode-go, glm-5.1, No variant, Confirm.
Output: Exit 0 and JSON for agent karl-orchestrator, model
        opencode-go/glm-5.1, and a changed sync result.
Observed state: .karl-ai/config.json contains the selected model and no variant.
Observed side effects: .opencode/agents/karl-orchestrator.md contains the selected
                       model but also variant: "high". A following sync --check
                       reports the projection as current.

Input: Esc at the first prompt.
Output: Exit 1 and model configuration cancelled.
Observed state: The config SHA-256 is unchanged.
Observed side effects: None.

Input: OpenCode absent from PATH; valid manual provider/model; decline confirm.
Output: Clear missing-client message, manual fallback, then exit 1.
Observed state: The config SHA-256 is unchanged.
Observed side effects: None.
```

## Findings

| Finding | Class | Severity | Evidence | Resolution |
|---|---|---|---|---|
| F-1: Selecting `No variant` can render an unselected compiled default variant. This fails the optional-variant and config-to-projection acceptance criteria. | Blocking | High | `tui/configure.go:177-182` returns an empty variant for `No variant`; `adapters/opencode/project.go:114` saves it; `adapters/opencode/render.go:57-63` applies a variant override only when it is non-empty. The real run saved `opencode-go/glm-5.1` without a variant at temp config lines 15-17, but rendered `variant: "high"` at temp agent line 5. | Open. Return to implementation. Represent an explicit cleared variant or otherwise stop default-variant inheritance when a confirmed model has no variant. Add an end-to-end test for an agent whose compiled default has a variant. |
| F-2: Depleted nonterminal input can panic instead of returning an error or cancellation. | Required | Medium | A real piped run supplied an invalid variant response and then EOF. Huh panicked with `index out of range [-1]`; the application frame was `tui/configure.go:293`. `RunWithSavedModel` deliberately supports nonterminal input through the accessible form at `tui/configure.go:48-53,303-309`. | Open. Convert EOF and exhausted accessible input into a controlled non-success result and add a regression test. |
| F-3: Interactive behavior remains verified only on Windows. | Advisory | Low | `RESEARCH.md:260-261` and `docs/DESIGN.md:448,456` record the platform gap. Automated and runtime review used Windows amd64. | Open, non-blocking. Add macOS and Linux checks when CI is established. |

## User Validation

| Field | Value |
|---|---|
| Status | Pending |
| Validated by |  |
| Date |  |
| Scenario | After F-1 and F-2 are resolved, configure an agent that has a default variant with a discovered model and `No variant`; confirm config and projection match. Also cancel once and confirm no write. |
| Notes | Do not request acceptance while blocking finding F-1 is open. Acceptance must be an explicit developer decision relayed by the orchestrator. |

## Foundation Reconciliation

| Artifact | Result |
|---|---|
| `changes/configure-models-tui/foundation/DESIGN.md` | Added as the reduced normal-form design, dependency, CLI, persistence, and reliability baseline for this change. |
| `changes/configure-models-tui/foundation/journeys/opencode-projection/JOURNEY.md` | Added as the reduced normal-form interactive configuration journey. |
| `docs/DESIGN.md` | Validated against the reduced artifact and implementation boundaries. The implementation commit already contains the semantic baseline update. |
| `docs/journeys/opencode-projection/JOURNEY.md` | Validated against the reduced journey. The implementation commit already contains the semantic baseline update. |
| Links, commands, schema, and terminology | Validated. The config schema stays version 1 with separate `model` and `variant` fields. `foundation_status` is `synced`; F-1 is an implementation defect against this baseline. |

## Archive Decision

| Requirement | Status |
|---|---|
| Implementation committed | Yes: `daddf75` |
| Blocking findings resolved | No: F-1 is open |
| User validation accepted | Pending |
| Foundation reconciled | Yes: synced |
| Ready to archive | No |
