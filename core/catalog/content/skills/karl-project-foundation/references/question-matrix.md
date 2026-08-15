# Adaptive Question Matrix

Question depth, requested coverage, and evidence are independent axes:

- **Complexity level** controls what Karl asks proactively.
- **Requested coverage** controls what artifacts and sections are produced.
- **Evidence state** controls whether Karl documents, confirms, or asks.

## Evidence States

| State | Meaning | Action |
|---|---|---|
| `Confirmed` | Direct evidence exists in the repository or developer statement | Document with its source; do not ask again |
| `Inferred` | Evidence supports a likely answer | State the inference; confirm when being wrong matters at the selected level |
| `Unknown` | Evidence is insufficient | Investigate discoverable facts; ask the developer only for decisions or business knowledge |

Never ask the developer how to inspect the repository. Facts are Karl's job;
business intent, preferences, and consequential decisions belong to the
developer.

## Question Families By Level

| Family | L1 Personal | L2 Product | L3 Business-Critical | L4 High-Assurance |
|---|---|---|---|---|
| Context & Domain | Purpose, scope, primary actors, core outcome | Additional actors, terms, rules, journeys, side effects | Context boundaries, invariants, sensitive concepts, failure consequences | Regulatory obligations, audit semantics, legal or safety outcomes |
| Journeys & Experience | Trigger, inputs, main flow, result, basic failure | Alternate flows, multiple actors, states, contracts, experience | Partial failures, compensation, retries, idempotency, accessibility | Manual recovery, evidence, traceability, exceptional authorization |
| Technical Design | App type, stack, structure, major tools, local validation | Architecture, persistence, interfaces, dependency versioning, CI | Migrations, compatibility, security, performance, resilience | Threat analysis, isolation, disaster recovery, hardened controls |
| Assurance & Operations | Format, static checks, unit or smoke tests | Integration, contract, end-to-end, release, logging | Security and performance tests, observability, recovery, service objectives | Independent verification, audit evidence, failover, compliance controls |

The matrix defines proactive depth, not allowed output. If an L1 developer asks
for a DBML model, threat model, contract suite, or UI/UX specification, include
it and ask the minimum questions required to make that artifact correct.

## Grilling Rounds

Build a dependency-aware decision tree. Ask only the current frontier: questions
whose prerequisites are already settled.

| Level | Typical Rounds | Questions Per Round |
|---|---:|---:|
| L1 | 1-2 | 3-5 |
| L2 | 2-4 | 4-6 |
| L3 | 3-6 | 4-6 |
| L4 | No fixed limit | Maximum 5 |

These are guidance, not quotas. Stop when no relevant decision remains open.
For each question:

1. State the decision and why it matters.
2. Offer concrete options when alternatives exist.
3. Give Karl's recommendation.
4. Wait for the developer before traversing dependent branches.

## Coverage Overrides

At any time the developer may:

- Add or remove an artifact from the requested output.
- Request deeper treatment of one family without changing the project level.
- Skip a noncritical question.
- Reopen a documented assumption.

Do not create empty optional folders or speculative documents merely because a
higher level permits them.
