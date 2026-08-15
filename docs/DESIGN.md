# System Design

This document describes the current technical baseline of `karl-ai`, a native
Go command-line tool. It is maintained in place and is not a change plan or a
historical log.

## 1. Design Scope & Architecture Goals

### Design Scope

The tool has two responsibilities. It manages the Karl change lifecycle through
a command line. It projects a client-neutral catalog of agents, commands, and
skills into supported AI clients. OpenCode is the first supported client.

### Architecture Goals

- Keep the core client-neutral. The core must not import Cobra, know OpenCode
  paths or model identifiers, or depend on the operating system filesystem.
- Keep client specifics in adapters.
- Make projection deterministic and idempotent.
- Never create a commit automatically.
- Keep the current Go CLI as the compatibility baseline.

### Non-Goals

- A graphical interface.
- A hosted service or multi-tenant system.
- Deleted Node/OpenCode prototype behavior is not an ongoing promise.

### Current System Characteristics

- One Go module, `github.com/carlos0934/karl-ai`, built as one binary.
- No persistence beyond project-local Markdown and JSON files.
- Projection and lifecycle operations do not make network calls at runtime.
  Model discovery delegates to the installed OpenCode client and can follow its
  cache and remote-catalog behavior.
- No CI, tags, or release process yet.

## 2. Architecture & System Organization

### Architecture Style

A layered, single-binary CLI. A client-neutral core holds domain rules. Adapters
hold filesystem, Git, and OpenCode specifics. The Cobra command tree wires the
two together.

### Main Architectural Areas

| Area | Purpose | Primary Dependencies |
|---|---|---|
| `core/lifecycle` | States, transitions, gates, change operations, domain errors | `core/documents`, `core/foundation` |
| `core/documents` | Frontmatter, Markdown sections, validators, embedded change templates | none internal |
| `core/foundation` | Required foundation artifacts and assurance levels | none internal |
| `core/catalog` | Client-neutral agents, commands, skills, prompts, references | `core/documents`, `core/foundation` |
| `cli` | Cobra command tree, flags, output, exit behavior, wiring | `core/lifecycle`, all adapters |
| `tui` | Interactive model selection flow | `adapters/opencode`, `core/catalog`, Bubble Tea, Huh |
| `adapters/filesystem` | Change persistence, atomic writes, projection state | `core/documents`, `core/lifecycle` |
| `adapters/git` | Repository checks, commit resolution, uncommitted files | `core/lifecycle` |
| `adapters/opencode` | Render and manage the OpenCode projection | `core/catalog`, `adapters/filesystem` |

### Dependency Direction

```text
cmd/karl-ai  ->  cli  ->  core/lifecycle, adapters/*, tui
cli          ->  tui  ->  adapters/opencode, core/catalog
adapters/*   ->  core/*
core/catalog ->  core/documents, core/foundation
core/lifecycle -> core/documents, core/foundation

core/* never imports Cobra, adapters, or operating-system specifics.
```

### Runtime Organization

```text
Single binary "karl-ai".
Commands write JSON to stdout and errors to stderr.
Exit code 0 on success, 1 on any error.
```

### External Integrations

| Integration | Direction | Purpose | Contract Reference |
|---|---|---|---|
| OpenCode projection | Outbound | Write managed agents, commands, skills, and `opencode.json` | Section 6 |
| OpenCode model discovery | Outbound | Read providers, models, and variants through `opencode models` | Section 6 |
| Git | Outbound | Resolve commits and list uncommitted files | Section 6 |
| OpenCode schema | Outbound | `$schema` value `https://opencode.ai/config.json` | Section 6 |

### Architecture Views

The [agent system diagram](./architecture/agent-system.png) gives a simplified,
human-readable view of the actors and workflow. Its editable Mermaid source is
[agent-system.mmd](./architecture/agent-system.mmd). The package layout above
remains the dependency view.

## 3. Technology Stack & Toolchain

### Runtime & Major Technologies

| Technology | Version | Role | Version Source |
|---|---|---|---|
| Go | 1.26 | Language and runtime | `go.mod` |
| Cobra | v1.9.1 | CLI framework | `go.mod` |
| Bubble Tea | v2.0.2 | Terminal UI runtime | `go.mod` |
| Huh | v2.0.3 | Terminal forms and selection fields | `go.mod` |

### Production Dependencies

