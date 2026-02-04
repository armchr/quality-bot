# Quality-Bot: Complete Technical Debt Detection System

A comprehensive Go-based agent for detecting code-level and architectural technical debt using code graph analysis, semantic search, and LLM-powered insights.

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Project Structure](#3-project-structure)
4. [Data Models](#4-data-models)
5. [Configuration](#5-configuration)
6. [Services](#6-services)
7. [Code-Level Detectors](#7-code-level-detectors)
8. [Architectural Debt Detectors](#8-architectural-debt-detectors)
9. [LLM-Enhanced Analysis](#9-llm-enhanced-analysis)
10. [CLI & Handlers](#10-cli--handlers)
11. [Build & Deployment](#11-build--deployment)
12. [Detection Capabilities Summary](#12-detection-capabilities-summary)

---

## 1. Overview

Quality-Bot analyzes codebases to detect various forms of technical debt including complexity issues, code smells, architectural violations, and maintainability problems. It leverages:

- **CodeAPI's Neo4j code graph** for structural analysis
- **Vector embeddings** for similarity-based detection
- **LLM integration** for semantic understanding and intelligent suggestions

### Key Characteristics

| Aspect | Description |
|--------|-------------|
| Language | Go |
| Interface | CLI-first with HTTP endpoint support planned |
| Architecture | Handler → Controller → Service layered |
| Detectors | Configurable with threshold customization |
| Analysis Types | Rule-based, graph-based, embedding-based, LLM-based |

### Detection Categories

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DETECTION CATEGORIES                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  CODE-LEVEL (Traditional)          ARCHITECTURAL                        │
│  ├── Complexity                    ├── Layering Violations              │
│  │   ├── Cyclomatic Complexity     ├── Missing Abstractions             │
│  │   └── Deep Nesting              ├── Package Coupling                 │
│  ├── Size                          ├── Monolithic Code                  │
│  │   ├── Long Methods              └── Inconsistent Patterns            │
│  │   ├── Long Parameter Lists                                           │
│  │   ├── God Classes               LLM-ENHANCED                         │
│  │   └── Large Files               ├── Error Handling Quality           │
│  ├── Coupling                      ├── Naming Quality                   │
│  │   ├── Feature Envy              ├── Business Logic Smells            │
│  │   ├── Inappropriate Intimacy    ├── Security Smells                  │
│  │   ├── High Dependencies         ├── Documentation Alignment          │
│  │   └── Primitive Obsession       ├── API Design Quality               │
│  └── Duplication                   ├── Test Quality                     │
│      └── Similar Code              ├── Concurrency Issues               │
│                                    ├── Resource Management              │
│                                    ├── Configuration Debt               │
│                                    ├── Domain Model Quality             │
│                                    └── Code Readability                 │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Architecture

### 2.1 Layered Architecture

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
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐       │ │
│  │  │ CodeAPI    │  │ Metrics    │  │ Detector   │  │ LLM        │       │ │
│  │  │ Client     │  │ Provider   │  │ Runner     │  │ Analyzer   │       │ │
│  │  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘       │ │
│  │        │               │               │               │               │ │
│  │  ┌─────┴───────────────┴───────────────┴───────────────┴─────┐        │ │
│  │  │                    Report Generator                        │        │ │
│  │  └────────────────────────────────────────────────────────────┘        │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│                           EXTERNAL SERVICES                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   CodeAPI    │  │   LLM API    │  │    Cache     │  │ File System  │    │
│  │   (graphs)   │  │  (Anthropic) │  │   (Redis)    │  │  (reports)   │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘    │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Layer Responsibilities

| Layer | Responsibility | Examples |
|-------|----------------|----------|
| **Handler** | Parse input (CLI args, HTTP requests), validate, call controllers, format output | `CLIHandler`, `HTTPHandler` |
| **Controller** | Business logic, orchestration, coordinate multiple services | `AnalysisController`, `ReportController` |
| **Service** | Single-responsibility operations, external API calls, data access | `CodeAPIClient`, `MetricsProvider`, `DetectorRunner`, `LLMAnalyzer` |

### 2.3 Service Layer Detail

```
┌─────────────────────────────────────────────────────────────────┐
│                    DetectorRunner                                │
│         (Registry + orchestration of detectors)                  │
│                                                                  │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────┐ │
│   │ Complexity  │ │    Size     │ │  Coupling   │ │ Layering │ │
│   │  Detector   │ │  Detector   │ │  Detector   │ │ Detector │ │
│   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └────┬─────┘ │
│          │               │               │              │       │
│   ┌──────┴───────────────┴───────────────┴──────────────┴────┐  │
│   │                   LLM Enhancement Layer                   │  │
│   │  (validates issues, enhances suggestions, adds context)   │  │
│   └──────────────────────────┬────────────────────────────────┘  │
└──────────────────────────────┼───────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    MetricsProvider                               │
│      (High-level metrics abstraction + caching)                  │
│                                                                  │
│   GetAllFunctionMetrics() → []FunctionMetrics                    │
│   GetAllClassMetrics()    → []ClassMetrics                       │
│   GetClassPairMetrics()   → []ClassPairMetrics                   │
│   GetPackageMetrics()     → []PackageMetrics                     │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CodeAPIClient                                 │
│         (Low-level HTTP client for CodeAPI)                      │
│                                                                  │
│   ExecuteCypher(query)    → raw results                          │
│   SearchSimilarCode(...)  → similarity matches                   │
│   GetSnippet(...)         → code snippet                         │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │   CodeAPI    │
                       │  (external)  │
                       └──────────────┘
```

### 2.4 CodeAPI Graph Model

The code graph uses these node labels and relationships:

**Node Labels:**

| Label | Description |
|-------|-------------|
| `FileScope` | Source file |
| `Class` | Class, struct, or interface |
| `Function` | Function or method |
| `Field` | Class field or property |
| `Variable` | Local variable |
| `Conditional` | If statements, switch statements |
| `Loop` | For, while, foreach loops |
| `Block` | Generic code blocks |
| `FunctionCall` | Function/method invocation site |

**Relationships:**

| Relationship | Description |
|--------------|-------------|
| `CONTAINS` | Hierarchical containment |
| `CONTAINS*` | Transitive containment |
| `CALLS` | Function invocation |
| `USES` | Variable/field usage |
| `BRANCH` | Conditional branch |
| `INHERITS_FROM` | Class inheritance |
| `INHERITS_FROM*` | Transitive inheritance |

---

## 3. Project Structure

```
quality-bot/
├── cmd/
│   └── quality-bot/
│       └── main.go                 # Entry point
├── src/
│   ├── config/
│   │   ├── config.go               # Configuration structs
│   │   ├── loader.go               # YAML/env loading
│   │   └── defaults.go             # Default values
│   ├── handler/
│   │   ├── cli/
│   │   │   ├── handler.go          # CLI handler
│   │   │   ├── analyze.go          # analyze command
│   │   │   ├── report.go           # report command
│   │   │   └── version.go          # version command
│   │   └── http/                   # (future)
│   │       ├── handler.go
│   │       ├── routes.go
│   │       └── middleware.go
│   ├── controller/
│   │   ├── analysis.go             # AnalysisController
│   │   ├── report.go               # ReportController
│   │   └── config.go               # ConfigController
│   ├── service/
│   │   ├── codeapi/
│   │   │   ├── client.go           # CodeAPIClient
│   │   │   ├── models.go           # Request/response types
│   │   │   └── queries.go          # Cypher query templates
│   │   ├── metrics/
│   │   │   ├── provider.go         # MetricsProvider
│   │   │   ├── function.go         # FunctionMetrics queries
│   │   │   ├── class.go            # ClassMetrics queries
│   │   │   ├── file.go             # FileMetrics queries
│   │   │   └── package.go          # PackageMetrics queries
│   │   ├── detector/
│   │   │   ├── runner.go           # DetectorRunner
│   │   │   ├── detector.go         # Detector interface
│   │   │   ├── complexity.go       # ComplexityDetector
│   │   │   ├── size.go             # SizeAndStructureDetector
│   │   │   ├── coupling.go         # CouplingDetector
│   │   │   ├── duplication.go      # DuplicationDetector
│   │   │   ├── layering.go         # LayeringDetector
│   │   │   ├── abstraction.go      # AbstractionDetector
│   │   │   ├── package_coupling.go # PackageCouplingDetector
│   │   │   ├── monolith.go         # MonolithDetector
│   │   │   ├── patterns.go         # PatternDetector
│   │   │   ├── llm_enhanced.go     # LLM enhancement wrapper
│   │   │   └── llm_detector.go     # LLM-only detector base
│   │   ├── llm/
│   │   │   ├── client.go           # LLM client interface
│   │   │   ├── anthropic.go        # Anthropic implementation
│   │   │   ├── prompts.go          # Prompt templates
│   │   │   └── cache.go            # Response caching
│   │   └── report/
│   │       ├── generator.go        # ReportGenerator
│   │       ├── json.go             # JSON formatter
│   │       ├── markdown.go         # Markdown formatter
│   │       └── sarif.go            # SARIF formatter
│   ├── model/
│   │   ├── issue.go                # DebtIssue, Severity, Category
│   │   ├── metrics.go              # Metric types
│   │   └── report.go               # AnalysisReport
│   └── util/
│       ├── logger.go               # Logging utilities
│       ├── patterns.go             # Glob/regex matching
│       └── concurrency.go          # Worker pool, semaphore
├── config/
│   └── config.example.yaml         # Example configuration
├── docs/
│   └── quality-bot-complete-design.md
├── prompts/                        # LLM prompt templates
│   ├── validate_issue.tmpl
│   ├── enhance_suggestion.tmpl
│   ├── detect_error_handling.tmpl
│   └── ...
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 4. Data Models

### 4.1 Issue Model

```go
// src/model/issue.go

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
    CategoryArchitecture Category = "architecture"
    CategoryReliability  Category = "reliability"
    CategorySecurity     Category = "security"
    CategoryMaintenance  Category = "maintenance"
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
    EntityType  string            `json:"entity_type"` // function, class, file, package
    Description string            `json:"description"`
    Metrics     map[string]any    `json:"metrics"`
    Suggestion  string            `json:"suggestion"`
    CodeSnippet string            `json:"code_snippet,omitempty"`

    // LLM-enhanced fields
    LLMAnalysis    string `json:"llm_analysis,omitempty"`
    IsFalsePositive bool  `json:"is_false_positive,omitempty"`
    Confidence     float64 `json:"confidence,omitempty"`
}

// AnalysisReport represents the complete analysis output
type AnalysisReport struct {
    RepoName    string        `json:"repo_name"`
    GeneratedAt time.Time     `json:"generated_at"`
    Summary     ReportSummary `json:"summary"`
    Issues      []DebtIssue   `json:"issues"`
}

// ReportSummary contains aggregated statistics
type ReportSummary struct {
    TotalIssues      int              `json:"total_issues"`
    ByCategory       map[Category]int `json:"by_category"`
    BySeverity       map[Severity]int `json:"by_severity"`
    HotspotFiles     []FileHotspot    `json:"hotspot_files"`
    DebtScore        float64          `json:"debt_score"`
    FalsePositives   int              `json:"false_positives_filtered,omitempty"`
    LLMTokensUsed    int              `json:"llm_tokens_used,omitempty"`
}

// FileHotspot represents a file with many issues
type FileHotspot struct {
    FilePath   string `json:"file_path"`
    IssueCount int    `json:"issue_count"`
}
```

### 4.2 Metrics Models

```go
// src/model/metrics.go

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
    CallerCount       int `json:"caller_count"`
    CalleeCount       int `json:"callee_count"`
    ExternalCalls     int `json:"external_calls"`
    OwnFieldUses      int `json:"own_field_uses"`
    ExternalFieldUses int `json:"external_field_uses"`
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

// PackageMetrics contains metrics for a logical package/module
type PackageMetrics struct {
    Name          string  `json:"name"`
    Path          string  `json:"path"`
    FileCount     int     `json:"file_count"`
    ClassCount    int     `json:"class_count"`
    FunctionCount int     `json:"function_count"`
    InternalCalls int     `json:"internal_calls"`
    ExternalCalls int     `json:"external_calls"`
    Afferent      int     `json:"afferent"`     // incoming dependencies (Ca)
    Efferent      int     `json:"efferent"`     // outgoing dependencies (Ce)
    Instability   float64 `json:"instability"`  // Ce / (Ca + Ce)
    Cohesion      float64 `json:"cohesion"`     // internal/total coupling
}

// PackagePairMetrics contains coupling metrics between two packages
type PackagePairMetrics struct {
    Package1    string `json:"package1"`
    Package2    string `json:"package2"`
    Calls1To2   int    `json:"calls_1_to_2"`
    Calls2To1   int    `json:"calls_2_to_1"`
    SharedTypes int    `json:"shared_types"`
}
```

---

## 5. Configuration

### 5.1 Configuration Structure

```go
// src/config/config.go

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
    LLM         LLMConfig         `yaml:"llm"`
}

// DetectorsConfig contains settings for all detectors
type DetectorsConfig struct {
    FailFast         bool                        `yaml:"fail_fast"`
    Complexity       ComplexityDetectorConfig    `yaml:"complexity"`
    SizeAndStructure SizeDetectorConfig          `yaml:"size_and_structure"`
    Coupling         CouplingDetectorConfig      `yaml:"coupling"`
    DeadCode         DeadCodeDetectorConfig      `yaml:"dead_code"`
    Duplication      DuplicationDetectorConfig   `yaml:"duplication"`
    Layering         LayeringDetectorConfig      `yaml:"layering"`
    Abstraction      AbstractionDetectorConfig   `yaml:"abstraction"`
    PackageCoupling  PackageCouplingConfig       `yaml:"package_coupling"`
    Monolith         MonolithDetectorConfig      `yaml:"monolith"`
    Patterns         PatternDetectorConfig       `yaml:"patterns"`
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

// DuplicationDetectorConfig contains duplication detector settings
type DuplicationDetectorConfig struct {
    Enabled             bool    `yaml:"enabled"`
    SimilarityThreshold float64 `yaml:"similarity_threshold"`
    MinLines            int     `yaml:"min_lines"`
    MaxFunctionsToCheck int     `yaml:"max_functions_to_check"`
    SkipTrivial         bool    `yaml:"skip_trivial"`
}

// LayeringDetectorConfig contains layer violation detection settings
type LayeringDetectorConfig struct {
    Enabled bool              `yaml:"enabled"`
    Layers  []LayerDefinition `yaml:"layers"`
    Rules   []LayerRule       `yaml:"rules"`
}

type LayerDefinition struct {
    Name     string   `yaml:"name"`
    Patterns []string `yaml:"patterns"`
}

type LayerRule struct {
    From      string   `yaml:"from"`
    Forbidden []string `yaml:"forbidden"`
}

// AbstractionDetectorConfig contains missing abstraction detection settings
type AbstractionDetectorConfig struct {
    Enabled            bool     `yaml:"enabled"`
    HighFanInThreshold int      `yaml:"high_fan_in_threshold"`
    InterfacePatterns  []string `yaml:"interface_patterns"`
}

// PackageCouplingConfig contains package coupling detection settings
type PackageCouplingConfig struct {
    Enabled                bool `yaml:"enabled"`
    BidirectionalThreshold int  `yaml:"bidirectional_threshold"`
    TotalCallThreshold     int  `yaml:"total_call_threshold"`
    PackageDepth           int  `yaml:"package_depth"`
}

// MonolithDetectorConfig contains monolith detection settings
type MonolithDetectorConfig struct {
    Enabled           bool    `yaml:"enabled"`
    MaxPackageClasses int     `yaml:"max_package_classes"`
    MaxPackageFiles   int     `yaml:"max_package_files"`
    MinCohesionRatio  float64 `yaml:"min_cohesion_ratio"`
    PackageDepth      int     `yaml:"package_depth"`
}

// PatternDetectorConfig contains pattern consistency detection settings
type PatternDetectorConfig struct {
    Enabled     bool                `yaml:"enabled"`
    Conventions []ConventionPattern `yaml:"conventions"`
}

type ConventionPattern struct {
    Name            string   `yaml:"name"`
    ClassPattern    string   `yaml:"class_pattern"`
    ExpectedMethods []string `yaml:"expected_methods"`
    ExpectedPackage string   `yaml:"expected_package"`
}

// LLMConfig contains LLM integration settings
type LLMConfig struct {
    Enabled     bool                  `yaml:"enabled"`
    Provider    string                `yaml:"provider"`
    Anthropic   AnthropicConfig       `yaml:"anthropic"`
    RateLimit   RateLimitConfig       `yaml:"rate_limit"`
    Cache       LLMCacheConfig        `yaml:"cache"`
    Cost        CostConfig            `yaml:"cost"`
    Enhancement EnhancementConfig     `yaml:"enhancement"`
    Detectors   LLMDetectorsConfig    `yaml:"detectors"`
}

type AnthropicConfig struct {
    APIKey    string `yaml:"api_key"`
    Model     string `yaml:"model"`
    MaxTokens int    `yaml:"max_tokens"`
}

type EnhancementConfig struct {
    EnhanceDetectors     []string `yaml:"enhance_detectors"`
    ValidateIssues       bool     `yaml:"validate_issues"`
    EnhanceSuggestions   bool     `yaml:"enhance_suggestions"`
    AssessSeverity       bool     `yaml:"assess_severity"`
    MinSeverityToEnhance string   `yaml:"min_severity_to_enhance"`
    MaxIssuesPerDetector int      `yaml:"max_issues_per_detector"`
}

type LLMDetectorsConfig struct {
    ErrorHandling LLMDetectorConfig `yaml:"error_handling"`
    Naming        LLMDetectorConfig `yaml:"naming"`
    BusinessLogic LLMDetectorConfig `yaml:"business_logic"`
    Security      LLMDetectorConfig `yaml:"security"`
    Documentation LLMDetectorConfig `yaml:"documentation"`
    TestQuality   LLMDetectorConfig `yaml:"test_quality"`
    Concurrency   LLMDetectorConfig `yaml:"concurrency"`
    Resources     LLMDetectorConfig `yaml:"resource_management"`
    DomainModel   LLMDetectorConfig `yaml:"domain_model"`
    Logging       LLMDetectorConfig `yaml:"logging"`
    Readability   LLMDetectorConfig `yaml:"readability"`
}

type LLMDetectorConfig struct {
    Enabled         bool     `yaml:"enabled"`
    BatchSize       int      `yaml:"batch_size"`
    MaxFunctions    int      `yaml:"max_functions"`
    MinFunctionSize int      `yaml:"min_function_size"`
    FocusPatterns   []string `yaml:"focus_patterns"`
}
```

### 5.2 Default Configuration

```go
// src/config/defaults.go

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
            Duplication: DuplicationDetectorConfig{
                Enabled:             true,
                SimilarityThreshold: 0.85,
                MinLines:            5,
                MaxFunctionsToCheck: 500,
                SkipTrivial:         true,
            },
            Layering: LayeringDetectorConfig{
                Enabled: true,
                Layers: []LayerDefinition{
                    {Name: "handler", Patterns: []string{"src/handler/**", "**/handler/**"}},
                    {Name: "controller", Patterns: []string{"src/controller/**"}},
                    {Name: "service", Patterns: []string{"src/service/**"}},
                    {Name: "repository", Patterns: []string{"src/repository/**", "**/db/**"}},
                },
                Rules: []LayerRule{
                    {From: "handler", Forbidden: []string{"repository"}},
                    {From: "repository", Forbidden: []string{"handler", "controller"}},
                },
            },
            Abstraction: AbstractionDetectorConfig{
                Enabled:            true,
                HighFanInThreshold: 5,
                InterfacePatterns:  []string{"*Interface", "*I", "I*"},
            },
            PackageCoupling: PackageCouplingConfig{
                Enabled:                true,
                BidirectionalThreshold: 3,
                TotalCallThreshold:     10,
                PackageDepth:           3,
            },
            Monolith: MonolithDetectorConfig{
                Enabled:           true,
                MaxPackageClasses: 20,
                MaxPackageFiles:   15,
                MinCohesionRatio:  0.3,
                PackageDepth:      3,
            },
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
        LLM: LLMConfig{
            Enabled:  false,
            Provider: "anthropic",
            Anthropic: AnthropicConfig{
                Model:     "claude-sonnet-4-20250514",
                MaxTokens: 4096,
            },
            RateLimit: RateLimitConfig{
                RequestsPerMinute: 60,
                TokensPerMinute:   100000,
            },
            Cache: LLMCacheConfig{
                Enabled: true,
                TTL:     24 * time.Hour,
            },
            Cost: CostConfig{
                BudgetPerRun: 5.00,
                BudgetPerDay: 50.00,
            },
            Enhancement: EnhancementConfig{
                EnhanceDetectors:     []string{"complexity", "coupling"},
                ValidateIssues:       true,
                EnhanceSuggestions:   true,
                MinSeverityToEnhance: "medium",
                MaxIssuesPerDetector: 50,
            },
        },
    }
}
```

### 5.3 Example Configuration File

```yaml
# config/config.yaml

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

  duplication:
    enabled: true
    similarity_threshold: 0.85
    min_lines: 5
    max_functions_to_check: 500

  layering:
    enabled: true
    layers:
      - name: ui
        patterns:
          - "src/ui/**"
          - "src/handler/**"
      - name: service
        patterns:
          - "src/service/**"
          - "src/controller/**"
      - name: data
        patterns:
          - "src/repository/**"
          - "src/db/**"
    rules:
      - from: ui
        forbidden: [data]
      - from: data
        forbidden: [ui]

  abstraction:
    enabled: true
    high_fan_in_threshold: 5
    interface_patterns: ["*Interface", "I*"]

  package_coupling:
    enabled: true
    bidirectional_threshold: 3
    total_call_threshold: 10
    package_depth: 3

  monolith:
    enabled: true
    max_package_classes: 20
    max_package_files: 15
    min_cohesion_ratio: 0.3

  patterns:
    enabled: true
    conventions:
      - name: handlers
        class_pattern: ".*Handler$"
        expected_package: "src/handler/**"
      - name: repositories
        class_pattern: ".*Repository$"
        expected_package: "src/repository/**"

exclusions:
  file_patterns:
    - "**/test/**"
    - "**/tests/**"
    - "**/generated/**"
    - "**/vendor/**"
    - "**/node_modules/**"
  class_patterns:
    - "^Test"
    - "Mock$"
    - "Stub$"
  function_patterns:
    - "^test_"
    - "^Test"

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

# LLM Configuration
llm:
  enabled: true
  provider: "anthropic"

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"
    max_tokens: 4096

  rate_limit:
    requests_per_minute: 60
    tokens_per_minute: 100000
    concurrent_requests: 5

  cache:
    enabled: true
    backend: "redis"
    redis_url: "${REDIS_URL:-redis://localhost:6379}"
    ttl: 24h

  cost:
    budget_per_run: 5.00
    budget_per_day: 50.00
    track_usage: true

  enhancement:
    enhance_detectors:
      - complexity
      - coupling
      - duplication
      - layering
    validate_issues: true
    enhance_suggestions: true
    min_severity_to_enhance: "medium"
    max_issues_per_detector: 50

  detectors:
    error_handling:
      enabled: true
      batch_size: 10
      max_functions: 500
      min_function_size: 5

    naming:
      enabled: true
      focus_patterns:
        - "src/service/**"
        - "src/handler/**"

    business_logic:
      enabled: true
      min_function_size: 10

    security:
      enabled: true

    test_quality:
      enabled: true
      focus_patterns:
        - "**/*_test.go"
        - "**/test_*.py"

    concurrency:
      enabled: true
      focus_patterns:
        - "**/*worker*"
        - "**/*async*"

    resource_management:
      enabled: true

    domain_model:
      enabled: true
      focus_patterns:
        - "src/domain/**"
        - "src/entity/**"

    logging:
      enabled: false

    readability:
      enabled: true
```

---

## 6. Services

### 6.1 CodeAPI Client

```go
// src/service/codeapi/client.go

package codeapi

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
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

// GetSnippet retrieves a code snippet by file and line range
func (c *Client) GetSnippet(ctx context.Context, repoName, filePath string, startLine, endLine int) (*SnippetResponse, error) {
    req := SnippetRequest{
        RepoName:  repoName,
        FilePath:  filePath,
        StartLine: startLine,
        EndLine:   endLine,
    }

    var resp SnippetResponse
    if err := c.post(ctx, "/codeapi/v1/snippet", req, &resp); err != nil {
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
```

### 6.2 Metrics Provider

```go
// src/service/metrics/provider.go

package metrics

import (
    "context"
    "sync"
)

// Provider provides high-level code metrics with caching
type Provider struct {
    client   *codeapi.Client
    repoName string
    cfg      config.CacheConfig

    mu              sync.RWMutex
    functionMetrics []model.FunctionMetrics
    classMetrics    []model.ClassMetrics
    fileMetrics     []model.FileMetrics
    classPairMetrics []model.ClassPairMetrics
    packageMetrics  []model.PackageMetrics
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
    // Similar pattern with class-specific query
    // ...
}

// GetClassPairMetrics retrieves coupling metrics between class pairs
func (p *Provider) GetClassPairMetrics(ctx context.Context) ([]model.ClassPairMetrics, error) {
    // Query for bidirectional class coupling
    // ...
}

// GetPackageMetrics aggregates metrics at package level
func (p *Provider) GetPackageMetrics(ctx context.Context, depth int) ([]model.PackageMetrics, error) {
    // Aggregate class metrics by package path
    // ...
}

// GetPackagePairMetrics aggregates class pair metrics at package level
func (p *Provider) GetPackagePairMetrics(ctx context.Context, depth int) ([]model.PackagePairMetrics, error) {
    // Aggregate class pair metrics by package
    // ...
}

// ClearCache clears all cached metrics
func (p *Provider) ClearCache() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.functionMetrics = nil
    p.classMetrics = nil
    p.fileMetrics = nil
    p.classPairMetrics = nil
    p.packageMetrics = nil
}
```

### 6.3 Detector Runner

```go
// src/service/detector/runner.go

package detector

import (
    "context"
    "fmt"
    "sync"
)

// Runner manages and runs all detectors
type Runner struct {
    detectors []Detector
    cfg       *config.Config
}

// NewRunner creates a new detector runner with all detectors registered
func NewRunner(metricsProvider *metrics.Provider, llmClient llm.Client, cfg *config.Config) *Runner {
    base := NewBaseDetector(metricsProvider, cfg)

    detectors := []Detector{
        // Traditional code-level detectors
        NewComplexityDetector(base, cfg.Detectors.Complexity),
        NewSizeAndStructureDetector(base, cfg.Detectors.SizeAndStructure),
        NewCouplingDetector(base, cfg.Detectors.Coupling),
        NewDuplicationDetector(base, cfg.Detectors.Duplication, metricsProvider),

        // Architectural detectors
        NewLayeringDetector(base, cfg.Detectors.Layering),
        NewAbstractionDetector(base, cfg.Detectors.Abstraction),
        NewPackageCouplingDetector(base, cfg.Detectors.PackageCoupling),
        NewMonolithDetector(base, cfg.Detectors.Monolith),
        NewPatternDetector(base, cfg.Detectors.Patterns),
    }

    // Add LLM-only detectors if enabled
    if cfg.LLM.Enabled && llmClient != nil {
        if cfg.LLM.Detectors.ErrorHandling.Enabled {
            detectors = append(detectors, NewErrorHandlingDetector(base, llmClient, cfg.LLM.Detectors.ErrorHandling))
        }
        if cfg.LLM.Detectors.Naming.Enabled {
            detectors = append(detectors, NewNamingDetector(base, llmClient, cfg.LLM.Detectors.Naming))
        }
        if cfg.LLM.Detectors.BusinessLogic.Enabled {
            detectors = append(detectors, NewBusinessLogicDetector(base, llmClient, cfg.LLM.Detectors.BusinessLogic))
        }
        if cfg.LLM.Detectors.Security.Enabled {
            detectors = append(detectors, NewSecurityDetector(base, llmClient, cfg.LLM.Detectors.Security))
        }
        // ... additional LLM detectors
    }

    // Wrap detectors with LLM enhancement if configured
    if cfg.LLM.Enabled && cfg.LLM.Enhancement.ValidateIssues {
        detectors = wrapWithLLMEnhancement(detectors, llmClient, cfg.LLM.Enhancement)
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

            sem <- struct{}{}
            defer func() { <-sem }()

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

    if err, ok := <-errChan; ok {
        return nil, err
    }

    return allIssues, nil
}
```

### 6.4 Detector Interface

```go
// src/service/detector/detector.go

package detector

import (
    "context"
)

// Detector is the interface for all debt detectors
type Detector interface {
    Name() string
    IsEnabled() bool
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

---

## 7. Code-Level Detectors

### 7.1 Complexity Detector

**Purpose**: Identifies functions with high cognitive load.

**Detects**:
- High cyclomatic complexity
- Deeply nested control flow

```go
// src/service/detector/complexity.go

type ComplexityDetector struct {
    BaseDetector
    cfg config.ComplexityDetectorConfig
}

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

        if fn.CyclomaticComplexity > d.cfg.CyclomaticModerate {
            issues = append(issues, d.createCCIssue(fn))
        }

        if fn.MaxNestingDepth > d.cfg.MaxNestingDepth {
            issues = append(issues, d.createNestingIssue(fn))
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

**Algorithm**:
```
FOR each function:
    CC = 1 + loops + branches
    IF CC > critical_threshold  → CRITICAL
    IF CC > high_threshold      → HIGH
    IF CC > moderate_threshold  → MEDIUM

    IF nesting_depth > threshold → Issue
```

**Thresholds**:
| Metric | Moderate | High | Critical |
|--------|----------|------|----------|
| Cyclomatic Complexity | 10 | 15 | 20 |
| Max Nesting Depth | 4 | 5 | 6 |

---

### 7.2 Size and Structure Detector

**Purpose**: Identifies code entities that violate single responsibility.

**Detects**:
- Long methods
- Long parameter lists
- God classes (too many methods/fields)
- Large files

```go
// src/service/detector/size.go

type SizeAndStructureDetector struct {
    BaseDetector
    cfg config.SizeDetectorConfig
}

func (d *SizeAndStructureDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    var issues []model.DebtIssue

    // Function analysis
    functions, _ := d.Metrics.GetAllFunctionMetrics(ctx)
    for _, fn := range functions {
        if fn.LineCount > d.cfg.MaxFunctionLines {
            issues = append(issues, d.createLongMethodIssue(fn))
        }
        if fn.ParameterCount > d.cfg.MaxParameters {
            issues = append(issues, d.createLongParamListIssue(fn))
        }
    }

    // Class analysis
    classes, _ := d.Metrics.GetAllClassMetrics(ctx)
    for _, cls := range classes {
        if cls.MethodCount > d.cfg.MaxClassMethods {
            issues = append(issues, d.createGodClassIssue(cls, "methods"))
        }
        if cls.FieldCount > d.cfg.MaxClassFields {
            issues = append(issues, d.createGodClassIssue(cls, "fields"))
        }
    }

    // File analysis
    files, _ := d.Metrics.GetAllFileMetrics(ctx)
    for _, f := range files {
        if f.LineCount > d.cfg.MaxFileLines {
            issues = append(issues, d.createLargeFileIssue(f))
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

**Thresholds**:
| Metric | Default |
|--------|---------|
| Function lines | 50 |
| Parameters | 5 |
| Class methods | 20 |
| Class fields | 15 |
| File lines | 500 |
| File functions | 20 |

---

### 7.3 Coupling Detector

**Purpose**: Identifies problematic dependencies between code entities.

**Detects**:
- Feature Envy (method uses other class's data)
- Inappropriate Intimacy (bidirectional tight coupling)
- High Dependency Count
- Primitive Obsession

```go
// src/service/detector/coupling.go

type CouplingDetector struct {
    BaseDetector
    cfg config.CouplingDetectorConfig
}

func (d *CouplingDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    var issues []model.DebtIssue

    // Feature Envy
    functions, _ := d.Metrics.GetAllFunctionMetrics(ctx)
    for _, fn := range functions {
        if fn.ExternalFieldUses > fn.OwnFieldUses &&
           fn.ExternalFieldUses > d.cfg.FeatureEnvyThreshold {
            issues = append(issues, d.createFeatureEnvyIssue(fn))
        }
    }

    // Inappropriate Intimacy
    pairs, _ := d.Metrics.GetClassPairMetrics(ctx)
    for _, pair := range pairs {
        if pair.Calls1To2 > d.cfg.IntimacyCallThreshold &&
           pair.Calls2To1 > d.cfg.IntimacyCallThreshold {
            issues = append(issues, d.createIntimacyIssue(pair))
        }
    }

    // High Coupling & Primitive Obsession
    classes, _ := d.Metrics.GetAllClassMetrics(ctx)
    for _, cls := range classes {
        if cls.DependencyCount > d.cfg.MaxDependencies {
            issues = append(issues, d.createHighCouplingIssue(cls))
        }
        if cls.PrimitiveFieldCount > d.cfg.PrimitiveFieldThreshold {
            issues = append(issues, d.createPrimitiveObsessionIssue(cls))
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

**Cypher Pattern for Feature Envy**:
```cypher
MATCH (f:Function)<-[:CONTAINS]-(c:Class)
MATCH (f)-[:USES]->(own:Field)<-[:CONTAINS]-(c)
MATCH (f)-[:USES]->(ext:Field)<-[:CONTAINS]-(other:Class)
WHERE other <> c
WITH f, c, count(DISTINCT own) as own_uses, count(DISTINCT ext) as ext_uses
WHERE ext_uses > own_uses
```

---

### 7.4 Duplication Detector

**Purpose**: Identifies similar or duplicate code.

**Detects**:
- Semantically similar functions
- Copy-paste code with minor variations

```go
// src/service/detector/duplication.go

type DuplicationDetector struct {
    BaseDetector
    cfg         config.DuplicationDetectorConfig
    codeapiClient *codeapi.Client
}

func (d *DuplicationDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    functions, _ := d.Metrics.GetAllFunctionMetrics(ctx)

    // Filter candidates
    candidates := d.filterCandidates(functions)

    var issues []model.DebtIssue
    seen := make(map[string]bool)

    for _, fn := range candidates {
        // Get code snippet
        snippet, _ := d.codeapiClient.GetSnippet(ctx, d.repoName, fn.FilePath, fn.StartLine, fn.EndLine)

        // Search for similar code
        resp, _ := d.codeapiClient.SearchSimilarCode(ctx, codeapi.SimilarCodeRequest{
            RepoName:    d.repoName,
            CodeSnippet: snippet.Code,
            Limit:       10,
        })

        for _, match := range resp.Matches {
            if match.Score >= d.cfg.SimilarityThreshold {
                key := d.makeKey(fn, match)
                if !seen[key] {
                    seen[key] = true
                    issues = append(issues, d.createDuplicationIssue(fn, match))
                }
            }
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

---

## 8. Architectural Debt Detectors

### 8.1 Layering Violations Detector

**Purpose**: Detects forbidden cross-layer dependencies.

```go
// src/service/detector/layering.go

type LayeringDetector struct {
    BaseDetector
    cfg          config.LayeringDetectorConfig
    forbiddenMap map[string]map[string]bool
}

func (d *LayeringDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    pairs, _ := d.Metrics.GetClassPairMetrics(ctx)

    var issues []model.DebtIssue

    for _, pair := range pairs {
        srcLayer := d.getLayer(pair.Class1File)
        dstLayer := d.getLayer(pair.Class2File)

        if pair.Calls1To2 > 0 && d.isForbidden(srcLayer, dstLayer) {
            issues = append(issues, d.createViolationIssue(pair, srcLayer, dstLayer))
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *LayeringDetector) getLayer(filePath string) string {
    for _, layer := range d.cfg.Layers {
        for _, pattern := range layer.Patterns {
            if matchGlob(pattern, filePath) {
                return layer.Name
            }
        }
    }
    return ""
}
```

**Example Configuration**:
```yaml
layering:
  layers:
    - name: ui
      patterns: ["src/handler/**", "src/ui/**"]
    - name: service
      patterns: ["src/service/**"]
    - name: data
      patterns: ["src/repository/**"]
  rules:
    - from: ui
      forbidden: [data]
```

---

### 8.2 Missing Abstractions Detector

**Purpose**: Detects concrete classes with high fan-in that should be interfaces.

```go
// src/service/detector/abstraction.go

type AbstractionDetector struct {
    BaseDetector
    cfg config.AbstractionDetectorConfig
}

func (d *AbstractionDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    classes, _ := d.Metrics.GetAllClassMetrics(ctx)

    var issues []model.DebtIssue

    for _, cls := range classes {
        if d.isInterface(cls.Name) {
            continue
        }

        if cls.DependentCount >= d.cfg.HighFanInThreshold {
            issues = append(issues, model.DebtIssue{
                Category:    model.CategoryArchitecture,
                Subcategory: "missing_abstraction",
                Severity:    d.calcSeverity(cls.DependentCount),
                EntityName:  cls.Name,
                Description: fmt.Sprintf(
                    "Concrete class %s has %d dependents - consider extracting an interface",
                    cls.Name, cls.DependentCount,
                ),
                Suggestion: "Extract interface for public methods",
            })
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

---

### 8.3 Package Coupling Detector

**Purpose**: Detects packages with excessive bidirectional dependencies.

```go
// src/service/detector/package_coupling.go

type PackageCouplingDetector struct {
    BaseDetector
    cfg config.PackageCouplingConfig
}

func (d *PackageCouplingDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    pairs, _ := d.Metrics.GetPackagePairMetrics(ctx, d.cfg.PackageDepth)

    var issues []model.DebtIssue

    for _, pair := range pairs {
        // Bidirectional tight coupling
        if pair.Calls1To2 >= d.cfg.BidirectionalThreshold &&
           pair.Calls2To1 >= d.cfg.BidirectionalThreshold {
            issues = append(issues, model.DebtIssue{
                Category:    model.CategoryArchitecture,
                Subcategory: "tight_coupling",
                Severity:    model.SeverityHigh,
                EntityName:  fmt.Sprintf("%s ↔ %s", pair.Package1, pair.Package2),
                EntityType:  "package_pair",
                Description: fmt.Sprintf(
                    "Bidirectional coupling: %d calls each direction",
                    min(pair.Calls1To2, pair.Calls2To1),
                ),
                Suggestion: "Introduce shared interface or extract common functionality",
            })
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

---

### 8.4 Monolith Detector

**Purpose**: Detects oversized packages with low cohesion.

```go
// src/service/detector/monolith.go

type MonolithDetector struct {
    BaseDetector
    cfg config.MonolithDetectorConfig
}

func (d *MonolithDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    packages, _ := d.Metrics.GetPackageMetrics(ctx, d.cfg.PackageDepth)

    var issues []model.DebtIssue

    for _, pkg := range packages {
        // Oversized package
        if pkg.ClassCount > d.cfg.MaxPackageClasses {
            issues = append(issues, d.createOversizedIssue(pkg))
        }

        // Low cohesion
        if pkg.ClassCount >= 3 && pkg.Cohesion < d.cfg.MinCohesionRatio {
            issues = append(issues, d.createLowCohesionIssue(pkg))
        }
    }

    return d.FilterBySeverity(issues), nil
}
```

---

### 8.5 Pattern Consistency Detector

**Purpose**: Detects inconsistent conventions across similar code.

```go
// src/service/detector/patterns.go

type PatternDetector struct {
    BaseDetector
    cfg              config.PatternDetectorConfig
    compiledPatterns map[string]*regexp.Regexp
}

func (d *PatternDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    classes, _ := d.Metrics.GetAllClassMetrics(ctx)

    var issues []model.DebtIssue

    // Check convention compliance
    for _, conv := range d.cfg.Conventions {
        re := d.compiledPatterns[conv.Name]
        for _, cls := range classes {
            if re.MatchString(cls.Name) {
                if conv.ExpectedPackage != "" && !matchGlob(conv.ExpectedPackage, cls.FilePath) {
                    issues = append(issues, d.createLocationIssue(cls, conv))
                }
            }
        }
    }

    // Detect naming inconsistencies
    issues = append(issues, d.detectNamingInconsistencies(classes)...)

    return d.FilterBySeverity(issues), nil
}
```

---

## 9. LLM-Enhanced Analysis

### 9.1 Overview

LLMs enhance analysis by providing:
- **Semantic understanding** of code intent
- **False positive filtering** through context reasoning
- **Context-specific suggestions** instead of generic templates
- **Detection of issues** invisible to structural analysis

### 9.2 LLM Client Interface

```go
// src/service/llm/client.go

package llm

// Client defines the interface for LLM interactions
type Client interface {
    Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error)
    BatchAnalyze(ctx context.Context, reqs []AnalysisRequest) ([]AnalysisResponse, error)
}

// AnalysisTask identifies the type of analysis
type AnalysisTask string

const (
    // Enhancement tasks
    TaskValidateIssue       AnalysisTask = "validate_issue"
    TaskEnhanceSuggestion   AnalysisTask = "enhance_suggestion"
    TaskAssessComplexity    AnalysisTask = "assess_complexity"

    // Detection tasks
    TaskDetectErrorHandling AnalysisTask = "detect_error_handling"
    TaskDetectNaming        AnalysisTask = "detect_naming"
    TaskDetectBusinessLogic AnalysisTask = "detect_business_logic"
    TaskDetectSecurity      AnalysisTask = "detect_security"
    TaskDetectTestQuality   AnalysisTask = "detect_test_quality"
    TaskDetectConcurrency   AnalysisTask = "detect_concurrency"
    TaskDetectResources     AnalysisTask = "detect_resources"
    TaskDetectDomainModel   AnalysisTask = "detect_domain_model"
    TaskDetectLogging       AnalysisTask = "detect_logging"
    TaskDetectReadability   AnalysisTask = "detect_readability"
)

type AnalysisRequest struct {
    Task    AnalysisTask     `json:"task"`
    Code    string           `json:"code"`
    Context *CodeContext     `json:"context,omitempty"`
    Issue   *model.DebtIssue `json:"issue,omitempty"`
}

type CodeContext struct {
    FilePath     string            `json:"file_path"`
    Language     string            `json:"language"`
    ClassName    string            `json:"class_name,omitempty"`
    FunctionName string            `json:"function_name,omitempty"`
    Callers      []string          `json:"callers,omitempty"`
    Callees      []string          `json:"callees,omitempty"`
    RelatedCode  map[string]string `json:"related_code,omitempty"`
}

type AnalysisResponse struct {
    Task           AnalysisTask    `json:"task"`
    Success        bool            `json:"success"`
    Result         json.RawMessage `json:"result"`
    TokensUsed     int             `json:"tokens_used"`
    ProcessingTime time.Duration   `json:"processing_time"`
}
```

### 9.3 LLM-Enhanced Detector Wrapper

```go
// src/service/detector/llm_enhanced.go

type LLMEnhancedDetector struct {
    BaseDetector
    wrapped   Detector
    llmClient llm.Client
    cfg       config.EnhancementConfig
}

func (d *LLMEnhancedDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    // Run traditional detection
    issues, err := d.wrapped.Detect(ctx)
    if err != nil {
        return nil, err
    }

    // Filter to issues worth enhancing
    toEnhance := d.filterForEnhancement(issues)

    // Validate issues (filter false positives)
    if d.cfg.ValidateIssues {
        toEnhance = d.validateIssues(ctx, toEnhance)
    }

    // Enhance suggestions
    if d.cfg.EnhanceSuggestions {
        toEnhance = d.enhanceSuggestions(ctx, toEnhance)
    }

    return d.mergeResults(issues, toEnhance), nil
}

func (d *LLMEnhancedDetector) validateIssues(ctx context.Context, issues []model.DebtIssue) []model.DebtIssue {
    var validated []model.DebtIssue

    for _, issue := range issues {
        code, _ := d.getCodeSnippet(ctx, issue)

        resp, _ := d.llmClient.Analyze(ctx, llm.AnalysisRequest{
            Task:  llm.TaskValidateIssue,
            Code:  code,
            Issue: &issue,
        })

        var result ValidationResult
        json.Unmarshal(resp.Result, &result)

        if result.IsValid {
            issue.Confidence = result.Confidence
            validated = append(validated, issue)
        }
    }

    return validated
}
```

### 9.4 LLM-Only Detectors

These detectors rely entirely on LLM semantic analysis:

#### Error Handling Detector

```go
// src/service/detector/llm_error_handling.go

type ErrorHandlingDetector struct {
    BaseDetector
    llmClient llm.Client
    cfg       config.LLMDetectorConfig
}

func (d *ErrorHandlingDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    functions, _ := d.Metrics.GetAllFunctionMetrics(ctx)
    functions = d.filterCandidates(functions)

    var issues []model.DebtIssue

    for _, fn := range functions {
        code, _ := d.getCode(ctx, fn)

        resp, _ := d.llmClient.Analyze(ctx, llm.AnalysisRequest{
            Task: llm.TaskDetectErrorHandling,
            Code: code,
            Context: &llm.CodeContext{
                FilePath:     fn.FilePath,
                FunctionName: fn.Name,
            },
        })

        detected := d.parseResults(resp.Result, fn)
        issues = append(issues, detected...)
    }

    return d.FilterBySeverity(issues), nil
}
```

**Detectable Issues**:
| Issue | Description |
|-------|-------------|
| Swallowed errors | Errors caught but not handled |
| Generic catch | Catching too broad exception types |
| Missing context | Errors without debugging info |
| Error masking | Wrapping errors and losing info |
| Missing propagation | Not returning errors |

#### Naming Quality Detector

**Detectable Issues**:
| Issue | Example |
|-------|---------|
| Misleading | `isValid()` returns error message |
| Too generic | `processData()`, `handleStuff()` |
| Abbreviations | `calcAvgMthlyRev()` |
| Wrong abstraction | `sendHTTPPostRequest()` |
| Negated booleans | `isNotEmpty` |

#### Business Logic Smell Detector

**Detectable Issues**:
| Smell | Description |
|-------|-------------|
| Magic Numbers | Hardcoded values with business meaning |
| Implicit State Machine | State transitions in if/else chains |
| Temporal Coupling | Methods must be called in order |
| Feature Flag Debt | Old feature flags still present |
| Mixed Abstraction | Business rules with infrastructure code |

#### Security Smell Detector

**Detectable Issues**:
| Category | Examples |
|----------|----------|
| Injection | SQL concatenation, command injection |
| Authentication | Hardcoded credentials, weak validation |
| Authorization | Missing access checks |
| Cryptography | Weak algorithms, hardcoded keys |
| Data Exposure | Logging sensitive data |

#### Test Quality Detector

**Detectable Issues**:
| Issue | Impact |
|-------|--------|
| Weak assertions | Don't verify behavior |
| Missing edge cases | Bugs in boundaries |
| Test implementation | Brittle tests |
| Missing negative tests | Unhandled errors |

### 9.5 Cost Optimization

```go
// Strategies for managing LLM costs

// 1. Tiered Analysis
type TieredAnalysis struct {
    CriticalTier TierConfig `yaml:"critical"` // Always analyze
    HighTier     TierConfig `yaml:"high"`     // If budget allows
    MediumTier   TierConfig `yaml:"medium"`   // Sample analysis
}

// 2. Caching
type LLMCache struct {
    codeCache    *lru.Cache // hash(code) -> result
    patternCache *lru.Cache // hash(pattern) -> template
    ttl          time.Duration
}

// 3. Batch Processing
func batchAnalyze(functions []FunctionInfo) {
    byFile := groupByFile(functions)
    for file, funcs := range byFile {
        // Single prompt for all functions in file
        prompt := buildBatchPrompt(file, funcs)
        results := llmClient.Analyze(ctx, prompt)
    }
}

// 4. Cost Tracking
type CostTracker struct {
    budget       float64
    spent        float64
    costPerToken float64
}

func (ct *CostTracker) CanAfford(tokens int) bool {
    return ct.spent + float64(tokens)*ct.costPerToken <= ct.budget
}
```

---

## 10. CLI & Handlers

### 10.1 CLI Handler

```go
// src/handler/cli/handler.go

package cli

import (
    "github.com/spf13/cobra"
)

type Handler struct {
    cfg        *config.Config
    configPath string
    rootCmd    *cobra.Command
}

func New() *Handler {
    h := &Handler{}
    h.setupCommands()
    return h
}

func (h *Handler) setupCommands() {
    h.rootCmd = &cobra.Command{
        Use:   "quality-bot",
        Short: "Technical debt detection agent",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            return h.loadConfig()
        },
    }

    h.rootCmd.PersistentFlags().StringVarP(&h.configPath, "config", "c", "", "Config file")

    h.rootCmd.AddCommand(h.analyzeCmd())
    h.rootCmd.AddCommand(h.versionCmd())
    h.rootCmd.AddCommand(h.detectorsCmd())
}

func Run() {
    handler := New()
    if err := handler.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 10.2 Analyze Command

```go
// src/handler/cli/analyze.go

func (h *Handler) analyzeCmd() *cobra.Command {
    var (
        repoName string
        output   string
        format   string
        timeout  time.Duration
    )

    cmd := &cobra.Command{
        Use:   "analyze",
        Short: "Analyze repository for technical debt",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := context.WithTimeout(context.Background(), timeout)
            defer cancel()

            analysisCtrl := controller.NewAnalysisController(h.cfg)
            report, err := analysisCtrl.Analyze(ctx, controller.AnalyzeRequest{
                RepoName: repoName,
            })
            if err != nil {
                return err
            }

            if output != "" {
                reportCtrl := controller.NewReportController(h.cfg)
                return reportCtrl.GenerateReports(report)
            }

            // Output to stdout
            json.NewEncoder(os.Stdout).Encode(report)
            return nil
        },
    }

    cmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name")
    cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory")
    cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format")
    cmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Minute, "Timeout")

    cmd.MarkFlagRequired("repo")
    return cmd
}
```

### 10.3 CLI Usage

```bash
# Basic analysis
quality-bot analyze --repo my-project

# With configuration file
quality-bot --config config.yaml analyze --repo my-project

# Output to directory
quality-bot analyze --repo my-project --output ./reports/

# With timeout
quality-bot analyze --repo my-project --timeout 10m

# List available detectors
quality-bot detectors

# Version
quality-bot version
```

---

## 11. Build & Deployment

### 11.1 Makefile

```makefile
.PHONY: build run test clean lint

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

dev:
	go run $(SRC_DIR) $(ARGS)

deps:
	go mod tidy && go mod download

build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(SRC_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(SRC_DIR)
```

### 11.2 Environment Variables

```bash
# Required for CodeAPI
export CODEAPI_URL=http://localhost:8181

# Required for LLM features
export ANTHROPIC_API_KEY=sk-ant-...

# Optional
export LLM_ENABLED=true
export LLM_BUDGET_PER_RUN=5.00
export REDIS_URL=redis://localhost:6379
export LOG_LEVEL=info
export OUTPUT_DIR=./reports
```

---

## 12. Detection Capabilities Summary

### 12.1 Complete Detection Matrix

| Category | Detector | Subcategory | Type | Status |
|----------|----------|-------------|------|--------|
| **Complexity** | ComplexityDetector | cyclomatic_complexity | Traditional | ✓ |
| | | deep_nesting | Traditional | ✓ |
| **Size** | SizeAndStructureDetector | long_method | Traditional | ✓ |
| | | long_parameter_list | Traditional | ✓ |
| | | god_class | Traditional | ✓ |
| | | large_file | Traditional | ✓ |
| **Coupling** | CouplingDetector | feature_envy | Traditional | ✓ |
| | | inappropriate_intimacy | Traditional | ✓ |
| | | high_coupling | Traditional | ✓ |
| | | primitive_obsession | Traditional | ✓ |
| **Duplication** | DuplicationDetector | similar_code | Embedding | ✓ |
| **Architecture** | LayeringDetector | layering_violation | Traditional | ✓ |
| | AbstractionDetector | missing_abstraction | Traditional | ✓ |
| | PackageCouplingDetector | tight_coupling | Traditional | ✓ |
| | | high_package_coupling | Traditional | ✓ |
| | MonolithDetector | monolithic_package | Traditional | ✓ |
| | | low_cohesion | Traditional | ✓ |
| | PatternDetector | inconsistent_location | Traditional | ✓ |
| | | inconsistent_naming | Traditional | ✓ |
| **Reliability** | ErrorHandlingDetector | swallowed_error | LLM | ✓ |
| | | generic_catch | LLM | ✓ |
| | | missing_context | LLM | ✓ |
| | ConcurrencyDetector | race_condition | LLM | ✓ |
| | | deadlock_risk | LLM | ✓ |
| | ResourceDetector | resource_leak | LLM | ✓ |
| **Security** | SecurityDetector | injection_risk | LLM | ✓ |
| | | auth_issue | LLM | ✓ |
| | | data_exposure | LLM | ✓ |
| **Maintenance** | NamingDetector | misleading_name | LLM | ✓ |
| | | generic_name | LLM | ✓ |
| | BusinessLogicDetector | magic_number | LLM | ✓ |
| | | implicit_state_machine | LLM | ✓ |
| | TestQualityDetector | weak_assertions | LLM | ✓ |
| | | missing_edge_cases | LLM | ✓ |
| | DomainModelDetector | anemic_model | LLM | ✓ |
| | ReadabilityDetector | low_readability | LLM | ✓ |
| **Dead Code** | DeadCodeDetector | unreachable_code | Future | Planned |

### 12.2 Severity Guidelines

| Severity | Criteria | Action |
|----------|----------|--------|
| **Critical** | Security vulnerabilities, data loss risk, production blockers | Immediate fix required |
| **High** | Significant maintainability impact, architectural violations | Fix in current sprint |
| **Medium** | Code smells, minor violations | Fix when touching code |
| **Low** | Style issues, minor improvements | Optional cleanup |

### 12.3 Value vs Cost Matrix (LLM Features)

| Enhancement | Value | Cost | Priority |
|-------------|-------|------|----------|
| False positive filtering | High | Low | 1 |
| Context-aware suggestions | High | Medium | 2 |
| Error handling detection | High | Medium | 3 |
| Security smell detection | High | Medium | 4 |
| Naming quality | Medium | Low | 5 |
| Business logic smells | High | High | 6 |
| Test quality analysis | Medium | Medium | 7 |
| Readability assessment | Medium | Medium | 8 |

---

## Appendix A: Cypher Query Reference

### Function Metrics Query
```cypher
MATCH (f:Function)
WHERE f.repo = $repo_name
OPTIONAL MATCH (c:Class)-[:CONTAINS]->(f)
OPTIONAL MATCH (f)-[:CONTAINS*]->(cond:Conditional)
OPTIONAL MATCH (f)-[:CONTAINS*]->(loop:Loop)
OPTIONAL MATCH (f)-[:CONTAINS*]->(:Conditional)-[br:BRANCH]->()
...
RETURN f.*, count metrics
```

### Class Pair Metrics Query
```cypher
MATCH (c1:Class)-[:CONTAINS]->(f1:Function)-[:CALLS]->(f2:Function)<-[:CONTAINS]-(c2:Class)
WHERE c1 <> c2 AND c1.repo = $repo_name
WITH c1, c2, count(*) as calls_1_to_2
MATCH (c2)-[:CONTAINS]->(f3:Function)-[:CALLS]->(f4:Function)<-[:CONTAINS]-(c1)
RETURN c1, c2, calls_1_to_2, count(*) as calls_2_to_1
```

### Feature Envy Query
```cypher
MATCH (f:Function)<-[:CONTAINS]-(c:Class)
MATCH (f)-[:USES]->(own:Field)<-[:CONTAINS]-(c)
MATCH (f)-[:USES]->(ext:Field)<-[:CONTAINS]-(other:Class)
WHERE other <> c
WITH f, c, count(DISTINCT own) as own_uses, count(DISTINCT ext) as ext_uses
WHERE ext_uses > own_uses AND ext_uses > $threshold
RETURN f, c, own_uses, ext_uses
```

---

## Appendix B: LLM Prompt Templates

### Issue Validation Prompt
```
Analyze this detected code issue:

Issue Type: {{.Issue.Subcategory}}
Severity: {{.Issue.Severity}}
Description: {{.Issue.Description}}

Code:
```{{.Context.Language}}
{{.Code}}
```

Determine if this is a true positive or false positive.
Consider:
1. Is this an intentional design pattern?
2. Is the complexity necessary for the problem?
3. Are there framework/library constraints?

Output JSON:
{
  "is_valid": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "..."
}
```

### Error Handling Detection Prompt
```
Analyze error handling in this code:

```{{.Context.Language}}
{{.Code}}
```

Identify issues:
1. Swallowed/ignored errors
2. Generic exception catching
3. Missing error context
4. Error masking
5. Missing propagation

Output JSON array of issues found.
```

---

*This document consolidates the complete design of quality-bot including traditional detectors, architectural analysis, and LLM-enhanced capabilities.*
