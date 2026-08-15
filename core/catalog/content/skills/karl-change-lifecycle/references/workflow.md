# Karl Change Workflow

Track changes that affect behavior, contracts, data, architecture, dependencies,
or journeys. Do not require a change package for formatting, typos, or comments
with no behavioral effect.

## Package

```text
changes/
|-- <change-name>/
|   |-- CHANGE.md
|   |-- PLAN.md
|   |-- TASKS.md
|   |-- RESEARCH.md
|   |-- REVIEW.md
|   `-- foundation/     # Only when applicable
`-- archive/
    `-- YYYY-MM-DD-<change-name>/
```

| File | Single Responsibility |
|---|---|
| `CHANGE.md` | Outcome, scope, acceptance, state, and relationships |
| `PLAN.md` | Decision basis citing research, technical approach, work-unit map, and integrated validation |
| `TASKS.md` | Self-contained vertical work units, their dependencies, and validation contracts |
| `RESEARCH.md` | Repository, documentation, and external evidence collected before planning |
| `REVIEW.md` | Actual evidence, findings, user validation, foundation reconciliation, and archive decision |
| `foundation/` | Reduced normal-form foundation artifacts affected by the change |

RESEARCH.md is the single source of repository and external evidence. PLAN.md
must not duplicate facts: each `Decision Basis` bullet cites a RESEARCH.md
section with `cites: <Section Name>`, and the plan gate verifies that every
cited section exists and has content. Work-unit dependencies are declared only
in TASKS.md, never in the PLAN.md work-unit map.

## States

```text
draft -> planned -> implementing -> reviewing -> validated -> archived
                         ^              |
                         +--------------+
```

Backward transitions are allowed from `implementing` to `planned`, and from
`reviewing` to `implementing` or `planned`. A validated change may return to
reviewing before archive. Use backward transitions when implementation teaches
something that invalidates the plan or evidence.

## Gates

### Plan

- Required files and sections are complete.
- Assurance level is selected.
- At least one valid work unit exists.
- No unresolved blocker prevents planning.

### Implement

- Plan gate still passes.
- The developer explicitly authorizes implementation.
- Declared dependencies are available.

### Review

- Every task checkbox is complete.
- Implementation is committed.
- The implementation reference resolves in Git.
- Review evidence can now be collected.

### Validate

- Evidence and runtime verification are recorded.
- User validation is accepted.
- Blocking findings are `none` or `resolved`.
- Foundation is `synced` or `not-required`.

### Archive

- Change state is `validated`.
- Dependencies are archived.
- Conflicts and blockers are empty.
- No uncommitted implementation files remain outside `docs/` and the active
  change package.
- `karl-ai change archive` moves the package but never commits it.

Archive is complete only after the developer explicitly requests and
successfully creates the final commit containing baseline updates and the move.

## Iteration

Keep artifacts coherent while implementation evolves. Update the current
change when intent remains the same. Start a new change when the original can
be completed independently or intent has materially changed.