| Dependency | Version Policy | Purpose |
|---|---|---|
| `charm.land/bubbletea/v2` | v2.0.2, pinned | Terminal UI runtime |
| `charm.land/huh/v2` | v2.0.3, pinned | Interactive forms |
| `charm.land/bubbles/v2` | v2.0.0, pinned | Huh key bindings |
| `github.com/spf13/cobra` | v1.9.1, pinned | Command tree |
| `github.com/spf13/pflag` | v1.0.6, indirect | Flag parsing |
| `github.com/inconshreveable/mousetrap` | v1.1.0, indirect | Windows click detection |

### Development & Validation Tools

| Tool | Version | Usage |
|---|---|---|
| `go` | 1.26 | build, test, vet |
| `git` | system | repository checks in the Git adapter |

### Dependency & Tool Versioning

- Runtime and tool versions are declared in `go.mod`.
- Dependency versions are resolved and locked by `go.sum`.
- Update policy: not defined.
- Compatibility policy: the current Go CLI is the baseline; client model
  defaults may change with new releases.
- Reproducibility requirements: `go build ./cmd/karl-ai` must produce the same
  binary from a clean checkout.
- The binary version defaults to `dev` and is injectable through a build-time
  `version` variable in `cmd/karl-ai/main.go`. No tagged release exists yet.

## 4. Project Structure & Conventions

### Directory Structure

