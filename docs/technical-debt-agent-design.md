# Technical Debt Detection Agent Design

This document describes the design of a Go-based agent that identifies code-level technical debt by leveraging the CodeAPI service for code graph analysis and semantic search.

## Overview

The Technical Debt Agent (quality-bot) analyzes codebases to detect various forms of code-level debt including complexity issues, code smells, and maintainability problems. It uses CodeAPI's Neo4j code graph for structural analysis and vector embeddings for similarity-based detection.

**Key characteristics:**
- Written in Go
- CLI-first design with HTTP endpoint support planned
- Handler → Controller → Service layered architecture
- Configurable detectors with threshold customization

---

## Architecture

### Layered Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              quality-bot                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                           HANDLERS                                      │ │
│  │  ┌──────────────────┐              ┌──────────────────┐                │ │
│  │  │   CLI Handler    │              │   HTTP Handler   │  (future)      │ │
│  │  │  (cobra/flags)   │              │     (gin/chi)    │                │ │
│  │  └────────┬─────────┘              └────────┬─────────┘                │ │
│  └───────────┼─────────────────────────────────┼──────────────────────────┘ │
│              │                                 │                             │
│              └─────────────┬───────────────────┘                             │
│                            │                                                 │
│  ┌─────────────────────────▼──────────────────────────────────────────────┐ │
│  │                         CONTROLLERS                                     │ │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │ │
│  │  │ AnalysisController│  │ ReportController │  │ ConfigController │      │ │
│  │  │ (orchestration)  │  │ (report gen)     │  │ (config mgmt)    │      │ │
│  │  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘      │ │
│  └───────────┼─────────────────────┼─────────────────────┼────────────────┘ │
│              │                     │                     │                   │
│              └─────────────────────┼─────────────────────┘                   │
│                                    │                                         │
│  ┌─────────────────────────────────▼──────────────────────────────────────┐ │
│  │                          SERVICES                                       │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │ │
│  │  │ CodeAPI      │  │ Metrics      │  │ Detector     │  │ Report     │  │ │
│  │  │ Client       │  │ Provider     │  │ Runner       │  │ Generator  │  │ │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └─────┬──────┘  │ │
│  └─────────┼─────────────────┼─────────────────┼────────────────┼─────────┘ │
│            │                 │                 │                │           │
├────────────┼─────────────────┼─────────────────┼────────────────┼───────────┤
│            │                 │                 │                │           │
│            ▼                 │                 │                ▼           │
│  ┌──────────────────┐        │                 │      ┌──────────────────┐  │
│  │   CodeAPI        │        │                 │      │   File System    │  │
│  │   (external)     │◄───────┴─────────────────┘      │   (reports)      │  │
│  └──────────────────┘                                 └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Responsibility | Examples |
|-------|----------------|----------|
| **Handler** | Parse input (CLI args, HTTP requests), validate, call controllers, format output | `CLIHandler`, `HTTPHandler` |
| **Controller** | Business logic, orchestration, coordinate multiple services | `AnalysisController`, `ReportController` |
| **Service** | Single-responsibility operations, external API calls, data access | `CodeAPIClient`, `MetricsProvider`, `DetectorRunner` |

---

## Service Layer Architecture

The service layer has three main components with distinct responsibilities:

```
┌─────────────────────────────────────────────────────────────┐
│                    DetectorRunner                            │
│         (Registry + orchestration of detectors)              │
│                                                              │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│   │ Complexity  │ │    Size     │ │  Coupling   │  ...      │
│   │  Detector   │ │  Detector   │ │  Detector   │           │
│   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘           │
└──────────┼───────────────┼───────────────┼──────────────────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    MetricsProvider                           │
│      (High-level metrics abstraction + caching)              │
│                                                              │
│   GetAllFunctionMetrics() → []FunctionMetrics                │
│   GetAllClassMetrics()    → []ClassMetrics                   │
│   GetClassPairMetrics()   → []ClassPairMetrics               │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    CodeAPIClient                             │
│         (Low-level HTTP client for CodeAPI)                  │
│                                                              │
│   ExecuteCypher(query)    → raw results                      │
│   SearchSimilarCode(...)  → similarity matches               │
│   GetFunctions(...)       → function list                    │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   CodeAPI    │
                    │  (external)  │
                    └──────────────┘
```

### Service Responsibilities

| Service | Responsibility | What It Knows About |
|---------|----------------|---------------------|
| **CodeAPIClient** | Makes HTTP calls to CodeAPI, handles retries/timeouts, serializes requests | Raw API endpoints, Cypher queries, HTTP details |
| **MetricsProvider** | Computes structured metrics from raw data, caches results, provides clean API for detectors | How to transform raw query results into `FunctionMetrics`, `ClassMetrics`, etc. |
| **DetectorRunner** | Runs detectors in parallel, manages concurrency, aggregates issues | Which detectors exist, how to orchestrate them |

### Why Three Layers?

**CodeAPIClient** (low-level):
```go
// Knows about HTTP, retries, raw Cypher
results, _ := client.ExecuteCypher(ctx, repo, `
    MATCH (f:Function) WHERE f.repo = $repo RETURN f.name, f.start_line...
`)
// Returns: []map[string]any (raw data)
```

**MetricsProvider** (abstraction):
```go
// Knows about metrics computation, caching
// Hides Cypher complexity from detectors
functions, _ := metricsProvider.GetAllFunctionMetrics(ctx)
// Returns: []FunctionMetrics (structured, cached)
```

**DetectorRunner** (orchestration):
```go
// Knows about running detectors, concurrency
issues, _ := detectorRunner.RunAll(ctx)
// Returns: []DebtIssue (aggregated from all detectors)
```

### Benefit: Clean Detector Code

Detectors don't need to know about Cypher queries, HTTP calls, or caching logic. They just consume metrics:

```go
func (d *ComplexityDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    // Simple, clean API - no Cypher here
    functions, err := d.Metrics.GetAllFunctionMetrics(ctx)

    for _, fn := range functions {
        if fn.CyclomaticComplexity > threshold {
            // Create issue
        }
    }
}
```

---

## Project Structure

