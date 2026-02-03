# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build              # Build binary to ./bin/quality-bot
make test               # Run all tests with verbose output
make test-coverage      # Generate coverage.out and coverage.html
make lint               # Run golangci-lint
make dev ARGS="..."     # Run directly with go run (no build step)
make deps               # go mod tidy && go mod download
```

Run analysis:
```bash
./bin/quality-bot analyze --repo <repo-name> --output ./reports --format markdown
```

## Architecture

**Layered architecture**: Handlers → Controllers → Services

```
cmd/quality-bot/main.go          Entry point, calls cli.Run()
        ↓
src/handler/cli/                 CLI commands (analyze, version, detectors)
        ↓
src/controller/                  Orchestration
  - analysis.go                  Runs detectors, aggregates issues, calculates debt score
  - report.go                    Generates output files
        ↓
src/service/
  - detector/                    Technical debt detectors (complexity, size, coupling, duplication)
  - codeapi/client.go            HTTP client for CodeAPI queries
  - metrics/provider.go          Metrics abstraction with caching
  - report/generator.go          Output formatting (JSON, Markdown, SARIF)
```

**Key patterns**:
- Detectors implement the `Detector` interface and are registered in `detector/runner.go`
- Parallel detector execution via `Runner` with configurable concurrency
- All code analysis queries go through CodeAPI (external service) via Cypher queries
- Configuration loaded from YAML with `${ENV_VAR:-default}` substitution

## Adding a New Detector

1. Create `src/service/detector/<name>.go` implementing the `Detector` interface
2. Register in `runner.go`'s `RegisterDetectors()` method
3. Add configuration struct in `src/config/config.go`
4. Add default thresholds in `src/config/defaults.go`

## Data Flow

1. CLI parses args → loads config from YAML
2. `AnalysisController` creates CodeAPI client and metrics provider
3. `Runner` executes registered detectors in parallel
4. Each detector queries CodeAPI for metrics, returns `[]DebtIssue`
5. Issues filtered by severity threshold, code snippets fetched if configured
6. `ReportController` generates output in requested formats

## Key Types

- `model.DebtIssue` - Single detected issue with severity, location, metrics
- `model.AnalysisReport` - Complete analysis output with summary and issues
- `model.Severity` - critical, high, medium, low
- `model.Category` - complexity, size, coupling, duplication, dead_code

## Configuration

See `config/config.example.yaml` for all options. Key sections:
- `codeapi` - CodeAPI connection settings
- `detectors` - Threshold values for each detector
- `exclusions` - File/class/function patterns to skip
- `output` - Formats, suggestions, metrics, code snippets
