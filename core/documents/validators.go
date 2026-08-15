package documents

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Validation struct {
	OK       bool           `json:"ok"`
	Errors   []string       `json:"errors"`
	Warnings []string       `json:"warnings"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Units    []WorkUnit     `json:"units,omitempty"`
}

type WorkUnit struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

var (
	commentPattern     = regexp.MustCompile(`(?s)<!--[\s\S]*?-->`)
	placeholderPattern = regexp.MustCompile(`<[^>\n]+>`)
	headingPattern     = regexp.MustCompile(`^(#{1,6})\s+`)
	workUnitPattern    = regexp.MustCompile(`^##\s+(\d+)\.\s+(.+)$`)
	incompleteBox      = regexp.MustCompile(`(?m)^- \[ \]`)
)

var assuranceLevels = stringSet("L1", "L2", "L3", "L4")
var userValidation = stringSet("pending", "accepted", "rejected")
var foundationStatus = stringSet("pending", "synced", "not-required")
var blockingFindings = stringSet("unknown", "open", "resolved", "none")
var horizontalTitles = stringSet("models", "services", "tests", "database", "infrastructure", "repositories", "controllers")

func ValidateChangeDocument(markdown string) Validation {
	errors := []string{}
	metadata := parseMetadata(markdown, &errors)
	for _, field := range []string{"name", "state", "created", "assurance_level"} {
		if !truthy(metadata[field]) {
			errors = append(errors, fmt.Sprintf("Missing CHANGE.md frontmatter field: %s", field))
		}
	}
	if state := stringValue(metadata["state"]); state != "" && !IsState(state) {
		errors = append(errors, fmt.Sprintf("Invalid change state: %s", state))
	}
	if level := stringValue(metadata["assurance_level"]); level != "" && !assuranceLevels[level] {
		errors = append(errors, "assurance_level must be one of L1, L2, L3, or L4")
	}
	for _, field := range []string{"depends_on", "conflicts_with", "blocked_by"} {
		if !isArray(metadata[field]) {
			errors = append(errors, fmt.Sprintf("%s must be an inline array in CHANGE.md frontmatter", field))
		}
	}
	requireSections(markdown, []string{"Outcome", "Scope", "Acceptance Criteria", "Foundation Areas"}, 2, &errors, true)
	return newValidation(errors, nil, metadata, nil)
}

func ValidatePlanDocument(markdown, researchMarkdown string) Validation {
	errors := []string{}
	requireSections(markdown, []string{"Decision Basis", "Technical Approach", "Work Units", "Integrated Validation"}, 2, &errors, true)

	workUnits := extractSection(markdown, "Work Units", 2)
	dataRow := regexp.MustCompile(`^\|\s*\d+\s*\|`)
	hasRow := false
	for _, line := range strings.Split(workUnits, "\n") {
		if dataRow.MatchString(strings.TrimSpace(line)) {
			hasRow = true
			break
		}
	}
	if !hasRow {
		errors = append(errors, "PLAN.md Work Units must contain at least one numbered row")
	}

	decisionBasis := extractSection(markdown, "Decision Basis", 2)
	citationPattern := regexp.MustCompile(`(?im)^\s*cites:\s*(.+?)\s*$`)
	citationMatches := citationPattern.FindAllStringSubmatch(decisionBasis, -1)
	if len(citationMatches) == 0 {
		errors = append(errors, "PLAN.md Decision Basis must cite at least one RESEARCH section")
	} else if researchMarkdown != "" {
		available := map[string]bool{}
		for _, title := range researchSectionTitles() {
			content, found := findSection(researchMarkdown, title, 2)
			if found && meaningful(content) {
				available[title] = true
			}
		}
		for _, match := range citationMatches {
			citation := match[1]
			if !available[citation] {
				errors = append(errors, fmt.Sprintf("PLAN.md cites unknown or incomplete RESEARCH section: %s", citation))
			}
		}
	}
	return newValidation(errors, nil, nil, nil)
}

func ValidateResearchDocument(markdown string) Validation {
	errors := []string{}
	requireSections(markdown, researchSectionTitles(), 2, &errors, true)
	return newValidation(errors, nil, nil, nil)
}

func ParseWorkUnits(markdown string) []WorkUnit {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	type start struct {
		index  int
		number int
		title  string
	}
	starts := []start{}
	for index, line := range lines {
		match := workUnitPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		starts = append(starts, start{index: index, number: number, title: strings.TrimSpace(match[2])})
	}
	units := make([]WorkUnit, 0, len(starts))
	for position, item := range starts {
		end := len(lines)
		if position+1 < len(starts) {
			end = starts[position+1].index
		}
		units = append(units, WorkUnit{Number: item.number, Title: item.title, Content: strings.TrimSpace(strings.Join(lines[item.index+1:end], "\n"))})
	}
	return units
}

func ValidateTasksDocument(markdown string, requireComplete bool) Validation {
	errors := []string{}
	warnings := []string{}
	units := ParseWorkUnits(markdown)
	if len(units) == 0 {
		errors = append(errors, "TASKS.md must contain at least one work unit using '## N. Name'")
		return newValidation(errors, warnings, nil, units)
	}

	allNumbers := map[int]bool{}
	for _, unit := range units {
		allNumbers[unit.Number] = true
	}
	seenNumbers := map[int]bool{}
	for index, unit := range units {
		if seenNumbers[unit.Number] {
			errors = append(errors, fmt.Sprintf("Duplicate work unit number: %d", unit.Number))
		}
		seenNumbers[unit.Number] = true
		if unit.Number != index+1 {
			errors = append(errors, "Work unit numbers must be sequential starting at 1")
		}
		if containsPlaceholder(unit.Title) {
			errors = append(errors, fmt.Sprintf("Work Unit %d has a placeholder title", unit.Number))
		}

		enabling := regexp.MustCompile(`(?i)\*\*Type:\*\*\s*Enabling Work`).MatchString(unit.Content)
		if horizontalTitles[strings.ToLower(unit.Title)] && !enabling {
			errors = append(errors, fmt.Sprintf("Work Unit %d is split by technical layer: %s", unit.Number, unit.Title))
		}
		if enabling && !regexp.MustCompile(`(?i)\*\*Required by:\*\*`).MatchString(unit.Content) {
			errors = append(errors, fmt.Sprintf("Enabling Work Unit %d must declare '**Required by:**' consumers", unit.Number))
		}
		requireSections(unit.Content, []string{"Outcome", "Acceptance", "Work", "Validation", "Complete When"}, 3, &errors, true)

		work := extractSection(unit.Content, "Work", 3)
		taskPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^- \[([ xX])\]\s+%d\.(\d+)\s+\*\*(Prepare|Implement|Validate):\*\*`, unit.Number))
		categories := map[string]bool{}
		taskIDs := map[string]bool{}
		for _, match := range taskPattern.FindAllStringSubmatch(work, -1) {
			categories[match[3]] = true
			if taskIDs[match[2]] {
				errors = append(errors, fmt.Sprintf("Duplicate task ID %d.%s", unit.Number, match[2]))
			}
			taskIDs[match[2]] = true
		}
		for _, category := range []string{"Prepare", "Implement", "Validate"} {
			if !categories[category] {
				errors = append(errors, fmt.Sprintf("Work Unit %d must include a %s task", unit.Number, category))
			}
		}

		dependsPattern := regexp.MustCompile(`(?i)\*\*Depends on:\*\*\s*(.+)`)
		dependsMatch := dependsPattern.FindStringSubmatch(unit.Content)
		if dependsMatch != nil && !regexp.MustCompile(`(?i)^None\.?$`).MatchString(strings.TrimSpace(dependsMatch[1])) {
			dependencyPattern := regexp.MustCompile(`(?i)(?:Work\s+)?Unit\s+(\d+)`)
			matches := dependencyPattern.FindAllStringSubmatch(dependsMatch[1], -1)
			if len(matches) == 0 {
				warnings = append(warnings, fmt.Sprintf("Work Unit %d has an unparsed dependency declaration", unit.Number))
			}
			for _, match := range matches {
				dependency, _ := strconv.Atoi(match[1])
				if !allNumbers[dependency] {
					errors = append(errors, fmt.Sprintf("Work Unit %d depends on unknown Work Unit %d", unit.Number, dependency))
				} else if dependency >= unit.Number {
					errors = append(errors, fmt.Sprintf("Work Unit %d must depend only on an earlier work unit", unit.Number))
				}
			}
		}
	}
	if requireComplete && incompleteBox.MatchString(markdown) {
		errors = append(errors, "TASKS.md contains incomplete checkboxes")
	}
	return newValidation(errors, warnings, nil, units)
}

func ValidateReviewDocument(markdown string, final bool) Validation {
	errors := []string{}
	metadata := parseMetadata(markdown, &errors)
	if !userValidation[stringValue(metadata["user_validation"])] {
		errors = append(errors, "REVIEW.md user_validation has an invalid value")
	}
	if !foundationStatus[stringValue(metadata["foundation_status"])] {
		errors = append(errors, "REVIEW.md foundation_status has an invalid value")
	}
	if !blockingFindings[stringValue(metadata["blocking_findings"])] {
		errors = append(errors, "REVIEW.md blocking_findings has an invalid value")
	}
	sections := []string{"Target", "Change-Specific Considerations", "Evidence", "Runtime Verification", "Findings", "User Validation", "Foundation Reconciliation", "Archive Decision"}
	requireSections(markdown, sections, 2, &errors, final)
	if final {
		if !truthy(metadata["implementation_ref"]) {
			errors = append(errors, "REVIEW.md implementation_ref is required")
		}
		if stringValue(metadata["user_validation"]) != "accepted" {
			errors = append(errors, "User validation must be accepted")
		}
		if status := stringValue(metadata["foundation_status"]); status != "synced" && status != "not-required" {
			errors = append(errors, "Foundation must be synced or not-required")
		}
		if findings := stringValue(metadata["blocking_findings"]); findings != "resolved" && findings != "none" {
			errors = append(errors, "Blocking findings must be resolved or none")
		}
		if !regexp.MustCompile(`(?i)\|\s*Status\s*\|\s*Accepted\s*\|`).MatchString(extractSection(markdown, "User Validation", 2)) {
			errors = append(errors, "User Validation table must record Status as Accepted")
		}
		if !regexp.MustCompile(`(?i)\|\s*Ready to archive\s*\|\s*Yes\s*\|`).MatchString(extractSection(markdown, "Archive Decision", 2)) {
			errors = append(errors, "Archive Decision must record Ready to archive as Yes")
		}
	}
	return newValidation(errors, nil, metadata, nil)
}

func MergeValidations(results ...Validation) Validation {
	errors := []string{}
	warnings := []string{}
	for _, result := range results {
		errors = append(errors, result.Errors...)
		warnings = append(warnings, result.Warnings...)
	}
	return newValidation(errors, warnings, nil, nil)
}

func IsState(value string) bool {
	return stringSet("draft", "planned", "implementing", "reviewing", "validated", "archived")[value]
}

func ArrayStrings(value any) []string {
	array, ok := value.([]any)
	if !ok {
		if stringsArray, ok := value.([]string); ok {
			return stringsArray
		}
		return nil
	}
	result := make([]string, 0, len(array))
	for _, item := range array {
		result = append(result, fmt.Sprint(item))
	}
	return result
}

func StringValue(value any) string { return stringValue(value) }

func newValidation(errors, warnings []string, metadata map[string]any, units []WorkUnit) Validation {
	if errors == nil {
		errors = []string{}
	}
	if warnings == nil {
		warnings = []string{}
	}
	return Validation{OK: len(errors) == 0, Errors: errors, Warnings: warnings, Metadata: metadata, Units: units}
}

func requireSections(markdown string, titles []string, level int, errors *[]string, complete bool) {
	for _, title := range titles {
		content, found := findSection(markdown, title, level)
		if !found {
			*errors = append(*errors, fmt.Sprintf("Missing required section: %s %s", strings.Repeat("#", level), title))
		} else if complete && !meaningful(content) {
			*errors = append(*errors, fmt.Sprintf("Incomplete required section: %s", title))
		}
	}
}

func extractSection(markdown, title string, level int) string {
	content, _ := findSection(markdown, title, level)
	return content
}

func findSection(markdown, title string, level int) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	marker := strings.Repeat("#", level) + " " + title
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == marker {
			start = index
			break
		}
	}
	if start == -1 {
		return "", false
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		match := headingPattern.FindStringSubmatch(lines[index])
		if match != nil && len(match[1]) <= level {
			end = index
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start+1:end], "\n")), true
}

func meaningful(value string) bool {
	stripped := strings.TrimSpace(commentPattern.ReplaceAllString(value, ""))
	codeFence := regexp.MustCompile("```[A-Za-z]*\\n?")
	stripped = strings.TrimSpace(strings.ReplaceAll(codeFence.ReplaceAllString(stripped, ""), "```", ""))
	return stripped != "" && stripped != "-" && !containsPlaceholder(stripped)
}

func containsPlaceholder(value string) bool {
	return placeholderPattern.MatchString(strings.TrimSpace(commentPattern.ReplaceAllString(value, "")))
}

func parseMetadata(markdown string, errors *[]string) map[string]any {
	parsed, err := ParseFrontmatter(markdown)
	if err != nil {
		*errors = append(*errors, err.Error())
		return map[string]any{}
	}
	return parsed.Data
}

func truthy(value any) bool {
	if value == nil || value == false || value == "" {
		return false
	}
	return true
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func isArray(value any) bool {
	switch value.(type) {
	case []any, []string:
		return true
	default:
		return false
	}
}

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func researchSectionTitles() []string {
	return []string{"Research Questions", "Repository Evidence", "External Sources", "Risks And Gaps", "Open Questions"}
}