```
quality-bot/
├── cmd/
│   └── quality-bot/
│       └── main.go              # Entry point
├── src/
│   ├── config/
│   │   ├── config.go            # Configuration structs
│   │   ├── loader.go            # YAML/env loading
│   │   └── defaults.go          # Default values
│   ├── handler/
│   │   ├── cli/
│   │   │   ├── handler.go       # CLI handler
│   │   │   ├── analyze.go       # analyze command
│   │   │   ├── report.go        # report command
│   │   │   └── version.go       # version command
│   │   └── http/                # (future)
│   │       ├── handler.go
│   │       ├── routes.go
│   │       └── middleware.go
│   ├── controller/
│   │   ├── analysis.go          # AnalysisController
│   │   ├── report.go            # ReportController
│   │   └── config.go            # ConfigController
│   ├── service/
│   │   ├── codeapi/
│   │   │   ├── client.go        # CodeAPIClient (HTTP client)
│   │   │   ├── models.go        # Request/response types
│   │   │   └── queries.go       # Cypher query templates
│   │   ├── metrics/
│   │   │   ├── provider.go      # MetricsProvider
│   │   │   ├── function.go      # FunctionMetrics queries
│   │   │   ├── class.go         # ClassMetrics queries
│   │   │   └── file.go          # FileMetrics queries
│   │   ├── detector/
│   │   │   ├── runner.go        # DetectorRunner (orchestration)
│   │   │   ├── detector.go      # Detector interface
│   │   │   ├── complexity.go    # ComplexityDetector
│   │   │   ├── size.go          # SizeAndStructureDetector
│   │   │   ├── coupling.go      # CouplingDetector
│   │   │   └── duplication.go   # DuplicationDetector
│   │   └── report/
│   │       ├── generator.go     # ReportGenerator
│   │       ├── json.go          # JSON formatter
│   │       ├── markdown.go      # Markdown formatter
│   │       └── sarif.go         # SARIF formatter
│   ├── model/
│   │   ├── issue.go             # DebtIssue, Severity, Category
│   │   ├── metrics.go           # Metric types
│   │   └── report.go            # AnalysisReport
│   └── util/
│       ├── logger.go            # Logging utilities
│       ├── patterns.go          # Glob/regex matching
│       └── concurrency.go       # Worker pool, semaphore
├── config/
│   └── config.example.yaml      # Example configuration
├── docs/
│   └── technical-debt-agent-design.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## CodeAPI Graph Model Reference

The code graph uses specific node labels:

| Label | Description |
|-------|-------------|
| `FileScope` | Source file |
| `Class` | Class, struct, or interface |
| `Function` | Function or method |
| `Field` | Class field or property |
| `Variable` | Local variable |
| `Conditional` | If statements, switch statements |
| `Loop` | For, while, foreach loops |
| `Block` | Generic code blocks (not control flow) |
| `FunctionCall` | Function/method invocation site |

Key relationships:
- `CONTAINS` - Hierarchical containment
- `CALLS` - Function invocation
- `USES` - Variable/field usage
- `BRANCH` - Conditional branch (from Conditional to branch block)
- `INHERITS_FROM` - Class inheritance

---

## Models

### src/model/issue.go

```go
package model

import "time"

// Severity represents the severity level of a debt issue
type Severity string

const (
    SeverityLow      Severity = "low"
    SeverityMedium   Severity = "medium"
    SeverityHigh     Severity = "high"
    SeverityCritical Severity = "critical"
)

// Category represents the category of technical debt
type Category string

const (
    CategoryComplexity   Category = "complexity"
    CategorySize         Category = "size"
    CategoryCoupling     Category = "coupling"
    CategoryDuplication  Category = "duplication"
    CategoryDeadCode     Category = "dead_code"
)

// DebtIssue represents a single detected technical debt issue
type DebtIssue struct {
    Category    Category          `json:"category"`
    Subcategory string            `json:"subcategory"`
    Severity    Severity          `json:"severity"`
    FilePath    string            `json:"file_path"`
    StartLine   int               `json:"start_line"`
    EndLine     int               `json:"end_line"`
    EntityName  string            `json:"entity_name"`
    EntityType  string            `json:"entity_type"` // function, class, file
    Description string            `json:"description"`
    Metrics     map[string]any    `json:"metrics"`
    Suggestion  string            `json:"suggestion"`
}

// AnalysisReport represents the complete analysis output
type AnalysisReport struct {
    RepoName    string            `json:"repo_name"`
    GeneratedAt time.Time         `json:"generated_at"`
    Summary     ReportSummary     `json:"summary"`
    Issues      []DebtIssue       `json:"issues"`
}

// ReportSummary contains aggregated statistics
type ReportSummary struct {
    TotalIssues   int                `json:"total_issues"`
    ByCategory    map[Category]int   `json:"by_category"`
    BySeverity    map[Severity]int   `json:"by_severity"`
    HotspotFiles  []FileHotspot      `json:"hotspot_files"`
    DebtScore     float64            `json:"debt_score"`
}

// FileHotspot represents a file with many issues
type FileHotspot struct {
    FilePath   string `json:"file_path"`
    IssueCount int    `json:"issue_count"`
}
```

### src/model/metrics.go

```go
package model

// FunctionMetrics contains metrics for a single function/method
type FunctionMetrics struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    FilePath  string `json:"file_path"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`
    ClassName string `json:"class_name,omitempty"`

    // Size metrics
    LineCount      int `json:"line_count"`
    ParameterCount int `json:"parameter_count"`

    // Complexity metrics
    CyclomaticComplexity int `json:"cyclomatic_complexity"`
    ConditionalCount     int `json:"conditional_count"`
    LoopCount            int `json:"loop_count"`
    BranchCount          int `json:"branch_count"`
    MaxNestingDepth      int `json:"max_nesting_depth"`

    // Coupling metrics
    CallerCount        int `json:"caller_count"`
    CalleeCount        int `json:"callee_count"`
    ExternalCalls      int `json:"external_calls"`
    OwnFieldUses       int `json:"own_field_uses"`
    ExternalFieldUses  int `json:"external_field_uses"`
}

// ClassMetrics contains metrics for a single class
type ClassMetrics struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    FilePath  string `json:"file_path"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`

    // Size metrics
    LineCount   int `json:"line_count"`
    MethodCount int `json:"method_count"`
    FieldCount  int `json:"field_count"`

    // Composition metrics
    PrimitiveFieldCount int `json:"primitive_field_count"`

    // Coupling metrics
    DependencyCount  int `json:"dependency_count"`
    DependentCount   int `json:"dependent_count"`
    InheritanceDepth int `json:"inheritance_depth"`
}

// FileMetrics contains metrics for a single file
type FileMetrics struct {
    Path     string `json:"path"`
    Language string `json:"language"`

    // Size metrics
    LineCount     int `json:"line_count"`
    FunctionCount int `json:"function_count"`
    ClassCount    int `json:"class_count"`

    // Aggregated complexity
    TotalCyclomaticComplexity int     `json:"total_cyclomatic_complexity"`
    MaxFunctionComplexity     int     `json:"max_function_complexity"`
    AvgFunctionComplexity     float64 `json:"avg_function_complexity"`
}

// ClassPairMetrics contains coupling metrics between two classes
type ClassPairMetrics struct {
    Class1Name        string `json:"class1_name"`
    Class1File        string `json:"class1_file"`
    Class2Name        string `json:"class2_name"`
    Class2File        string `json:"class2_file"`
    Calls1To2         int    `json:"calls_1_to_2"`
    Calls2To1         int    `json:"calls_2_to_1"`
    SharedFieldAccess int    `json:"shared_field_access"`
}
```

---

## Configuration

### src/config/config.go

```go
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
    MaxParallelDetectors      int  `yaml:"max_parallel_detectors"`
    MetricsBatchSize          int  `yaml:"metrics_batch_size"`
    SimilaritySearchWorkers   int  `yaml:"similarity_search_workers"`
    RateLimitEnabled          bool `yaml:"rate_limit_enabled"`
    RateLimitRequestsPerSec   int  `yaml:"rate_limit_requests_per_sec"`
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
    Enabled              bool `yaml:"enabled"`
    CyclomaticModerate   int  `yaml:"cyclomatic_moderate"`
    CyclomaticHigh       int  `yaml:"cyclomatic_high"`
    CyclomaticCritical   int  `yaml:"cyclomatic_critical"`
    MaxNestingDepth      int  `yaml:"max_nesting_depth"`
}

// SizeDetectorConfig contains size detector settings
type SizeDetectorConfig struct {
    Enabled           bool `yaml:"enabled"`
    MaxFunctionLines  int  `yaml:"max_function_lines"`
    MaxParameters     int  `yaml:"max_parameters"`
    MaxClassMethods   int  `yaml:"max_class_methods"`
    MaxClassFields    int  `yaml:"max_class_fields"`
    MaxFileLines      int  `yaml:"max_file_lines"`
    MaxFileFunctions  int  `yaml:"max_file_functions"`
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
    Enabled             bool     `yaml:"enabled"`
    EntryPoints         []string `yaml:"entry_points"`
    EntryPointPatterns  []string `yaml:"entry_point_patterns"`
}

