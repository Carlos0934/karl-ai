# Karl Project Foundation

## Activation Contract

Run when the developer explicitly requests a Karl project foundation workshop,
or accepts Karl's recommendation to establish one. Recommend it when a new or
existing project lacks reliable context, a journey map, or an engineering
baseline. Do not activate merely to create docs, explore code, plan one change,
or deepen one existing section.

## Hard Rules

- Require developer acceptance before starting.
- Inspect existing projects before asking questions.
- Discover facts; ask the developer for intent and consequential decisions.
- Let Karl recommend `L1-L4`; let the developer select the final level.
- Use the level only to limit proactive questioning, never to limit coverage.
- Do not fabricate unknowns or create empty optional artifacts.
- Keep project analysis metadata only in `docs/CONTEXT.md` frontmatter.
- Do not implement product code or produce a change plan.

## Decision Gates

| Gate | Choice |
|---|---|
| Initialization mode | New, existing, or hybrid |
| Evidence | Confirmed, inferred, or unknown |
| Question depth | Selected complexity level |
| Output coverage | Developer request plus justified baseline artifacts |

Load `references/complexity-levels.md` and
`references/question-matrix.md` before classification or grilling.

## Execution Steps

1. Confirm scope, documentation root, initialization mode, and requested output.
2. Gather repository evidence for existing or hybrid projects.
3. Recommend a complexity level with evidence; obtain the developer's choice.
4. Build the current question frontier across context, journeys, technical
   design, and assurance. Ask in rounds with a recommendation per decision.
5. Load `references/documentation-model.md`; generate only applicable artifacts
   from templates and reconcile existing documents instead of overwriting them.
6. Update foundation, project, and analysis state in `docs/CONTEXT.md`
   frontmatter; keep its analysis summary concise and evidence-based.
7. Validate links, terminology, commands, contracts, and unresolved claims.

## Output Contract

Produce `docs/CONTEXT.md` and `docs/DESIGN.md`. Add journeys, architecture, data
models, contracts, or UI/UX artifacts only when requested or justified. Return
foundation, project, and analysis state; selected mode and level; files created
or updated; evidence-based assumptions; and unresolved blockers.

## Resources

- `templates/CONTEXT.template.md`
- `templates/JOURNEY.template.md`
- `templates/DESIGN.template.md`
- `references/documentation-model.md`
- `references/complexity-levels.md`
- `references/question-matrix.md`
