# System Design: Interactive Model Configuration

## Scope

`karl-ai models configure` is the only interactive model-configuration entry.
`init` and all existing noninteractive commands keep their contracts. The flow
configures one Karl agent at a time for OpenCode.

## Architecture

```text
cli -> tui -> adapters/opencode -> OpenCode executable
             adapters/opencode -> adapters/filesystem
tui -> core/catalog
```

- `cli` owns Cobra wiring, standard streams, JSON output, and exit behavior.
- `tui` owns unpersisted client, agent, provider, model, variant, and
  confirmation state.
- `adapters/opencode` owns client discovery, config persistence, and projection
  sync.
- `core/catalog` supplies agent identifiers only. It contains no provider or
  model identifier.

## Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| Bubble Tea | v2.0.2 | Terminal UI runtime |
| Huh | v2.0.3 | Select, input, confirmation, validation, and cancellation fields |
| Bubbles | v2.0.0 | Huh key bindings |

## Discovery Contract

- The OpenCode adapter runs `opencode models` in the `--root` project to list
  providers.
- It runs `opencode models --verbose <provider>` in the same project to list
  models, names, and model-specific variants.
- Provider and model values are not compiled into the TUI.
- A missing executable, command failure, or empty result gives a clear manual
  `provider/model` fallback.
- A saved model absent from discovery remains available and is marked stale.

## CLI Contract

| Command | Output | Exit behavior |
|---|---|---|
| `karl-ai models configure [--root PATH]` | Interactive form on stderr; saved agent, model, optional variant, and sync result as JSON on stdout | 0 on success; 1 on error or cancellation |

The flow order is client, agent, provider, model, optional variant, and confirm.
The command does not run through `init` and does not add an OpenCode slash
command.

## Persistence And Projection

The command uses the existing version 1 config shape:

```json
{
  "version": 1,
  "opencode": {
    "agents": {
      "<agent-id>": {
        "model": "<provider/model>",
        "variant": "<optional separate variant>"
      }
    }
  }
}
```

- Confirmation atomically replaces `.karl-ai/config.json` after changing only
  the selected agent value.
- A confirmed `No variant` selection persists an explicit empty variant. The
  rendered agent then has no variant and does not inherit the compiled default.
- A legacy override that omits `variant` retains the compiled default variant.
- Save runs before automatic sync.
- Sync never forces drifted managed files. If sync fails after save, the command
  reports the partial result and leaves drifted files unchanged.
- Cancellation, exhausted accessible input, and invalid manual input do not
  write configuration. Exhausted input returns a controlled error and does not
  panic.

## Validation Baseline

- Discovery parsing is tested with injected command output.
- TUI tests cover order, variants, cancellation, exhausted input, missing
  discovery, stale saved models, and manual validation.
- Adapter tests cover one-agent persistence, atomic write failure, default
  stability, and drift refusal.
- CLI tests cover end-to-end save and sync plus existing noninteractive output
  and exit behavior.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/karl-ai` are required.
