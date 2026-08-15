# karl-ai

`karl-ai` is a native Go CLI that centralizes Karl's project workflow and
projects its agents, commands, and skills into supported AI clients.

OpenCode is the first supported client. Its `.opencode/` directory is generated
from Karl's client-neutral catalog and managed with a project-local manifest.

## Install

```sh
go install github.com/carlos0934/karl-ai/cmd/karl-ai@latest
```

## OpenCode Lifecycle

Initialize Karl in a consumer project:

```sh
karl-ai init opencode --root /path/to/project
```

This creates `.karl-ai/config.json`, projects the Karl agents, commands, skills,
and resources into `.opencode/`, merges Karl's required shared OpenCode settings,
and records owned-file hashes in `.karl-ai/manifest.json`.

Configure one agent model and optional variant without editing JSON:

```sh
karl-ai models configure --root /path/to/project
```

After confirmation, the command atomically saves only the selected agent
override and synchronizes the projection. It refuses to replace drifted managed
files without an explicit `sync opencode --force`.

If OpenCode cannot list models, the command shows the error and requests a
manual `provider/model` reference. It validates the reference before saving. A
configured model that is no longer listed remains available as a stale choice.

Reconcile the projection after upgrading `karl-ai` or changing overrides:

```sh
karl-ai sync opencode --root /path/to/project
karl-ai sync opencode --root /path/to/project --check
```

`--check` performs no writes and exits unsuccessfully when generated state is
not current. Karl rejects edits to managed files; use `--force` only when those
edits should be replaced by the catalog projection.

Remove the managed projection:

```sh
karl-ai uninstall opencode --root /path/to/project
```

Uninstall preserves `.karl-ai/config.json`, unrelated `.opencode/` files and
settings, and OpenCode values changed after installation. It rejects drifted
owned files unless `--force` is supplied.

## Change Lifecycle

```sh
karl-ai change new add-refunds --level L2
karl-ai change list
karl-ai change status add-refunds
karl-ai change validate add-refunds plan
karl-ai change transition add-refunds planned
karl-ai change archive add-refunds
```

Use `--root /path/to/project` when the target is not the current directory.
The archive command moves the validated package but never creates a commit.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/karl-ai
```

The maintained project context and technical baseline live in
[docs/CONTEXT.md](docs/CONTEXT.md) and [docs/DESIGN.md](docs/DESIGN.md).
