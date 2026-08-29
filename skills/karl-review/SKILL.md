---
name: karl-review
description: Independently evaluate a resulting change against its original expected outcome and return PASS, FAIL, or BLOCKED with evidence.
---

# Reviewer procedure

Function:

```text
expected outcome + resulting state
→ independent evaluation
→ PASS / FAIL / BLOCKED
```

## Steps

```text
1. Read the original objective, expected outcome, resulting state, and constraints.
2. Reconstruct evidence from the resulting state. Do not treat WORKER conclusions as facts.
3. Inspect behavior, tests, diffs, and relevant state as needed.
4. If evaluation cannot be completed safely: return BLOCKED.
5. If the result does not satisfy the expected outcome: return FAIL with findings.
6. If every required criterion is satisfied: return PASS with evidence.
7. Do not modify or repair the implementation.
```

Workspace writes are allowed only for tests and temporary validation artifacts. Leave no persistent changes.

`FAIL` = the result can be evaluated and does not satisfy the target.
`BLOCKED` = evaluation cannot be completed responsibly.

## PASS

```markdown
## Status

PASS

## Evidence

- <Criterion and supporting evidence>
```

## FAIL

```markdown
## Status

FAIL

## Findings

### <Severity> - <Unsatisfied condition>

Evidence: <Observable behavior or relevant state>

Recommended direction: <Brief corrective direction>
```

Do not provide a replacement implementation.

## BLOCKED

```markdown
## Status

BLOCKED

## Reason

<Why a reliable decision cannot be made>

## Needed decision

<Missing information, authority, or external decision>
```
