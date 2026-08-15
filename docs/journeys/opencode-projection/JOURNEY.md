# Journey: OpenCode Projection Lifecycle

## Purpose

Install, keep current, and remove Karl's agents, commands, and skills in a
developer's OpenCode client.

## Trigger

The developer runs `karl-ai init opencode`, `karl-ai sync opencode`,
`karl-ai uninstall opencode`, or `karl-ai models configure` in a project
directory.

## Actors

### Initiating Actor

- Developer.

### Participating Actors

- None. The projection is produced by the tool, not by an agent.

### Receiving Actors

- Developer: receives a valid managed projection, a current check, or a clean
  removal.

## Business Context

The OpenCode projection is generated from Karl's client-neutral catalog. The
project stores its source configuration in `.karl-ai/config.json`. The tool
records what it owns in `.karl-ai/manifest.json`.

## Inputs

| Input | Provided By | Meaning |
|---|---|---|
| `--root` | Developer | Project directory, default current directory |
| `.karl-ai/config.json` | Project | Agent model and variant overrides |
| `.karl-ai/manifest.json` | Project | Previously owned files and shared settings |

## Preconditions

- The tool can write into the project directory.
- For sync and uninstall, an installed manifest must exist.

## Actor Interactions

### CLI Interface

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai init opencode [--root PATH]` |
| Arguments and Options | `--root` |
| Standard Output | JSON: `operation`, `client`, `changed`, `paths` |
| Standard Error | Error text on failure |
| Exit Codes | 0 success, 1 error |
| Business Result | `.karl-ai/config.json`, `.karl-ai/manifest.json`, and `.opencode/` are created |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai sync opencode [--root PATH] [--check] [--force]` |
| Arguments and Options | `--root`, `--check`, `--force` |
| Standard Output | JSON: `operation`, `client`, `changed`, `paths` |
| Standard Error | Drift or out-of-sync error text |
| Exit Codes | 0 success, 1 error or out of sync |
| Business Result | Projection matches the catalog; shared settings are current |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai uninstall opencode [--root PATH] [--force]` |
| Arguments and Options | `--root`, `--force` |
| Standard Output | JSON: `operation`, `client`, `changed`, `paths` |
| Standard Error | Drift error text on refusal |
| Exit Codes | 0 success, 1 error |
| Business Result | Owned files removed; unrelated files and config preserved |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai models configure [--root PATH]` |
| Arguments and Options | `--root` |
| Standard Output | JSON: selected `agent`, `model`, optional `variant`, and `sync` result |
| Standard Error | Interactive form; clear error when save or sync fails |
| Exit Codes | 0 success, 1 error or cancellation |
| Business Result | The selected agent override is atomically saved, then the projection is synchronized |

## Actor Experience & Interface Behavior

### Experience Goal

The developer installs or updates Karl in one command and trusts that hand edits
are never silently lost.

### Experience States

| State | Actor Perceives | Actor Can Do | Expected Feedback |
|---|---|---|---|
| Initial | No `.opencode/` or no manifest | Run init | JSON result listing changed paths |
| Processing | Command running | Wait | No intermediate output |
| Model selection | Interactive model form | Confirm or cancel | Selected model and variant summary |
| Discovery unavailable | OpenCode cannot list choices | Enter a manual model or cancel | Clear error and manual `provider/model` prompt |
| Stale model | Saved model is not in discovered choices | Keep or replace it | Saved model labeled configured and stale |
| Validation Error | Drift or unknown config | Correct or force | Error naming the drifted paths |
| Success | Current or removed projection | Continue | JSON result with `changed` paths |
| Failure | Refusal or not installed | Read error and retry | Single-line error |

### Interaction Requirements

- Drift is reported as a list of paths, never silently overwritten.
- `--check` writes nothing and reports the changed paths.
- Uninstall keeps files not listed in the manifest.
- Discovery failure or an empty discovery result offers a manual entry path and
  never writes a value without confirmation.
- Manual model entry must use `provider/model`; variants remain a separate
  optional field.
- A saved model that discovery does not return remains configured and is marked
  stale rather than removed.

## Main Business Flow

1. The developer runs init in a project.
2. The tool writes `.karl-ai/config.json` if it is missing.
3. The tool renders agents, commands, skills, and shared settings from the
   catalog and overrides.