// DuplicationDetectorConfig contains duplication detector settings
type DuplicationDetectorConfig struct {
    Enabled              bool    `yaml:"enabled"`
    SimilarityThreshold  float64 `yaml:"similarity_threshold"`
    MinLines             int     `yaml:"min_lines"`
    MaxFunctionsToCheck  int     `yaml:"max_functions_to_check"`
    SkipTrivial          bool    `yaml:"skip_trivial"`
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
```

### src/config/loader.go

```go
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
        submatches := re.FindStringSubmatch(match)
        if len(submatches) < 2 {
            return match
        }

        varName := submatches[1]
        defaultVal := ""
        if len(submatches) >= 3 {
            defaultVal = submatches[2]
        }

        if val, exists := os.LookupEnv(varName); exists {
            return val
        }

        return defaultVal
    })
}
```

### src/config/defaults.go

```go
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
```

### Example Configuration File (config/config.example.yaml)

```yaml
# quality-bot configuration
#
# Environment variables can be referenced using:
#   ${VAR_NAME}           - substitutes the value (empty if not set)
#   ${VAR_NAME:-default}  - substitutes the value, or "default" if not set

agent:
  name: "quality-bot"
  version: "1.0.0"

codeapi:
  url: "${CODEAPI_URL:-http://localhost:8181}"
  timeout: 30s
  retry:
    max_attempts: 3
    backoff_factor: 1.5
    initial_delay: 100ms
    max_delay: 5s

concurrency:
  max_parallel_detectors: 5
  metrics_batch_size: 100
  similarity_search_workers: 3
  rate_limit_enabled: false
  rate_limit_requests_per_sec: 10

cache:
  enabled: true
  ttl: 1h
  max_size_mb: 256

detectors:
  fail_fast: false

  complexity:
    enabled: true
    cyclomatic_moderate: 10
    cyclomatic_high: 15
    cyclomatic_critical: 20
    max_nesting_depth: 4

  size_and_structure:
    enabled: true
    max_function_lines: 50
    max_parameters: 5
    max_class_methods: 20
    max_class_fields: 15
    max_file_lines: 500
    max_file_functions: 20

  coupling:
    enabled: true
    max_dependencies: 10
    feature_envy_threshold: 3
    intimacy_call_threshold: 3
    primitive_field_threshold: 8

  # dead_code: (FUTURE FEATURE - not yet implemented)
  #   enabled: false
  #   entry_points:
  #     - "main"
  #     - "init"
  #     - "__init__"
  #   entry_point_patterns:
  #     - "^test_"
  #     - "^Test"

  duplication:
    enabled: true
    similarity_threshold: 0.85
    min_lines: 5
    max_functions_to_check: 500

exclusions:
  file_patterns:
    - "**/test/**"
    - "**/tests/**"
    - "**/generated/**"
    - "**/vendor/**"
  class_patterns:
    - "^Test"
    - "Mock$"
  function_patterns:
    - "^test_"

severity:
  min_severity: "low"
  overrides:
    primitive_obsession: "low"

output:
  formats:
    - json
    - markdown
  output_dir: "${OUTPUT_DIR:-./reports}"
  include_suggestions: true
  include_metrics: true
  include_code_snippets: false
  max_issues_per_category: 100
  hotspots_top_n: 10

logging:
  level: "${LOG_LEVEL:-info}"
  format: "text"
  include_timestamp: true
```

---

## Services

### src/service/codeapi/client.go

```go
package codeapi

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "quality-bot/src/config"
)

// Client provides access to CodeAPI endpoints
type Client struct {
    baseURL    string
    httpClient *http.Client
    retryConf  config.RetryConfig
}

// NewClient creates a new CodeAPI client
func NewClient(cfg config.CodeAPIConfig) *Client {
    return &Client{
        baseURL: cfg.URL,
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
        retryConf: cfg.Retry,
    }
}

// ExecuteCypher executes a Cypher query against the code graph
func (c *Client) ExecuteCypher(ctx context.Context, repoName, query string) ([]map[string]any, error) {
    req := CypherRequest{
        RepoName: repoName,
        Query:    query,
    }

    var resp CypherResponse
    if err := c.post(ctx, "/codeapi/v1/cypher", req, &resp); err != nil {
        return nil, err
    }

    return resp.Results, nil
}

