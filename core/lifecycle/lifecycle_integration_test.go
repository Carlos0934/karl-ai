package lifecycle_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	filesystemadapter "github.com/carlos0934/karl-ai/adapters/filesystem"
	gitadapter "github.com/carlos0934/karl-ai/adapters/git"
	"github.com/carlos0934/karl-ai/core/documents"
	"github.com/carlos0934/karl-ai/core/lifecycle"
)

func TestNewRequiresProjectFoundation(t *testing.T) {
	root, service := makeProject(t, false, false)
	_, err := service.Create("add-refunds", "L2", "2026-08-14")
	if err == nil || !strings.Contains(err.Error(), "Project foundation is required first") {
		t.Fatalf("Create() error = %v, root = %s", err, root)
	}
}

func TestNewScaffoldsOnlyRequiredChangeFiles(t *testing.T) {
	_, service := makeProject(t, true, false)
	created, err := service.Create("add-refunds", "L2", "2026-08-14")
	if err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(created.Path, "CHANGE.md"))
	parsed, err := documents.ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if created.State != "draft" || parsed.Data["assurance_level"] != "L2" {
		t.Fatalf("created = %#v, metadata = %#v", created, parsed.Data)
	}
	if _, err := os.Stat(filepath.Join(created.Path, "foundation", "CONTEXT.md")); !os.IsNotExist(err) {
		t.Fatalf("foundation artifact unexpectedly scaffolded: %v", err)
	}
}

func TestPlanGateRejectsUnresolvedScaffoldPlaceholders(t *testing.T) {
	_, service := makeProject(t, true, false)
	if _, err := service.Create("add-refunds", "L2", "2026-08-14"); err != nil {
		t.Fatal(err)
	}
	gate, err := service.Validate("add-refunds", "plan")
	if err != nil {
		t.Fatal(err)
	}
	assertGateError(t, gate, "Incomplete required section")
}

func TestInvalidStateTransitionsAreRejected(t *testing.T) {
	_, service := makeProject(t, true, false)
	if _, err := service.Create("add-refunds", "L2", "2026-08-14"); err != nil {
		t.Fatal(err)
	}
	_, err := service.Transition("add-refunds", "reviewing")
	if err == nil || !strings.Contains(err.Error(), "Invalid state transition") {
		t.Fatalf("Transition() error = %v", err)
	}
}

func TestFullLifecycleArchivesWithoutCreatingACommit(t *testing.T) {
	root, service := makeProject(t, true, true)
	slug := "add-refunds"
	createReadyPackage(t, root, service, slug, false)
	if _, err := service.Transition(slug, "planned"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Transition(slug, "implementing"); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(root, "refund.js"), "export const refund = true;\n")
	git(t, root, "add", "refund.js")
	git(t, root, "commit", "-m", "feat: implement refund fixture")
	reference := git(t, root, "rev-parse", "HEAD")
	writeFile(t, filepath.Join(root, "changes", slug, "TASKS.md"), lifecycleTasks(true))
	writeFile(t, filepath.Join(root, "changes", slug, "REVIEW.md"), reviewMarkdown(reference, false, "not-required"))
	if _, err := service.Transition(slug, "reviewing"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "changes", slug, "REVIEW.md"), reviewMarkdown(reference, true, "not-required"))
	if _, err := service.Transition(slug, "validated"); err != nil {
		t.Fatal(err)
	}

	before := git(t, root, "rev-parse", "HEAD")
	archived, err := service.Archive(slug, "2026-08-14")
	if err != nil {
		t.Fatal(err)
	}
	after := git(t, root, "rev-parse", "HEAD")
	parsed, err := documents.ParseFrontmatter(readFile(t, filepath.Join(archived.Path, "CHANGE.md")))
	if err != nil {
		t.Fatal(err)
	}
	if !archived.CommitRequired || before != after || parsed.Data["state"] != "archived" {
		t.Fatalf("archived = %#v, commits %s -> %s, state = %v", archived, before, after, parsed.Data["state"])
	}
}

