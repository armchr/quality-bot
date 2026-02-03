package detector

import (
	"context"
	"fmt"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/util"
)

// SizeAndStructureDetector detects size-related issues in code entities
type SizeAndStructureDetector struct {
	BaseDetector
	cfg config.SizeDetectorConfig
}

// NewSizeAndStructureDetector creates a new size and structure detector
func NewSizeAndStructureDetector(base BaseDetector, cfg config.SizeDetectorConfig) *SizeAndStructureDetector {
	return &SizeAndStructureDetector{
		BaseDetector: base,
		cfg:          cfg,
	}
}

// Name returns the detector name
func (d *SizeAndStructureDetector) Name() string {
	return "size_structure"
}

// IsEnabled returns whether the detector is enabled
func (d *SizeAndStructureDetector) IsEnabled() bool {
	return d.cfg.Enabled
}

// Detect runs size and structure detection
func (d *SizeAndStructureDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
	util.Debug("Size detector: starting analysis")
	var issues []model.DebtIssue

	// Detect function-level issues
	funcIssues, err := d.detectFunctionIssues(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, funcIssues...)
	util.Debug("Size detector: found %d function-level issues", len(funcIssues))

	// Detect class-level issues
	classIssues, err := d.detectClassIssues(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, classIssues...)
	util.Debug("Size detector: found %d class-level issues", len(classIssues))

	// Detect file-level issues
	fileIssues, err := d.detectFileIssues(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, fileIssues...)
	util.Debug("Size detector: found %d file-level issues", len(fileIssues))

	return d.FilterBySeverity(issues), nil
}

func (d *SizeAndStructureDetector) detectFunctionIssues(ctx context.Context) ([]model.DebtIssue, error) {
	functions, err := d.Metrics.GetAllFunctionMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, fn := range functions {
		if d.ShouldExclude(fn.FilePath, fn.ClassName, fn.Name) {
			continue
		}

		// Check function length
		if fn.LineCount > d.cfg.MaxFunctionLines {
			issues = append(issues, d.createLongMethodIssue(fn))
		}

		// Check parameter count
		if fn.ParameterCount > d.cfg.MaxParameters {
			issues = append(issues, d.createLongParameterListIssue(fn))
		}
	}

	return issues, nil
}

func (d *SizeAndStructureDetector) detectClassIssues(ctx context.Context) ([]model.DebtIssue, error) {
	classes, err := d.Metrics.GetAllClassMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, cls := range classes {
		if d.ShouldExclude(cls.FilePath, cls.Name, "") {
			continue
		}

		// Check method count (god class)
		if cls.MethodCount > d.cfg.MaxClassMethods {
			issues = append(issues, d.createGodClassMethodIssue(cls))
		}

		// Check field count
		if cls.FieldCount > d.cfg.MaxClassFields {
			issues = append(issues, d.createGodClassFieldIssue(cls))
		}
	}

	return issues, nil
}

func (d *SizeAndStructureDetector) detectFileIssues(ctx context.Context) ([]model.DebtIssue, error) {
	files, err := d.Metrics.GetAllFileMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, file := range files {
		if d.ShouldExclude(file.Path, "", "") {
			continue
		}

		// Check file line count
		if file.LineCount > d.cfg.MaxFileLines {
			issues = append(issues, d.createLargeFileLineIssue(file))
		}

		// Check function count
		if file.FunctionCount > d.cfg.MaxFileFunctions {
			issues = append(issues, d.createLargeFileFunctionIssue(file))
		}
	}

	return issues, nil
}

func (d *SizeAndStructureDetector) createLongMethodIssue(fn model.FunctionMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if fn.LineCount > d.cfg.MaxFunctionLines*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "long_method",
		Severity:    severity,
		FilePath:    fn.FilePath,
		StartLine:   fn.StartLine,
		EndLine:     fn.EndLine,
		EntityName:  fn.Name,
		EntityType:  "function",
		Description: fmt.Sprintf("Method is too long (%d lines, threshold: %d)", fn.LineCount, d.cfg.MaxFunctionLines),
		Metrics: map[string]any{
			"line_count": fn.LineCount,
			"threshold":  d.cfg.MaxFunctionLines,
		},
		Suggestion: "Extract smaller, single-purpose methods",
	}
}

