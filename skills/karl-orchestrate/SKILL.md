---
name: karl-orchestrate
description: Coordinate non-trivial development through a shallow implement-verify workflow with independent verification and bounded repairs.
---

# Controlled Development Orchestration

Act as MAIN. Own user intent, the expected outcome, acceptance criteria, workflow state, delegation, repair count, terminal status, and the final response.

## Invariants

- Only MAIN coordinates work.
- Workers return to MAIN and never communicate directly.
- Delegation depth is one: workers must not delegate.
- Implementation authority and verification authority remain separate.
- Pass only context that changes a worker's decisions.
- Do not forward full reasoning transcripts between workers.
- Allow at most two repair attempts.

## State

Preserve only:

- original request
- expected outcome and acceptance criteria
- current stage
- implementation outcome
- verification outcome
- repair count
- unresolved issues

Use these stages:

```text
RECEIVED -> IMPLEMENTING -> VERIFYING
VERIFYING -> COMPLETED | REPAIRING | BLOCKED
REPAIRING -> VERIFYING
VERIFYING -> UNRESOLVED after the repair limit
```

## Workflow

1. Derive an observable expected outcome and concise acceptance criteria.
2. Handle work directly only when it is trivial, non-behavioral, and independent verification has negligible value.
3. Delegate implementation to `karl-implementer` with the task, expected outcome, relevant scope, material constraints, and required return shape.
4. Stop as `BLOCKED` if implementation cannot continue responsibly.
5. Delegate independent verification to `karl-verifier`. Send the original objective, acceptance criteria, resulting state, and relevant constraints. Do not send the implementer's reasoning.
6. Complete only on verifier `PASS` with no blocking issue.
7. On verifier `FAIL`, convert concrete findings into a repair task for the implementer, then request a fresh independent verification.
8. Stop as `UNRESOLVED` after two failed repair attempts.

## Implementation Delegation

```markdown
## Task

<What to achieve>

## Expected outcome

- <Observable criterion>

## Scope

<Only relevant boundaries>

## Constraints

<Only decision-changing constraints>

## Return

Report status, result, validation, and unresolved issues.
```

Omit empty sections.

## Verification Delegation

Send only the objective, acceptance criteria, resulting state, and constraints. Ask for `PASS`, `FAIL`, or `BLOCKED` with evidence. Do not ask the verifier to repair findings.

## Control Decisions

After a worker outcome, choose only one of:

```text
continue | verify | repair | complete | stop blocked | stop unresolved
```
