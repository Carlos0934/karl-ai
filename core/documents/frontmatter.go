package documents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Field struct {
	Key   string
	Value any
}

type Frontmatter struct {
	Data  map[string]any
	Body  string
	order []string
}

var plainScalar = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func ParseFrontmatter(markdown string) (Frontmatter, error) {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return Frontmatter{}, fmt.Errorf("Missing YAML frontmatter opening delimiter")
	}

	closing := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			closing = i
			break
		}
	}
	if closing == -1 {
		return Frontmatter{}, fmt.Errorf("Missing YAML frontmatter closing delimiter")
	}

	data := make(map[string]any)
	order := make([]string, 0, closing-1)
	linePattern := regexp.MustCompile(`^([A-Za-z0-9_-]+):\s*(.*)$`)
	for _, line := range lines[1:closing] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		match := linePattern.FindStringSubmatch(line)
		if match == nil {
			return Frontmatter{}, fmt.Errorf("Unsupported frontmatter line: %s", line)
		}
		if _, exists := data[match[1]]; !exists {
			order = append(order, match[1])
		}
		data[match[1]] = parseScalar(match[2])
	}

	body := strings.Join(lines[closing+1:], "\n")
	body = strings.TrimPrefix(body, "\n")
	return Frontmatter{Data: data, Body: body, order: order}, nil
}

func SerializeFrontmatter(fields []Field, body string) string {
	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		lines = append(lines, field.Key+": "+formatScalar(field.Value))
	}
	return "---\n" + strings.Join(lines, "\n") + "\n---\n\n" + strings.TrimLeft(body, "\n")
}

func UpdateFrontmatter(markdown string, updates map[string]any) (string, error) {
	parsed, err := ParseFrontmatter(markdown)
	if err != nil {
		return "", err
	}
	for key, value := range updates {
		parsed.Data[key] = value
	}

	fields := make([]Field, 0, len(parsed.Data))
	seen := make(map[string]bool)
	for _, key := range parsed.order {
		fields = append(fields, Field{Key: key, Value: parsed.Data[key]})
		seen[key] = true
	}
	remaining := make([]string, 0)
	for key := range parsed.Data {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		fields = append(fields, Field{Key: key, Value: parsed.Data[key]})
	}
	return SerializeFrontmatter(fields, parsed.Body), nil
}

func parseScalar(raw string) any {
	value := strings.TrimSpace(raw)
	switch value {
	case "":
		return ""
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		if strings.TrimSpace(value[1:len(value)-1]) == "" {
			return []any{}
		}
		var result []any
		if json.Unmarshal([]byte(value), &result) == nil {
			return result
		}
		parts := strings.Split(value[1:len(value)-1], ",")
		result = make([]any, 0, len(parts))
		for _, part := range parts {
			result = append(result, strings.Trim(strings.TrimSpace(part), "'\""))
		}
		return result
	}

	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		if value[0] == '"' {
			var decoded string
			if json.Unmarshal([]byte(value), &decoded) == nil {
				return decoded
			}
		}
		return value[1 : len(value)-1]
	}
	return value
}

func formatScalar(value any) string {
	if value == nil {
		return "null"
	}
	switch typed := value.(type) {
	case []any, []string:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(typed)
	}
	text := fmt.Sprint(value)
	if text != "" && plainScalar.MatchString(text) {
		return text
	}
	encoded, _ := json.Marshal(text)
	return string(encoded)
}