4. The tool records owned paths and their hashes in `.karl-ai/manifest.json`.
5. The tool writes the projection into `.opencode/` and returns the changed
   paths.

### Configure One Agent Model

1. The developer selects a client, agent, provider, model, and optional variant.
   A saved model absent from discovery is kept as a stale choice.
2. On confirmation, the tool atomically writes only that agent's `model` and
   `variant` fields in `.karl-ai/config.json`.
3. The tool runs the same drift-safe sync used by `karl-ai sync opencode`.
4. On success, it reports the saved selection and sync result. On sync failure,
   it reports that the config was saved and leaves drifted files unchanged.

## Alternate Flows

### Keep The Projection Current

1. The developer runs sync after an upgrade or an override change.
2. The tool compares desired content to the manifest and current files.
3. Unchanged files are left as they are; changed files are replaced; stale files
   are removed.

### Check Without Writing

1. The developer runs sync with `--check`.
2. The tool computes the desired projection and reports differences.
3. It writes nothing and exits 1 when the projection is not current.

### Replace Drift

1. The developer runs sync with `--force`.
2. The tool replaces drifted managed files and shared settings with the catalog
   values.

### Manual Model Entry

1. OpenCode discovery fails, times out, or returns no provider or model choices.
2. The tool explains the failure and asks for a manual `provider/model`
   reference and optional separate variant.
3. The tool validates the reference before confirmation. An invalid reference
   is not saved.

## Failure Flows

### Drift Refusal

1. A managed file or shared setting differs from both the manifest and the
   desired render.
2. The tool refuses and reports the paths.
3. The developer resolves the difference or uses `--force`.

### Model Sync Failure After Save

1. The developer confirms a model selection.
2. The atomic config write succeeds, but sync detects drift or another error.
3. The tool reports the sync failure and keeps the saved config for a later
   retry. It does not overwrite drifted files.

### Stale Saved Model

1. The saved agent model is not in the discovered model list.
2. The tool labels the saved value as configured and stale and keeps it as a
   selectable choice.
3. The developer can retain it without rewriting config, or select and confirm
   a replacement.

### Missing Installation

1. The developer runs uninstall without a manifest.
2. The tool reports that the projection is not installed.

### Unknown Override

1. `config.json` names an agent that is not in the catalog.
2. The tool rejects the config and reports the unknown identifier.

## Business Rules Applied

- Managed files that drift are not overwritten without explicit force.
- The client-neutral catalog is the source of truth.
- Uninstall preserves files and settings not owned by Karl.

## State Changes

| Entity or Concept | Previous State | New State | Condition |
|---|---|---|---|
| Projection | not installed | installed | init succeeds |
| Managed file | drifted | replaced | sync with force |
| Shared setting | user value | Karl value | sync restores Karl values |
| Projection | installed | removed | uninstall succeeds |
| Agent override | selected model and variant | saved and synchronized | models configure succeeds |

## Side Effects

| Side Effect | Caused By | Business Consequence | Reversible |
|---|---|---|---|
| Files written under `.opencode/` | init or sync | Client gains Karl content | Yes |
| Managed files replaced | sync with force | Hand edits lost | Partial |
| `opencode.json` merged | init or sync | Shared settings set | Yes |
| Manifest removed | uninstall | Ownership record deleted | Yes |
| Agent override saved | models configure | Selected agent uses the chosen model after sync | Yes, by another confirmed selection |

## Expected Outcomes

- A valid, current projection after init or sync.
- A clean removal after uninstall.

## Postconditions

- `.karl-ai/config.json` always remains after uninstall.
- Unrelated `.opencode/` files and user settings remain.

## Acceptance Criteria

- `karl-ai sync opencode --check` exits 0 when current and 1 when not current.
- Sync is idempotent: a second sync changes nothing.
- Drift is rejected without force and replaced with force.
- Uninstall removes only owned files and keeps `.karl-ai/config.json`.
- A confirmed model selection updates only the selected agent and never forces
  drifted projection files to be replaced.

## Related Journeys

- [Change Lifecycle](./../change-lifecycle/JOURNEY.md)

## Related Documents

- [System Context](../../CONTEXT.md)
- [System Design](../../DESIGN.md)
