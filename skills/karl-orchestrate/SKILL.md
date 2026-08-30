---
name: karl-orchestrate
description: Coordinate non-trivial development through a shallow change-review workflow with independent review and bounded repairs.
---

# Orchestrator procedure

Function:

```text
understand → delegate → collect → decide → route
```

## Topology

```text
                   ORCHESTRATOR
                  /      |      \
                 ↓       ↓       ↓
             WORKER   REVIEWER  SCOUT
```

Only ORCHESTRATOR coordinates. Subagents do not communicate. Scout evidence reaches WORKER and REVIEWER only through ORCHESTRATOR.

```text
WORKER → ORCHESTRATOR → REVIEWER
REVIEWER → ORCHESTRATOR → WORKER
SCOUT → ORCHESTRATOR (evidence only)
```

Delegation depth is one.

## Research routing

ORCHESTRATOR dispatches karl-scout in exactly two situations:

```text
pre-delegation: context for a correct delegation packet is missing
                (unknown area, unclear scope or acceptance detail)
post-FAIL:      a finding is likely answered by evidence the
                orchestrator, worker, or reviewer has not gathered,
                especially external documentation
```

Do not dispatch SCOUT for user-information requests without a change objective, or when ORCHESTRATOR can resolve the gap by reading a file directly at trivial cost.

## Happy path

```text
RECEIVE → UNDERSTAND → IMPLEMENT → REVIEW → COMPLETE
```

```text
USER
  ↓
ORCHESTRATOR
  ↓ interpret request
  ↓ delegate change
WORKER
  ↓ outcome
ORCHESTRATOR
  ↓ delegate independent review
REVIEWER
  ↓ PASS
ORCHESTRATOR
  ↓ final response
```

## Failure path

```text
REVIEWER
  ↓ FAIL + findings
ORCHESTRATOR
  ↓ repair delegation
WORKER
  ↓ repair outcome
ORCHESTRATOR
  ↓ new review
REVIEWER
```

ORCHESTRATOR converts findings into the next delegation. REVIEWER does not instruct WORKER.

## Repair limit

```text
max repair attempts = 2
```

```text
REVIEW
  ├── PASS → COMPLETE
  └── FAIL → REPAIR → REVIEW
```

After the limit: `UNRESOLVED`. Stop.

## State

Preserve only:

```text
original request
expected outcome
current stage
worker outcome
reviewer outcome
scout outcome
repair count
unresolved issues
```

```text
RECEIVED
   ↓
IMPLEMENTING
   ↓
REVIEWING
   ├──────── PASS ───────→ COMPLETED
   ├──────── FAIL ───────→ REPAIRING → REVIEWING
   └────── BLOCKED ──────→ BLOCKED
```

RESEARCHING is an optional stage. It is reachable from pre-delegation routing and from REPAIRING, and returns to the same routing decision:

```text
UNDERSTAND ──→ RESEARCHING ──┐
REPAIRING ───→ RESEARCHING ──┴─→ routing decision again
```

After the repair limit: `REVIEWING → FAIL → UNRESOLVED`.

Terminal states: `COMPLETED` | `BLOCKED` | `UNRESOLVED`.

## Steps

```text
1. Receive the user request.
2. Derive an observable expected outcome and acceptance criteria.
3. If the work is trivial, non-behavioral, and independent review has negligible value:
     handle it directly and return.
4. Delegate change work to karl-worker.
5. Receive the WORKER outcome.
6. If BLOCKED: stop BLOCKED.
7. Delegate independent review to karl-reviewer.
8. Receive the REVIEWER outcome.
9. If PASS: stop COMPLETED.
10. If BLOCKED: stop BLOCKED.
11. While FAIL and repair_count < 2:
      create a repair task from original expected outcome + current findings
      delegate to karl-worker
      receive the WORKER outcome
      delegate independent review again
      if PASS: stop COMPLETED
12. Stop UNRESOLVED.
```

## Delegation

Pass only context that changes decisions: intended outcome, success criteria, relevant boundaries, non-obvious constraints.

Do not forward one subagent's reasoning to the other subagent.

Every delegation item must change something evaluable against a finding or a goal question. Items that only re-run prior validation or touch state for formatting are not delegation work.

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

To REVIEWER send:

```text
original objective
expected outcome
current resulting state
relevant constraints
```

Ask for `PASS`, `FAIL`, or `BLOCKED` with evidence. Do not ask REVIEWER to repair.

To SCOUT send the same packet with:

```text
a research goal instead of an implementation task
explicit goal questions
relevant source boundaries
```

Ask for `COMPLETE`, `PARTIAL`, or `BLOCKED` with evidence, gaps, and dead ends. Do not ask SCOUT to interpret or recommend.

## Outcomes to collect

WORKER:

```text
PASS | FAIL | BLOCKED
result
validation
remaining issues
```

REVIEWER:

```text
PASS | FAIL | BLOCKED
```

plus evidence or findings.

SCOUT:

```text
COMPLETE | PARTIAL | BLOCKED
evidence
gaps
dead ends
```

plus goal coverage. ORCHESTRATOR is the sole interpreter of scout evidence; scout recommendations do not exist by design.

`FAIL` = the work can be evaluated and does not satisfy the target.
`BLOCKED` = the work cannot be completed or evaluated safely.
`PARTIAL` = useful evidence is returned plus explicit gaps.
`COMPLETE` = every goal question is resolved with a citation.

## Control decisions

After an outcome, choose one:

```text
continue | review | repair | complete | escalate | stop blocked | stop unresolved
```

Complete only when:

```text
requested outcome is implemented
AND independent review is PASS
AND no blocking issue remains
```

If review could not run, say so. Do not treat WORKER success as completion.

## Channels

```text
USER → ORCHESTRATOR
ORCHESTRATOR → WORKER
WORKER → ORCHESTRATOR
ORCHESTRATOR → REVIEWER
REVIEWER → ORCHESTRATOR
ORCHESTRATOR → SCOUT
SCOUT → ORCHESTRATOR
```

No peer-to-peer channel. Repair reuses the same channels.

## Rules

1. ORCHESTRATOR owns orchestration.
2. Subagents do not coordinate directly.
3. Delegation depth stays shallow.
4. WORKER changes the system.
5. REVIEWER evaluates the system independently.
6. REVIEWER does not repair.
7. ORCHESTRATOR preserves the original expected outcome.
8. Delegate only decision-relevant context.
9. Do not forward full subagent transcripts.
10. Subagent outcomes are concise and evidence-oriented.
11. Repair loops are bounded.
12. A new role requires a real responsibility boundary.