func (d *SizeAndStructureDetector) createLongParameterListIssue(fn model.FunctionMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if fn.ParameterCount > d.cfg.MaxParameters*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "long_parameter_list",
		Severity:    severity,
		FilePath:    fn.FilePath,
		StartLine:   fn.StartLine,
		EndLine:     fn.EndLine,
		EntityName:  fn.Name,
		EntityType:  "function",
		Description: fmt.Sprintf("Too many parameters (%d, threshold: %d)", fn.ParameterCount, d.cfg.MaxParameters),
		Metrics: map[string]any{
			"parameter_count": fn.ParameterCount,
			"threshold":       d.cfg.MaxParameters,
		},
		Suggestion: "Consider using a parameter object or builder pattern",
	}
}

func (d *SizeAndStructureDetector) createGodClassMethodIssue(cls model.ClassMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if cls.MethodCount > d.cfg.MaxClassMethods*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "god_class",
		Severity:    severity,
		FilePath:    cls.FilePath,
		StartLine:   cls.StartLine,
		EndLine:     cls.EndLine,
		EntityName:  cls.Name,
		EntityType:  "class",
		Description: fmt.Sprintf("Class has too many methods (%d, threshold: %d)", cls.MethodCount, d.cfg.MaxClassMethods),
		Metrics: map[string]any{
			"method_count": cls.MethodCount,
			"threshold":    d.cfg.MaxClassMethods,
		},
		Suggestion: "Split into smaller, focused classes following Single Responsibility Principle",
	}
}

func (d *SizeAndStructureDetector) createGodClassFieldIssue(cls model.ClassMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if cls.FieldCount > d.cfg.MaxClassFields*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "god_class",
		Severity:    severity,
		FilePath:    cls.FilePath,
		StartLine:   cls.StartLine,
		EndLine:     cls.EndLine,
		EntityName:  cls.Name,
		EntityType:  "class",
		Description: fmt.Sprintf("Class has too many fields (%d, threshold: %d)", cls.FieldCount, d.cfg.MaxClassFields),
		Metrics: map[string]any{
			"field_count": cls.FieldCount,
			"threshold":   d.cfg.MaxClassFields,
		},
		Suggestion: "Consider breaking into smaller classes or extracting value objects",
	}
}

func (d *SizeAndStructureDetector) createLargeFileLineIssue(file model.FileMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if file.LineCount > d.cfg.MaxFileLines*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "large_file",
		Severity:    severity,
		FilePath:    file.Path,
		StartLine:   1,
		EndLine:     file.LineCount,
		EntityName:  file.Path,
		EntityType:  "file",
		Description: fmt.Sprintf("File is too large (%d lines, threshold: %d)", file.LineCount, d.cfg.MaxFileLines),
		Metrics: map[string]any{
			"line_count": file.LineCount,
			"threshold":  d.cfg.MaxFileLines,
		},
		Suggestion: "Split into multiple files organized by responsibility",
	}
}

func (d *SizeAndStructureDetector) createLargeFileFunctionIssue(file model.FileMetrics) model.DebtIssue {
	severity := model.SeverityMedium
	if file.FunctionCount > d.cfg.MaxFileFunctions*2 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategorySize,
		Subcategory: "large_file",
		Severity:    severity,
		FilePath:    file.Path,
		StartLine:   1,
		EndLine:     file.LineCount,
		EntityName:  file.Path,
		EntityType:  "file",
		Description: fmt.Sprintf("File has too many functions (%d, threshold: %d)", file.FunctionCount, d.cfg.MaxFileFunctions),
		Metrics: map[string]any{
			"function_count": file.FunctionCount,
			"threshold":      d.cfg.MaxFileFunctions,
		},
		Suggestion: "Split into multiple files organized by feature or domain",
	}
}
