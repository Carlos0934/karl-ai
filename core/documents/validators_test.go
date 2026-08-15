package documents

import (
	"strings"
	"testing"
)

const validResearch = `# Change Research

## Research Questions

- Which order states permit a refund?

## Repository Evidence

- Order state transitions live in orders.js.

## External Sources

- None required for this fixture.

## Risks And Gaps

- No missing evidence.

## Open Questions

- None.
`

const validPlan = `# Change Plan

See [CHANGE.md](./CHANGE.md) for outcome, scope, and acceptance criteria.

## Decision Basis

- Reuse the existing HTTP boundary.
  cites: Repository Evidence

## Technical Approach

Add an isolated refund lifecycle through the existing HTTP boundary.

## Work Units

| Unit | Deliverable |
|---|---|
| 1 | Submit and retrieve a refund request |

## Cross-Unit Constraints

- Order state remains unchanged.

## Integrated Validation

- Execute the complete HTTP scenario and regression tests.

## Recovery

Remove the isolated refund module and routes.
`

func TestWorkUnitsRequirePrepareImplementAndValidateTasks(t *testing.T) {
	checked := ValidateTasksDocument(tasksMarkdown(false, false), false)
	assertValidationError(t, checked, "must include a Validate task")
}

func TestResearchDocumentRequiresEvidenceSections(t *testing.T) {
	checked := ValidateResearchDocument("# Change Research\n\n## Research Questions\n\nPlan facts only.")
	assertValidationError(t, checked, "Missing required section: ## Repository Evidence")
}

func TestPlanDecisionsMustCiteResearch(t *testing.T) {
	checked := ValidatePlanDocument(strings.Replace(validPlan, "\n  cites: Repository Evidence\n", "\n", 1), validResearch)
	assertValidationError(t, checked, "must cite at least one RESEARCH section")
}

func TestPlanCitationsMustReferenceExistingResearchSections(t *testing.T) {
	checked := ValidatePlanDocument(strings.Replace(validPlan, "Repository Evidence", "Nonexistent Section", 1), validResearch)
	assertValidationError(t, checked, "unknown or incomplete RESEARCH section: Nonexistent Section")
}

func TestPlanDecisionsWithValidCitationsPass(t *testing.T) {
	checked := ValidatePlanDocument(validPlan, validResearch)
	if !checked.OK || len(checked.Errors) != 0 {
		t.Fatalf("validation = %#v", checked)
	}
}

func TestFrontmatterUpdatePreservesOrderAndFormatsValues(t *testing.T) {
	markdown := "---\r\nname: 'sample'\r\nstate: draft\r\ndepends_on: [one, 'two']\r\n---\r\n\r\n# Body\r\n"
	updated, err := UpdateFrontmatter(markdown, map[string]any{"state": "planned"})
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := "---\nname: sample\nstate: planned\ndepends_on: [\"one\",\"two\"]\n---\n\n# Body"
	if !strings.HasPrefix(updated, wantPrefix) {
		t.Fatalf("updated frontmatter = %q", updated)
	}
}

func tasksMarkdown(complete, includeValidate bool) string {
	mark := " "
	if complete {
		mark = "x"
	}
	validate := ""
	if includeValidate {
		validate = "- [" + mark + "] 1.3 **Validate:** Execute focused and runtime verification."
	}
	return `# Change Tasks

## 1. Submit And Retrieve A Refund Request

**Status:** Planned  
**Type:** Vertical Slice  
**Depends on:** None

### Outcome

A customer can submit and retrieve one pending refund request.

### Acceptance

- Eligible requests become pending and can be retrieved.

### Deliverables

| Deliverable | Expected Result |
|---|---|
| HTTP behavior | Submit and retrieve operations work |

### Work

- [` + mark + `] 1.1 **Prepare:** Confirm order eligibility boundaries.
- [` + mark + `] 1.2 **Implement:** Build the behavior through HTTP.
` + validate + `

### Validation

| Check | Method | Expected Result |
|---|---|---|
| HTTP behavior | npm test | Tests pass |

#### Runtime Scenario

Given: an eligible order
When: a refund is submitted
Then: it can be retrieved as pending

### Rollback

Remove the isolated refund module and routes.

### Complete When

- [` + mark + `] Outcome is observable.
- [` + mark + `] Acceptance conditions pass.
- [` + mark + `] Automated and runtime checks pass.
`
}

func assertValidationError(t *testing.T, checked Validation, want string) {
	t.Helper()
	if checked.OK || !strings.Contains(strings.Join(checked.Errors, "\n"), want) {
		t.Fatalf("validation errors = %q, want substring %q", checked.Errors, want)
	}
}
