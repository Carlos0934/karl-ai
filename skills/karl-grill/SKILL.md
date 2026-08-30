---
name: karl-grill
description: Grill the user about a plan, decision, or idea through three fixed rounds of sliced questions, then report gaps and risks.
---

# Grill procedure

Function:

```text
subject → design tree
→ round 1 understand → round 2 constrain and scope → round 3 validate
→ gaps and risks → user confirmation
```

## Design tree

Map the subject (plan, decision, or idea) as a design tree: every decision
branches into the decisions that hang off it.

Work the tree in exactly three rounds. Each round takes one fixed slice:

```text
Round 1 — Understand the proposal.
  Intent, the problem being solved, the affected parties, the current shape,
  and the outcome that would count as success.

Round 2 — Necessary information, then constraints and scope.
  Facts the session must know to proceed, technical and organizational
  constraints, and the in-scope / out-of-scope boundary.

Round 3 — Confirm the details that validate the decision.
  Success criteria, failure modes, and what must be true for the plan,
  decision, or idea to be done and safe to act on.
```

A question belongs to a round only when every prerequisite for asking it is
settled before that round starts; a question that depends on an unheard answer
belongs to a later round, not this one.

## Asking a round

Ask the whole slice in one round, then wait for the user's answers before the
next round. Answers reshape the tree: settled decisions unblock the questions
that depend on them, so recompute the slice at the start of every round. A
round with no open decisions states that and passes; do not invent questions
to fill it.

One question carries one decision and looks like:

```text
❓ **Q1** — **<question title>**: <body — at most 3 short paragraphs>

- **A)** <option one — short, one to two lines>
- **B)** <option two>
- **C)** <option three>
- **D)** <option four>

➡️ Recommended: **<choice>** — <one-line rationale>
```

Options are aligned list items with bold letter labels, never inline in
prose; only the recommendation carries rationale.

## Facts are yours, decisions are the user's

Never ask the user for a fact you could look up yourself. When a question
needs an environment fact (filesystem, tools, code, documentation), dispatch
a scout sub-agent to find it; if the harness has no scout, search directly. An
unfinished exploration is an unsettled prerequisite: only the questions
downstream of it wait — ask the rest of the round now. Report found facts to
the user as context inside the round; only decisions go to the user as
questions.

## Closing

The session is done when all three rounds are complete and nothing is silently
assumed: every branch of the tree visited. Report the residue as gaps and
risks, classified by consequence:

```markdown
## Gaps and risks

### Blocking — technical, architecture, or security decisions

- <gap or risk> — <why it changes the design>

### Deferred — missing implementation details

- <gap or risk> — <why it can be resolved during implementation>
```

Omit empty sections. Rank by consequence, not by ease: technical and
planning-spec gaps outrank implementation-detail gaps.

Ask the user to confirm the shared understanding. Do not act on the plan,
decision, or idea until the user confirms.
