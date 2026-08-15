package lifecycle

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/carlos0934/karl-ai/core/documents"
	"github.com/carlos0934/karl-ai/core/foundation"
)

type Package struct {
	Directory string
	Files     map[string]string
	Metadata  map[string]any
}

type ChangeItem struct {
	Name  string `json:"name"`
	State any    `json:"state"`
}

type ChangeList struct {
	Active   []ChangeItem `json:"active"`
	Archived []string     `json:"archived"`
}

type Store interface {
	Missing(paths []string) ([]string, error)
	CreateChange(slug string, files map[string]string) (string, error)
	ReadChange(slug string) (Package, error)
	WriteChange(slug, content string) error
	ArchivedDependencyExists(dependency string) (bool, error)
	FoundationFileCount(slug string) (int, error)
	ArchiveChange(slug, date, updatedChange, originalChange string) (string, error)
	ListChanges() (ChangeList, error)
}

type Git interface {
	IsRepository() bool
	ResolvesCommit(reference string) bool
	UncommittedFiles() ([]string, error)
}

type Service struct {
	store Store
	git   Git
	now   func() time.Time
}

type Created struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Path  string `json:"path"`
}

type GateResult struct {
	OK       bool     `json:"ok"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Gate     string   `json:"gate"`
	State    any      `json:"state"`
}

type Transitioned struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

type Archived struct {
	Name           string `json:"name"`
	State          string `json:"state"`
	Path           string `json:"path"`
	CommitRequired bool   `json:"commitRequired"`
}

type Status struct {
	Name     string                `json:"name"`
	State    any                   `json:"state"`
	Metadata map[string]any        `json:"metadata"`
	Gates    map[string]GateResult `json:"gates"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var refRangePattern = regexp.MustCompile(`\.{2,3}`)

func New(store Store, git Git) *Service {
	return &Service{store: store, git: git, now: time.Now}
}

func (service *Service) Create(slug, level, date string) (Created, error) {
	if err := assertSlug(slug); err != nil {
		return Created{}, err
	}
	if level == "" {
		level = "pending"
	}
	if level != "pending" && level != "L1" && level != "L2" && level != "L3" && level != "L4" {
		return Created{}, fmt.Errorf("Level must be L1, L2, L3, or L4")
	}
	missing, err := service.store.Missing(foundation.RequiredArtifacts)
	if err != nil {
		return Created{}, err
	}
	if len(missing) > 0 {
		return Created{}, fmt.Errorf("Project foundation is required first. Missing: %s", strings.Join(missing, ", "))
	}
	if date == "" {
		date = service.currentDate()
	}

	replacements := map[string]string{
		"<change-name>":  slug,
		"<created-date>": date,
		"<Change Title>": titleFromSlug(slug),
	}
	files := make(map[string]string, len(documents.RequiredChangeFiles))
	for _, name := range documents.RequiredChangeFiles {
		templateName := strings.TrimSuffix(name, ".md")
		content, templateErr := documents.Template(templateName)
		if templateErr != nil {
			return Created{}, templateErr
		}
		for needle, replacement := range replacements {
			content = strings.ReplaceAll(content, needle, replacement)
		}
		if name == "CHANGE.md" && level != "pending" {
			content, templateErr = documents.UpdateFrontmatter(content, map[string]any{"assurance_level": level})
			if templateErr != nil {
				return Created{}, templateErr
			}
		}
		files[name] = content
	}
	path, err := service.store.CreateChange(slug, files)
	if err != nil {
		return Created{}, err
	}
	return Created{Name: slug, State: "draft", Path: path}, nil
}

func (service *Service) Validate(slug, gate string) (GateResult, error) {
	if gate != "plan" && gate != "implement" && gate != "review" && gate != "archive" {
		return GateResult{}, fmt.Errorf("Unknown validation gate: %s", gate)
	}
	change, err := service.store.ReadChange(slug)
	if err != nil {
		return GateResult{}, err
	}
	changeValidation := documents.ValidateChangeDocument(change.Files["CHANGE.md"])
	planValidation := documents.ValidatePlanDocument(change.Files["PLAN.md"], change.Files["RESEARCH.md"])
	researchValidation := documents.ValidateResearchDocument(change.Files["RESEARCH.md"])
	tasksValidation := documents.ValidateTasksDocument(change.Files["TASKS.md"], gate != "plan")
	reviewValidation := documents.ValidateReviewDocument(change.Files["REVIEW.md"], gate == "review" || gate == "archive")

	selected := []documents.Validation{changeValidation, planValidation, researchValidation, tasksValidation}
	if gate != "plan" {
		selected = append(selected, reviewValidation)
	}
	merged := documents.MergeValidations(selected...)
	errors := append([]string{}, merged.Errors...)
	warnings := append([]string{}, merged.Warnings...)
	metadata := changeValidation.Metadata
	if metadata == nil {
		metadata = change.Metadata
	}

	if blocked := documents.ArrayStrings(metadata["blocked_by"]); len(blocked) > 0 {
		errors = append(errors, fmt.Sprintf("Change is blocked by: %s", strings.Join(blocked, ", ")))
	}
	if gate != "plan" {
		errors = append(errors, service.validateGitReference(documents.StringValue(reviewValidation.Metadata["implementation_ref"]))...)
	}
	if gate == "archive" {
		if documents.StringValue(metadata["state"]) != "validated" {
			errors = append(errors, "Change state must be validated before archive")
		}
		if conflicts := documents.ArrayStrings(metadata["conflicts_with"]); len(conflicts) > 0 {
			errors = append(errors, fmt.Sprintf("conflicts_with must be empty before archive: %s", strings.Join(conflicts, ", ")))
		}
		for _, dependency := range documents.ArrayStrings(metadata["depends_on"]) {
			exists, dependencyErr := service.store.ArchivedDependencyExists(dependency)
			if dependencyErr != nil {
				return GateResult{}, dependencyErr
			}
			if !exists {
				errors = append(errors, fmt.Sprintf("Dependency must be archived first: %s", dependency))
			}
		}
		count, countErr := service.store.FoundationFileCount(slug)
		if countErr != nil {
			return GateResult{}, countErr
		}
		status := documents.StringValue(reviewValidation.Metadata["foundation_status"])
		if status == "synced" && count == 0 {
			errors = append(errors, "foundation_status is synced but change/foundation contains no files")
		}
		if status == "not-required" && count > 0 {
			errors = append(errors, "foundation_status is not-required but change/foundation contains files")
		}
		errors = append(errors, service.uncommittedImplementationErrors(slug)...)
	}
	return GateResult{OK: len(errors) == 0, Errors: errors, Warnings: warnings, Gate: gate, State: metadata["state"]}, nil
}

func (service *Service) Transition(slug, to string) (Transitioned, error) {
	change, err := service.store.ReadChange(slug)
	if err != nil {
		return Transitioned{}, err
	}
	from := documents.StringValue(change.Metadata["state"])
	if err := AssertTransition(from, to); err != nil {
		return Transitioned{}, err
	}
	if gate := GateForTransition(from, to); gate != "" {
		result, validationErr := service.Validate(slug, gate)
		if validationErr != nil {
			return Transitioned{}, validationErr
		}
		if !result.OK {
			return Transitioned{}, fmt.Errorf("Cannot transition %s -> %s:\n- %s", from, to, strings.Join(result.Errors, "\n- "))
		}
	}
	updated, err := documents.UpdateFrontmatter(change.Files["CHANGE.md"], map[string]any{"state": to})
	if err != nil {
		return Transitioned{}, err
	}
	if err := service.store.WriteChange(slug, updated); err != nil {
		return Transitioned{}, err
	}
	return Transitioned{Name: slug, From: from, To: to}, nil
}

func (service *Service) Archive(slug, date string) (Archived, error) {
	if date == "" {
		date = service.currentDate()
	}
	if !datePattern.MatchString(date) {
		return Archived{}, fmt.Errorf("Archive date must use YYYY-MM-DD")
	}
	gate, err := service.Validate(slug, "archive")
	if err != nil {
		return Archived{}, err
	}
	if !gate.OK {
		return Archived{}, fmt.Errorf("Archive blocked:\n- %s", strings.Join(gate.Errors, "\n- "))
	}
	change, err := service.store.ReadChange(slug)
	if err != nil {
		return Archived{}, err
	}
	updated, err := documents.UpdateFrontmatter(change.Files["CHANGE.md"], map[string]any{"state": "archived"})
	if err != nil {
		return Archived{}, err
	}
	path, err := service.store.ArchiveChange(slug, date, updated, change.Files["CHANGE.md"])
	if err != nil {
		return Archived{}, err
	}
	return Archived{Name: slug, State: "archived", Path: path, CommitRequired: true}, nil
}

func (service *Service) Status(slug string) (Status, error) {
	change, err := service.store.ReadChange(slug)
	if err != nil {
		return Status{}, err
	}
	gates := make(map[string]GateResult, 4)
	for _, gate := range []string{"plan", "implement", "review", "archive"} {
		result, validationErr := service.Validate(slug, gate)
		if validationErr != nil {
			return Status{}, validationErr
		}
		gates[gate] = result
	}
	return Status{Name: slug, State: change.Metadata["state"], Metadata: change.Metadata, Gates: gates}, nil
}

func (service *Service) List() (ChangeList, error) {
	return service.store.ListChanges()
}

func (service *Service) validateGitReference(reference string) []string {
	if !service.git.IsRepository() {
		return []string{"Project must be a Git repository before implementation can be reviewed"}
	}
	refs := []string{}
	for _, item := range refRangePattern.Split(reference, -1) {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			refs = append(refs, trimmed)
		}
	}
	if len(refs) == 0 {
		return []string{"REVIEW.md implementation_ref is required"}
	}
	errors := []string{}
	for _, ref := range refs {
		if !service.git.ResolvesCommit(ref) {
			errors = append(errors, fmt.Sprintf("Implementation reference does not resolve to a commit: %s", ref))
		}
	}
	return errors
}

func (service *Service) uncommittedImplementationErrors(slug string) []string {
	files, err := service.git.UncommittedFiles()
	if err != nil {
		return []string{"Project must be a Git repository before archive"}
	}
	errors := []string{}
	allowed := []string{"docs/", "changes/" + slug + "/"}
	for _, file := range files {
		normalized := strings.Trim(strings.ReplaceAll(file, "\\", "/"), "\"")
		if !strings.HasPrefix(normalized, allowed[0]) && !strings.HasPrefix(normalized, allowed[1]) {
			errors = append(errors, fmt.Sprintf("Uncommitted implementation file blocks archive: %s", normalized))
		}
	}
	return errors
}

func (service *Service) currentDate() string {
	return service.now().UTC().Format("2006-01-02")
}

func assertSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("Change name must be lowercase kebab-case")
	}
	return nil
}

func titleFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for index, part := range parts {
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
