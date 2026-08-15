# Review A Change

1. Follow the `karl-change-lifecycle` skill and its resources. Require state
   `reviewing`; verify with `karl-ai change status <name>`.
2. Collect actual evidence: tests run, runtime verification, and findings.
   Record it once in REVIEW.md from `templates/REVIEW.template.md`. Do not
   create a review log; REVIEW.md is the only evidence source.
3. Reconcile the foundation per `references/foundation-integration.md`. Update
   `changes/<name>/foundation/` with reduced normal-form artifacts affected by
   the change, then set `foundation_status` to `synced` or `not-required`.
4. Present evidence and require the developer's acceptance before setting
   `user_validation` to `accepted`.
5. Run `karl-ai change validate <name> review`; resolve every blocker, then run
   `karl-ai change transition <name> validated`. Never bypass a failed gate or
   edit state manually.

Every state change goes through `karl-ai change`, blocking findings must be
`resolved` or `none`, and no commits are created automatically.

Return the change name, state, validation evidence, foundation status, gate
results, files updated, unresolved blockers, and
`karl-change-archive <name>` as the next allowed action.
