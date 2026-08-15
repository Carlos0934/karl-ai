# Plan A Tracked Change

This command is explicit developer acceptance to plan a tracked change.

1. Follow the `karl-change-lifecycle` skill and its resources.
2. Require `docs/CONTEXT.md` and `docs/DESIGN.md`. If missing, stop and
   recommend the `karl-foundation` command before planning.
3. Inspect active changes with `karl-ai change list`. Reuse an active change
   when intent is the same; plan a new one when intent is independent or
   materially different.
4. Create the package with
   `karl-ai change new <name> [--level L1|L2|L3|L4]`. It scaffolds CHANGE.md,
   PLAN.md, TASKS.md, RESEARCH.md, and REVIEW.md.
5. Research: delegate to `karl-searcher`, the only author of
   `changes/<name>/RESEARCH.md`, to collect repository facts, project
   documentation, and external sources when they apply. The orchestrator and
   planner never author RESEARCH.md.
6. Establish outcome, scope, acceptance criteria, assurance level, work units,
   and integrated validation. Ask only unresolved decisions using
   `references/questioning-guide.md`; give a concrete recommendation for each.
7. Complete CHANGE.md, PLAN.md, and TASKS.md from templates. In PLAN.md
   `Decision Basis`, cite RESEARCH.md sections with `cites: <Section Name>`;
   never duplicate research facts. Declare work-unit dependencies only in
   TASKS.md.
8. Define self-contained vertical work units; never split by technical layer.
   Load `references/work-units.md` before writing PLAN.md and TASKS.md.
9. Run `karl-ai change validate <name> plan`; resolve every reported blocker,
   then run `karl-ai change transition <name> planned`. Never bypass a failed
   gate or edit state manually.

Every state change goes through `karl-ai change`, REVIEW.md is the only review
evidence source, and no commits are created automatically.

Return the change name, state, assurance level, work units, gate results, files
created, unresolved decisions, and `karl-change-implement <name>` as the next
allowed action after explicit developer authorization.