func TestArchiveRejectsUncommittedImplementationFiles(t *testing.T) {
	root, service, slug, reference := readyValidatedChange(t)
	writeFile(t, filepath.Join(root, "uncommitted.js"), "unfinished\n")
	gate, err := service.Validate(slug, "archive")
	if err != nil {
		t.Fatal(err)
	}
	_ = reference
	assertGateError(t, gate, "Uncommitted implementation file blocks archive")
}

func TestArchiveRequiresDependenciesToBeArchived(t *testing.T) {
	root, service, slug, reference := readyValidatedChange(t)
	writeFile(t, filepath.Join(root, "changes", slug, "CHANGE.md"), changeMarkdown(slug, "validated", `["shared-auth"]`))
	writeFile(t, filepath.Join(root, "changes", slug, "REVIEW.md"), reviewMarkdown(reference, true, "not-required"))
	gate, err := service.Validate(slug, "archive")
	if err != nil {
		t.Fatal(err)
	}
	assertGateError(t, gate, "Dependency must be archived first: shared-auth")
}

func TestArchiveRejectsSyncedFoundationWithoutArtifacts(t *testing.T) {
	root, service, slug, reference := readyValidatedChange(t)
	writeFile(t, filepath.Join(root, "changes", slug, "REVIEW.md"), reviewMarkdown(reference, true, "synced"))
	gate, err := service.Validate(slug, "archive")
	if err != nil {
		t.Fatal(err)
	}
	assertGateError(t, gate, "change/foundation contains no files")
}

func TestArchiveRejectsNotRequiredFoundationWithArtifacts(t *testing.T) {
	root, service, slug, _ := readyValidatedChange(t)
	writeFile(t, filepath.Join(root, "changes", slug, "foundation", "CONTEXT.md"), "# Context\n")
	gate, err := service.Validate(slug, "archive")
	if err != nil {
		t.Fatal(err)
	}
	assertGateError(t, gate, "change/foundation contains files")
}

func TestImplementationGateRejectsAnUnresolvableGitReference(t *testing.T) {
	root, service := makeProject(t, true, true)
	createReadyPackage(t, root, service, "add-refunds", false)
	writeFile(t, filepath.Join(root, "changes", "add-refunds", "REVIEW.md"), reviewMarkdown("missing-ref", false, "pending"))
	gate, err := service.Validate("add-refunds", "implement")
	if err != nil {
		t.Fatal(err)
	}
	assertGateError(t, gate, "Implementation reference does not resolve to a commit: missing-ref")
}

func TestListReportsSortedActiveArchivedAndInvalidChanges(t *testing.T) {
	root, service := makeProject(t, true, false)
	if _, err := service.Create("zeta-change", "L1", "2026-08-14"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "changes", "alpha-change"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "changes", "archive", "2026-08-13-old-change"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := service.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Active) != 2 || result.Active[0].Name != "alpha-change" || result.Active[0].State != "invalid" || len(result.Archived) != 1 {
		t.Fatalf("List() = %#v", result)
	}
}

func readyValidatedChange(t *testing.T) (string, *lifecycle.Service, string, string) {
	t.Helper()
	root, service := makeProject(t, true, true)
	slug := "add-refunds"
	createReadyPackage(t, root, service, slug, true)
	writeFile(t, filepath.Join(root, "refund.js"), "export const refund = true;\n")
	git(t, root, "add", "refund.js")
	git(t, root, "commit", "-m", "feat: implement refund fixture")
	reference := git(t, root, "rev-parse", "HEAD")
	writeFile(t, filepath.Join(root, "changes", slug, "CHANGE.md"), changeMarkdown(slug, "validated", `[]`))
	writeFile(t, filepath.Join(root, "changes", slug, "REVIEW.md"), reviewMarkdown(reference, true, "not-required"))
	return root, service, slug, reference
}

