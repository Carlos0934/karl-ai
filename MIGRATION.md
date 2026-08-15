# Karl Minimal System Migration

This document is the complete migration contract for reducing Karl to its
smallest useful form.

The migration is intentionally destructive. Obsolete workflow components will
be deleted, not deprecated, emulated, archived, or preserved behind compatibility
layers. Git history is the archive.

## 1. Decision Summary

Karl stops being a change-lifecycle system. It becomes:

1. A small set of explicit AI agents and portable skills.
2. A client-neutral catalog projected through client adapters.
3. A configuration tool for installing Karl and selecting agent models.

The target system keeps a simplified `karl-orchestrator`, but the orchestrator
is no longer a workflow engine. It routes only explicit user intent and never
assumes that a request is a tracked change.

| Area | Target decision |
|---|---|
| User-facing stages | Explore, Plan, Implement |
| Primary entry | Simplified `karl-orchestrator` |
| Direct agent access | `karl-explore`, `karl-plan`, and `karl-implement` remain directly selectable |
| Automatic chaining | None |
| Change lifecycle | Removed |
| `changes/` | Removed |
| Foundation workflow | Removed |
| Foundation documents | Not required or managed by Karl |
| Assurance levels and gates | Removed |
| Review and archive agents | Removed |
| Canonical commands | Removed |
| Canonical behavior | Portable skills |
| Temporary work documents | `RESEARCH.md` and `PLAN.md` only |
| Work document location | `.karl-ai/work/`, ignored by Git |
| Projection manifest | Removed |
| Project configuration | `.karl-ai/config.json` remains |
| Product responsibility | Configure Karl and adapt the same catalog to supported AI clients |

## 2. Why We Are Deleting the Current System

The current system carries more process than the product needs:

- Seven workflow agents.
- Five change artifacts for every change.
- Six lifecycle states.
- Four validation gates.
- A required `changes/` hierarchy and archive.
- Foundation state, project state, analysis state, assurance levels, timestamps,
  journeys, and reconciliation.
- A manifest with hashes for every projected file.
- Repeated state across Markdown frontmatter, generated files, Git, and
  conversation.

This creates synchronization work instead of user value. A small change can
require research, planning, tasks, review evidence, state transitions, commits,
archive movement, and foundation reconciliation before it is considered done.

The migration accepts the loss of this process. Karl will rely on explicit user
intent, temporary working documents, automated tests, and Git history.

### Demolition principles

- Prefer deletion over compatibility wrappers.
- Keep one source of truth for every behavior.
- Do not preserve lifecycle state after the lifecycle is removed.
- Do not replace `changes/` with another renamed change-package hierarchy.
- Do not replace the manifest with another hash inventory.
- Do not make optional project documentation a gate.
- Do not add a new agent when one skill or one bounded worker instance is enough.
- Do not make agents chain automatically.

## 3. Target Product Responsibility

`karl-ai` remains a native Go CLI, but its responsibility narrows.

### Karl continues to do

- Install Karl agents and skills into a supported AI client.
- Synchronize Karl-owned generated files after an upgrade.
- Remove Karl-owned generated files.
- Store project-local agent model configuration.
- Discover clients, providers, models, and variants.
- Offer `karl-ai models configure` for interactive model selection.
- Keep a client-neutral catalog and client-specific adapters.
- Add future clients without copying the behavioral contracts.

### Karl stops doing

- Creating or managing change packages.
- Tracking lifecycle states.
- Validating plan, implementation, review, or archive gates.
- Resolving implementation commit references.
- Managing change dependencies or conflicts.
- Creating or reconciling foundation documents.
- Selecting assurance levels.
- Maintaining review evidence.
- Archiving changes.
- Treating every behavior request as a formal change.

### Target CLI

```text
karl-ai init <client> [--root PATH]
karl-ai sync <client> [--root PATH] [--check] [--force]
karl-ai uninstall <client> [--root PATH] [--force]
karl-ai models configure [--root PATH]
karl-ai version
karl-ai completion <shell>
```

The complete `karl-ai change ...` command tree is removed.

## 4. Target Runtime Model

### Public agents

