package detector

import (
	"context"
	"fmt"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/util"
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
	util.Debug("Complexity detector: fetching function metrics")
	functions, err := d.Metrics.GetAllFunctionMetrics(ctx)
	if err != nil {
		return nil, err
	}

	util.Debug("Complexity detector: analyzing %d functions", len(functions))
	var issues []model.DebtIssue
	excluded := 0

	for _, fn := range functions {
		if d.ShouldExclude(fn.FilePath, fn.ClassName, fn.Name) {
			excluded++
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

	util.Debug("Complexity detector: %d functions excluded by filters", excluded)
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
