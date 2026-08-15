# Foundation Integration

`karl-change-lifecycle` consumes the baseline created by
`karl-project-foundation`:

```text
docs/CONTEXT.md
docs/DESIGN.md
docs/journeys/
docs/architecture/
docs/data-models/
docs/contracts/
docs/uiux/
```

Do not start a tracked change without `docs/CONTEXT.md` and `docs/DESIGN.md`.
Recommend the foundation workshop when either file is absent.

## Change Foundation

Create `changes/<name>/foundation/` only when the change affects the baseline.
Use normal project formats, not delta syntax. Include only relevant documents
and sections. Keep them synchronized with what implementation actually becomes.

## Reconciliation

Before validation:

1. Compare change foundation artifacts with implemented behavior.
2. Reconcile applicable content into `docs/` semantically.
3. Validate links, diagrams, schemas, contracts, commands, and terminology.
4. Set `foundation_status` in REVIEW.md to `synced`.

Use `not-required` only when implementation changes no context, journey,
technical baseline, contract, data model, architecture, or experience. Explain
that conclusion in REVIEW.md.

The lifecycle CLI validates state and evidence markers. It never performs a
semantic foundation merge.
