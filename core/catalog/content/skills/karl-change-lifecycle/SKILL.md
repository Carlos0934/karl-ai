# Karl Change Lifecycle

## Activation Contract

Use for work that changes behavior, contracts, data, architecture,
dependencies, or journeys. Do not require it for formatting, typos, comments,
or read-only analysis. Activate on an explicit phase request or when Karl
recommends tracking the requested work and the developer accepts.

This workflow can be driven through the catalog commands `karl-change-new`,
`karl-change-implement`, `karl-change-review`, and `karl-change-archive`, which
enforce the same gates as `karl-ai change`.

## Hard Rules

- Require `docs/CONTEXT.md` and `docs/DESIGN.md`; recommend the
  `karl-project-foundation` skill when absent.
- Inspect foundation, source, tests, contracts, and active changes before asking.
- Use `karl-ai change` for creation, state transitions, validation, and archive;
  never bypass a failed gate or move changes manually.
- Let the developer authorize implementation, validation, archive, and commits.
- Plan self-contained vertical work units; do not split by technical layer.
- Mark tasks complete only after their required behavior is demonstrated.
- Record actual evidence once in `REVIEW.md`; do not create a review log.
- Never commit automatically. Archive only prepares the final move.

## Decision Gates

| Situation | Action |
|---|---|
| Foundation missing | Stop and recommend the foundation workshop |
| Outcome or fit unclear | Run the adaptive questions in `references/questioning-guide.md` |
| Same intent, refined approach | Update the active change |
| Independent or materially different intent | Create a new change |
| Gate fails | Resolve its reported blockers; do not force progress |

## Execution Steps

1. Load `references/workflow.md`, `references/questioning-guide.md`, and the
   relevant project foundation documents.
2. Establish the change name and assurance level, then create the package with
   `karl-ai change new <name> --root <project>`.
3. Complete research in the scaffolded `RESEARCH.md` before planning:
   repository facts with file references, documentation, and external sources.
   Delegate research to the research specialist; the planner never writes
   research content.
4. Establish outcome, scope, acceptance, technical preferences, work units,
   and validation; ask only unresolved decisions. Complete CHANGE.md, PLAN.md,
   and TASKS.md from templates. In PLAN.md `Decision Basis`, cite RESEARCH.md
   sections with `cites: <Section Name>`; never duplicate research facts.
5. Validate and transition through `planned`, `implementing`, `reviewing`, and
   `validated`; return to an earlier state when findings invalidate artifacts.
6. During implementation, complete one vertical unit at a time and keep its
   tests and relevant foundation artifacts with it.
7. Record final checks, findings, user validation, and foundation status in
   REVIEW.md; reconcile `foundation/` per
   `references/foundation-integration.md`.
8. Run `karl-ai change archive` only after its gate passes. Report that archive
   remains incomplete until the developer explicitly requests the final commit.

## Output Contract

Return the active change, state, gate results, unresolved decisions, files
created or updated, validation evidence, and next allowed actions. On archive,
return the destination and state clearly that no commit was created.

## Resources

- `templates/CHANGE.template.md`
- `templates/PLAN.template.md`
- `templates/TASKS.template.md`
- `templates/RESEARCH.template.md`
- `templates/REVIEW.template.md`
- `references/workflow.md`
- `references/questioning-guide.md`
- `references/work-units.md`
- `references/foundation-integration.md`