```text
.
|-- cmd/karl-ai/            binary entrypoint
|-- cli/                    Cobra command tree
|-- tui/                    interactive model selection
|-- core/
|   |-- catalog/            client-neutral content
|   |-- documents/          frontmatter, validators, change templates
|   |-- foundation/         foundation artifacts and assurance levels
|   `-- lifecycle/          states, transitions, gates
|-- adapters/
|   |-- filesystem/         change persistence and atomic writes
|   |-- git/                repository checks
|   `-- opencode/           OpenCode projection
|-- .karl-ai/               source configuration (config.json, manifest.json)
|-- docs/                   foundation artifacts (this document set)
|-- go.mod / go.sum         module and lockfile
|-- README.md               usage and command reference
`-- .opencode/              untracked generated projection (dogfooding)
```

### Directory Responsibilities

| Path | Contains | Must Not Contain |
|---|---|---|
| `cmd/karl-ai` | The `main` package and version variable | Business logic |
| `cli` | Command tree and dependency wiring | Domain rules |
| `core/lifecycle` | States, gates, change operations | Cobra or adapter imports |
| `core/catalog` | Agent, command, skill content | Client model identifiers |
| `adapters/opencode` | OpenCode rendering and manifest | Core domain rules |
| `tui` | Unpersisted interactive model selection | Configuration writes or client catalogs |
| `.karl-ai/` | `config.json`, `manifest.json` (source of truth) | Hand-written projection files |
| `.opencode/` | Generated projection (untracked runtime) | Source configuration |
| `docs/` | Foundation baseline documents | Change plans or code |

### Module Organization

- One Go module. Core packages never import adapters or `github.com/spf13/cobra`.
- Adapters satisfy the small interfaces declared in `core/lifecycle`
  (`Store`, `Git`).

### Naming Conventions

- Change names use lowercase kebab-case.
- Agent and command identifiers use the `karl-*` prefix.
- Adapter packages import core types by full path.

### Configuration Rules

- Source configuration lives in `.karl-ai/config.json`.
- Generated ownership lives in `.karl-ai/manifest.json`.
- Client-specific model identifiers never enter the core catalog.

### Error Handling Conventions

- Domain and adapter errors return descriptive messages.
- The `cli` package returns `ErrValidationFailed` when a gate fails; `main`
  suppresses that message because the JSON was already printed.
- Unknown change options report `Unknown option: <name>`.

### Logging Conventions

- No logging library. Results are JSON on stdout; errors are text on stderr.

## 5. Data Models & Persistence

### Domain Model

- `Change`: a tracked unit of work identified by a kebab-case slug.
- `Lifecycle state`: one of six states with legal transitions.
- `Gate`: a validation result (`ok`, `errors`, `warnings`, `gate`, `state`).
- `Projection`: the set of owned client files described by a manifest.
- `Assurance level`: `L1` through `L4`.

### Persistence Model

| Model | Representation | Reference |
|---|---|---|
| Change package | Five Markdown files with YAML frontmatter | `changes/<name>/` |
| Projection ownership | JSON manifest with SHA-256 hashes | `.karl-ai/manifest.json` |
| Client overrides | JSON config | `.karl-ai/config.json` |
| Shared client settings | JSON document | `.opencode/opencode.json` |

### Persistence Rules

- A change package is `CHANGE.md`, `PLAN.md`, `TASKS.md`, `RESEARCH.md`, and
  `REVIEW.md`.
- `CHANGE.md` frontmatter holds `name`, `state`, `created`, `assurance_level`,
  `depends_on`, `conflicts_with`, `blocked_by`.
- `REVIEW.md` frontmatter holds `implementation_ref`, `user_validation`,
  `foundation_status`, `blocking_findings`.
- Archive moves the package to `changes/archive/<YYYY-MM-DD>-<name>/`.
- The manifest records the Karl version, client, and content hash of every
  generated file.

### Migration Rules

- None. There is no database and no schema migration.

### Data Validation Rules

- Frontmatter is parsed and re-serialized deterministically.
- Required sections are checked per document.
- Placeholders left in a scaffold fail the plan gate.
- Work unit numbers must be sequential and each unit must have Prepare,
  Implement, and Validate tasks.

## 6. Contracts & External Interfaces

### Contract Strategy

| Interface Type | Contract Format | Location |
|---|---|---|
| CLI | Command, flag, output, and exit-code specification | This section |
| Config | JSON schema by Go struct | `adapters/opencode/render.go` |
| Manifest | JSON schema by Go struct | `adapters/opencode/project.go` |
| Projection | Markdown files with frontmatter | `.opencode/` |

### CLI Interface Inventory

| Command | Flags | Output | Exit Codes |
|---|---|---|---|
| `karl-ai init opencode` | `--root` | JSON result | 0 success, 1 error |
| `karl-ai sync opencode` | `--root`, `--check`, `--force` | JSON result | 0 success, 1 error or out of sync |
| `karl-ai uninstall opencode` | `--root`, `--force` | JSON result | 0 success, 1 error |
| `karl-ai models configure` | `--root` | Saved selection and sync JSON on stdout; interactive form on stderr | 0 success, 1 error or cancellation |
| `karl-ai change new <name>` | `--root`, `--level` | JSON result | 0 success, 1 error |
| `karl-ai change list` | `--root`, `--json` | JSON result | 0 success, 1 error |
| `karl-ai change status <name>` | `--root`, `--json` | JSON result | 0 success, 1 error |
| `karl-ai change validate <name> <gate>` | `--root`, `--json` | JSON result | 1 when gate fails |
| `karl-ai change transition <name> <state>` | `--root` | JSON result | 0 success, 1 error |
| `karl-ai change archive <name>` | `--root`, `--date` | JSON result | 0 success, 1 error |
| `karl-ai version` | none | version text | 0 |
| `karl-ai completion <shell>` | none | shell script | 0 |

- `<gate>` is one of `plan`, `implement`, `review`, `archive`.
- `<state>` is one of `planned`, `implementing`, `reviewing`, `validated`.
- `<level>` defaults to `pending` and accepts `L1` through `L4`.
- Noninteractive command output is JSON. The `--json` flag is accepted for
  compatibility but does not change output. `models configure` writes its
  saved selection and automatic sync result as JSON on stdout and its
  interactive form on stderr.
- `sync --check` writes nothing and exits 1 when the projection is not current.

### Lifecycle State Contract

```text
draft -> planned -> implementing -> reviewing -> validated -> archived
Backward: implementing -> planned; reviewing -> implementing | planned
```

Each forward transition runs the matching gate: plan, implement, review.
Archive runs the archive gate and never creates a commit.

### Config Contract

```json
{
  "version": 1,
  "opencode": {
    "agents": { "<agent-id>": { "model": "<id>", "variant": "<optional>" } }
  }
}
```

Unknown agent override identifiers are rejected.

### Manifest Contract

```json
{
  "version": 1,
  "karl_version": "<version>",
  "client": "opencode",
  "files": [ { "path": ".opencode/...", "sha256": "<hex>" } ],
  "opencode": { "...": "previous shared values" }
}
```

Manifest paths must be sorted, relative, and under `.opencode/`. Hashes must be
valid SHA-256 hex. Sync compares current file content to the manifest and to the
desired render to detect drift.

### Shared OpenCode Settings Contract

The projector merges three values into `.opencode/opencode.json` without
deleting user fields: `$schema` is set to `https://opencode.ai/config.json`,
`default_agent` is set to `karl-orchestrator`, and
`permission.task["karl-*"]` is set to `allow`. Uninstall restores the previous
values recorded at init.

### Projection Contract

