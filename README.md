# Quality Bot

A technical debt detection agent that analyzes codebases using [CodeAPI](https://github.com/armchr/codeapi) to identify code quality issues, complexity hotspots, and structural problems.

## Features

- **Complexity Detection**: Identifies functions with high cyclomatic complexity and deep nesting
- **Size Analysis**: Detects oversized functions, classes, and files
- **Coupling Detection**: Finds feature envy, high coupling, inappropriate intimacy, and primitive obsession
- **Code Duplication**: Uses semantic similarity search to find duplicate code patterns
- **Multiple Output Formats**: JSON, Markdown, and SARIF for CI/CD integration
- **Configurable Thresholds**: All detection thresholds are customizable via YAML
- **Parallel Execution**: Runs detectors concurrently for faster analysis

## Prerequisites

- **Go 1.21+**
- **CodeAPI** running and accessible (for code graph queries)
- A repository indexed in CodeAPI

## Quick Start

### 1. Build

```bash
make build
# or
go build -o bin/quality-bot ./cmd/quality-bot
```

### 2. Configure

```bash
cp config/config.example.yaml config/config.yaml
# Edit config/config.yaml with your settings
```

### 3. Run Analysis

```bash
./bin/quality-bot analyze --repo <repo-name> --output ./reports --format markdown
```

## CLI Commands

### analyze

Run technical debt analysis on a repository.

```bash
./bin/quality-bot analyze --repo <repo-name> [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--repo` | `-r` | Repository name (required, must be indexed in CodeAPI) |
| `--output` | `-o` | Output directory for reports |
| `--format` | `-f` | Output format: `json`, `markdown`, `sarif` |
| `--config` | `-c` | Path to configuration file |
| `--timeout` | `-t` | Analysis timeout (default: 5m) |

### detectors

List available detectors and their status.

```bash
./bin/quality-bot detectors
```

### version

Show version information.

```bash
./bin/quality-bot version
```

## Configuration

Configuration is done via YAML with environment variable substitution support:

```yaml
# Reference environment variables with ${VAR} or ${VAR:-default}
codeapi:
  url: "${CODEAPI_URL:-http://localhost:8181}"

logging:
  level: "${LOG_LEVEL:-info}"  # debug, info, warn, error
```

### Key Configuration Sections

#### CodeAPI Connection

```yaml
codeapi:
  url: "http://localhost:8181"
  timeout: 30s
  retry:
    max_attempts: 3
    backoff_factor: 1.5
```

#### Detector Thresholds

```yaml
detectors:
  complexity:
    enabled: true
    cyclomatic_moderate: 10    # Medium severity threshold
    cyclomatic_high: 15        # High severity threshold
    cyclomatic_critical: 20    # Critical severity threshold
    max_nesting_depth: 4

  size_and_structure:
    enabled: true
    max_function_lines: 50
    max_parameters: 5
    max_class_methods: 20
    max_class_fields: 15
    max_file_lines: 500

  coupling:
    enabled: true
    max_dependencies: 10
    feature_envy_threshold: 3
    intimacy_call_threshold: 3

  duplication:
    enabled: true
    similarity_threshold: 0.85
    min_lines: 5
```

#### Exclusions

```yaml
exclusions:
  file_patterns:
    - "**/test/**"
    - "**/vendor/**"
  class_patterns:
    - "^Test"
    - "Mock$"
  function_patterns:
    - "^test_"
```

#### Output Options

```yaml
output:
  formats: [json, markdown]
  output_dir: "./reports"
  include_suggestions: true
  include_metrics: true
  include_code_snippets: false
  max_issues_per_category: 100
```

See `config/config.example.yaml` for full configuration options.

## Detectors

### Complexity Detector

Identifies functions with high cognitive complexity:

- **Cyclomatic Complexity**: Counts decision points (branches, loops)
- **Deep Nesting**: Detects deeply nested control flow

### Size & Structure Detector

Finds oversized code entities:

- **Long Methods**: Functions exceeding line thresholds
- **Long Parameter Lists**: Functions with too many parameters
- **God Classes**: Classes with too many methods or fields
- **Large Files**: Files with excessive lines or functions

### Coupling Detector

Detects problematic dependencies between code entities:

- **Feature Envy**: Methods that use other classes' data more than their own
- **High Coupling**: Classes with too many dependencies
- **Inappropriate Intimacy**: Bidirectional tight coupling between classes
- **Primitive Obsession**: Classes with excessive primitive fields

### Duplication Detector

Uses CodeAPI's semantic similarity search to find:

- **Similar Code**: Functions with high semantic similarity scores
- Configurable similarity threshold (default: 85%)

## Output Formats

### JSON

Structured output suitable for programmatic processing:

```json
{
  "repo_name": "my-project",
  "generated_at": "2024-01-15T10:30:00Z",
  "summary": {
    "total_issues": 42,
    "debt_score": 35.5,
    "by_severity": {"critical": 2, "high": 10, "medium": 20, "low": 10}
  },
  "issues": [...]
}
```

### Markdown

Human-readable report with tables and formatted issues.

### SARIF

Static Analysis Results Interchange Format for CI/CD integration (GitHub Code Scanning, etc.).

## Architecture

```
quality-bot/
├── cmd/quality-bot/       # CLI entry point
├── src/
│   ├── config/            # Configuration loading
│   ├── controller/        # Orchestration layer
│   ├── handler/cli/       # CLI command handlers
│   ├── model/             # Data models
│   ├── service/
│   │   ├── codeapi/       # CodeAPI client
│   │   ├── detector/      # Debt detectors
│   │   ├── metrics/       # Metrics provider with caching
│   │   └── report/        # Report generation
│   └── util/              # Logging utilities
└── config/                # Configuration files
```

## Known Limitations

1. **Else-if Chain Nesting**: The nesting depth calculation counts `else if` chains as nested structures (since they're represented as `else { if { } }` in the AST). This can inflate nesting depth for functions with long if/else-if chains.

2. **Dead Code Detection**: Not yet implemented (planned for future release).

3. **CodeAPI Dependency**: Requires a running CodeAPI instance with indexed repositories.

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.
