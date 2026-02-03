# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-02

### Added

- **Core Analysis Engine**
  - Layered architecture: Handler → Controller → Service
  - Parallel detector execution with configurable concurrency
  - Metrics provider with caching support

- **Detectors**
  - Complexity detector: cyclomatic complexity, deep nesting detection
  - Size & structure detector: long methods, god classes, large files
  - Coupling detector: feature envy, high coupling, inappropriate intimacy, primitive obsession
  - Duplication detector: semantic similarity-based code clone detection

- **CodeAPI Integration**
  - Cypher query execution for code graph analysis
  - Semantic similarity search for duplication detection
  - Code snippet fetching for issue context
  - Automatic retry with exponential backoff

- **Configuration**
  - YAML-based configuration with environment variable substitution
  - Support for `${VAR}` and `${VAR:-default}` syntax
  - Configurable thresholds for all detectors
  - File/class/function exclusion patterns

- **Output Formats**
  - JSON: structured output for programmatic processing
  - Markdown: human-readable reports with tables
  - SARIF: CI/CD integration (GitHub Code Scanning compatible)

- **CLI Interface**
  - `analyze` command for running analysis
  - `detectors` command for listing available detectors
  - `version` command for version information
  - Configurable timeout and output options

- **Logging**
  - Configurable log levels (debug, info, warn, error)
  - Optional file output
  - Timestamps and caller information options

- **Report Features**
  - Issue categorization by severity and type
  - Debt score calculation
  - Hotspot file identification
  - Optional code snippets in reports
  - Actionable suggestions for each issue

### Known Limitations

- Else-if chains are counted as nested structures, which can inflate nesting depth metrics
- Dead code detection is planned for a future release
- Requires CodeAPI to be running with indexed repositories

## [Unreleased]

### Planned

- Dead code detection (unreachable code, unused functions)
- Trend analysis (comparing across runs)
- Integration with more CI/CD platforms
- Custom detector plugin system
- IDE integrations
