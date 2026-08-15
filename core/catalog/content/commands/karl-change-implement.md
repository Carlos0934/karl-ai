# Implement A Planned Change

This command is explicit developer authorization to start implementation.
Confirm that authorization before any state change.

1. Follow the `karl-change-lifecycle` skill and its resources.
2. Identify the change with `karl-ai change status <name>`. If the name is
   omitted, use the only active change or ask which one. Require state
   `planned`, or a documented return from `reviewing`.
3. Re-run `karl-ai change validate <name> plan`; resolve every blocker before
   proceeding. Then run `karl-ai change transition <name> implementing`.
4. Execute one vertical work unit at a time. Keep code, tests, and affected
   foundation artifacts together in the same unit. Mark a task complete only
   after its required behavior is demonstrated.
5. Keep PLAN.md and TASKS.md coherent while implementation evolves. Update the
   current change when intent is unchanged; recommend a new change when intent
   has materially changed.
6. When every task checkbox is complete, request the developer to create the
   implementation commit. Only after it exists, record the commit or range as
   `implementation_ref` in REVIEW.md frontmatter.
7. Run `karl-ai change validate <name> implement`; resolve every blocker, then
   run `karl-ai change transition <name> reviewing`. Never bypass a failed gate
   or edit state manually.

Every state change goes through `karl-ai change`, REVIEW.md is the only review
evidence source, and the implementation commit is created only when the
developer explicitly requests it.

Return the change name, state, completed work units, gate results, files
changed, unresolved blockers, and `karl-change-review <name>` as the next
allowed action.