| Agent | Mode | Responsibility |
|---|---|---|
| `karl-orchestrator` | Primary | Interpret explicit user intent and route to Explore, Plan, or Implement |
| `karl-explore` | All | Research one objective and write a closed recommendation |
| `karl-plan` | All | Convert completed research into a functional specification |
| `karl-implement` | All | Implement and validate the functional specification |

`mode: all` allows Explore, Plan, and Implement to be selected directly or
called by the orchestrator when the client supports both uses.

### Internal agents

| Agent | Mode | Responsibility |
|---|---|---|
| `karl-explore-worker` | Hidden subagent | Research one bounded evidence track |
| `karl-plan-worker` | Hidden subagent | Execute one assigned stage against the shared `PLAN.md` |

Internal agents do not ask the user, delegate, write product code, or own final
documents beyond their assigned write boundary.

### Canonical skills

| Skill | Loaded by | Responsibility |
|---|---|---|
| `karl-explore` | `karl-explore` | Coordinate direct or explicit parallel research and write `RESEARCH.md` |
| `karl-explore-track` | `karl-explore-worker` | Return evidence for one bounded research track |
| `karl-plan` | `karl-plan` | Coordinate direct or explicit multi-stage planning |
| `karl-plan-build` | `karl-plan-worker` | Create the initial functional `PLAN.md` |
| `karl-plan-challenge` | `karl-plan-worker` | Add missing, unsafe, duplicate, and failure behavior |
| `karl-plan-simplify` | `karl-plan-worker` | Remove redundancy without removing required behavior |
| `karl-implement` | `karl-implement` | Implement and validate the current research and plan |

Skills are the canonical behavior contracts. Agent files contain identity,
permissions, and the instruction to load the correct skill. Client-specific
commands may exist as optional aliases, but they are not canonical catalog
content.

## 5. Simplified Orchestrator

The orchestrator remains because a single conversational entry is useful. Its
authority is deliberately small.

### Orchestrator responsibilities

- Read the user request.
- Detect an explicit request to explore, plan, or implement.
- Delegate only the requested stage.
- Ask one short question when the requested stage is ambiguous.
- Recommend Explore when the user asks for analysis without a selected stage.
- Report the selected agent result to the user.
- Request explicit permission before any commit.

### Orchestrator prohibitions

- Do not create lifecycle state.
- Do not create `changes/` content.
- Do not assume that a new idea is a tracked change.
- Do not run Explore, then Plan, then Implement automatically.
- Do not create research or plan documents itself.
- Do not perform review, archive, or foundation reconciliation.
- Do not maintain a conversation-level workflow state machine.

### Routing behavior

```text
User explicitly asks to explore
    -> karl-explore

User explicitly asks to plan
    -> karl-plan

User explicitly asks to implement
    -> karl-implement

User request is ambiguous
    -> ask: Explore, Plan, or Implement?
```

Agents remain directly selectable. The orchestrator is a convenience, not a
mandatory gateway.

## 6. Explore

### Direct exploration

`karl-explore` researches the objective directly and writes one final document.

### Explicit parallel exploration

Parallel exploration is enabled only when the user includes `--parallel` or
explicitly requests parallel exploration.

```text
karl-explore --parallel
    ├── repository track
    ├── documentation track
    └── runtime verification track
```

Rules:

- Use two or three workers.
- Use one delegation level.
- Workers do not write files.
- Workers do not ask the user.
- Workers return verified answers, resources, and risks.
- Only `karl-explore` writes `RESEARCH.md`.
- `karl-explore` resolves contradictory evidence.
- `karl-explore` asks the user when evidence cannot resolve a consequential
  product decision.
- The final research contains one selected conclusion.

### Research escalation

Explore asks the user when the missing answer changes:

- Product behavior.
- Scope.
- Compatibility.
- Security or privacy.
- Destructive behavior.
- Cost or required external services.
- Authority between a human, machine, and agent.
- A contradiction that remains after verification.

Explore does not ask when the answer is available in source, tests,
configuration, documentation, or a safe runtime check.

## 7. Research Document Contract

Path:

```text
.karl-ai/work/RESEARCH.md
```