func makeProject(t *testing.T, foundation, repository bool) (string, *lifecycle.Service) {
	t.Helper()
	root := t.TempDir()
	if foundation {
		writeFile(t, filepath.Join(root, "docs", "CONTEXT.md"), "# Context\n")
		writeFile(t, filepath.Join(root, "docs", "DESIGN.md"), "# Design\n")
	}
	if repository {
		git(t, root, "init")
		git(t, root, "config", "user.email", "test@example.com")
		git(t, root, "config", "user.name", "Test User")
		git(t, root, "add", ".")
		git(t, root, "commit", "-m", "chore: initialize fixture")
	}
	return root, lifecycle.New(filesystemadapter.New(root), gitadapter.New(root))
}

func createReadyPackage(t *testing.T, root string, service *lifecycle.Service, slug string, complete bool) {
	t.Helper()
	if _, err := service.Create(slug, "L2", "2026-08-14"); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "changes", slug)
	writeFile(t, filepath.Join(directory, "CHANGE.md"), changeMarkdown(slug, "draft", `[]`))
	writeFile(t, filepath.Join(directory, "PLAN.md"), integrationPlan)
	writeFile(t, filepath.Join(directory, "TASKS.md"), lifecycleTasks(complete))
	writeFile(t, filepath.Join(directory, "RESEARCH.md"), integrationResearch)
}

func changeMarkdown(slug, state, dependencies string) string {
	return `---
name: ` + slug + `
state: ` + state + `
created: 2026-08-14
assurance_level: L2
depends_on: ` + dependencies + `
conflicts_with: []
blocked_by: []
---

# Add Refund Requests

## Outcome

Customers can submit and retrieve a refund request.

## Scope

### Included

- Submit and retrieve refunds.

### Excluded

- Payment execution.

## Acceptance Criteria

- Eligible requests become pending.

## Foundation Areas

- None for this fixture.
`
}

const integrationResearch = `# Change Research

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

const integrationPlan = `# Change Plan

## Decision Basis

- Reuse the existing HTTP boundary.
  cites: Repository Evidence

## Technical Approach

Add an isolated refund lifecycle.

## Work Units

| Unit | Deliverable |
|---|---|
| 1 | Submit and retrieve a refund request |

## Integrated Validation

- Execute the complete HTTP scenario.
`

func lifecycleTasks(complete bool) string {
	mark := " "
	if complete {
		mark = "x"
	}
	return `# Change Tasks

## 1. Submit And Retrieve A Refund Request

**Status:** Planned
**Type:** Vertical Slice
**Depends on:** None

### Outcome

A customer can submit and retrieve one request.

### Acceptance

- Eligible requests become pending.

### Work

- [` + mark + `] 1.1 **Prepare:** Confirm eligibility.
- [` + mark + `] 1.2 **Implement:** Build HTTP behavior.
- [` + mark + `] 1.3 **Validate:** Execute verification.

### Validation

- HTTP tests pass.

### Complete When

- [` + mark + `] Outcome is observable.
`
}

func reviewMarkdown(reference string, final bool, status string) string {
	user, findings, accepted, ready := "pending", "unknown", "Pending", "No"
	if final {
		user, findings, accepted, ready = "accepted", "none", "Accepted", "Yes"
	}
	return `---
implementation_ref: "` + reference + `"
user_validation: ` + user + `
foundation_status: ` + status + `
blocking_findings: ` + findings + `
---

# Change Review

## Target

Validate the implementation.

## Change-Specific Considerations

- Order state remains unchanged.

## Evidence

| Check | Command Or Method | Result |
|---|---|---|
| Refund tests | npm test | 8 passed |

## Runtime Verification

Input: eligible order; Output: pending refund.

## Findings

| Finding | Severity | Resolution |
|---|---|---|
| None | None | No action required |

## User Validation

| Field | Value |
|---|---|
| Status | ` + accepted + ` |

## Foundation Reconciliation

| Artifact | Result |
|---|---|
| None | Not required |

## Archive Decision

| Requirement | Status |
|---|---|
| Ready to archive | ` + ready + ` |
`
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func assertGateError(t *testing.T, gate lifecycle.GateResult, want string) {
	t.Helper()
	if gate.OK || !strings.Contains(strings.Join(gate.Errors, "\n"), want) {
		t.Fatalf("gate errors = %q, want substring %q", gate.Errors, want)
	}
}
