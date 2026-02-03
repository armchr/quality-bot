package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading from YAML files
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads configuration from a YAML file with environment variable substitution.
// Environment variables can be referenced in the YAML using:
//   - ${VAR_NAME} - substitutes the value of VAR_NAME, empty string if not set
//   - ${VAR_NAME:-default} - substitutes VAR_NAME or "default" if not set
func (l *Loader) Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Resolve config file path
	filePath := l.resolveConfigPath(configPath)
	if filePath == "" {
		// No config file found, use defaults
		return cfg, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables in the YAML content
	expandedData := l.expandEnvVars(string(data))

	// Parse YAML
	if err := yaml.Unmarshal([]byte(expandedData), cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

func (l *Loader) resolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}

	// Try default locations
	defaults := []string{
		"config.yaml",
		"config/config.yaml",
		filepath.Join(os.Getenv("HOME"), ".quality-bot", "config.yaml"),
	}

	for _, path := range defaults {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// expandEnvVars expands environment variable references in the input string.
// Supports two formats:
//   - ${VAR_NAME} - replaced with the value of VAR_NAME (empty if not set)
//   - ${VAR_NAME:-default} - replaced with VAR_NAME value, or "default" if not set
func (l *Loader) expandEnvVars(input string) string {
	// Pattern matches ${VAR} or ${VAR:-default}
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name and optional default
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		defaultVal := ""
		if len(submatches) >= 3 {
			defaultVal = submatches[2]
		}

		// Get environment variable value
		if val, exists := os.LookupEnv(varName); exists {
			return val
		}

		return defaultVal
	})
}
