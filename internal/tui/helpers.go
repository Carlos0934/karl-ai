package tui

import (
	"fmt"
	"sort"
	"time"
	"unicode"

	"github.com/carlos0934/karl-ai/internal/domain"
)

func stageSelection(pending map[string]Selection, original domain.AgentConfig, choice Selection) {
	if choice.Model == original.Model && choice.Variant == configVariant(original) {
		delete(pending, choice.Agent)
		return
	}
	pending[choice.Agent] = choice
}

func selectionConfig(selection Selection) domain.AgentConfig {
	variant := selection.Variant
	return domain.AgentConfig{
		Model:   selection.Model,
		Variant: &variant,
	}
}

func configVariant(config domain.AgentConfig) string {
	if config.Variant == nil {
		return ""
	}
	return *config.Variant
}

func formatModel(model, variant string) string {
	if variant == "" {
		return model
	}
	return fmt.Sprintf("%s (variant: %s)", model, variant)
}

func changeCount(count int) string {
	if count == 1 {
		return "1 change"
	}
	return fmt.Sprintf("%d changes", count)
}

func orderedSelections(pending map[string]Selection) []Selection {
	result := make([]Selection, 0, len(pending))
	for _, selection := range pending {
		result = append(result, selection)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Agent < result[j].Agent
	})
	return result
}

func orderedVariants(variants []string) []string {
	unique := map[string]bool{}
	for _, v := range variants {
		if v != "" {
			unique[v] = true
		}
	}
	result := make([]string, 0, len(unique))
	for v := range unique {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		return naturalLess(result[i], result[j])
	})
	return result
}

type modelChoice struct {
	model      domain.Model
	discovered bool
}

func orderedModelChoices(models []domain.Model, effective, original domain.AgentConfig) []modelChoice {
	choices := make([]modelChoice, 0, len(models))
	for _, model := range models {
		choices = append(choices, modelChoice{model: model, discovered: true})
	}
	sort.Slice(choices, func(i, j int) bool {
		leftDate, leftValid := releaseDate(choices[i].model.ReleaseDate)
		rightDate, rightValid := releaseDate(choices[j].model.ReleaseDate)
		if leftValid != rightValid {
			return leftValid
		}
		if leftValid && !leftDate.Equal(rightDate) {
			return leftDate.After(rightDate)
		}
		return naturalLess(choices[i].model.ID, choices[j].model.ID)
	})
	return choices
}

func modelChoiceLabel(choice modelChoice, effective, original domain.AgentConfig) string {
	label := choice.model.Name
	if label == "" {
		label = choice.model.ID
	}
	if choice.model.ReleaseDate != "" {
		label += " - " + choice.model.ReleaseDate
	}
	pending := configsDiffer(original, effective)
	configuredModel := choice.model.ID == original.Model
	pendingModel := pending && choice.model.ID == effective.Model
	if configuredModel {
		label += " [current]"
	}
	if pendingModel {
		label += " [pending]"
	}
	if configuredModel && pendingModel {
		label += fmt.Sprintf(" (current: %s; pending: %s)", variantDisplay(configVariant(original)), variantDisplay(configVariant(effective)))
	}
	return label
}

func modelContextDescription(original, effective domain.AgentConfig) string {
	if original.Model == "" {
		if effective.Model == "" {
			return "Choose a model."
		}
		return "Pending: " + formatModel(effective.Model, configVariant(effective))
	}
	description := "Current: " + formatModel(original.Model, configVariant(original))
	if configsDiffer(original, effective) {
		description += " | Pending: " + formatModel(effective.Model, configVariant(effective))
	}
	return description
}

func variantContextDescription(selectedModel string, original, effective domain.AgentConfig) string {
	description := "Selected model: " + selectedModel
	if original.Model != "" {
		description += " | Current: " + formatModel(original.Model, configVariant(original))
	}
	if configsDiffer(original, effective) && effective.Model != "" {
		description += " | Pending: " + formatModel(effective.Model, configVariant(effective))
	}
	return description
}

func configsDiffer(original, effective domain.AgentConfig) bool {
	return original.Model != effective.Model || configVariant(original) != configVariant(effective)
}

func variantDisplay(variant string) string {
	if variant == "" {
		return "none"
	}
	return variant
}

func releaseDate(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	return parsed, err == nil
}

func naturalLess(left, right string) bool {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	i, j := 0, 0
	for i < len(leftRunes) && j < len(rightRunes) {
		if unicode.IsDigit(leftRunes[i]) && unicode.IsDigit(rightRunes[j]) {
			leftNum := 0
			for i < len(leftRunes) && unicode.IsDigit(leftRunes[i]) {
				leftNum = leftNum*10 + int(leftRunes[i]-'0')
				i++
			}
			rightNum := 0
			for j < len(rightRunes) && unicode.IsDigit(rightRunes[j]) {
				rightNum = rightNum*10 + int(rightRunes[j]-'0')
				j++
			}
			if leftNum != rightNum {
				return leftNum < rightNum
			}
			continue
		}
		if leftRunes[i] != rightRunes[j] {
			return leftRunes[i] < rightRunes[j]
		}
		i++
		j++
	}
	return len(leftRunes) < len(rightRunes)
}

func validateModelReference(value string) error {
	return domain.ValidateModelReference(value)
}

func validateVariant(value string) error {
	return domain.ValidateVariant(value)
}
