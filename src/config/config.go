package config

import "time"

// Config is the root configuration structure
type Config struct {
	Agent       AgentConfig       `yaml:"agent"`
	CodeAPI     CodeAPIConfig     `yaml:"codeapi"`
	Concurrency ConcurrencyConfig `yaml:"concurrency"`
	Cache       CacheConfig       `yaml:"cache"`
	Detectors   DetectorsConfig   `yaml:"detectors"`
	Exclusions  ExclusionsConfig  `yaml:"exclusions"`
	Severity    SeverityConfig    `yaml:"severity"`
	Output      OutputConfig      `yaml:"output"`
	Logging     LoggingConfig     `yaml:"logging"`
}

// AgentConfig contains agent metadata
type AgentConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// CodeAPIConfig contains CodeAPI connection settings
type CodeAPIConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
	Retry   RetryConfig   `yaml:"retry"`
}

// RetryConfig contains retry settings for API calls
type RetryConfig struct {
	MaxAttempts   int           `yaml:"max_attempts"`
	BackoffFactor float64       `yaml:"backoff_factor"`
	InitialDelay  time.Duration `yaml:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	RetryOnStatus []int         `yaml:"retry_on_status"`
}

// ConcurrencyConfig contains concurrency settings
type ConcurrencyConfig struct {
	MaxParallelDetectors    int  `yaml:"max_parallel_detectors"`
	MetricsBatchSize        int  `yaml:"metrics_batch_size"`
	SimilaritySearchWorkers int  `yaml:"similarity_search_workers"`
	RateLimitEnabled        bool `yaml:"rate_limit_enabled"`
	RateLimitRequestsPerSec int  `yaml:"rate_limit_requests_per_sec"`
}

// CacheConfig contains caching settings
type CacheConfig struct {
	Enabled   bool          `yaml:"enabled"`
	TTL       time.Duration `yaml:"ttl"`
	MaxSizeMB int           `yaml:"max_size_mb"`
}

// DetectorsConfig contains settings for all detectors
type DetectorsConfig struct {
	FailFast         bool                      `yaml:"fail_fast"`
	Complexity       ComplexityDetectorConfig  `yaml:"complexity"`
	SizeAndStructure SizeDetectorConfig        `yaml:"size_and_structure"`
	Coupling         CouplingDetectorConfig    `yaml:"coupling"`
	DeadCode         DeadCodeDetectorConfig    `yaml:"dead_code"`
	Duplication      DuplicationDetectorConfig `yaml:"duplication"`
}

// ComplexityDetectorConfig contains complexity detector settings
type ComplexityDetectorConfig struct {
	Enabled            bool `yaml:"enabled"`
	CyclomaticModerate int  `yaml:"cyclomatic_moderate"`
	CyclomaticHigh     int  `yaml:"cyclomatic_high"`
	CyclomaticCritical int  `yaml:"cyclomatic_critical"`
	MaxNestingDepth    int  `yaml:"max_nesting_depth"`
}

// SizeDetectorConfig contains size detector settings
type SizeDetectorConfig struct {
	Enabled          bool `yaml:"enabled"`
	MaxFunctionLines int  `yaml:"max_function_lines"`
	MaxParameters    int  `yaml:"max_parameters"`
	MaxClassMethods  int  `yaml:"max_class_methods"`
	MaxClassFields   int  `yaml:"max_class_fields"`
	MaxFileLines     int  `yaml:"max_file_lines"`
	MaxFileFunctions int  `yaml:"max_file_functions"`
}

// CouplingDetectorConfig contains coupling detector settings
type CouplingDetectorConfig struct {
	Enabled                 bool `yaml:"enabled"`
	MaxDependencies         int  `yaml:"max_dependencies"`
	FeatureEnvyThreshold    int  `yaml:"feature_envy_threshold"`
	IntimacyCallThreshold   int  `yaml:"intimacy_call_threshold"`
	PrimitiveFieldThreshold int  `yaml:"primitive_field_threshold"`
}

// DeadCodeDetectorConfig contains dead code detector settings (FUTURE FEATURE)
type DeadCodeDetectorConfig struct {
	Enabled            bool     `yaml:"enabled"`
	EntryPoints        []string `yaml:"entry_points"`
	EntryPointPatterns []string `yaml:"entry_point_patterns"`
}

// DuplicationDetectorConfig contains duplication detector settings
type DuplicationDetectorConfig struct {
	Enabled             bool    `yaml:"enabled"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
	MinLines            int     `yaml:"min_lines"`
	MaxFunctionsToCheck int     `yaml:"max_functions_to_check"`
	SkipTrivial         bool    `yaml:"skip_trivial"`
}

// ExclusionsConfig contains exclusion patterns
type ExclusionsConfig struct {
	FilePatterns     []string `yaml:"file_patterns"`
	Files            []string `yaml:"files"`
	ClassPatterns    []string `yaml:"class_patterns"`
	FunctionPatterns []string `yaml:"function_patterns"`
	Languages        []string `yaml:"languages"`
}

// SeverityConfig contains severity settings
type SeverityConfig struct {
	MinSeverity string            `yaml:"min_severity"`
	Overrides   map[string]string `yaml:"overrides"`
}

// OutputConfig contains output settings
type OutputConfig struct {
	Formats              []string `yaml:"formats"`
	OutputDir            string   `yaml:"output_dir"`
	IncludeSuggestions   bool     `yaml:"include_suggestions"`
	IncludeMetrics       bool     `yaml:"include_metrics"`
	IncludeCodeSnippets  bool     `yaml:"include_code_snippets"`
	MaxIssuesPerCategory int      `yaml:"max_issues_per_category"`
	HotspotsTopN         int      `yaml:"hotspots_top_n"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level            string `yaml:"level"`
	Format           string `yaml:"format"` // text, json
	File             string `yaml:"file"`
	IncludeTimestamp bool   `yaml:"include_timestamp"`
	IncludeCaller    bool   `yaml:"include_caller"`
}
