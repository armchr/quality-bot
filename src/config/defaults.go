package config

import "time"

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:        "quality-bot",
			Version:     "1.0.0",
			Description: "Technical debt detection agent",
		},
		CodeAPI: CodeAPIConfig{
			URL:     "http://localhost:8181",
			Timeout: 30 * time.Second,
			Retry: RetryConfig{
				MaxAttempts:   3,
				BackoffFactor: 1.5,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				RetryOnStatus: []int{502, 503, 504},
			},
		},
		Concurrency: ConcurrencyConfig{
			MaxParallelDetectors:    5,
			MetricsBatchSize:        100,
			SimilaritySearchWorkers: 3,
			RateLimitEnabled:        false,
			RateLimitRequestsPerSec: 10,
		},
		Cache: CacheConfig{
			Enabled:   true,
			TTL:       1 * time.Hour,
			MaxSizeMB: 256,
		},
		Detectors: DetectorsConfig{
			FailFast: false,
			Complexity: ComplexityDetectorConfig{
				Enabled:            true,
				CyclomaticModerate: 10,
				CyclomaticHigh:     15,
				CyclomaticCritical: 20,
				MaxNestingDepth:    4,
			},
			SizeAndStructure: SizeDetectorConfig{
				Enabled:          true,
				MaxFunctionLines: 50,
				MaxParameters:    5,
				MaxClassMethods:  20,
				MaxClassFields:   15,
				MaxFileLines:     500,
				MaxFileFunctions: 20,
			},
			Coupling: CouplingDetectorConfig{
				Enabled:                 true,
				MaxDependencies:         10,
				FeatureEnvyThreshold:    3,
				IntimacyCallThreshold:   3,
				PrimitiveFieldThreshold: 8,
			},
			// DeadCode: Future feature - disabled by default
			DeadCode: DeadCodeDetectorConfig{
				Enabled:     false,
				EntryPoints: []string{"main", "init", "__init__", "__str__"},
				EntryPointPatterns: []string{
					"^test_", "^Test", "^setUp$", "^tearDown$",
				},
			},
			Duplication: DuplicationDetectorConfig{
				Enabled:             true,
				SimilarityThreshold: 0.85,
				MinLines:            5,
				MaxFunctionsToCheck: 500,
				SkipTrivial:         true,
			},
		},
		Exclusions: ExclusionsConfig{
			FilePatterns: []string{
				"**/test/**", "**/tests/**", "**/generated/**",
				"**/vendor/**", "**/node_modules/**",
			},
			ClassPatterns:    []string{"^Test", "Mock$", "Stub$"},
			FunctionPatterns: []string{"^test_"},
		},
		Severity: SeverityConfig{
			MinSeverity: "low",
			Overrides:   map[string]string{},
		},
		Output: OutputConfig{
			Formats:              []string{"json"},
			OutputDir:            ".",
			IncludeSuggestions:   true,
			IncludeMetrics:       true,
			IncludeCodeSnippets:  false,
			MaxIssuesPerCategory: 100,
			HotspotsTopN:         10,
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "text",
			IncludeTimestamp: true,
			IncludeCaller:    false,
		},
	}
}
