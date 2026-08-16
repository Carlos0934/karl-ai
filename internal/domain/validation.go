package domain

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// ValidateModelReference validates the persisted provider/model form.
func ValidateModelReference(value string) error {
	provider, model, ok := strings.Cut(value, "/")
	if value != strings.TrimSpace(value) || !ok || provider == "" || model == "" || strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.ContainsRune(value, '#') {
		return errors.New("model reference must be trimmed provider/model and contain no whitespace or #")
	}
	return nil
}

// ValidateVariant validates a persisted optional model variant.
func ValidateVariant(value string) error {
	if value != strings.TrimSpace(value) || strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.ContainsRune(value, '#') {
		return errors.New("variant must be trimmed and contain no whitespace or #")
	}
	return nil
}

// ValidateManagedPattern ensures pattern strings are secure and safe relative paths.
func ValidateManagedPattern(pattern string) error {
	if pattern == "" || pattern == "." {
		return fmt.Errorf("path must not be empty")
	}
	if strings.Contains(pattern, "\\") || path.IsAbs(pattern) || filepath.IsAbs(pattern) || filepath.VolumeName(pattern) != "" || hasWindowsDrivePrefix(pattern) {
		return fmt.Errorf("path must be relative and slash-normalized")
	}
	if path.Clean(pattern) != pattern {
		return fmt.Errorf("path must be normalized")
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == ".." {
			return fmt.Errorf("path must not contain ..")
		}
	}

	if !strings.ContainsAny(pattern, "*?[") {
		return nil
	}
	if strings.ContainsAny(pattern, "?[") || !strings.HasSuffix(pattern, "/**") {
		return fmt.Errorf("wildcards are limited to directory * patterns ending in /**")
	}
	prefix := strings.TrimSuffix(pattern, "/**")
	if prefix == "" || strings.Contains(prefix, "*") {
		return fmt.Errorf("wildcards are limited to directory * patterns ending in /**")
	}
	return nil
}

func hasWindowsDrivePrefix(value string) bool {
	return len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z'))
}