Format:

```markdown
# Research

## Goal

<The question, system, constraints, and completion condition.>

## Conclusion

<One selected direction that directly answers the Goal.>

## Verified Answers and Resources

| Verified answer | Resource | What it proves |
|---|---|---|

## Risks and Mitigations

| Risk or limitation | Effect | Chosen mitigation |
|---|---|---|
```

Quality rules:

- The Goal identifies what must be answered and when research is complete.
- The Conclusion selects one direction instead of transferring choices to Plan.
- Every verified answer has a precise path, test, command, or authoritative URL.
- `What it proves` explains why the resource supports the answer.
- Every important risk has one chosen mitigation.
- Consequential user decisions are resolved before the final file is written.
- The file uses four sections and stays below 2,000 lines.

## 8. Plan

### Direct planning

Without explicit multi-stage activation, `karl-plan` reads `RESEARCH.md` and
writes `PLAN.md` directly.

### Explicit multi-stage planning

Multi-stage planning is enabled only when the user includes `--multi-stage` or
explicitly requests multi-stage planning.

```text
Domain Builder
    -> Scenario Challenger
        -> Simplifier
            -> Karl Plan validation
```

The stages share one `PLAN.md`. They do not write concurrently.

### Stage 1: Build

The Build worker loads `karl-plan-build` and creates the first complete
functional plan.

It defines:

- Context.
- Human, machine, and agent actors.
- Functional concepts.
- One desired-flow ASCII diagram.
- Outcome.
- Scope.
- Core Given/When/Then scenarios.

### Stage 2: Challenge

The Challenge worker loads `karl-plan-challenge`, reads the shared file, and
adds or corrects behavior for:

- Rejection.
- Uncertainty.
- Duplicate actions and events.
- Unavailable actors.
- External failure.
- Events that occur out of order.
- Conflicting human, machine, and agent authority.
- Desired-flow paths without a scenario.
- Scenarios outside Outcome or Scope.

### Stage 3: Simplify

The Simplify worker loads `karl-plan-simplify`, reads the challenged file, and
removes:

- Duplicate concepts.
- Multiple names for the same concept.
- Actors without responsibility.
- Context that does not affect behavior.
- Equivalent scenarios.
- Technical terminology.
- Unnecessary flow complexity.

Simplification cannot remove behavior required by Outcome, Scope, a desired
flow path, or an accepted risk mitigation.

### Shared-file rule

```text
Workers total: 3
Concurrent writers: 0
Order: Build -> Challenge -> Simplify
Shared document: .karl-ai/work/PLAN.md
Final validator: karl-plan
```

## 9. Plan Document Contract

Path:

```text
.karl-ai/work/PLAN.md
```

Format:

````markdown
# Plan

## Context

<Current situation, opportunity, or requested capability.>

**Actors**

| Actor | Type | Responsibility |
|---|---|---|

**Concepts**

| Concept | Meaning |
|---|---|

**Desired Flow**

```text
<Small human-readable ASCII flow>
```

## Outcome

<Observable functional result.>

## Scope

Included:

- <Included behavior>

Excluded:

- <Explicit boundary>

## Specification

**Scenario 1 — <Observable behavior>**

Given:

- <Initial functional condition>

When:

- <Actor action or domain event>

Then:

- <Observable result>
````

Plan rules:

- Context appears before Outcome.
- Actors are Human, Machine, or Agent.
- Every actor performs an action, provides information, has authority, or
  receives an outcome.
- Concepts use domain language, not class, endpoint, table, queue, or service
  names.
- The desired flow is an orientation aid, not a state matrix.
- Specification uses the same actor and concept names defined in Context.
- Every important desired-flow path has a scenario.
- Scenarios define behavior, not implementation.
- The file has four sections and stays below 2,000 lines.
- `PLAN.md` is immutable during implementation.

## 10. Implement

`karl-implement` reads:

```text
.karl-ai/work/RESEARCH.md
.karl-ai/work/PLAN.md
```

It does not edit either document.

Responsibilities:

1. Read Research for verified evidence, constraints, and the selected direction.
2. Read Plan for Context, Outcome, Scope, and Specification.
3. Inspect relevant source and tests.
4. Ask the user when an implementation choice changes observable behavior,
   compatibility, security, destructive effects, or material cost.
5. Select conventional reversible implementation details directly.
6. Modify product code and tests.
7. Validate the specified scenarios.
8. Report behavior, files, commands, results, blockers, and commit readiness in
   conversation.

Implement does not create `TASKS.md`, `REVIEW.md`, execution state, or a new
working document. Git diff and tests are the implementation evidence.

## 11. Temporary Workspace

The only Karl working files are:

```text
.karl-ai/work/
├── RESEARCH.md
└── PLAN.md
```

The directory is ignored:

```gitignore
/.karl-ai/work/
```

Properties:

- One active research and plan pair.
- No archive.
- No state frontmatter.
- No timestamps or assurance level.
- No concurrent work packages.
- Starting a new objective replaces the current temporary documents only after
  explicit user intent.

`.karl-ai/config.json` remains project configuration and may remain tracked.

## 12. Client-Neutral Catalog and Adapters

The catalog remains the source of portable Karl behavior:

```text
catalog
├── agents
└── skills
```

Canonical commands are removed. A client adapter may generate optional command
aliases when the client supports them, but the alias contains no behavioral
contract.

Each adapter owns:

- Client file paths.
- Agent and skill syntax.
- Model and variant fields.
- Permission translation.
- Client discovery.
- Installation, synchronization, and removal.

The core catalog does not contain client model identifiers or client paths.

### Future client contract

A future adapter must be able to:

1. Render the six agent definitions.
2. Render the seven skills and their local support files.
3. Store model selections through the client section of `.karl-ai/config.json`.
4. Discover available providers, models, and variants when supported.
5. Synchronize only Karl-owned output.
6. Remove only Karl-owned output.

## 13. Projection Without a Manifest

`.karl-ai/manifest.json` is removed.

The replacement ownership contract is intentionally smaller:

- Karl writes only fixed, namespaced paths such as `karl-*` agents and skills.
- Every generated file contains a stable `Generated by karl-ai` marker.
- Sync overwrites only a known Karl path that contains the marker.
- Sync refuses to overwrite a known path without the marker unless the user
  explicitly forces it.
- Uninstall deletes only known paths that contain the marker.
- Karl does not maintain a file hash inventory.
- Karl does not attempt to restore arbitrary historical user content.

### Shared client configuration

Karl should avoid modifying shared client configuration that it cannot safely
restore without a manifest. For OpenCode, generated agents and their own
frontmatter carry their permissions and model configuration.

The adapter should not require `default_agent` or global task-permission changes
in `opencode.json`. Users can select the simplified orchestrator or one of the
three stage agents directly.

This is an accepted tradeoff: simpler ownership and uninstall behavior are more
important than automatically changing shared client defaults.

## 14. Repository Deletion Map

### Delete lifecycle code

```text
core/lifecycle/
```

This removes state transitions, gates, change operations, dependency ordering,
archive checks, and lifecycle integration tests.

### Delete change document code

```text
core/documents/
```

This removes change frontmatter, templates, validators, and their tests. Keep no
replacement validator for temporary work documents unless a concrete failure
proves one is necessary.

### Delete foundation code

```text
core/foundation/
```

This removes foundation templates, assurance concepts, required artifacts, and
foundation gates.

### Delete obsolete adapters

```text
adapters/git/
adapters/filesystem/store.go
```

Keep the atomic-write and managed-file primitives required by configuration and
projection. Remove Git and change-package persistence when no remaining caller
uses them.

### Delete change content

```text
changes/
```

Delete active and archived change packages. Git history already preserves their
content.

### Delete foundation documents

```text
docs/CONTEXT.md
docs/DESIGN.md
docs/journeys/change-lifecycle/
docs/journeys/opencode-projection/
```

Karl no longer creates, validates, requires, or reconciles project foundation
documents. Project teams may keep their own documentation, but it is outside
Karl's contract.

The existing architecture diagram is removed or replaced only if it accurately
describes the minimal target system. It is never a gate.

### Delete obsolete catalog agents