// SearchSimilarCode finds semantically similar code
func (c *Client) SearchSimilarCode(ctx context.Context, req SimilarCodeRequest) (*SimilarCodeResponse, error) {
    var resp SimilarCodeResponse
    if err := c.post(ctx, "/api/v1/searchSimilarCode", req, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

// GetFunctions retrieves functions from a repository
func (c *Client) GetFunctions(ctx context.Context, repoName string, filePath string) (*FunctionsResponse, error) {
    req := FunctionsRequest{
        RepoName: repoName,
        FilePath: filePath,
    }

    var resp FunctionsResponse
    if err := c.post(ctx, "/codeapi/v1/functions", req, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
    var lastErr error

    for attempt := 0; attempt <= c.retryConf.MaxAttempts; attempt++ {
        if attempt > 0 {
            delay := c.calculateBackoff(attempt)
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }

        err := c.doPost(ctx, path, body, result)
        if err == nil {
            return nil
        }

        lastErr = err
        if !c.shouldRetry(err) {
            break
        }
    }

    return lastErr
}

func (c *Client) doPost(ctx context.Context, path string, body any, result any) error {
    jsonBody, err := json.Marshal(body)
    if err != nil {
        return fmt.Errorf("marshaling request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonBody))
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
    }

    if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
        return fmt.Errorf("decoding response: %w", err)
    }

    return nil
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
    delay := float64(c.retryConf.InitialDelay)
    for i := 0; i < attempt; i++ {
        delay *= c.retryConf.BackoffFactor
    }
    if delay > float64(c.retryConf.MaxDelay) {
        delay = float64(c.retryConf.MaxDelay)
    }
    return time.Duration(delay)
}

func (c *Client) shouldRetry(err error) bool {
    if apiErr, ok := err.(*APIError); ok {
        for _, code := range c.retryConf.RetryOnStatus {
            if apiErr.StatusCode == code {
                return true
            }
        }
    }
    return false
}

// APIError represents an error response from CodeAPI
type APIError struct {
    StatusCode int
    Body       string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("CodeAPI error (status %d): %s", e.StatusCode, e.Body)
}
```

### src/service/metrics/provider.go

```go
package metrics

import (
    "context"
    "sync"

    "quality-bot/src/config"
    "quality-bot/src/model"
    "quality-bot/src/service/codeapi"
)

// Provider provides high-level code metrics with caching.
// It abstracts away Cypher queries and provides a clean API for detectors.
type Provider struct {
    client   *codeapi.Client
    repoName string
    cfg      config.CacheConfig

    // Cached metrics
    mu              sync.RWMutex
    functionMetrics []model.FunctionMetrics
    classMetrics    []model.ClassMetrics
    fileMetrics     []model.FileMetrics
}

// NewProvider creates a new metrics provider
func NewProvider(client *codeapi.Client, repoName string, cfg config.CacheConfig) *Provider {
    return &Provider{
        client:   client,
        repoName: repoName,
        cfg:      cfg,
    }
}

// GetAllFunctionMetrics retrieves metrics for all functions
func (p *Provider) GetAllFunctionMetrics(ctx context.Context) ([]model.FunctionMetrics, error) {
    p.mu.RLock()
    if p.functionMetrics != nil {
        defer p.mu.RUnlock()
        return p.functionMetrics, nil
    }
    p.mu.RUnlock()

    p.mu.Lock()
    defer p.mu.Unlock()

    // Double-check after acquiring write lock
    if p.functionMetrics != nil {
        return p.functionMetrics, nil
    }

    metrics, err := p.fetchFunctionMetrics(ctx)
    if err != nil {
        return nil, err
    }

    if p.cfg.Enabled {
        p.functionMetrics = metrics
    }

    return metrics, nil
}

func (p *Provider) fetchFunctionMetrics(ctx context.Context) ([]model.FunctionMetrics, error) {
    query := `
    MATCH (f:Function)
    WHERE f.repo = $repo_name

    OPTIONAL MATCH (c:Class)-[:CONTAINS]->(f)
    OPTIONAL MATCH (f)-[:CONTAINS*]->(cond:Conditional)
    OPTIONAL MATCH (f)-[:CONTAINS*]->(loop:Loop)
    OPTIONAL MATCH (f)-[:CONTAINS*]->(:Conditional)-[br:BRANCH]->()
    OPTIONAL MATCH path = (f)-[:CONTAINS*]->(nested)
    WHERE nested:Conditional OR nested:Loop
    OPTIONAL MATCH (caller:Function)-[:CALLS]->(f)
    OPTIONAL MATCH (f)-[:CALLS]->(callee:Function)
    OPTIONAL MATCH (f)-[:CALLS]->(ext:Function)<-[:CONTAINS]-(other:Class)
    WHERE other <> c
    OPTIONAL MATCH (f)-[:USES]->(own_field:Field)<-[:CONTAINS]-(c)
    OPTIONAL MATCH (f)-[:USES]->(ext_field:Field)<-[:CONTAINS]-(ext_class:Class)
    WHERE ext_class <> c

    WITH f, c,
         count(DISTINCT cond) as conditional_count,
         count(DISTINCT loop) as loop_count,
         count(DISTINCT br) as branch_count,
         max(length(path)) as max_nesting_depth,
         count(DISTINCT caller) as caller_count,
         count(DISTINCT callee) as callee_count,
         count(DISTINCT other) as external_calls,
         count(DISTINCT own_field) as own_field_uses,
         count(DISTINCT ext_field) as external_field_uses

    RETURN
        f.id as id,
        f.name as name,
        f.file_path as file_path,
        f.start_line as start_line,
        f.end_line as end_line,
        c.name as class_name,
        COALESCE(f.param_count, 0) as parameter_count,
        (f.end_line - f.start_line) as line_count,
        (1 + loop_count + branch_count) as cyclomatic_complexity,
        conditional_count,
        loop_count,
        branch_count,
        COALESCE(max_nesting_depth, 0) as max_nesting_depth,
        caller_count,
        callee_count,
        external_calls,
        own_field_uses,
        external_field_uses
    `

    results, err := p.client.ExecuteCypher(ctx, p.repoName, query)
    if err != nil {
        return nil, err
    }

    metrics := make([]model.FunctionMetrics, 0, len(results))
    for _, r := range results {
        metrics = append(metrics, model.FunctionMetrics{
            ID:                   getString(r, "id"),
            Name:                 getString(r, "name"),
            FilePath:             getString(r, "file_path"),
            StartLine:            getInt(r, "start_line"),
            EndLine:              getInt(r, "end_line"),
            ClassName:            getString(r, "class_name"),
            LineCount:            getInt(r, "line_count"),
            ParameterCount:       getInt(r, "parameter_count"),
            CyclomaticComplexity: getInt(r, "cyclomatic_complexity"),
            ConditionalCount:     getInt(r, "conditional_count"),
            LoopCount:            getInt(r, "loop_count"),
            BranchCount:          getInt(r, "branch_count"),
            MaxNestingDepth:      getInt(r, "max_nesting_depth"),
            CallerCount:          getInt(r, "caller_count"),
            CalleeCount:          getInt(r, "callee_count"),
            ExternalCalls:        getInt(r, "external_calls"),
            OwnFieldUses:         getInt(r, "own_field_uses"),
            ExternalFieldUses:    getInt(r, "external_field_uses"),
        })
    }

    return metrics, nil
}

// GetAllClassMetrics retrieves metrics for all classes
func (p *Provider) GetAllClassMetrics(ctx context.Context) ([]model.ClassMetrics, error) {
    // Similar implementation to GetAllFunctionMetrics
    // ... (omitted for brevity)
    return nil, nil
}

// GetAllFileMetrics retrieves metrics for all files
func (p *Provider) GetAllFileMetrics(ctx context.Context) ([]model.FileMetrics, error) {
    // Similar implementation to GetAllFunctionMetrics
    // ... (omitted for brevity)
    return nil, nil
}

// GetClassPairMetrics retrieves coupling metrics between class pairs
func (p *Provider) GetClassPairMetrics(ctx context.Context) ([]model.ClassPairMetrics, error) {
    // Similar implementation
    return nil, nil
}

// ClearCache clears all cached metrics
func (p *Provider) ClearCache() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.functionMetrics = nil
    p.classMetrics = nil
    p.fileMetrics = nil
}

// Helper functions
func getString(m map[string]any, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

func getInt(m map[string]any, key string) int {
    switch v := m[key].(type) {
    case int:
        return v
    case int64:
        return int(v)
    case float64:
        return int(v)
    }
    return 0
}
```

### src/service/detector/detector.go

```go
package detector

import (
    "context"

    "quality-bot/src/config"
    "quality-bot/src/model"
    "quality-bot/src/service/metrics"
    "quality-bot/src/util"
)

// Detector is the interface for all debt detectors
type Detector interface {
    // Name returns the detector name
    Name() string

    // IsEnabled returns whether the detector is enabled
    IsEnabled() bool

    // Detect runs the detection and returns found issues
    Detect(ctx context.Context) ([]model.DebtIssue, error)
}

// BaseDetector provides common functionality for detectors
type BaseDetector struct {
    Metrics    *metrics.Provider
    Cfg        *config.Config
    Exclusions *util.ExclusionMatcher
}

// NewBaseDetector creates a new base detector
func NewBaseDetector(metricsProvider *metrics.Provider, cfg *config.Config) BaseDetector {
    return BaseDetector{
        Metrics:    metricsProvider,
        Cfg:        cfg,
        Exclusions: util.NewExclusionMatcher(cfg.Exclusions),
    }
}

// ShouldExclude checks if an entity should be excluded
func (b *BaseDetector) ShouldExclude(filePath, className, funcName string) bool {
    return b.Exclusions.Matches(filePath, className, funcName)
}

// FilterBySeverity filters issues by minimum severity
func (b *BaseDetector) FilterBySeverity(issues []model.DebtIssue) []model.DebtIssue {
    minSev := model.Severity(b.Cfg.Severity.MinSeverity)
    order := []model.Severity{
        model.SeverityLow, model.SeverityMedium,
        model.SeverityHigh, model.SeverityCritical,
    }

    minIdx := 0
    for i, s := range order {
        if s == minSev {
            minIdx = i
            break
        }
    }

    filtered := make([]model.DebtIssue, 0, len(issues))
    for _, issue := range issues {
        for i, s := range order {
            if s == issue.Severity && i >= minIdx {
                filtered = append(filtered, issue)
                break
            }
        }
    }

    return filtered
}
```

### src/service/detector/runner.go

```go
package detector

import (
    "context"
    "fmt"
    "sync"

    "quality-bot/src/config"
    "quality-bot/src/model"
    "quality-bot/src/service/metrics"
)

// Runner manages and runs all detectors.
// It handles detector registration, parallel execution, and result aggregation.
type Runner struct {
    detectors []Detector
    cfg       *config.Config
}

// NewRunner creates a new detector runner with all detectors registered
func NewRunner(metricsProvider *metrics.Provider, cfg *config.Config) *Runner {
    base := NewBaseDetector(metricsProvider, cfg)

    detectors := []Detector{
        NewComplexityDetector(base, cfg.Detectors.Complexity),
        NewSizeAndStructureDetector(base, cfg.Detectors.SizeAndStructure),
        NewCouplingDetector(base, cfg.Detectors.Coupling),
        NewDuplicationDetector(base, cfg.Detectors.Duplication, metricsProvider),
        // DeadCodeDetector - planned for future release
    }

    return &Runner{
        detectors: detectors,
        cfg:       cfg,
    }
}

// RunAll executes all enabled detectors and returns combined issues
func (r *Runner) RunAll(ctx context.Context) ([]model.DebtIssue, error) {
    var (
        allIssues []model.DebtIssue
        mu        sync.Mutex
        wg        sync.WaitGroup
        errChan   = make(chan error, len(r.detectors))
        sem       = make(chan struct{}, r.cfg.Concurrency.MaxParallelDetectors)
    )

    for _, d := range r.detectors {
        if !d.IsEnabled() {
            continue
        }

        wg.Add(1)
        go func(detector Detector) {
            defer wg.Done()

            sem <- struct{}{}        // Acquire semaphore
            defer func() { <-sem }() // Release semaphore

            issues, err := detector.Detect(ctx)
            if err != nil {
                if r.cfg.Detectors.FailFast {
                    errChan <- fmt.Errorf("detector %s: %w", detector.Name(), err)
                }
                return
            }

            mu.Lock()
            allIssues = append(allIssues, issues...)
            mu.Unlock()
        }(d)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    if err, ok := <-errChan; ok {
        return nil, err
    }

    return allIssues, nil
}

// GetDetector returns a detector by name
func (r *Runner) GetDetector(name string) Detector {
    for _, d := range r.detectors {
        if d.Name() == name {
            return d
        }
    }
    return nil
}

// ListDetectors returns names of all registered detectors
func (r *Runner) ListDetectors() []string {
    names := make([]string, len(r.detectors))
    for i, d := range r.detectors {
        names[i] = d.Name()
    }
    return names
}
```

### src/service/detector/complexity.go

```go
package detector

import (
    "context"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// ComplexityDetector detects complexity issues in functions
type ComplexityDetector struct {
    BaseDetector
    cfg config.ComplexityDetectorConfig
}

// NewComplexityDetector creates a new complexity detector
func NewComplexityDetector(base BaseDetector, cfg config.ComplexityDetectorConfig) *ComplexityDetector {
    return &ComplexityDetector{
        BaseDetector: base,
        cfg:          cfg,
    }
}

// Name returns the detector name
func (d *ComplexityDetector) Name() string {
    return "complexity"
}

// IsEnabled returns whether the detector is enabled
func (d *ComplexityDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

// Detect runs complexity detection
func (d *ComplexityDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    functions, err := d.Metrics.GetAllFunctionMetrics(ctx)
    if err != nil {
        return nil, err
    }

    var issues []model.DebtIssue

    for _, fn := range functions {
        if d.ShouldExclude(fn.FilePath, fn.ClassName, fn.Name) {
            continue
        }

        // Check cyclomatic complexity
        if fn.CyclomaticComplexity > d.cfg.CyclomaticModerate {
            issues = append(issues, d.createCCIssue(fn))
        }

        // Check nesting depth
        if fn.MaxNestingDepth > d.cfg.MaxNestingDepth {
            issues = append(issues, d.createNestingIssue(fn))
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *ComplexityDetector) createCCIssue(fn model.FunctionMetrics) model.DebtIssue {
    cc := fn.CyclomaticComplexity

    var severity model.Severity
    switch {
    case cc > d.cfg.CyclomaticCritical:
        severity = model.SeverityCritical
    case cc > d.cfg.CyclomaticHigh:
        severity = model.SeverityHigh
    default:
        severity = model.SeverityMedium
    }

    return model.DebtIssue{
        Category:    model.CategoryComplexity,
        Subcategory: "cyclomatic_complexity",
        Severity:    severity,
        FilePath:    fn.FilePath,
        StartLine:   fn.StartLine,
        EndLine:     fn.EndLine,
        EntityName:  fn.Name,
        EntityType:  "function",
        Description: fmt.Sprintf("High cyclomatic complexity (CC=%d)", cc),
        Metrics: map[string]any{
            "cyclomatic_complexity": cc,
            "conditionals":          fn.ConditionalCount,
            "loops":                 fn.LoopCount,
            "branches":              fn.BranchCount,
        },
        Suggestion: d.ccSuggestion(cc),
    }
}

func (d *ComplexityDetector) createNestingIssue(fn model.FunctionMetrics) model.DebtIssue {
    depth := fn.MaxNestingDepth

    var severity model.Severity
    switch {
    case depth > 6:
        severity = model.SeverityCritical
    case depth > 5:
        severity = model.SeverityHigh
    default:
        severity = model.SeverityMedium
    }

    return model.DebtIssue{
        Category:    model.CategoryComplexity,
        Subcategory: "deep_nesting",
        Severity:    severity,
        FilePath:    fn.FilePath,
        StartLine:   fn.StartLine,
        EndLine:     fn.EndLine,
        EntityName:  fn.Name,
        EntityType:  "function",
        Description: fmt.Sprintf("Deeply nested control flow (depth=%d)", depth),
        Metrics: map[string]any{
            "nesting_depth": depth,
        },
        Suggestion: "Reduce nesting with early returns, guard clauses, or extract methods",
    }
}

func (d *ComplexityDetector) ccSuggestion(cc int) string {
    switch {
    case cc > d.cfg.CyclomaticCritical:
        return "Split into multiple smaller functions; consider strategy or state pattern"
    case cc > d.cfg.CyclomaticHigh:
        return "Extract conditional logic into separate methods"
    default:
        return "Consider simplifying conditionals or extracting helper methods"
    }
}
```

---

## Detector Algorithms

This section describes the high-level algorithms and detection logic for each detector.

### ComplexityDetector

**Purpose**: Identifies functions with high cognitive load that are difficult to understand, test, and maintain.

**Detects**:
- High cyclomatic complexity
- Deeply nested control flow

**Algorithm**:

```
FOR each function in repository:
    1. Skip if function matches exclusion patterns

    2. CYCLOMATIC COMPLEXITY CHECK:
       - Calculate CC = 1 + loops + branches
         (where branches = number of BRANCH relationships from Conditional nodes)
       - Classify severity:
         * CC > critical_threshold  → CRITICAL
         * CC > high_threshold      → HIGH
         * CC > moderate_threshold  → MEDIUM
       - If CC exceeds moderate threshold, create issue

    3. NESTING DEPTH CHECK:
       - Find maximum path length from function to any nested Conditional/Loop
       - If depth > max_nesting_threshold, create issue
       - Severity based on depth (>6 critical, >5 high, else medium)
```

**Cypher Query Pattern**:
```cypher
MATCH (f:Function)-[:CONTAINS*]->(cond:Conditional)
MATCH (f)-[:CONTAINS*]->(loop:Loop)
MATCH (f)-[:CONTAINS*]->(:Conditional)-[br:BRANCH]->()
RETURN f, count(DISTINCT loop) + count(DISTINCT br) + 1 as cyclomatic_complexity
```

---

### SizeAndStructureDetector

**Purpose**: Identifies code entities that have grown too large, violating single responsibility principle.

**Detects**:
- Long methods (too many lines)
- Long parameter lists
- God classes (too many methods/fields)
- Large files (too many lines/functions)

**Algorithm**:

```
1. FUNCTION SIZE ANALYSIS:
   FOR each function:
       - line_count = end_line - start_line
       - IF line_count > max_function_lines:
           Create "long_method" issue
           Severity: line_count > 2x threshold → HIGH, else MEDIUM

       - IF parameter_count > max_parameters:
           Create "long_parameter_list" issue
           Severity: params > 2x threshold → HIGH, else MEDIUM

2. CLASS SIZE ANALYSIS:
   FOR each class:
       - IF method_count > max_class_methods:
           Create "god_class" issue (too many methods)
           Severity based on how much threshold is exceeded

       - IF field_count > max_class_fields:
           Create "god_class" issue (too many fields)

3. FILE SIZE ANALYSIS:
   FOR each file:
       - IF line_count > max_file_lines:
           Create "large_file" issue

       - IF function_count > max_file_functions:
           Create "large_file" issue (too many functions)
```

**Thresholds** (configurable):
| Metric | Default Threshold |
|--------|-------------------|
| Function lines | 50 |
| Parameters | 5 |
| Class methods | 20 |
| Class fields | 15 |
| File lines | 500 |
| File functions | 20 |

---

### CouplingDetector

**Purpose**: Identifies problematic dependencies between code entities that make the codebase rigid and hard to change.

**Detects**:
- Feature Envy (method uses other class's data more than its own)
- Inappropriate Intimacy (two classes too tightly coupled)
- High Dependency Count (class depends on too many others)
- Primitive Obsession (class has too many primitive fields)

**Algorithm**:

```
1. FEATURE ENVY DETECTION:
   FOR each method in a class:
       - Count own_field_uses (fields from same class)
       - Count external_field_uses (fields from other classes)

       - IF external_field_uses > own_field_uses
          AND external_field_uses > feature_envy_threshold:
           Create "feature_envy" issue
           The method "envies" the class whose fields it uses most

   Cypher Pattern:
   MATCH (f:Function)<-[:CONTAINS]-(c:Class)
   MATCH (f)-[:USES]->(own:Field)<-[:CONTAINS]-(c)
   MATCH (f)-[:USES]->(ext:Field)<-[:CONTAINS]-(other:Class)
   WHERE other <> c
   WITH f, c, count(DISTINCT own) as own_uses, count(DISTINCT ext) as ext_uses
   WHERE ext_uses > own_uses

2. INAPPROPRIATE INTIMACY DETECTION:
   FOR each pair of classes (A, B):
       - Count calls from A's methods to B's methods
       - Count calls from B's methods to A's methods
       - Count shared field access

       - IF bidirectional_calls > intimacy_threshold:
           Create "inappropriate_intimacy" issue
           Both classes are too tightly coupled

   Cypher Pattern:
   MATCH (c1:Class)-[:CONTAINS]->(f1:Function)-[:CALLS]->(f2:Function)<-[:CONTAINS]-(c2:Class)
   WHERE c1 <> c2
   WITH c1, c2, count(*) as calls_1_to_2
   MATCH (c2)-[:CONTAINS]->(f3:Function)-[:CALLS]->(f4:Function)<-[:CONTAINS]-(c1)
   WITH c1, c2, calls_1_to_2, count(*) as calls_2_to_1
   WHERE calls_1_to_2 > threshold AND calls_2_to_1 > threshold

3. HIGH COUPLING DETECTION:
   FOR each class:
       - Count distinct classes it depends on (via CALLS relationships)

       - IF dependency_count > max_dependencies:
           Create "high_coupling" issue

   Cypher Pattern:
   MATCH (c:Class)-[:CONTAINS]->(:Function)-[:CALLS]->(:Function)<-[:CONTAINS]-(dep:Class)
   WHERE c <> dep
   WITH c, count(DISTINCT dep) as dependency_count
   WHERE dependency_count > threshold

4. PRIMITIVE OBSESSION DETECTION:
   FOR each class:
       - Count fields with primitive types (string, int, bool, float, etc.)

       - IF primitive_field_count > primitive_threshold:
           Create "primitive_obsession" issue
           Suggests creating value objects or domain types
```

**Severity Assignment**:
- Feature Envy: HIGH if ratio > 3:1, else MEDIUM
- Inappropriate Intimacy: HIGH if bidirectional, MEDIUM if unidirectional
- High Coupling: Based on how much threshold is exceeded
- Primitive Obsession: MEDIUM (design smell, not critical)

---

### DuplicationDetector

**Purpose**: Identifies similar or duplicate code that violates DRY (Don't Repeat Yourself) principle.

**Detects**:
- Semantically similar functions (may not be exact text matches)
- Copy-paste code with minor variations

**Algorithm**:

```
1. CANDIDATE SELECTION:
   - Fetch all functions with line_count >= min_lines
   - Skip trivial functions if configured (getters, setters, constructors)
   - Limit to max_functions_to_check (performance optimization)

2. SIMILARITY SEARCH (using CodeAPI embeddings):
   FOR each function F:
       - Use CodeAPI's searchSimilarCode endpoint
       - Query: Find functions similar to F's code/embedding
       - Filter results: similarity_score >= similarity_threshold

       - FOR each similar function G where G != F:
           - Skip if same file and overlapping line ranges
           - Skip if already reported (F,G) pair
           - Create "code_duplication" issue
             Include both function locations and similarity score

3. RESULT DEDUPLICATION:
   - Track reported pairs to avoid (A,B) and (B,A) duplicates
   - Group related duplicates (if A~B and B~C, may all be related)
```

**CodeAPI Integration**:
```go
// Request to find similar code
req := SimilarCodeRequest{
    RepoName:   repoName,
    SourceCode: function.SourceCode,  // or use embedding ID
    TopK:       10,
    MinScore:   similarityThreshold,
}

resp, _ := codeAPIClient.SearchSimilarCode(ctx, req)

for _, match := range resp.Matches {
    if match.Score >= threshold && match.FunctionID != function.ID {
        // Create duplication issue
    }
}
```

**Thresholds** (configurable):
| Setting | Default | Description |
|---------|---------|-------------|
| similarity_threshold | 0.85 | Minimum cosine similarity to consider duplicate |
| min_lines | 5 | Ignore functions smaller than this |
| max_functions_to_check | 500 | Performance limit |
| skip_trivial | true | Skip getters/setters/constructors |

**Output Example**:
```json
{
  "category": "duplication",
  "subcategory": "similar_code",
  "severity": "medium",
  "file_path": "src/service/user.go",
  "entity_name": "validateUser",
  "description": "Function is 92% similar to validateAccount in src/service/account.go:45",
  "metrics": {
    "similarity_score": 0.92,
    "duplicate_file": "src/service/account.go",
    "duplicate_function": "validateAccount",
    "duplicate_line": 45
  },
  "suggestion": "Extract common validation logic into a shared function"
}
```

---

## Controllers

### src/controller/analysis.go

```go
package controller

import (
    "context"
    "time"

    "quality-bot/src/config"
    "quality-bot/src/model"
    "quality-bot/src/service/codeapi"
    "quality-bot/src/service/detector"
    "quality-bot/src/service/metrics"
)

// AnalysisController orchestrates the debt analysis process
type AnalysisController struct {
    cfg *config.Config
}

// NewAnalysisController creates a new analysis controller
func NewAnalysisController(cfg *config.Config) *AnalysisController {
    return &AnalysisController{cfg: cfg}
}

// AnalyzeRequest represents a request to analyze a repository
type AnalyzeRequest struct {
    RepoName   string
    Detectors  []string // Optional: specific detectors to run (empty = all)
}

// Analyze runs the full analysis pipeline
func (c *AnalysisController) Analyze(ctx context.Context, req AnalyzeRequest) (*model.AnalysisReport, error) {
    // Create CodeAPI client
    codeapiClient := codeapi.NewClient(c.cfg.CodeAPI)

    // Create metrics provider
    metricsProvider := metrics.NewProvider(codeapiClient, req.RepoName, c.cfg.Cache)

    // Create detector runner
    detectorRunner := detector.NewRunner(metricsProvider, c.cfg)

    // Run all detectors
    issues, err := detectorRunner.RunAll(ctx)
    if err != nil {
        return nil, err
    }

    // Apply global filters
    issues = c.applyGlobalFilters(issues)

    // Generate report
    report := &model.AnalysisReport{
        RepoName:    req.RepoName,
        GeneratedAt: time.Now().UTC(),
        Issues:      issues,
        Summary:     c.generateSummary(issues),
    }

    return report, nil
}

func (c *AnalysisController) applyGlobalFilters(issues []model.DebtIssue) []model.DebtIssue {
    maxPerCategory := c.cfg.Output.MaxIssuesPerCategory
    if maxPerCategory <= 0 {
        return issues
    }

    byCategory := make(map[model.Category][]model.DebtIssue)
    for _, issue := range issues {
        if len(byCategory[issue.Category]) < maxPerCategory {
            byCategory[issue.Category] = append(byCategory[issue.Category], issue)
        }
    }

    var filtered []model.DebtIssue
    for _, catIssues := range byCategory {
        filtered = append(filtered, catIssues...)
    }

    return filtered
}

func (c *AnalysisController) generateSummary(issues []model.DebtIssue) model.ReportSummary {
    byCategory := make(map[model.Category]int)
    bySeverity := make(map[model.Severity]int)
    byFile := make(map[string]int)

    for _, issue := range issues {
        byCategory[issue.Category]++
        bySeverity[issue.Severity]++
        byFile[issue.FilePath]++
    }

    // Find hotspots
    type fileCount struct {
        path  string
        count int
    }
    var files []fileCount
    for path, count := range byFile {
        files = append(files, fileCount{path, count})
    }
    // Sort by count descending
    for i := 0; i < len(files); i++ {
        for j := i + 1; j < len(files); j++ {
            if files[j].count > files[i].count {
                files[i], files[j] = files[j], files[i]
            }
        }
    }

    topN := c.cfg.Output.HotspotsTopN
    if topN > len(files) {
        topN = len(files)
    }

    hotspots := make([]model.FileHotspot, topN)
    for i := 0; i < topN; i++ {
        hotspots[i] = model.FileHotspot{
            FilePath:   files[i].path,
            IssueCount: files[i].count,
        }
    }

    return model.ReportSummary{
        TotalIssues:  len(issues),
        ByCategory:   byCategory,
        BySeverity:   bySeverity,
        HotspotFiles: hotspots,
        DebtScore:    c.calculateDebtScore(issues),
    }
}

func (c *AnalysisController) calculateDebtScore(issues []model.DebtIssue) float64 {
    if len(issues) == 0 {
        return 0
    }

    weights := map[model.Severity]int{
        model.SeverityLow:      1,
        model.SeverityMedium:   3,
        model.SeverityHigh:     7,
        model.SeverityCritical: 15,
    }

    var total int
    for _, issue := range issues {
        total += weights[issue.Severity]
    }

    score := float64(total) / 10.0
    if score > 100 {
        score = 100
    }

    return score
}
```

### src/controller/report.go

```go
package controller

import (
    "os"
    "path/filepath"

    "quality-bot/src/config"
    "quality-bot/src/model"
    "quality-bot/src/service/report"
)

// ReportController handles report generation
type ReportController struct {
    cfg *config.Config
}

// NewReportController creates a new report controller
func NewReportController(cfg *config.Config) *ReportController {
    return &ReportController{cfg: cfg}
}

// GenerateReports generates reports in all configured formats
func (c *ReportController) GenerateReports(analysisReport *model.AnalysisReport) error {
    reportGenerator := report.NewGenerator(c.cfg.Output)

    for _, format := range c.cfg.Output.Formats {
        output, err := reportGenerator.Generate(analysisReport, format)
        if err != nil {
            return err
        }

        // Determine output path
        outputPath := c.getOutputPath(analysisReport.RepoName, format)

        // Ensure directory exists
        if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
            return err
        }

        // Write file
        if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
            return err
        }
    }

    return nil
}

func (c *ReportController) getOutputPath(repoName, format string) string {
    ext := format
    if format == "markdown" {
        ext = "md"
    }

    filename := repoName + "-debt-report." + ext
    return filepath.Join(c.cfg.Output.OutputDir, filename)
}
```

---

## Handlers

### src/handler/cli/handler.go

```go
package cli

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "quality-bot/src/config"
)

// Handler handles CLI commands
type Handler struct {
    cfg        *config.Config
    configPath string
    rootCmd    *cobra.Command
}

// New creates a new CLI handler
func New() *Handler {
    h := &Handler{}
    h.setupCommands()
    return h
}

func (h *Handler) setupCommands() {
    h.rootCmd = &cobra.Command{
        Use:   "quality-bot",
        Short: "Technical debt detection agent",
        Long:  "Analyzes codebases to detect technical debt using CodeAPI",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            return h.loadConfig()
        },
    }

    // Global flags
    h.rootCmd.PersistentFlags().StringVarP(&h.configPath, "config", "c", "",
        "Path to configuration file")

    // Add subcommands
    h.rootCmd.AddCommand(h.analyzeCmd())
    h.rootCmd.AddCommand(h.versionCmd())
    h.rootCmd.AddCommand(h.detectorsCmd())
}

func (h *Handler) loadConfig() error {
    loader := config.NewLoader("QUALITY_BOT")
    cfg, err := loader.Load(h.configPath)
    if err != nil {
        return fmt.Errorf("loading configuration: %w", err)
    }
    h.cfg = cfg
    return nil
}

// Execute runs the CLI
func (h *Handler) Execute() error {
    return h.rootCmd.Execute()
}

// Run is the main entry point
func Run() {
    handler := New()
    if err := handler.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### src/handler/cli/analyze.go

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"

    "quality-bot/src/controller"
)

func (h *Handler) analyzeCmd() *cobra.Command {
    var (
        repoName   string
        outputFile string
        format     string
        timeout    time.Duration
    )

    cmd := &cobra.Command{
        Use:   "analyze",
        Short: "Analyze a repository for technical debt",
        Long:  "Runs all enabled detectors against a repository and generates a report",
        RunE: func(cmd *cobra.Command, args []string) error {
            if repoName == "" {
                return fmt.Errorf("--repo is required")
            }

            ctx, cancel := context.WithTimeout(context.Background(), timeout)
            defer cancel()

            // Run analysis
            analysisCtrl := controller.NewAnalysisController(h.cfg)
            report, err := analysisCtrl.Analyze(ctx, controller.AnalyzeRequest{
                RepoName: repoName,
            })
            if err != nil {
                return fmt.Errorf("analysis failed: %w", err)
            }

            // Output results
            if outputFile != "" {
                // Generate report files
                reportCtrl := controller.NewReportController(h.cfg)
                if err := reportCtrl.GenerateReports(report); err != nil {
                    return fmt.Errorf("generating reports: %w", err)
                }
                fmt.Printf("Reports written to %s\n", h.cfg.Output.OutputDir)
            } else {
                // Output to stdout
                output, err := json.MarshalIndent(report, "", "  ")
                if err != nil {
                    return err
                }
                fmt.Println(string(output))
            }

            // Print summary
            fmt.Fprintf(os.Stderr, "\nAnalysis complete:\n")
            fmt.Fprintf(os.Stderr, "  Total issues: %d\n", report.Summary.TotalIssues)
            fmt.Fprintf(os.Stderr, "  Debt score: %.1f/100\n", report.Summary.DebtScore)

            return nil
        },
    }

    cmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
    cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path")
    cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format (json, markdown, sarif)")
    cmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Minute, "Analysis timeout")

    cmd.MarkFlagRequired("repo")

    return cmd
}
```

### src/handler/cli/version.go

```go
package cli

import (
    "fmt"

    "github.com/spf13/cobra"
)

func (h *Handler) versionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print version information",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("quality-bot %s\n", h.cfg.Agent.Version)
        },
    }
}

