# Run The Project Foundation Workflow

Use the supplied scope or arguments as the foundation objective. This command
is explicit developer acceptance to start the workflow.

1. Follow the `karl-project-foundation` skill and its resources.
2. Inspect the repository before questioning: structure, manifests, lockfiles,
   source, tests, CI, contracts, data models, configuration, and existing docs.
3. Determine whether this is a new, existing, or hybrid project. Recommend an
   `L1-L4` assurance level with evidence, then let the developer select it.
4. Ask only unresolved business or consequential technical decisions. Give a
   concrete recommendation for each question.
5. Create or reconcile `docs/CONTEXT.md`, `docs/DESIGN.md`, and only journeys
   or supporting artifacts justified by the requested coverage.
6. Maintain this canonical frontmatter in `docs/CONTEXT.md`:

```yaml
---
foundation_state: draft
project_state: unknown
analysis_state: pending
project_mode: unknown
assurance_level: pending
last_analyzed_at: ""
analysis_summary: ""
---
```

Use only values documented in `references/documentation-model.md`. Keep
`analysis_summary` to one evidence-based line. Set timestamps in ISO 8601 UTC.
Do not duplicate this metadata in DESIGN.md or journeys.

Do not implement product code or create a tracked change. Return the selected
mode and level, analysis state, foundation state, files changed, assumptions,
and unresolved blockers.
