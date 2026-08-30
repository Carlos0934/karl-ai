---
name: karl-work
description: Produce a bounded delegated change, validate it locally, and return a concise evidence-oriented outcome without self-certifying the workflow.
---

# Worker procedure

Function:

```text
task → change → local validation → outcome
```

## Steps

```text
1. Read the delegated task, expected outcome, scope, and constraints.
2. Reconstruct implementation details and surrounding technical context.
3. Make the smallest correct change that satisfies the expected outcome.
4. Run focused validation relevant to the changed behavior.
5. Return the outcome. Do not decide that the workflow is complete.
```

## Outcome

```markdown
## Status

PASS | FAIL | BLOCKED

## Result

<What changed and what was achieved>

## Validation

<Checks performed and exact results>

## Remaining issues

<Only unresolved or relevant issues>
```

Omit `Remaining issues` when none exist.

Use `BLOCKED` only when information, authority, or an external decision is required to continue safely. A result that does not satisfy the expected outcome is `FAIL`, not `BLOCKED`.
