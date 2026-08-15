# Vertical Work Units

A work unit is a complete, reviewable deliverable. It includes the minimum
domain behavior, interface, data, tests, and documentation required to make its
outcome real.

## Required Shape

Every `## N. Name` group in TASKS.md must contain:

- `### Outcome`
- `### Acceptance`
- `### Work`
- `### Validation`
- `### Complete When`

The work checklist must include at least one `Prepare`, `Implement`, and
`Validate` task using the unit's number.

## Rules

- Split by observable behavior, not file type or architectural layer.
- Keep tests and relevant docs with the behavior they verify.
- Leave the repository coherent after each completed unit.
- State runtime validation or an explicit reason it is not applicable.
- Define a rollback boundary independent of commit creation.
- Design each unit as a commit candidate; a separate commit is recommended but
  not mandatory.

## Enabling Work

Use an `Enabling Work` unit only when at least two vertical units depend on a
shared prerequisite, duplication is unreasonable, and the prerequisite can be
validated independently. Name its consumers and validation explicitly. Do not
create generic model, service, infrastructure, or test phases.

## Evidence

TASKS.md defines how each unit will be validated. Record actual commands,
results, runtime observations, findings, and user validation once in REVIEW.md;
do not create a separate review log.