- Seven agents, five slash commands, and two skills with references and
  templates.
- The orchestrator is `mode: primary`; specialists are `mode: subagent` with
  `hidden: true`.
- Each agent carries scoped edit and task permissions.
- Commands receive `$ARGUMENTS` and never carry an `agent:` frontmatter field.

### Compatibility & Contract Versioning

- Config and manifest carry a `version` field; unsupported versions are
  rejected.
- The OpenCode client format is not pinned to a released OpenCode version.
  Compatibility relies on the current schema URL and frontmatter shape.

## 7. Testing & Code Validation

### Testing Methodology

Tests target domain rules directly and drive the filesystem and Git through
temporary directories. Golden and behavioral checks cover the OpenCode
projection. CLI regression tests fix the noninteractive command output and
exit-code baseline. A scripted model configuration run uses injected discovery
to verify that only the selected agent configuration and projection change.

### Test Levels

| Level | Purpose | Scope | Tool | Location |
|---|---|---|---|---|
| Unit | Frontmatter, validators, catalog, state machine | single package | `go test` | `core/documents`, `core/catalog` |
| Integration | Lifecycle over real temp dirs and Git | cross-package | `go test` | `core/lifecycle` |
| CLI | Command surface, flags, exit behavior | `cli` | `go test` | `cli/root_test.go` |
| TUI | Selection order, discovery fallback, stale models, manual validation, and cancellation | `tui` | `go test` | `tui/configure_test.go` |
| Projection | Render determinism, drift, merge, uninstall | `adapters/opencode` | `go test` | `adapters/opencode/project_test.go` |

### Test Data & Fixtures

- Integration fixtures build a temporary project with `docs/CONTEXT.md` and
  `docs/DESIGN.md`, and a Git repository where required.
- No external test data is used.

### Validation Sequence

```text
Format -> Static analysis -> Unit -> Integration
```

### Quality Gates

- `go vet ./...` must pass.
- `go build ./cmd/karl-ai` must pass.
- All tests must pass: 70 tests across 10 packages after the model selection
  flow was added.

### Local Validation Commands

```text
go test ./...
go vet ./...
go build ./cmd/karl-ai
```

### CI Validation Commands

```text
None. No CI workflow exists yet.
```

## 8. CLI Experience Baseline

- Commands report results as pretty-printed JSON on stdout.
- Errors are single-line messages on stderr.
- `models configure` saves only the selected agent model and variant, runs a
  drift-safe sync, and writes its result as JSON on stdout. It renders the
  interactive form on stderr; cancellation returns exit code 1.
- When OpenCode model discovery fails or returns no choices, `models configure`
  shows the failure and requests a validated manual `provider/model` reference.
  A saved model absent from discovery is shown as configured and stale, so the
  developer can retain or replace it.
- A failing `change validate` prints its JSON result and exits 1.
- `change --help` and subcommand help print a fixed usage text that lists
  states, gates, and options.

## 9. Operational & Quality Baseline

### Security Baseline

- Karl does not manage secrets or authentication. Model discovery delegates any
  network or cache behavior to the installed OpenCode client.
- Agent permissions are scoped per specialist in the projection.

### Configuration & Secrets

- No secrets. `.env*` patterns are ignored.
- Configuration is project-local JSON only.

### Observability

- None beyond command output and exit codes.

### Performance Expectations

- Not defined. Operations are local and expected to be interactive-fast.

### Reliability Requirements

- Projection operations must be idempotent and must not overwrite drift without
  force.
- Writes use atomic file replacement.

### Build & Release Requirements

- Not established. There are no tags, releases, or CI pipelines.
- The only history is one initial commit on `master`.
- Verification has only been run on Windows.

## 10. Risks & Open Items

### Risks

- No CI, tag, or release process exists; version is `dev`.
- OpenCode client compatibility is unpinned to a released OpenCode version.
- Verification is Windows-only; cross-platform behavior is unproven.
- History is a single initial commit; no release baseline exists.
- The `--json` flag on `list`, `status`, and `validate` is a no-op because all
  output is always JSON.

### Open Items (non-blocking)

- Adoption by external developers is unproven.
- Release versioning and injection of the `version` variable are not defined.
- The OpenCode schema and frontmatter compatibility policy is not stated.

## Related Documents

- [System Context](./CONTEXT.md)
- [OpenCode Projection Journey](./journeys/opencode-projection/JOURNEY.md)
- [Change Lifecycle Journey](./journeys/change-lifecycle/JOURNEY.md)
