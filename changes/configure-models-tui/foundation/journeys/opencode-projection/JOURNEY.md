# Journey: Configure One OpenCode Agent Model

## Purpose

Let a developer select and apply one Karl agent model and optional variant
without editing JSON.

## Trigger

The developer runs `karl-ai models configure [--root PATH]` in an initialized
project.

## Actor

- Developer.

## Inputs

| Input | Meaning |
|---|---|
| `--root` | Target project, default current directory |
| `.karl-ai/config.json` | Current per-agent model and variant values |
| OpenCode discovery | Project-scoped providers, models, and variants |

## Main Flow

1. The tool presents OpenCode as the available client.
2. The developer selects one Karl agent.
3. OpenCode discovers providers in the target project.
4. The developer selects a provider.
5. OpenCode discovers that provider's models and variants in the target
   project.
6. The developer selects a model and an optional separate variant.
7. The tool presents a summary and waits for confirmation.
8. On confirmation, the tool atomically saves only the selected agent value.
9. The tool runs drift-safe OpenCode sync without force.
10. The tool writes the saved selection and sync result as JSON.

A confirmed `No variant` selection stores an explicit empty variant and renders
no variant frontmatter. An existing override that omits `variant` keeps its
compiled default variant.

## Alternate And Failure Flows

### Cancellation

Esc, Ctrl+C, or a declined confirmation exits unsuccessfully and writes
nothing.

### Exhausted Accessible Input

Input that ends before a selection is complete returns a controlled
non-success result, writes nothing, and never panics.

### Discovery Unavailable

A missing OpenCode executable, discovery error, or empty result displays the
error and requests a manual `provider/model` value and optional separate
variant. The tool validates the value and still requires confirmation.

### Stale Saved Model

A saved model absent from discovery is labeled configured and stale. The
developer can retain it without a config rewrite or select a replacement.

### Write Failure

An atomic config-write failure reports an error and preserves the prior config.
Sync does not run.

### Sync Failure After Save

The command reports that the config was saved but sync failed. Drifted managed
files remain unchanged, and the developer can resolve drift and retry sync.

## Postconditions

- On success, config and the selected rendered agent agree on model and
  optional variant.
- Other agent values and compiled defaults do not change.
- Existing noninteractive command output and exit behavior do not change.

## Acceptance Criteria

- Provider and model choices come from project-scoped live discovery, except
  for a marked stale value or confirmed manual fallback.
- A confirmed selection updates one agent and automatically runs drift-safe
  sync.
- `No variant` produces no rendered variant, while an omitted legacy variant
  retains the compiled default.
- Cancellation, exhausted input, invalid input, and write failure do not
  change config.
- Drift is reported and not overwritten.
