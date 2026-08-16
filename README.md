# karl-ai

`karl-ai` is a native Go CLI that installs and configures Karl's reserved
OpenCode agent.

## Install

```sh
go install github.com/carlos0934/karl-ai/cmd/karl-ai@latest
```

## OpenCode

Initialize Karl in a project:

```sh
karl-ai init opencode --root /path/to/project
```

The projection contains only:

```text
.opencode/agents/karl-orchestrator.md
```

The generated agent is a primary placeholder with no assigned responsibilities,
skills, delegated agents, work documents, or persistent workflow state. Its
model and optional variant are stored directly in the agent's Markdown frontmatter.

Configure its model interactively:

```sh
karl-ai models configure --root /path/to/project
```

Synchronize or check the projection:

```sh
karl-ai sync opencode --root /path/to/project
karl-ai sync opencode --root /path/to/project --check
```

Remove the generated projection:

```sh
karl-ai uninstall opencode --root /path/to/project
```

Karl does not create `.opencode/skills/`, manifests, shared OpenCode configuration,
or lifecycle documents.

## Development

```sh
go vet ./...
go test -count=1 ./...
go build ./cmd/karl-ai
go mod tidy -diff
go mod verify
```