```text
karl-searcher
karl-foundation
karl-planner
karl-implementer
karl-reviewer
karl-archiver
```

Replace them with:

```text
karl-orchestrator
karl-explore
karl-explore-worker
karl-plan
karl-plan-worker
karl-implement
```

### Delete obsolete commands

```text
karl-change-new
karl-change-implement
karl-change-review
karl-change-archive
karl-foundation
```

No canonical replacement commands are required.

### Delete obsolete skills

```text
karl-change-lifecycle
karl-project-foundation
```

Replace them with the seven target skills listed in this document.

### Delete the manifest

```text
.karl-ai/manifest.json
```

Replace hash ownership with fixed namespaced paths and generated markers.

## 15. Target Repository Shape

```text
.
├── cmd/karl-ai/
├── cli/
├── core/
│   └── catalog/
│       └── content/
│           ├── agents/
│           │   ├── karl-orchestrator.md
│           │   ├── karl-explore.md
│           │   ├── karl-explore-worker.md
│           │   ├── karl-plan.md
│           │   ├── karl-plan-worker.md
│           │   └── karl-implement.md
│           └── skills/
│               ├── karl-explore/
│               ├── karl-explore-track/
│               ├── karl-plan/
│               ├── karl-plan-build/
│               ├── karl-plan-challenge/
│               ├── karl-plan-simplify/
│               └── karl-implement/
├── adapters/
│   ├── filesystem/
│   └── opencode/
├── tui/
├── .karl-ai/
│   ├── config.json
│   └── work/                ignored
│       ├── RESEARCH.md
│       └── PLAN.md
├── README.md
├── MIGRATION.md
├── go.mod
└── go.sum
```

There is no `changes/`, lifecycle package, foundation package, manifest,
canonical command catalog, review agent, or archive agent.

## 16. Migration Sequence

The migration uses reviewable implementation slices, but it does not create a
new change package. This `MIGRATION.md` is the migration contract.

### Slice 1: Introduce the target behavioral catalog

- Add the six target agent definitions.
- Add the seven target skills.
- Add Research and Plan templates and quality references.
- Add catalog tests for names, modes, permissions, and skill assignments.
- Keep the old system temporarily so the new catalog can be verified.

Complete when:

- The new agents and skills render correctly.
- Explore direct and parallel contracts are testable.
- Plan direct and multi-stage contracts are testable.
- Implement can read the temporary documents without mutating them.

### Slice 2: Add the temporary workspace

- Add `/.karl-ai/work/` to `.gitignore`.
- Ensure Explore can write only `RESEARCH.md`.
- Ensure Plan stages can write only `PLAN.md`.
- Ensure Implement cannot edit either document.
- Verify the 2,000-line limits through agent contracts, not a new lifecycle
  validator.

Complete when:

- A user can run Explore, Plan, and Implement with one active work pair.
- No work document appears in Git status.

### Slice 3: Simplify the orchestrator

- Replace lifecycle phase detection with explicit Explore, Plan, Implement
  routing.
- Remove state reads, artifact reconciliation, review acceptance, archive, and
  commit-flow logic.
- Keep explicit commit authorization.
- Test ambiguous-request questioning and explicit routing.

Complete when:

- The orchestrator never assumes a tracked change.
- No agent is chained automatically.
- Direct agent selection remains possible.

### Slice 4: Simplify projection ownership

- Add generated ownership markers.
- Replace manifest hash checks with fixed path and marker checks.
- Stop modifying shared `opencode.json` defaults and task permissions.
- Update sync, check, force, and uninstall behavior.
- Remove manifest generation and parsing.

Complete when:

- Init and sync produce the target six agents and seven skills.
- Check detects missing, outdated, or unowned paths without a manifest.
- Force replaces only explicitly targeted Karl paths.
- Uninstall removes only marker-owned Karl files.
- `.karl-ai/manifest.json` is not created.

### Slice 5: Remove lifecycle and foundation

- Remove the `change` CLI tree.
- Remove lifecycle, document, foundation, Git, and change-store code.
- Remove old lifecycle and foundation tests.
- Remove old agents, commands, skills, templates, and references.
- Delete `changes/` and foundation documents.

