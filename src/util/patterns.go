package util

import (
	"path/filepath"
	"regexp"
	"strings"

	"quality-bot/src/config"
)

// ExclusionMatcher matches entities against exclusion patterns
type ExclusionMatcher struct {
	filePatterns     []string
	files            []string
	classPatterns    []*regexp.Regexp
	functionPatterns []*regexp.Regexp
}

// NewExclusionMatcher creates a new exclusion matcher from config
func NewExclusionMatcher(cfg config.ExclusionsConfig) *ExclusionMatcher {
	m := &ExclusionMatcher{
		filePatterns: cfg.FilePatterns,
		files:        cfg.Files,
	}

	// Compile class patterns
	for _, p := range cfg.ClassPatterns {
		if re, err := regexp.Compile(p); err == nil {
			m.classPatterns = append(m.classPatterns, re)
		}
	}

	// Compile function patterns
	for _, p := range cfg.FunctionPatterns {
		if re, err := regexp.Compile(p); err == nil {
			m.functionPatterns = append(m.functionPatterns, re)
		}
	}

	return m
}

// Matches checks if an entity should be excluded
func (m *ExclusionMatcher) Matches(filePath, className, funcName string) bool {
	// Check exact file matches
	for _, f := range m.files {
		if filePath == f {
			return true
		}
	}

	// Check file patterns (glob)
	for _, pattern := range m.filePatterns {
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		// Also try matching with ** patterns
		if matchDoubleGlob(pattern, filePath) {
			return true
		}
	}

	// Check class patterns
	if className != "" {
		for _, re := range m.classPatterns {
			if re.MatchString(className) {
				return true
			}
		}
	}

	// Check function patterns
	if funcName != "" {
		for _, re := range m.functionPatterns {
			if re.MatchString(funcName) {
				return true
			}
		}
	}

	return false
}

// matchDoubleGlob handles ** patterns in globs
func matchDoubleGlob(pattern, path string) bool {
	// Handle ** patterns by converting to a simpler check
	if strings.Contains(pattern, "**") {
		// Convert ** to a regex-like check
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check if path contains both parts
			if prefix == "" && suffix != "" {
				return strings.HasSuffix(path, suffix) || strings.Contains(path, "/"+suffix)
			}
			if suffix == "" && prefix != "" {
				return strings.HasPrefix(path, prefix) || strings.Contains(path, prefix+"/")
			}
			if prefix != "" && suffix != "" {
				return strings.Contains(path, prefix) && strings.Contains(path, suffix)
			}
		}
	}
	return false
}

// MatchGlob matches a path against a glob pattern
func MatchGlob(pattern, path string) bool {
	if strings.Contains(pattern, "**") {
		return matchDoubleGlob(pattern, path)
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}
