# Journey: Change Lifecycle

## Purpose

Move one tracked change from creation through planning, implementation, review,
and validation to archive, without creating a commit automatically.

## Trigger

The developer runs `karl-ai change new <name>` in a project that has a
foundation.

## Actors

### Initiating Actor

- Developer.

### Participating Actors

- Karl Searcher, Karl Planner, Karl Implementer, Karl Reviewer, and Karl
  Archiver complete the change artifacts and gates.
- Karl Orchestrator routes the work and asks the user for decisions.

### Receiving Actors

- Developer: receives a validated and archived change and a proposed commit.

## Business Context

A change package lives in `changes/<name>/` and holds five Markdown artifacts.
Lifecycle state lives in `CHANGE.md` frontmatter. Four gates control forward
movement.

## Inputs

| Input | Provided By | Meaning |
|---|---|---|
| `<name>` | Developer | Kebab-case change slug |
| `--level` | Developer | Assurance level `L1` through `L4` |
| `--root` | Developer | Project directory |
| Foundation | Project | `docs/CONTEXT.md` and `docs/DESIGN.md` must exist |

## Preconditions

- A foundation exists.
- The change name is lowercase kebab-case.
- For implement and later gates, the project is a Git repository.

## Actor Interactions

### CLI Interface

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai change new <name> [--level L1|L2|L3|L4] [--root PATH]` |
| Arguments and Options | `<name>`, `--level`, `--root` |
| Standard Output | JSON: `name`, `state`, `path` |
| Standard Error | Error text on failure |
| Exit Codes | 0 success, 1 error |
| Business Result | A draft change package with five scaffolded files |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai change validate <name> <gate> [--root PATH]` |
| Arguments and Options | `<name>`, `<gate>` |
| Standard Output | JSON: `ok`, `errors`, `warnings`, `gate`, `state` |
| Standard Error | None for a failed gate; message already in JSON |
| Exit Codes | 0 pass, 1 fail |
| Business Result | The named gate passes or lists its errors |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai change transition <name> <state> [--root PATH]` |
| Arguments and Options | `<name>`, `<state>` |
| Standard Output | JSON: `name`, `from`, `to` |
| Standard Error | Error text on illegal transition |
| Exit Codes | 0 success, 1 error |
| Business Result | `CHANGE.md` state updates after the gate passes |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai change archive <name> [--root PATH] [--date YYYY-MM-DD]` |
| Arguments and Options | `<name>`, `--date` |
| Standard Output | JSON: `name`, `state`, `path`, `commitRequired` |
| Standard Error | Error text on blocked archive |
| Exit Codes | 0 success, 1 error |
| Business Result | Package moves to `changes/archive/<date>-<name>/` |

| Field | Value |
|---|---|
| Actor | Developer |
| Command | `karl-ai change list` and `karl-ai change status <name>` |
| Arguments and Options | `--root`, `--json` |
| Standard Output | JSON list or status with gates |
| Standard Error | Error text on failure |
| Exit Codes | 0 success, 1 error |
| Business Result | Active and archived changes, or one change's gate results |

## Actor Experience & Interface Behavior

### Experience Goal

The developer sees one clear state and gate result at every step, and never has
a commit created without asking.

### Experience States

| State | Actor Perceives | Actor Can Do | Expected Feedback |
|---|---|---|---|
| Initial | Draft scaffold with placeholders | Fill research, plan, tasks | Plan gate reports placeholders |
| Processing | Specialists working | Wait or interrupt | Gate results on command |
| Validation Error | Gate error list | Fix artifacts and re-run | Named errors, not generic |
| Success | Gate passes, state moves | Continue or authorize | JSON with new state |
| Failure | Illegal transition or blocked archive | Read error and retry | Single-line error |

### Interaction Requirements

- State is read from files, never assumed from conversation.
- A failing gate lists every error so the developer can fix them together.

## Main Business Flow

1. The developer creates a change with a name and level.
2. The searcher completes `RESEARCH.md`; the planner completes `PLAN.md` and
   `TASKS.md`.
3. The developer transitions the change through planned, implementing, and
   reviewing; each step runs its gate.
4. The reviewer records evidence and the developer accepts the review.
5. The change reaches validated and the archive gate passes.
6. The archiver moves the package to the archive and proposes a commit.

## Alternate Flows

### Backward Transition

1. The developer moves a change from implementing back to planned, or from
   reviewing back to implementing or planned.
2. No gate runs on a backward move.

### Change With Dependencies

1. The change declares `depends_on` in its frontmatter.
2. Archive requires each dependency to already be archived.

### Foundation Reconciliation

1. The review records `foundation_status` as `synced` or `not-required`.
2. Archive checks that the change's `foundation/` folder matches the recorded
   status.

## Failure Flows

### Missing Foundation

1. The developer runs `change new` before the foundation exists.
2. The tool reports the missing artifacts and creates nothing.

### Illegal Transition

1. The developer requests a state not reachable from the current state.
2. The tool rejects the request and reports the invalid transition.

### Blocked Archive

1. Dependencies are not archived, or conflicts remain, or unrelated files are
   uncommitted.
2. The tool refuses and reports each blocker.

### Unresolvable Reference

1. `REVIEW.md` names an implementation reference that does not resolve to a
   commit.
2. The implement or later gate fails with that reference.

## Business Rules Applied

- A foundation must exist before a change is created.
- A change moves only along legal transitions.
- Dependencies must be archived first.
- No commit is created automatically.

## State Changes

| Entity or Concept | Previous State | New State | Condition |
|---|---|---|---|
| Change | draft | planned | plan gate passes |
| Change | planned | implementing | plan gate passes |
| Change | implementing | reviewing | implement gate passes |
| Change | reviewing | validated | review gate passes |
| Change | validated | archived | archive gate passes |

## Side Effects

| Side Effect | Caused By | Business Consequence | Reversible |
|---|---|---|---|
| Five files created under `changes/<name>/` | new | Change is trackable | Yes |
| `CHANGE.md` state updated | transition | Change moves forward | Yes |
| Package moved to archive | archive | Change leaves active set | Yes |
| No commit created | archive | Integration is deferred to the user | Yes |

## Expected Outcomes

- A validated change package with complete artifacts.
- An archived package and a proposed integration commit.

## Postconditions

- The package is under `changes/archive/<date>-<name>/`.
- No commit was created by the tool.

## Acceptance Criteria

- `change new` refuses without a foundation.
- `change validate` returns `ok: true` only when all gate rules pass.
- `change transition` rejects illegal states.
- `change archive` moves the package and never creates a commit.

## Related Journeys

- [OpenCode Projection Lifecycle](./../opencode-projection/JOURNEY.md)

## Related Documents

- [System Context](../../CONTEXT.md)
- [System Design](../../DESIGN.md)