Complete when:

- No production import references removed packages.
- CLI help contains no lifecycle command.
- Repository search finds no lifecycle state, gate, foundation-state, assurance,
  review, archive, or change-package contract in active product content.

### Slice 6: Reconcile the product surface

- Rewrite README around installation, client projection, model configuration,
  and the explicit agents.
- Update the architecture diagram or remove it.
- Remove obsolete dependencies.
- Regenerate the OpenCode projection.
- Run the complete quality suite.

Complete when:

- Documentation describes only the minimal system.
- Generated OpenCode content matches the target catalog.
- The working tree contains no obsolete generated lifecycle content.

## 17. Validation Contract

### Catalog

- Exactly six Karl agents exist.
- Exactly seven Karl skills exist.
- No canonical Karl commands exist.
- Only the orchestrator is primary-only.
- Explore, Plan, and Implement are directly selectable and delegable.
- Workers are hidden subagents.
- Each agent can load only its assigned skills.

### Explore

- Direct Explore creates one four-section `RESEARCH.md`.
- `--parallel` creates two or three read-only workers.
- Workers do not write files or ask the user.
- Research contains one selected conclusion.
- Verified answers contain precise resources.
- Consequential decisions are resolved with the user.

### Plan

- Direct Plan creates one four-section `PLAN.md`.
- `--multi-stage` runs Build, Challenge, and Simplify in order.
- Only one stage writes at a time.
- All stages edit the same `PLAN.md`.
- Context defines actors, concepts, and a readable ASCII flow.
- Specification uses strict Context vocabulary.
- Important flow paths have Given/When/Then scenarios.
- No technical design or implementation state appears in Plan.

### Implement

- Implement reads Research and Plan without modifying them.
- Implement changes only product and test files in approved scope.
- Implement asks the user for consequential implementation decisions.
- Implement reports validation evidence in conversation.
- Implement creates no commit without explicit user authorization.

### CLI and adapters

- `init`, `sync`, `uninstall`, `models configure`, `version`, and completion work.
- `change` commands do not exist.
- OpenCode discovery and model configuration remain functional.
- Future client adapters can render the same agent and skill catalog.
- No manifest is created.
- Marker-based sync and uninstall preserve non-Karl files.

### Repository cleanup

- `changes/` does not exist.
- `core/lifecycle/` does not exist.
- `core/foundation/` does not exist.
- Change templates and validators do not exist.
- Foundation workflow content does not exist.
- `.karl-ai/manifest.json` does not exist.
- `.karl-ai/work/` is ignored.
- README and generated projection describe only the target system.

### Project quality

```text
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/karl-ai
go mod tidy -diff
go mod verify
```

All commands must pass.

## 18. Accepted Tradeoffs

This migration intentionally gives up:

- Durable per-change research and plans.
- Lifecycle gates and state transitions.
- Dedicated review and archive agents.
- Assurance-level questioning.
- Required project foundations and journeys.
- Change dependency and conflict tracking.
- Manifest hash-based drift detection.
- Automatic restoration of shared client configuration.
- A project-local archive of completed change packages.

The replacement is deliberately smaller:

- Explicit Explore, Plan, and Implement intent.
- Two temporary human-readable documents.
- Functional Given/When/Then specifications.
- Automated tests and runtime checks.
- Explicit commit authorization.
- Git history.
- A portable catalog rendered by client adapters.

## 19. Final Definition of Done

The migration is complete when a new user can:

1. Install Karl for a supported client.
2. Configure agent models through the TUI.
3. Select the simplified orchestrator or a stage agent directly.
4. Run Explore directly or with explicit `--parallel` research.
5. Produce one closed temporary `RESEARCH.md`.
6. Run Plan directly or with explicit `--multi-stage` planning.
7. Produce one functional temporary `PLAN.md`.
8. Run Implement against the immutable Research and Plan documents.
9. Validate behavior and request a commit explicitly.
10. Synchronize or uninstall Karl without a lifecycle, foundation, change
    package, or manifest.

At that point Karl is no longer a process framework. It is a small configurable
agent system and a general client adapter.
