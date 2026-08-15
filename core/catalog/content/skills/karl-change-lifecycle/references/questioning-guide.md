# Change Discovery And Planning Questions

Inspect the foundation, active changes, source, tests, contracts, and project
tooling before questioning the developer. Treat repository facts as Karl's
responsibility. Ask about intent, preferences, and consequential decisions.

## Frontier Order

1. **Outcome:** actor, observable result, success, and explicit non-goals.
2. **Behavior:** journeys, states, failures, side effects, and compatibility.
3. **Preferences:** simplicity, extensibility, dependencies, delivery shape,
   and explicitly preferred technical approaches.
4. **Technical fit:** architecture, interfaces, data, migration, security,
   operations, and UI/UX only when applicable.
5. **Work units:** vertical deliverables, dependencies, rollback boundaries,
   and reviewable commit candidates.
6. **Validation:** focused checks, runtime scenarios, failure paths,
   regressions, integrated behavior, and user validation.

Ask only the current dependency-aware frontier. Give Karl's recommended answer
for each decision. Do not ask a downstream question while its prerequisites
remain unresolved.

## Evidence Treatment

| Evidence | Action |
|---|---|
| Confirmed | Use it and cite its repository source; do not ask again |
| Inferred | State the inference and confirm it when being wrong matters |
| Unknown fact | Investigate it |
| Unknown decision | Ask the developer with a recommendation |

## Assurance

Default to the project's foundation level, then raise the change level when its
own consequences require more assurance. Karl recommends; the developer selects
the effective level. The level controls proactive questioning, not which
artifacts the developer may request.

## Completion

Planning is complete when the change has one coherent outcome, bounded scope,
verifiable acceptance, a technically compatible approach, vertical work units,
and validation that can demonstrate the behavior through its real boundary.