func (h *Handler) detectorsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "detectors",
        Short: "List available detectors",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Available detectors:")
            fmt.Println("  - complexity     : Cyclomatic complexity and nesting depth")
            fmt.Println("  - size_structure : Long methods, large classes/files, parameter lists")
            fmt.Println("  - coupling       : Feature envy, inappropriate intimacy, dependencies")
            fmt.Println("  - duplication    : Similar code detection")
            fmt.Println("")
            fmt.Println("Planned (future):")
            fmt.Println("  - dead_code      : Unused functions and classes")
        },
    }
}
```

### Future: src/handler/http/handler.go (Placeholder)

```go
package http

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "quality-bot/src/config"
    "quality-bot/src/controller"
)

// Handler handles HTTP requests
type Handler struct {
    cfg          *config.Config
    analysisCtrl *controller.AnalysisController
    reportCtrl   *controller.ReportController
}

// New creates a new HTTP handler
func New(cfg *config.Config) *Handler {
    return &Handler{
        cfg:          cfg,
        analysisCtrl: controller.NewAnalysisController(cfg),
        reportCtrl:   controller.NewReportController(cfg),
    }
}

// SetupRoutes configures HTTP routes
func (h *Handler) SetupRoutes(r *gin.Engine) {
    api := r.Group("/api/v1")
    {
        api.POST("/analyze", h.handleAnalyze)
        api.GET("/health", h.handleHealth)
        api.GET("/detectors", h.handleListDetectors)
    }
}

