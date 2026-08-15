# Build `karl-ai` as a native, cross-agent Go CLI

Replace the project-local Node/OpenCode prototype with one native Go binary.
The Go core is the source of truth; client directories such as `.opencode/`
are generated projections in consumer projects, not source directories in
this repository.

## Decisions

| Topic | Decision |
|---|---|
| Repository and module | `github.com/carlos0934/karl-ai` |
| Executable | `karl-ai` |
| Go layout | One Go module with separate `core`, `cli`, and `adapters` packages |
| First client | OpenCode |
| Consumer state | Project-local `.karl-ai/config.json` and `.karl-ai/manifest.json` |
| OpenCode output | Managed `.opencode/` projection in consumer projects |
| CLI framework | Cobra, constructed directly without `cobra-cli` generation |
| Compatibility | Preserve the current lifecycle behavior and tests before deleting Node code |
| Commits | Never create commits automatically |

## Target Structure

```text
.
|-- cmd/
|   `-- karl-ai/
|       `-- main.go
|-- core/
|   |-- catalog/
|   |   `-- content/
|   |-- documents/
|   |   `-- templates/
|   |-- foundation/
|   `-- lifecycle/
|-- cli/
|-- adapters/
|   |-- filesystem/
|   |-- git/
|   `-- opencode/
|       `-- templates/
|-- go.mod
|-- go.sum
|-- README.md
|-- PLAN.md
`-- .gitignore
```

## Package Boundaries

| Package | Responsibility |
|---|---|
| `core/lifecycle` | States, transitions, gates, change operations, and domain errors |
| `core/documents` | Frontmatter, Markdown sections, validators, and embedded document templates |
| `core/foundation` | Required foundation artifacts and assurance concepts |
| `core/catalog` | Client-neutral agents, commands, skills, prompts, and references |
| `cli` | Cobra command tree, flags, output formatting, exit behavior, and dependency wiring |
| `adapters/filesystem` | Project discovery, change package persistence, projection state, and atomic writes |
| `adapters/git` | Repository checks, commit reference resolution, and uncommitted-file discovery |
| `adapters/opencode` | Render and manage OpenCode agents, commands, skills, and `opencode.json` |

The core must not import Cobra or know OpenCode paths, model identifiers, Git
commands, or operating-system filesystem details.

## CLI Surface

```text
karl-ai init opencode [--root PATH]
karl-ai sync opencode [--root PATH] [--check] [--force]
karl-ai uninstall opencode [--root PATH] [--force]

karl-ai change new <name> [--level L1|L2|L3|L4] [--root PATH]
karl-ai change list [--root PATH] [--json]
karl-ai change status <name> [--root PATH] [--json]
karl-ai change validate <name> <plan|implement|review|archive> [--root PATH] [--json]
karl-ai change transition <name> <state> [--root PATH]
karl-ai change archive <name> [--root PATH] [--date YYYY-MM-DD]

karl-ai version
karl-ai completion <shell>
```

## Managed Projection

`karl-ai init opencode` creates project-local management state:

```text
.karl-ai/
|-- config.json
`-- manifest.json
```

The OpenCode adapter projects the catalog into:

```text
.opencode/
|-- agents/
|-- commands/
|-- skills/
`-- opencode.json
```

The manifest records the Karl version, client, relative path, and content hash
for every generated file. `sync` is idempotent. It refuses to overwrite drift
unless `--force` is provided. `uninstall` removes only files still owned by the
manifest and preserves unrelated OpenCode configuration.

OpenCode model and variant defaults live in the OpenCode adapter. Projects can
override them in `.karl-ai/config.json`; client-specific model IDs never enter
the core catalog.

## Work Units

### 1. Establish the Go application

- Create `go.mod`, the binary entrypoint, Cobra root command, version command,
  shared I/O, and project-root handling.
- Add a clean repository `.gitignore` and concise README.
- Verify `go test ./...`, `go vet ./...`, and `go build ./cmd/karl-ai`.

### 2. Port lifecycle core and persistence

- Port states and legal transitions exactly.
- Port frontmatter parsing and deterministic serialization.
- Port CHANGE, PLAN, TASKS, RESEARCH, and REVIEW validation.
- Port scaffolding, list, status, gate validation, transitions, and archive.
- Keep Git behind a core interface implemented by `adapters/git`.
- Reproduce all 16 existing tests, including citation, dependency, Git, and
  archive behavior.

### 3. Centralize portable content

- Move lifecycle and foundation document templates into embedded core assets.
- Extract agent role prompts without OpenCode frontmatter.
- Extract skill instructions and references as client-neutral catalog content.
- Replace Node script paths in prompts with `karl-ai change ...` commands.
- Keep RESEARCH ownership, PLAN citation, REVIEW evidence, and no-auto-commit
  rules unchanged.

### 4. Implement the OpenCode projection

- Render seven agents with OpenCode frontmatter, models, variants, and
  permissions.
- Render five slash commands and two skills.
- Merge the Karl fields into `opencode.json` without deleting user fields.
- Implement init, sync, drift detection, forced replacement, and uninstall.
- Add golden tests for paths and rendered content.
- Add integration tests proving idempotence and preservation of user files.

### 5. Remove the prototype

Delete only after the Go tests and projection parity checks pass:

```text
.agents/
.opencode/
diagrams/
scripts/
node_modules/
package.json
package-lock.json
skills-lock.json
```

Remove `docs/` only if it remains empty. Do not delete consumer-facing content
until its canonical or projected replacement is covered by a test.

## Compatibility Checklist

- [x] Foundation is required before creating a change.
- [x] Change names use lowercase kebab-case.
- [x] Required package files are scaffolded from embedded templates.
- [x] Scaffold placeholders fail the plan gate.
- [x] PLAN decisions cite complete RESEARCH sections.
- [x] Work units contain Prepare, Implement, and Validate tasks.
- [x] States and backward transitions match the prototype.
- [x] Implementation references resolve to Git commits.
- [x] Archive rejects unrelated uncommitted files.
- [x] Dependencies must be archived first.
- [x] Foundation status matches foundation artifacts.
- [x] Archive moves the package and never creates a commit.
- [x] Text and JSON output have stable exit behavior.
- [x] OpenCode generation is deterministic and idempotent.
- [x] Drift is not overwritten without explicit force.
- [x] Uninstall preserves files not owned by Karl.

## Final Verification

```text
go test ./...
go vet ./...
go build ./cmd/karl-ai

karl-ai init opencode --root <temporary-project>
karl-ai sync opencode --check --root <temporary-project>
karl-ai change new example-change --level L2 --root <temporary-project>
karl-ai uninstall opencode --root <temporary-project>
```

The migration is complete when the Go implementation satisfies the checklist,
the temporary OpenCode projection is valid, and no Node, Mermaid, `.agents`,
or source `.opencode` artifacts remain in this repository.
