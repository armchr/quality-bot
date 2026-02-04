# LLM-Enhanced Code Quality Analysis

This document describes how Large Language Models (LLMs) can enhance technical debt detection beyond traditional rule-based and metric-based approaches. It covers improvements to existing detectors and entirely new analysis capabilities that LLMs make possible.

## Table of Contents

1. [Overview](#overview)
2. [Current Limitations](#current-limitations)
3. [LLM Integration Architecture](#llm-integration-architecture)
4. [Enhanced Existing Detectors](#enhanced-existing-detectors)
5. [New LLM-Enabled Detectors](#new-llm-enabled-detectors)
6. [Implementation Details](#implementation-details)
7. [Cost Optimization Strategies](#cost-optimization-strategies)
8. [Configuration](#configuration)

---

## Overview

Traditional static analysis tools detect code issues through:
- **Metric thresholds**: Cyclomatic complexity > 15, line count > 50
- **Pattern matching**: Regex for naming conventions, glob for file paths
- **Graph analysis**: Call graphs, dependency graphs via Cypher queries
- **Similarity search**: Vector embeddings for code duplication

These approaches have fundamental limitations — they analyze code **structure** without understanding code **meaning**. LLMs bridge this gap by providing semantic understanding, enabling:

| Capability | Traditional | With LLM |
|------------|-------------|----------|
| Complexity assessment | Count branches | Understand if complexity is necessary |
| Suggestions | Generic templates | Context-specific refactoring advice |
| False positive handling | Manual exclusion lists | Automated reasoning about exceptions |
| Pattern detection | Predefined rules | Recognize emergent anti-patterns |
| Documentation | Not analyzed | Verify code-documentation alignment |
| Business logic | Structure only | Understand intent and logic flaws |

---

## Current Limitations

### Metric-Based Detection

```
Current: if fn.CyclomaticComplexity > 15 → Issue

Problems:
- Parser with CC=25 is necessary complexity
- State machine with CC=20 is appropriate
- Simple function with CC=12 might be poorly written
- No distinction between "complex" and "complicated"
```

### Pattern-Based Detection

```
Current: if path matches "src/handler/**" → handler layer

Problems:
- Misplaced files not detected
- Mixed responsibilities in correct location not detected
- Can't understand actual class purpose
- Domain-specific conventions not recognized
```

### Suggestion Generation

```
Current: "Consider splitting this function into smaller parts"

Problems:
- No specific guidance on HOW to split
- Doesn't consider the specific code context
- Same suggestion for all high-complexity issues
- No risk assessment for the refactoring
```

---

## LLM Integration Architecture

### Service Layer Addition

```
┌─────────────────────────────────────────────────────────────────────┐
│                         quality-bot                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                      SERVICES                                    │ │
│  │                                                                  │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │ │
│  │  │ CodeAPI      │  │ Metrics      │  │ LLM Analyzer         │  │ │
│  │  │ Client       │  │ Provider     │  │ (NEW)                │  │ │
│  │  └──────────────┘  └──────────────┘  └──────────────────────┘  │ │
│  │                                                                  │ │
│  │  ┌──────────────────────────────────────────────────────────┐   │ │
│  │  │                   Detector Runner                         │   │ │
│  │  │                                                           │   │ │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐            │   │ │
│  │  │  │ Traditional│ │ Traditional│ │ LLM-Only   │            │   │ │
│  │  │  │ Detectors  │ │ + LLM      │ │ Detectors  │            │   │ │
│  │  │  │            │ │ Enhanced   │ │ (NEW)      │            │   │ │
│  │  │  └────────────┘ └────────────┘ └────────────┘            │   │ │
│  │  └──────────────────────────────────────────────────────────┘   │ │
│  │                                                                  │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│  External Services:                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │   CodeAPI    │  │  LLM API     │  │   Cache      │               │
│  │   (graphs)   │  │  (Anthropic) │  │   (Redis)    │               │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. Traditional Detection
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │   Metrics   │ ──▶ │  Detector   │ ──▶ │   Issues    │
   │  Provider   │     │  (rules)    │     │   (raw)     │
   └─────────────┘     └─────────────┘     └─────────────┘
                                                  │
                                                  ▼
2. LLM Enrichment
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │   Issues    │ ──▶ │    LLM      │ ──▶ │  Enhanced   │
   │   + Code    │     │  Analyzer   │     │   Issues    │
   └─────────────┘     └─────────────┘     └─────────────┘
                                                  │
                                                  ▼
3. LLM-Only Detection
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │    Code     │ ──▶ │    LLM      │ ──▶ │    New      │
   │  Snippets   │     │  Detector   │     │   Issues    │
   └─────────────┘     └─────────────┘     └─────────────┘
                                                  │
                                                  ▼
4. Final Report
   ┌─────────────────────────────────────────────────────┐
   │  - Validated issues (false positives removed)       │
   │  - Context-specific suggestions                     │
   │  - New semantic issues                              │
   │  - Prioritized remediation plan                     │
   └─────────────────────────────────────────────────────┘
```

---

## Enhanced Existing Detectors

### 1. Complexity Detector Enhancement

**Current Behavior**: Flags functions with cyclomatic complexity > threshold

**LLM Enhancement**: Classify complexity as necessary vs accidental

```go
type ComplexityAssessment struct {
    Issue              model.DebtIssue `json:"issue"`
    ComplexityType     string          `json:"complexity_type"`     // "necessary", "accidental", "domain"
    Confidence         float64         `json:"confidence"`
    Reasoning          string          `json:"reasoning"`
    IsReducible        bool            `json:"is_reducible"`
    ReductionStrategy  string          `json:"reduction_strategy,omitempty"`
    EstimatedNewCC     int             `json:"estimated_new_cc,omitempty"`
}
```

**Prompt Template**:
```
Analyze this function's complexity (cyclomatic complexity = {{.CC}}):

```{{.Language}}
{{.Code}}
```

Classify the complexity:
1. NECESSARY: Inherent to the problem (parsers, state machines, protocol handlers)
2. ACCIDENTAL: Can be reduced through refactoring
3. DOMAIN: Business rules that are inherently complex

If ACCIDENTAL, describe specific refactoring steps to reduce complexity.

Output JSON:
{
  "complexity_type": "necessary|accidental|domain",
  "reasoning": "...",
  "is_reducible": true/false,
  "reduction_strategy": "..." // if reducible
}
```

**Example Output**:
```json
{
  "complexity_type": "accidental",
  "reasoning": "This function handles three separate concerns: input validation (lines 5-20), business logic (lines 22-45), and response formatting (lines 47-60). The nested conditionals in the validation section can be flattened using early returns.",
  "is_reducible": true,
  "reduction_strategy": "1. Extract validateInput() for lines 5-20\n2. Extract formatResponse() for lines 47-60\n3. Use early returns instead of nested if-else\n4. Consider using a validation library for common checks",
  "estimated_new_cc": 8
}
```

---

### 2. Coupling Detector Enhancement

**Current Behavior**: Detects feature envy via field usage ratios

**LLM Enhancement**: Understand if coupling is appropriate for the design pattern

```go
type CouplingAssessment struct {
    Issue           model.DebtIssue `json:"issue"`
    DesignPattern   string          `json:"design_pattern,omitempty"` // "visitor", "strategy", "facade", etc.
    IsIntentional   bool            `json:"is_intentional"`
    Reasoning       string          `json:"reasoning"`
    Alternative     string          `json:"alternative,omitempty"`
}
```

**Patterns LLM Can Recognize**:

| Detected Issue | LLM Recognition | Verdict |
|----------------|-----------------|---------|
| Feature envy in `JsonSerializer.serialize(User)` | Visitor pattern implementation | Intentional |
| High coupling in `OrderFacade` | Facade pattern - meant to aggregate | Intentional |
| Inappropriate intimacy A↔B | Both classes should be merged | Actual debt |
| Method uses 5 external classes | Orchestration method (valid) | Intentional |

---

### 3. Duplication Detector Enhancement

**Current Behavior**: Finds semantically similar code via embeddings

**LLM Enhancement**: Determine if duplication is problematic and how to consolidate

```go
type DuplicationAssessment struct {
    Issue              model.DebtIssue   `json:"issue"`
    DuplicationType    string            `json:"duplication_type"` // "exact", "structural", "behavioral"
    IsProblematic      bool              `json:"is_problematic"`
    Reasoning          string            `json:"reasoning"`
    ConsolidationPlan  *ConsolidationPlan `json:"consolidation_plan,omitempty"`
}

type ConsolidationPlan struct {
    Strategy          string   `json:"strategy"` // "extract_function", "template_method", "strategy_pattern"
    SharedAbstraction string   `json:"shared_abstraction"`
    AffectedFiles     []string `json:"affected_files"`
    CodeExample       string   `json:"code_example"`
}
```

**Non-Problematic Duplication Examples**:
- Test fixtures with similar structure (intentional)
- Generated code from different sources
- Protocol buffer / API client implementations
- Boilerplate required by frameworks

---

### 4. Size Detector Enhancement

**Current Behavior**: Flags large classes/functions based on line count

**LLM Enhancement**: Identify actual responsibility violations vs legitimate size

```go
type SizeAssessment struct {
    Issue              model.DebtIssue    `json:"issue"`
    ResponsibilityCount int               `json:"responsibility_count"`
    Responsibilities   []Responsibility   `json:"responsibilities"`
    SizeJustified      bool               `json:"size_justified"`
    SplitRecommendation *SplitPlan        `json:"split_recommendation,omitempty"`
}

type Responsibility struct {
    Name        string `json:"name"`
    LineRange   string `json:"line_range"`
    Description string `json:"description"`
}

type SplitPlan struct {
    NewClasses []ProposedClass `json:"new_classes"`
    Rationale  string          `json:"rationale"`
}

type ProposedClass struct {
    Name            string   `json:"name"`
    Responsibilities []string `json:"responsibilities"`
    Methods         []string `json:"methods"`
}
```

---

### 5. Layering Detector Enhancement

**Current Behavior**: Path-based layer detection

**LLM Enhancement**: Analyze actual code responsibilities to classify layers

```go
type LayerAssessment struct {
    FilePath           string   `json:"file_path"`
    ClassName          string   `json:"class_name"`
    ConfiguredLayer    string   `json:"configured_layer"`    // from path patterns
    ActualLayer        string   `json:"actual_layer"`        // LLM-determined
    Confidence         float64  `json:"confidence"`
    Responsibilities   []string `json:"responsibilities"`
    LayerViolations    []string `json:"layer_violations"`
    Recommendation     string   `json:"recommendation"`
}
```

**Prompt Template**:
```
Analyze this class and determine its architectural layer:

```{{.Language}}
{{.Code}}
```

Layers:
- PRESENTATION: HTTP handlers, CLI commands, UI components
- APPLICATION: Use cases, orchestration, DTOs
- DOMAIN: Business entities, value objects, domain services
- INFRASTRUCTURE: Database, external APIs, file system

Identify:
1. Primary layer based on responsibilities
2. Any layer violations (e.g., domain class making HTTP calls)
3. Mixed responsibilities that should be separated
```

---

## New LLM-Enabled Detectors

These detectors are only possible with LLM semantic understanding.

### 6. Error Handling Quality Detector

**Purpose**: Identify poor error handling patterns that metrics can't detect

```go
type ErrorHandlingIssue struct {
    model.DebtIssue
    ErrorPattern     string `json:"error_pattern"`
    ProblematicCode  string `json:"problematic_code"`
    Impact           string `json:"impact"`
    CorrectApproach  string `json:"correct_approach"`
}
```

**Detectable Issues**:

| Pattern | Example | Problem |
|---------|---------|---------|
| Swallowed errors | `catch (e) {}` | Silent failures, hard to debug |
| Generic catch | `catch (Exception e)` | Catches unintended exceptions |
| Error message without context | `return errors.New("failed")` | No debugging info |
| Inconsistent error types | Mix of errors and panics | Unpredictable behavior |
| Missing error propagation | Not returning error from called function | Lost error information |
| Log and throw | Log error then rethrow same error | Duplicate logging |
| Null/nil returns | Return nil instead of error | Nil pointer risks |

**Prompt Template**:
```
Analyze error handling in this code:

```{{.Language}}
{{.Code}}
```

Identify error handling issues:
1. Swallowed/ignored errors
2. Generic exception catching
3. Errors without sufficient context
4. Inconsistent error handling patterns
5. Missing error propagation
6. Potential nil/null pointer risks from error returns

For each issue, explain the impact and suggest the correct approach.
```

---

### 7. Naming Quality Detector

**Purpose**: Evaluate if names accurately describe behavior

```go
type NamingIssue struct {
    model.DebtIssue
    CurrentName     string   `json:"current_name"`
    ActualBehavior  string   `json:"actual_behavior"`
    NamingProblems  []string `json:"naming_problems"`
    SuggestedNames  []string `json:"suggested_names"`
    Confidence      float64  `json:"confidence"`
}
```

**Detectable Issues**:

| Issue Type | Example | Problem |
|------------|---------|---------|
| Misleading | `isValid()` returns error message | Name implies boolean |
| Too generic | `processData()`, `handleStuff()` | Doesn't describe what it does |
| Abbreviations | `calcAvgMthlyRev()` | Hard to read |
| Wrong abstraction level | `sendHTTPPostRequest()` | Implementation detail in name |
| Negated booleans | `isNotEmpty`, `disableValidation` | Cognitive overhead |
| Type in name | `userList`, `stringName` | Redundant with type system |
| Inconsistent | `getUser()` vs `fetchAccount()` | Same pattern, different verbs |

---

### 8. Business Logic Smell Detector

**Purpose**: Detect logic-level issues invisible to structural analysis

```go
type BusinessLogicSmell struct {
    model.DebtIssue
    SmellType       string `json:"smell_type"`
    Evidence        string `json:"evidence"`
    BusinessImpact  string `json:"business_impact"`
    Refactoring     string `json:"refactoring"`
}
```

**Detectable Smells**:

| Smell | Description | Example |
|-------|-------------|---------|
| **Magic Numbers** | Hardcoded values with business meaning | `if amount > 10000` |
| **Implicit State Machine** | State transitions hidden in if/else chains | Order status handling |
| **Primitive Obsession (semantic)** | Using strings for domain concepts | `status = "pending"` |
| **Temporal Coupling** | Methods must be called in specific order | `init()` before `process()` |
| **Feature Flags Debt** | Old feature flags still in code | Dead conditional branches |
| **Mixed Abstraction** | High-level and low-level code mixed | Business rules with SQL |
| **Shotgun Surgery Risk** | Change requires modifying many places | Spread business rules |
| **Data Clumps (semantic)** | Related data always passed together | `(userId, userName, userEmail)` |

**Prompt Template**:
```
Analyze this code for business logic issues:

```{{.Language}}
{{.Code}}
```

Look for:
1. Magic numbers/strings that represent business rules
2. Implicit state machines that should be explicit
3. Business logic mixed with infrastructure code
4. Temporal coupling (required call order)
5. Scattered business rules that should be centralized
6. Domain concepts represented as primitives

For each issue, explain the business impact and suggest improvements.
```

---

### 9. Security Smell Detector

**Purpose**: Identify potential security issues through semantic analysis

```go
type SecuritySmell struct {
    model.DebtIssue
    VulnerabilityType string   `json:"vulnerability_type"`
    CWE               string   `json:"cwe,omitempty"`
    Severity          string   `json:"severity"` // "critical", "high", "medium", "low"
    AttackVector      string   `json:"attack_vector"`
    Remediation       string   `json:"remediation"`
    References        []string `json:"references,omitempty"`
}
```

**Detectable Issues**:

| Category | Examples |
|----------|----------|
| **Injection** | SQL concatenation, command injection, LDAP injection |
| **Authentication** | Hardcoded credentials, weak password validation |
| **Authorization** | Missing access checks, IDOR vulnerabilities |
| **Cryptography** | Weak algorithms, hardcoded keys, improper IV usage |
| **Data Exposure** | Logging sensitive data, verbose error messages |
| **Input Validation** | Missing sanitization, improper type coercion |
| **Session Management** | Predictable tokens, missing expiration |
| **Configuration** | Debug mode in production, exposed endpoints |

**Note**: LLM analysis complements but doesn't replace dedicated security scanners (SAST/DAST).

---

### 10. Documentation-Code Alignment Detector

**Purpose**: Verify documentation matches actual behavior

```go
type DocumentationIssue struct {
    model.DebtIssue
    IssueType          string `json:"issue_type"`
    DocumentedBehavior string `json:"documented_behavior"`
    ActualBehavior     string `json:"actual_behavior"`
    Location           string `json:"location"` // doc comment, README, etc.
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Stale Comments** | Code changed but comment wasn't updated |
| **Missing Documentation** | Public API without documentation |
| **Parameter Mismatch** | Documented params don't match signature |
| **Return Value Mismatch** | Documented return doesn't match actual |
| **Side Effect Not Documented** | Function modifies state without mention |
| **Exception Not Documented** | Throws exceptions not in docs |
| **Outdated Examples** | Code examples that no longer work |
| **TODO/FIXME Debt** | Long-standing TODO comments |

---

### 11. API Design Quality Detector

**Purpose**: Evaluate API design consistency and best practices

```go
type APIDesignIssue struct {
    model.DebtIssue
    Endpoint        string `json:"endpoint"`
    IssueType       string `json:"issue_type"`
    CurrentDesign   string `json:"current_design"`
    BestPractice    string `json:"best_practice"`
    SuggestedChange string `json:"suggested_change"`
}
```

**Detectable Issues**:

| Category | Issues |
|----------|--------|
| **REST Violations** | Verbs in URLs, wrong HTTP methods, inconsistent pluralization |
| **Versioning** | Missing version, inconsistent versioning strategy |
| **Naming** | Inconsistent naming conventions across endpoints |
| **Response Format** | Inconsistent response structures, missing pagination |
| **Error Responses** | Inconsistent error formats, missing error codes |
| **Authentication** | Inconsistent auth requirements across endpoints |
| **Idempotency** | Non-idempotent operations without safeguards |

---

### 12. Test Quality Detector

**Purpose**: Evaluate test effectiveness beyond coverage metrics

```go
type TestQualityIssue struct {
    model.DebtIssue
    TestFile        string   `json:"test_file"`
    IssueType       string   `json:"issue_type"`
    AffectedTests   []string `json:"affected_tests"`
    Impact          string   `json:"impact"`
    Improvement     string   `json:"improvement"`
}
```

**Detectable Issues**:

| Issue | Description | Impact |
|-------|-------------|--------|
| **Weak Assertions** | Tests that only check "no error" | Don't verify behavior |
| **Missing Edge Cases** | No tests for boundary conditions | Bugs in edge cases |
| **Test Implementation** | Tests tied to implementation details | Brittle tests |
| **Shared Mutable State** | Tests depend on order | Flaky tests |
| **Missing Negative Tests** | No tests for error conditions | Unhandled errors |
| **Overlapping Tests** | Multiple tests check same thing | Maintenance burden |
| **Test Data Issues** | Hardcoded dates, random data | Future failures |
| **Missing Integration Tests** | Only unit tests, no integration | Integration bugs |

**Prompt Template**:
```
Analyze these tests for the function `{{.FunctionName}}`:

Function under test:
```{{.Language}}
{{.FunctionCode}}
```

Tests:
```{{.Language}}
{{.TestCode}}
```

Evaluate:
1. Are assertions verifying actual behavior or just "no error"?
2. Are edge cases covered (empty input, boundaries, nulls)?
3. Are error conditions tested?
4. Are tests testing behavior or implementation?
5. Is there unnecessary overlap between tests?
6. What important scenarios are missing?
```

---

### 13. Concurrency Issue Detector

**Purpose**: Identify potential race conditions and synchronization issues

```go
type ConcurrencyIssue struct {
    model.DebtIssue
    IssueType       string   `json:"issue_type"`
    SharedResources []string `json:"shared_resources"`
    RaceCondition   string   `json:"race_condition,omitempty"`
    Remediation     string   `json:"remediation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Unprotected Shared State** | Multiple goroutines access shared data without sync |
| **Lock Ordering** | Potential deadlock from inconsistent lock order |
| **Missing Synchronization** | Data race between read and write |
| **Premature Optimization** | Lock-free code that's incorrect |
| **Channel Misuse** | Unbuffered channels causing deadlock |
| **Context Ignorance** | Not respecting context cancellation |
| **Goroutine Leaks** | Goroutines that never terminate |

---

### 14. Resource Management Detector

**Purpose**: Identify resource leaks and improper cleanup

```go
type ResourceIssue struct {
    model.DebtIssue
    ResourceType    string `json:"resource_type"` // "file", "connection", "transaction", etc.
    IssueType       string `json:"issue_type"`    // "leak", "improper_cleanup", "missing_timeout"
    CodePath        string `json:"code_path"`
    Remediation     string `json:"remediation"`
}
```

**Detectable Issues**:

| Resource | Issues |
|----------|--------|
| **Files** | Not closed, closed in wrong place, missing error handling on close |
| **Connections** | Leaks, no pool limits, missing timeouts |
| **Transactions** | Not committed/rolled back, nested transaction issues |
| **Locks** | Not released on error paths, held too long |
| **Memory** | Unbounded caches, large allocations in loops |
| **Goroutines** | Never terminate, accumulate over time |

---

### 15. Configuration Debt Detector

**Purpose**: Identify configuration-related technical debt

```go
type ConfigurationIssue struct {
    model.DebtIssue
    IssueType       string `json:"issue_type"`
    ConfigItem      string `json:"config_item"`
    CurrentState    string `json:"current_state"`
    Recommendation  string `json:"recommendation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Hardcoded Values** | Configuration that should be externalized |
| **Environment Coupling** | Code with environment-specific logic |
| **Dead Feature Flags** | Feature flags for fully rolled out features |
| **Missing Defaults** | Required config without sensible defaults |
| **Inconsistent Config** | Same setting configured differently |
| **Secret Exposure** | Credentials in code or config files |
| **Config Duplication** | Same values repeated across files |

---

### 16. Code Evolution Debt Detector

**Purpose**: Identify debt accumulated through code evolution

```go
type EvolutionDebt struct {
    model.DebtIssue
    DebtType        string   `json:"debt_type"`
    History         string   `json:"history"` // How the debt accumulated
    RelatedCommits  []string `json:"related_commits,omitempty"`
    Recommendation  string   `json:"recommendation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Deprecated Usage** | Using deprecated APIs internally |
| **Migration Incomplete** | Partial migration to new patterns |
| **Compatibility Layers** | Temporary compatibility code still present |
| **Dead Code** | Code unreachable but not removed |
| **Orphaned Tests** | Tests for removed functionality |
| **Version Debt** | Outdated dependencies with known issues |
| **Abandoned Refactoring** | Partially completed refactoring |

---

### 17. Domain Model Quality Detector

**Purpose**: Evaluate domain model design quality

```go
type DomainModelIssue struct {
    model.DebtIssue
    IssueType       string   `json:"issue_type"`
    AffectedClasses []string `json:"affected_classes"`
    DomainConcept   string   `json:"domain_concept"`
    Recommendation  string   `json:"recommendation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Anemic Domain Model** | Entities with only getters/setters, logic elsewhere |
| **Missing Value Objects** | Primitives used for domain concepts |
| **Broken Aggregates** | Aggregate boundaries not enforced |
| **Missing Domain Events** | Important state changes not captured |
| **Leaky Abstraction** | Domain exposes persistence details |
| **Missing Invariants** | Business rules not enforced in model |
| **God Entity** | Entity with too many responsibilities |

---

### 18. Logging Quality Detector

**Purpose**: Evaluate logging practices

```go
type LoggingIssue struct {
    model.DebtIssue
    IssueType       string `json:"issue_type"`
    LogStatement    string `json:"log_statement"`
    Context         string `json:"context"`
    Recommendation  string `json:"recommendation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Missing Context** | Log without request ID, user ID, etc. |
| **Sensitive Data** | Logging passwords, tokens, PII |
| **Inconsistent Levels** | Errors logged as info, info as debug |
| **Missing Error Logs** | Error paths without logging |
| **Excessive Logging** | Logging in tight loops |
| **String Concatenation** | Performance issues in disabled log levels |
| **Missing Structured Data** | Text logs instead of structured |

---

### 19. Dependency Health Detector

**Purpose**: Analyze dependency usage quality

```go
type DependencyIssue struct {
    model.DebtIssue
    Dependency      string `json:"dependency"`
    IssueType       string `json:"issue_type"`
    CurrentVersion  string `json:"current_version,omitempty"`
    Recommendation  string `json:"recommendation"`
}
```

**Detectable Issues**:

| Issue | Description |
|-------|-------------|
| **Over-dependence** | Heavy library for simple use case |
| **Underutilization** | Library included but barely used |
| **Conflicting Libraries** | Multiple libraries for same purpose |
| **Wrapper Absence** | Direct use of library throughout codebase |
| **Version Spread** | Different versions of same library |
| **Transitive Risk** | Risky transitive dependencies |

---

### 20. Code Readability Detector

**Purpose**: Holistic readability assessment beyond metrics

```go
type ReadabilityIssue struct {
    model.DebtIssue
    IssueType       string  `json:"issue_type"`
    ReadabilityScore float64 `json:"readability_score"` // 0-1
    Factors         []ReadabilityFactor `json:"factors"`
    Improvements    []string `json:"improvements"`
}

type ReadabilityFactor struct {
    Factor    string  `json:"factor"`
    Score     float64 `json:"score"`
    Details   string  `json:"details"`
}
```

**Evaluated Factors**:

| Factor | What's Evaluated |
|--------|------------------|
| **Naming Clarity** | Are names self-documenting? |
| **Code Organization** | Logical grouping and ordering |
| **Abstraction Level** | Consistent abstraction throughout |
| **Comment Quality** | Helpful comments, not obvious ones |
| **Control Flow** | Clear, linear control flow |
| **Side Effects** | Obvious side effects |
| **Cognitive Load** | How much to hold in memory |

---

## Implementation Details

### LLM Service Interface

```go
// src/service/llm/client.go
package llm

import "context"

// Client defines the interface for LLM interactions
type Client interface {
    // Analyze sends code for analysis with a specific task
    Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error)

    // BatchAnalyze processes multiple requests efficiently
    BatchAnalyze(ctx context.Context, reqs []AnalysisRequest) ([]AnalysisResponse, error)
}

// AnalysisRequest represents a single analysis request
type AnalysisRequest struct {
    Task        AnalysisTask   `json:"task"`
    Code        string         `json:"code"`
    Context     *CodeContext   `json:"context,omitempty"`
    Issue       *model.DebtIssue `json:"issue,omitempty"` // For enrichment tasks
    Options     *TaskOptions   `json:"options,omitempty"`
}

// AnalysisTask identifies the type of analysis
type AnalysisTask string

const (
    // Enhancement tasks (enrich existing issues)
    TaskValidateIssue        AnalysisTask = "validate_issue"
    TaskEnhanceSuggestion    AnalysisTask = "enhance_suggestion"
    TaskAssessComplexity     AnalysisTask = "assess_complexity"
    TaskAssessCoupling       AnalysisTask = "assess_coupling"

    // Detection tasks (find new issues)
    TaskDetectErrorHandling  AnalysisTask = "detect_error_handling"
    TaskDetectNaming         AnalysisTask = "detect_naming"
    TaskDetectBusinessLogic  AnalysisTask = "detect_business_logic"
    TaskDetectSecurity       AnalysisTask = "detect_security"
    TaskDetectDocAlignment   AnalysisTask = "detect_doc_alignment"
    TaskDetectAPIDesign      AnalysisTask = "detect_api_design"
    TaskDetectTestQuality    AnalysisTask = "detect_test_quality"
    TaskDetectConcurrency    AnalysisTask = "detect_concurrency"
    TaskDetectResources      AnalysisTask = "detect_resources"
    TaskDetectConfig         AnalysisTask = "detect_config"
    TaskDetectDomainModel    AnalysisTask = "detect_domain_model"
    TaskDetectLogging        AnalysisTask = "detect_logging"
    TaskDetectReadability    AnalysisTask = "detect_readability"

    // Classification tasks
    TaskClassifyLayer        AnalysisTask = "classify_layer"
)

// CodeContext provides additional context for analysis
type CodeContext struct {
    FilePath      string            `json:"file_path"`
    Language      string            `json:"language"`
    ClassName     string            `json:"class_name,omitempty"`
    FunctionName  string            `json:"function_name,omitempty"`
    Callers       []string          `json:"callers,omitempty"`
    Callees       []string          `json:"callees,omitempty"`
    RelatedCode   map[string]string `json:"related_code,omitempty"` // name -> code
    ProjectInfo   *ProjectInfo      `json:"project_info,omitempty"`
}

// ProjectInfo provides project-level context
type ProjectInfo struct {
    Name           string   `json:"name"`
    Language       string   `json:"primary_language"`
    Framework      string   `json:"framework,omitempty"`
    Conventions    []string `json:"conventions,omitempty"`
}

// TaskOptions customizes analysis behavior
type TaskOptions struct {
    MaxTokens       int     `json:"max_tokens,omitempty"`
    Temperature     float64 `json:"temperature,omitempty"`
    FocusAreas      []string `json:"focus_areas,omitempty"`
    IgnorePatterns  []string `json:"ignore_patterns,omitempty"`
}

// AnalysisResponse contains the LLM analysis result
type AnalysisResponse struct {
    Task            AnalysisTask    `json:"task"`
    Success         bool            `json:"success"`
    Result          json.RawMessage `json:"result"` // Task-specific result
    TokensUsed      int             `json:"tokens_used"`
    ProcessingTime  time.Duration   `json:"processing_time"`
    Error           string          `json:"error,omitempty"`
}
```

### Anthropic Client Implementation

```go
// src/service/llm/anthropic.go
package llm

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/anthropics/anthropic-sdk-go"
)

type AnthropicClient struct {
    client      *anthropic.Client
    model       string
    maxTokens   int
    prompts     *PromptTemplates
    cache       Cache
    rateLimiter *RateLimiter
}

func NewAnthropicClient(cfg AnthropicConfig) (*AnthropicClient, error) {
    client := anthropic.NewClient(cfg.APIKey)

    prompts, err := LoadPromptTemplates(cfg.PromptsPath)
    if err != nil {
        return nil, fmt.Errorf("loading prompts: %w", err)
    }

    return &AnthropicClient{
        client:      client,
        model:       cfg.Model,
        maxTokens:   cfg.MaxTokens,
        prompts:     prompts,
        cache:       NewCache(cfg.Cache),
        rateLimiter: NewRateLimiter(cfg.RateLimit),
    }, nil
}

func (c *AnthropicClient) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error) {
    // Check cache first
    cacheKey := c.buildCacheKey(req)
    if cached, ok := c.cache.Get(cacheKey); ok {
        return cached, nil
    }

    // Rate limiting
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }

    // Build prompt
    prompt, err := c.prompts.Build(req.Task, req)
    if err != nil {
        return nil, fmt.Errorf("building prompt: %w", err)
    }

    // Call API
    start := time.Now()
    resp, err := c.client.Messages.Create(ctx, anthropic.MessageCreateParams{
        Model:     c.model,
        MaxTokens: c.getMaxTokens(req),
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
        },
        System: anthropic.NewTextBlock(c.prompts.SystemPrompt(req.Task)),
    })
    if err != nil {
        return nil, fmt.Errorf("API call: %w", err)
    }

    // Parse response
    result := &AnalysisResponse{
        Task:           req.Task,
        Success:        true,
        TokensUsed:     resp.Usage.InputTokens + resp.Usage.OutputTokens,
        ProcessingTime: time.Since(start),
    }

    // Extract JSON from response
    content := resp.Content[0].Text
    result.Result, err = extractJSON(content)
    if err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("parsing response: %v", err)
    }

    // Cache successful results
    if result.Success {
        c.cache.Set(cacheKey, result)
    }

    return result, nil
}
```

### Prompt Templates

```go
// src/service/llm/prompts.go
package llm

import (
    "embed"
    "text/template"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

type PromptTemplates struct {
    templates map[AnalysisTask]*template.Template
    systems   map[AnalysisTask]string
}

func LoadPromptTemplates(customPath string) (*PromptTemplates, error) {
    pt := &PromptTemplates{
        templates: make(map[AnalysisTask]*template.Template),
        systems:   make(map[AnalysisTask]string),
    }

    // Load embedded default templates
    // Load custom templates from customPath if provided
    // Parse and cache templates

    return pt, nil
}

func (pt *PromptTemplates) Build(task AnalysisTask, req AnalysisRequest) (string, error) {
    tmpl, ok := pt.templates[task]
    if !ok {
        return "", fmt.Errorf("unknown task: %s", task)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, req); err != nil {
        return "", err
    }

    return buf.String(), nil
}

func (pt *PromptTemplates) SystemPrompt(task AnalysisTask) string {
    if system, ok := pt.systems[task]; ok {
        return system
    }
    return defaultSystemPrompt
}

const defaultSystemPrompt = `You are a senior software engineer performing code quality analysis.
Provide specific, actionable feedback based on the code provided.
Output your analysis as valid JSON matching the requested schema.
Be direct and technical. Focus on issues that have real impact on maintainability, reliability, or security.`
```

### Example Prompt Template

```
{{/* prompts/detect_error_handling.tmpl */}}

Analyze error handling in this {{.Context.Language}} code:

```{{.Context.Language}}
{{.Code}}
```

{{if .Context.FunctionName}}
Function: {{.Context.FunctionName}}
{{end}}

{{if .Context.ClassName}}
Class: {{.Context.ClassName}}
{{end}}

Identify error handling issues from this list:
1. Swallowed/ignored errors - errors caught but not handled
2. Generic exception catching - catching too broad exception types
3. Missing error context - errors without useful debugging information
4. Inconsistent patterns - different error handling styles mixed together
5. Error masking - wrapping errors and losing original information
6. Missing propagation - not returning errors from called functions
7. Panic/exception misuse - using panic/throw for normal control flow

For each issue found, provide:
- The specific code location (line numbers if possible)
- The problematic pattern
- The potential impact
- A suggested fix with example code

Output JSON array:
[
  {
    "issue_type": "swallowed_error|generic_catch|missing_context|inconsistent|masking|missing_propagation|panic_misuse",
    "location": "line X" or "lines X-Y",
    "problematic_code": "the specific code with the issue",
    "impact": "what can go wrong",
    "suggestion": "how to fix it",
    "example_fix": "corrected code example"
  }
]

If no issues found, return empty array: []
```

### LLM-Enhanced Detector Base

```go
// src/service/detector/llm_enhanced.go
package detector

import (
    "context"

    "quality-bot/src/model"
    "quality-bot/src/service/llm"
)

// LLMEnhancedDetector wraps a traditional detector with LLM enhancement
type LLMEnhancedDetector struct {
    BaseDetector
    wrapped     Detector
    llmClient   llm.Client
    cfg         LLMEnhancementConfig
}

type LLMEnhancementConfig struct {
    Enabled           bool    `yaml:"enabled"`
    ValidateIssues    bool    `yaml:"validate_issues"`    // Filter false positives
    EnhanceSuggestions bool   `yaml:"enhance_suggestions"` // Add specific advice
    MinSeverity       string  `yaml:"min_severity"`        // Only enhance high+ severity
    MaxIssuesPerRun   int     `yaml:"max_issues_per_run"`  // Cost control
}

func NewLLMEnhancedDetector(
    base BaseDetector,
    wrapped Detector,
    llmClient llm.Client,
    cfg LLMEnhancementConfig,
) *LLMEnhancedDetector {
    return &LLMEnhancedDetector{
        BaseDetector: base,
        wrapped:      wrapped,
        llmClient:    llmClient,
        cfg:          cfg,
    }
}

func (d *LLMEnhancedDetector) Name() string {
    return d.wrapped.Name() + "_llm_enhanced"
}

func (d *LLMEnhancedDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    // First, run traditional detection
    issues, err := d.wrapped.Detect(ctx)
    if err != nil {
        return nil, err
    }

    if !d.cfg.Enabled {
        return issues, nil
    }

    // Filter to issues worth enhancing
    toEnhance := d.filterForEnhancement(issues)

    // Enhance with LLM
    enhanced, err := d.enhanceIssues(ctx, toEnhance)
    if err != nil {
        // Log error but return original issues
        log.Warn("LLM enhancement failed", "error", err)
        return issues, nil
    }

    // Merge enhanced issues back
    return d.mergeEnhanced(issues, enhanced), nil
}

func (d *LLMEnhancedDetector) enhanceIssues(ctx context.Context, issues []model.DebtIssue) ([]EnhancedIssue, error) {
    var enhanced []EnhancedIssue

    for _, issue := range issues {
        // Get code snippet for context
        code, err := d.getCodeSnippet(ctx, issue)
        if err != nil {
            continue
        }

        // Validate issue (check for false positive)
        if d.cfg.ValidateIssues {
            valid, err := d.validateIssue(ctx, issue, code)
            if err != nil {
                continue
            }
            if !valid.IsValid {
                // Mark as false positive, will be filtered out
                enhanced = append(enhanced, EnhancedIssue{
                    Original:       issue,
                    IsFalsePositive: true,
                    Reasoning:      valid.Reasoning,
                })
                continue
            }
        }

        // Enhance suggestion
        if d.cfg.EnhanceSuggestions {
            suggestion, err := d.enhanceSuggestion(ctx, issue, code)
            if err != nil {
                continue
            }
            enhanced = append(enhanced, EnhancedIssue{
                Original:         issue,
                EnhancedSuggestion: suggestion,
            })
        }
    }

    return enhanced, nil
}
```

### LLM-Only Detector

```go
// src/service/detector/llm_detector.go
package detector

import (
    "context"

    "quality-bot/src/model"
    "quality-bot/src/service/llm"
)

// LLMDetector is a detector that relies entirely on LLM analysis
type LLMDetector struct {
    BaseDetector
    llmClient   llm.Client
    task        llm.AnalysisTask
    category    model.Category
    subcategory string
    cfg         LLMDetectorConfig
}

type LLMDetectorConfig struct {
    Enabled         bool     `yaml:"enabled"`
    BatchSize       int      `yaml:"batch_size"`        // Functions per request
    MaxFunctions    int      `yaml:"max_functions"`     // Total functions to analyze
    MinFunctionSize int      `yaml:"min_function_size"` // Skip trivial functions
    FocusPatterns   []string `yaml:"focus_patterns"`    // Prioritize matching files
}

func NewErrorHandlingDetector(base BaseDetector, llm llm.Client, cfg LLMDetectorConfig) *LLMDetector {
    return &LLMDetector{
        BaseDetector: base,
        llmClient:    llm,
        task:         llm.TaskDetectErrorHandling,
        category:     model.CategoryReliability,
        subcategory:  "error_handling",
        cfg:          cfg,
    }
}

func (d *LLMDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    if !d.cfg.Enabled {
        return nil, nil
    }

    // Get functions to analyze
    functions, err := d.Metrics.GetAllFunctionMetrics(ctx)
    if err != nil {
        return nil, err
    }

    // Filter and prioritize
    functions = d.filterFunctions(functions)

    var allIssues []model.DebtIssue

    // Process in batches
    for i := 0; i < len(functions); i += d.cfg.BatchSize {
        end := min(i+d.cfg.BatchSize, len(functions))
        batch := functions[i:end]

        issues, err := d.analyzeBatch(ctx, batch)
        if err != nil {
            // Log and continue with next batch
            log.Warn("Batch analysis failed", "batch", i, "error", err)
            continue
        }

        allIssues = append(allIssues, issues...)
    }

    return d.FilterBySeverity(allIssues), nil
}

func (d *LLMDetector) analyzeBatch(ctx context.Context, functions []model.FunctionMetrics) ([]model.DebtIssue, error) {
    var issues []model.DebtIssue

    for _, fn := range functions {
        // Get code
        code, err := d.getCode(ctx, fn)
        if err != nil {
            continue
        }

        // Build request
        req := llm.AnalysisRequest{
            Task: d.task,
            Code: code,
            Context: &llm.CodeContext{
                FilePath:     fn.FilePath,
                FunctionName: fn.Name,
                ClassName:    fn.ClassName,
            },
        }

        // Analyze
        resp, err := d.llmClient.Analyze(ctx, req)
        if err != nil {
            continue
        }

        // Parse results
        detected, err := d.parseResults(resp.Result, fn)
        if err != nil {
            continue
        }

        issues = append(issues, detected...)
    }

    return issues, nil
}
```

---

## Cost Optimization Strategies

### 1. Tiered Analysis

```yaml
llm:
  tiers:
    # Tier 1: Always analyze (most critical)
    critical:
      enabled: true
      severities: ["critical"]
      tasks: ["validate_issue", "enhance_suggestion"]

    # Tier 2: Analyze if budget allows
    high:
      enabled: true
      severities: ["high"]
      tasks: ["validate_issue"]
      max_cost: 0.50  # USD per run

    # Tier 3: Sample analysis
    medium:
      enabled: true
      severities: ["medium"]
      sample_rate: 0.1  # Analyze 10% of issues
      tasks: ["validate_issue"]
```

### 2. Intelligent Caching

```go
type LLMCache struct {
    // Code-based cache (same code = same analysis)
    codeCache    *lru.Cache  // hash(code) -> result

    // Pattern-based cache (similar patterns = similar issues)
    patternCache *lru.Cache  // hash(pattern) -> result template

    ttl          time.Duration
}

// Cache hit rate is typically 40-60% for similar codebases
```

### 3. Batch Processing

```go
// Instead of one API call per function, batch related functions
func (d *LLMDetector) batchAnalyze(ctx context.Context, functions []FunctionInfo) {
    // Group by file for context efficiency
    byFile := groupByFile(functions)

    for file, fileFuncs := range byFile {
        // Single prompt with multiple functions
        prompt := buildBatchPrompt(file, fileFuncs)
        // One API call for all functions in file
        results := d.llmClient.Analyze(ctx, prompt)
        // Parse and distribute results
    }
}
```

### 4. Progressive Enhancement

```go
// Start with fast heuristics, use LLM only when uncertain
func (d *HybridDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    issues, _ := d.traditionalDetector.Detect(ctx)

    var needsLLM []model.DebtIssue
    var confident []model.DebtIssue

    for _, issue := range issues {
        if d.isHighConfidence(issue) {
            confident = append(confident, issue)
        } else {
            needsLLM = append(needsLLM, issue)
        }
    }

    // Only send uncertain cases to LLM
    enhanced := d.llmEnhance(ctx, needsLLM)

    return append(confident, enhanced...), nil
}
```

### 5. Cost Tracking and Budgets

```go
type CostTracker struct {
    budget       float64
    spent        float64
    mu           sync.Mutex
    costPerToken float64
}

func (ct *CostTracker) CanAfford(estimatedTokens int) bool {
    ct.mu.Lock()
    defer ct.mu.Unlock()

    estimated := float64(estimatedTokens) * ct.costPerToken
    return ct.spent + estimated <= ct.budget
}

func (ct *CostTracker) Record(tokens int) {
    ct.mu.Lock()
    defer ct.mu.Unlock()

    ct.spent += float64(tokens) * ct.costPerToken
}
```

---

## Configuration

### Full Configuration Schema

```yaml
# config/config.yaml

llm:
  # Global enable/disable
  enabled: true

  # Provider configuration
  provider: "anthropic"  # "anthropic", "openai", "azure", "local"

  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-sonnet-4-20250514"  # or claude-3-haiku for cost savings
    max_tokens: 4096

  # Alternative providers
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"

  local:
    endpoint: "http://localhost:8080"
    model: "codellama-34b"

  # Rate limiting
  rate_limit:
    requests_per_minute: 60
    tokens_per_minute: 100000
    concurrent_requests: 5

  # Caching
  cache:
    enabled: true
    backend: "redis"  # "memory", "redis", "file"
    redis_url: "${REDIS_URL:-redis://localhost:6379}"
    ttl: 24h
    max_entries: 10000

  # Cost control
  cost:
    budget_per_run: 5.00      # USD
    budget_per_day: 50.00     # USD
    track_usage: true
    alert_threshold: 0.8      # Alert at 80% budget

  # Enhancement configuration
  enhancement:
    # Which existing detectors to enhance
    enhance_detectors:
      - complexity
      - coupling
      - duplication
      - layering

    # Enhancement tasks
    validate_issues: true       # Filter false positives
    enhance_suggestions: true   # Add specific advice
    assess_severity: true       # Refine severity based on context

    # Filtering
    min_severity_to_enhance: "medium"
    max_issues_per_detector: 50

  # LLM-only detectors
  detectors:
    error_handling:
      enabled: true
      batch_size: 10
      max_functions: 500
      min_function_lines: 5

    naming:
      enabled: true
      focus_patterns:
        - "src/service/**"
        - "src/handler/**"

    business_logic:
      enabled: true
      min_complexity: 5  # Only analyze complex functions

    security:
      enabled: true
      severity_override: "high"  # Security issues are always high+

    documentation:
      enabled: false  # Disabled by default

    test_quality:
      enabled: true
      test_patterns:
        - "**/*_test.go"
        - "**/test_*.py"

    api_design:
      enabled: true
      api_patterns:
        - "src/handler/**"
        - "src/api/**"

    concurrency:
      enabled: true
      focus_patterns:
        - "**/*worker*"
        - "**/*async*"
        - "**/*concurrent*"

    resource_management:
      enabled: true
      resource_types:
        - "file"
        - "connection"
        - "transaction"

    domain_model:
      enabled: true
      entity_patterns:
        - "src/domain/**"
        - "src/entity/**"
        - "src/model/**"

    logging:
      enabled: true
      min_function_lines: 10

    readability:
      enabled: true
      min_complexity: 3

    configuration:
      enabled: true
      config_patterns:
        - "**/*.yaml"
        - "**/*.json"
        - "**/config*"

# Custom prompts (optional, overrides built-in)
prompts:
  path: "./prompts"  # Directory with custom .tmpl files
```

### Environment Variables

```bash
# Required
export ANTHROPIC_API_KEY="sk-ant-..."

# Optional
export LLM_ENABLED=true
export LLM_MODEL="claude-sonnet-4-20250514"
export LLM_BUDGET_PER_RUN=5.00
export LLM_CACHE_ENABLED=true
export REDIS_URL="redis://localhost:6379"
```

---

## Summary

### Detection Capabilities Matrix

| Detector | Type | Traditional | LLM-Enhanced | LLM-Only |
|----------|------|-------------|--------------|----------|
| Complexity | Existing | Threshold | + Necessity assessment | - |
| Size | Existing | Line count | + Responsibility analysis | - |
| Coupling | Existing | Graph metrics | + Pattern recognition | - |
| Duplication | Existing | Embeddings | + Consolidation plans | - |
| Layering | Existing | Path patterns | + Actual responsibility | - |
| Error Handling | New | - | - | Full |
| Naming Quality | New | - | - | Full |
| Business Logic | New | - | - | Full |
| Security Smells | New | - | - | Full |
| Documentation | New | - | - | Full |
| API Design | New | - | - | Full |
| Test Quality | New | - | - | Full |
| Concurrency | New | - | - | Full |
| Resources | New | - | - | Full |
| Configuration | New | - | - | Full |
| Domain Model | New | - | - | Full |
| Logging | New | - | - | Full |
| Readability | New | - | - | Full |

### Value vs Cost Assessment

| Enhancement | Value | API Cost | Priority |
|-------------|-------|----------|----------|
| False positive filtering | High | Low | 1 |
| Context-aware suggestions | High | Medium | 2 |
| Error handling detection | High | Medium | 3 |
| Security smell detection | High | Medium | 4 |
| Naming quality | Medium | Low | 5 |
| Business logic smells | High | High | 6 |
| Test quality analysis | Medium | Medium | 7 |
| Documentation alignment | Medium | Low | 8 |
| Readability assessment | Medium | Medium | 9 |

### Implementation Phases

**Phase 1**: Enhancement of existing detectors
- False positive filtering
- Context-aware suggestions
- Estimated: 40% reduction in false positives, 3x improvement in suggestion quality

**Phase 2**: High-value LLM-only detectors
- Error handling quality
- Security smells
- Naming quality
- Estimated: 15-20% more actionable issues found

**Phase 3**: Comprehensive semantic analysis
- Business logic smells
- Test quality
- Domain model analysis
- API design quality
- Estimated: Full semantic coverage of code quality