func (h *Handler) handleAnalyze(c *gin.Context) {
    // Implementation for HTTP endpoint
}

func (h *Handler) handleHealth(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (h *Handler) handleListDetectors(c *gin.Context) {
    // Implementation
}
```

---

## Entry Point

### cmd/quality-bot/main.go

```go
package main

import (
    "quality-bot/src/handler/cli"
)

func main() {
    cli.Run()
}
```

---

## CLI Usage

```bash
# Basic usage
quality-bot analyze --repo my-project

# With configuration file
quality-bot --config config.yaml analyze --repo my-project

# Output to file
quality-bot analyze --repo my-project --output ./reports/

# With timeout
quality-bot analyze --repo my-project --timeout 10m

# List detectors
quality-bot detectors

# Version
quality-bot version
```

### Environment Variable Substitution

Configuration is loaded exclusively from YAML files. Environment variables can be referenced within the YAML using substitution syntax:

```yaml
# In config.yaml:
codeapi:
  url: "${CODEAPI_URL:-http://localhost:8181}"  # Uses CODEAPI_URL or default

output:
  output_dir: "${OUTPUT_DIR:-./reports}"        # Uses OUTPUT_DIR or default

logging:
  level: "${LOG_LEVEL:-info}"                   # Uses LOG_LEVEL or default
```

**Supported syntax:**
- `${VAR_NAME}` - Substitutes the value of VAR_NAME (empty string if not set)
- `${VAR_NAME:-default}` - Substitutes VAR_NAME value, or "default" if not set

**Example usage:**
```bash
# Set environment variables
export CODEAPI_URL=http://codeapi.internal:8181
export LOG_LEVEL=debug

# Run with config that references these variables
quality-bot --config config.yaml analyze --repo my-project
```

---

## Detection Capabilities Matrix

| Debt Type | Metrics Used | Detector | Status |
|-----------|--------------|----------|--------|
| Cyclomatic Complexity | `FunctionMetrics.CyclomaticComplexity` | ComplexityDetector | ✓ |
| Deep Nesting | `FunctionMetrics.MaxNestingDepth` | ComplexityDetector | ✓ |
| Long Methods | `FunctionMetrics.LineCount` | SizeAndStructureDetector | ✓ |
| Long Parameter Lists | `FunctionMetrics.ParameterCount` | SizeAndStructureDetector | ✓ |
| Large Classes | `ClassMetrics.MethodCount`, `FieldCount` | SizeAndStructureDetector | ✓ |
| Large Files | `FileMetrics.LineCount`, `FunctionCount` | SizeAndStructureDetector | ✓ |
| Feature Envy | `FunctionMetrics.OwnFieldUses`, `ExternalFieldUses` | CouplingDetector | ✓ |
| High Coupling | `ClassMetrics.DependencyCount` | CouplingDetector | ✓ |
| Inappropriate Intimacy | `ClassPairMetrics.Calls*` | CouplingDetector | ✓ |
| Primitive Obsession | `ClassMetrics.PrimitiveFieldCount` | CouplingDetector | ✓ |
| Code Duplication | Semantic similarity search | DuplicationDetector | ✓ |
| Dead Code (functions) | `FunctionMetrics.CallerCount` | DeadCodeDetector | Future |
| Dead Code (classes) | `ClassMetrics` (unused) | DeadCodeDetector | Future |

---

## Build & Run

### Makefile

```makefile
.PHONY: build run test clean

BINARY=quality-bot
BUILD_DIR=./bin
SRC_DIR=./cmd/quality-bot

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(SRC_DIR)

run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

lint:
	golangci-lint run ./...

# Development
dev:
	go run $(SRC_DIR) $(ARGS)

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(SRC_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(SRC_DIR)
```

---

## Future Enhancements

### Planned Detectors

1. **Dead Code Detection**: Identify unused functions and classes by analyzing call graphs
   - Track functions with zero callers (excluding entry points)
   - Detect orphaned classes with no instantiation or inheritance
   - Requires careful handling of reflection, callbacks, and framework entry points

### Platform Features

1. **HTTP API**: Add REST endpoints using the existing controller layer
2. **WebSocket Support**: Real-time analysis progress updates
3. **Database Storage**: Store analysis history for trend tracking
4. **LLM Integration**: Use code summaries for nuanced smell detection
5. **IDE Plugins**: VS Code extension for real-time feedback
6. **CI/CD Integration**: GitHub Actions, Jenkins plugins
7. **Custom Rules**: User-defined detection rules via configuration
