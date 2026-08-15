---
foundation_state: established
project_state: development
analysis_state: complete
project_mode: existing
assurance_level: L2
last_analyzed_at: "2026-08-15T16:39:53Z"
analysis_summary: "Go 1.26 CLI verified against source: 70 tests pass across 10 packages and go vet and go build pass; no CI, tags, or releases exist."
---

# System Context

This document defines the business context of Karl, the development workflow
that the `karl-ai` command-line tool carries out. Technology and implementation
details live in [DESIGN.md](./DESIGN.md).

## 1. Context Identity & Scope

### Context Name

Karl project workflow.

### Purpose

Karl gives a software team one structured way to plan, implement, review, and
archive changes. It also keeps the team's working baseline, called the
foundation, current. Karl installs a set of specialist assistant agents and
commands into a developer's AI client so that the workflow runs the same way in
every project.

### This Context Covers

- The change lifecycle and its gates.
- The project foundation and its required artifacts.
- The managed installation of Karl agents, commands, and skills into a client.
- The rules that apply across every change and every client.

### This Context Does Not Cover

- The internal implementation of the `karl-ai` tool.
- The Go module layout, dependencies, or build commands.
- The concrete model identifiers or file formats used by a client.
- The visual appearance of the AI client.

### Related Contexts

- None. This repository holds one bounded context.

## 2. Actors & External Systems

### Actors

| Actor | Type | Role In This Context |
|---|---|---|
| Developer | Human | Starts changes, selects the assurance level, authorizes implementation, accepts review, and requests commits. |
| Karl Orchestrator | Non-human | The single user-facing agent. Routes work, owns gates and decisions, and talks to the user. |
| Karl Searcher | Non-human | Collects research evidence for a change or a foundation. |
| Karl Foundation | Non-human | Establishes and refreshes the project foundation. |
| Karl Planner | Non-human | Builds a researched change plan and passes the plan gate. |
| Karl Implementer | Non-human | Implements one vertical work unit at a time and passes the implement gate. |
| Karl Reviewer | Non-human | Reviews behavior, records evidence, and owns the review gate. |
| Karl Archiver | Non-human | Archives a validated change and prepares the final integration commit. |

### External Systems

| External System | Used For |
|---|---|
| OpenCode | Hosts the projected Karl agents, commands, and skills, and merges Karl's shared settings. |
| Git | Checks repository state, resolves commit references, and lists uncommitted files. |

### Boundary Rules

- The client-neutral catalog is the single source of truth for Karl content.
- Client directories are generated projections. A developer does not edit them
  by hand.
- Karl never creates a commit without an explicit user request.
- The developer is the only actor that accepts review and authorizes commits.

## 3. Domain Concepts, Language & Relationships

### Canonical Terms

| Term | Definition | Required Usage |
|---|---|---|
| Foundation | The project working baseline: `docs/CONTEXT.md`, `docs/DESIGN.md`, and journeys. | Required before a change is created. |
| Change | One tracked unit of behavior-changing work. | Use for all work that changes behavior. |
| Change package | The directory `changes/<name>` that holds the five change artifacts. | Use for the tracked folder. |
| Assurance level | One of `L1`, `L2`, `L3`, or `L4`. The depth of scrutiny applied to a change. | Select one at creation. |
| Gate | A validation check that must pass before a state transition. | Four gates: plan, implement, review, archive. |
| Lifecycle state | One of `draft`, `planned`, `implementing`, `reviewing`, `validated`, `archived`. | Stored in `CHANGE.md` frontmatter. |
| Projection | The installed, managed copy of Karl content in a client, such as `.opencode/`. | Use for generated client output. |
| Drift | A managed file whose content no longer matches the recorded source. | Never overwrite without explicit force. |
| Work unit | One vertical slice of implementation with Prepare, Implement, and Validate tasks. | Use in plans and tasks. |

### Core Concepts

- The lifecycle moves a change from draft to archived through four gates.
- The foundation is established once and refreshed when the code or contracts
  change.
- The projection is installed, checked, and removed through three operations:
  init, sync, and uninstall.

### Relationships

- A foundation is required before any change.
- A change may depend on other changes. Dependencies must be archived first.
- A projection is derived from the catalog. The manifest records what the
  projection owns.

## 4. Business Capabilities & Journey Map

### Capabilities

| Capability | Purpose |
|---|---|
| Projection management | Install, keep current, and remove Karl agents, commands, and skills in a client. |
| Change lifecycle | Create, validate, transition, and archive tracked changes. |
| Foundation establishment | Establish and refresh the working baseline. |

### Journeys

| Journey | Trigger | Actors | Outcome | Specification |
|---|---|---|---|---|
| OpenCode projection | Developer runs `init`, `sync`, or `uninstall` | Developer | A current or removed managed projection | [JOURNEY.md](./journeys/opencode-projection/JOURNEY.md) |
| Change lifecycle | Developer runs `change new` | Developer, Karl specialists | A validated and archived change | [JOURNEY.md](./journeys/change-lifecycle/JOURNEY.md) |

## 5. Business Rules & Invariants

### Business Rules

- A foundation must exist before a change is created.
- A change moves only along the legal state transitions.
- No commit is created automatically. Implementation and archive commits require
  an explicit user request.
- A dependency must be archived before the change that depends on it.
- Managed client files that drift are not overwritten without explicit force.
- Change names use lowercase kebab-case.
- The client-neutral catalog is the source of truth. Client directories are
  generated.

### Invariants

- Lifecycle state lives in `CHANGE.md` frontmatter, never in conversation.
- Archive moves the package and never creates a commit.
- Uninstall keeps files and settings not owned by Karl.

## 6. Business Outcomes & Side Effects

### Expected Outcomes

- A current, valid client projection.
- A validated and archived change.
- A coherent foundation.

### Significant Side Effects

| Side Effect | Caused By | Business Consequence |
|---|---|---|
| Client files are written into `.opencode/` | init or sync | The client gains Karl agents, commands, and skills. |
| Managed files are replaced | sync with force | Hand edits to managed files are lost. |
| `CHANGE.md` frontmatter is updated | transition or archive | The change moves to a new state. |
| The package is moved to `changes/archive/` | archive | The change leaves the active set. |

## Related Documents

- [System Design](./DESIGN.md)
- [OpenCode Projection Journey](./journeys/opencode-projection/JOURNEY.md)
- [Change Lifecycle Journey](./journeys/change-lifecycle/JOURNEY.md)
