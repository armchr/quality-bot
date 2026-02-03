package detector

import (
	"context"
	"fmt"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/util"
)

// CouplingDetector detects coupling-related issues between code entities
type CouplingDetector struct {
	BaseDetector
	cfg config.CouplingDetectorConfig
}

// NewCouplingDetector creates a new coupling detector
func NewCouplingDetector(base BaseDetector, cfg config.CouplingDetectorConfig) *CouplingDetector {
	return &CouplingDetector{
		BaseDetector: base,
		cfg:          cfg,
	}
}

// Name returns the detector name
func (d *CouplingDetector) Name() string {
	return "coupling"
}

// IsEnabled returns whether the detector is enabled
func (d *CouplingDetector) IsEnabled() bool {
	return d.cfg.Enabled
}

// Detect runs coupling detection
func (d *CouplingDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
	util.Debug("Coupling detector: starting analysis")
	var issues []model.DebtIssue

	// Detect feature envy
	featureEnvyIssues, err := d.detectFeatureEnvy(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, featureEnvyIssues...)
	util.Debug("Coupling detector: found %d feature envy issues", len(featureEnvyIssues))

	// Detect high coupling
	highCouplingIssues, err := d.detectHighCoupling(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, highCouplingIssues...)
	util.Debug("Coupling detector: found %d high coupling issues", len(highCouplingIssues))

	// Detect inappropriate intimacy
	intimacyIssues, err := d.detectInappropriateIntimacy(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, intimacyIssues...)
	util.Debug("Coupling detector: found %d inappropriate intimacy issues", len(intimacyIssues))

	// Detect primitive obsession
	primitiveIssues, err := d.detectPrimitiveObsession(ctx)
	if err != nil {
		return nil, err
	}
	issues = append(issues, primitiveIssues...)
	util.Debug("Coupling detector: found %d primitive obsession issues", len(primitiveIssues))

	return d.FilterBySeverity(issues), nil
}

func (d *CouplingDetector) detectFeatureEnvy(ctx context.Context) ([]model.DebtIssue, error) {
	functions, err := d.Metrics.GetAllFunctionMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, fn := range functions {
		if d.ShouldExclude(fn.FilePath, fn.ClassName, fn.Name) {
			continue
		}

		// Feature envy: method uses external fields more than its own class fields
		if fn.ClassName != "" &&
			fn.ExternalFieldUses > fn.OwnFieldUses &&
			fn.ExternalFieldUses > d.cfg.FeatureEnvyThreshold {

			severity := model.SeverityMedium
			ratio := float64(fn.ExternalFieldUses) / float64(max(fn.OwnFieldUses, 1))
			if ratio > 3 {
				severity = model.SeverityHigh
			}

			issues = append(issues, model.DebtIssue{
				Category:    model.CategoryCoupling,
				Subcategory: "feature_envy",
				Severity:    severity,
				FilePath:    fn.FilePath,
				StartLine:   fn.StartLine,
				EndLine:     fn.EndLine,
				EntityName:  fn.Name,
				EntityType:  "function",
				Description: fmt.Sprintf("Method uses %d external fields vs %d own fields", fn.ExternalFieldUses, fn.OwnFieldUses),
				Metrics: map[string]any{
					"external_field_uses": fn.ExternalFieldUses,
					"own_field_uses":      fn.OwnFieldUses,
					"ratio":               ratio,
				},
				Suggestion: "Consider moving this method to the class whose data it uses most",
			})
		}
	}

	return issues, nil
}

func (d *CouplingDetector) detectHighCoupling(ctx context.Context) ([]model.DebtIssue, error) {
	classes, err := d.Metrics.GetAllClassMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, cls := range classes {
		if d.ShouldExclude(cls.FilePath, cls.Name, "") {
			continue
		}

		if cls.DependencyCount > d.cfg.MaxDependencies {
			severity := model.SeverityMedium
			if cls.DependencyCount > d.cfg.MaxDependencies*2 {
				severity = model.SeverityHigh
			}

			issues = append(issues, model.DebtIssue{
				Category:    model.CategoryCoupling,
				Subcategory: "high_coupling",
				Severity:    severity,
				FilePath:    cls.FilePath,
				StartLine:   cls.StartLine,
				EndLine:     cls.EndLine,
				EntityName:  cls.Name,
				EntityType:  "class",
				Description: fmt.Sprintf("Class depends on %d other classes (threshold: %d)", cls.DependencyCount, d.cfg.MaxDependencies),
				Metrics: map[string]any{
					"dependency_count": cls.DependencyCount,
					"threshold":        d.cfg.MaxDependencies,
				},
				Suggestion: "Reduce dependencies by introducing abstractions or reorganizing responsibilities",
			})
		}
	}

	return issues, nil
}

func (d *CouplingDetector) detectInappropriateIntimacy(ctx context.Context) ([]model.DebtIssue, error) {
	classPairs, err := d.Metrics.GetClassPairMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue
	reported := make(map[string]bool)

	for _, pair := range classPairs {
		// Skip if either class is excluded
		if d.ShouldExclude(pair.Class1File, pair.Class1Name, "") ||
			d.ShouldExclude(pair.Class2File, pair.Class2Name, "") {
			continue
		}

		// Check for inappropriate intimacy (bidirectional high coupling)
		if pair.Calls1To2 > d.cfg.IntimacyCallThreshold &&
			pair.Calls2To1 > d.cfg.IntimacyCallThreshold {

			// Create a unique key to avoid duplicate reports
			key := pair.Class1Name + ":" + pair.Class2Name
			if pair.Class1Name > pair.Class2Name {
				key = pair.Class2Name + ":" + pair.Class1Name
			}

			if reported[key] {
				continue
			}
			reported[key] = true

			severity := model.SeverityHigh // Bidirectional is always more serious

			issues = append(issues, model.DebtIssue{
				Category:    model.CategoryCoupling,
				Subcategory: "inappropriate_intimacy",
				Severity:    severity,
				FilePath:    pair.Class1File,
				StartLine:   1,
				EndLine:     1,
				EntityName:  fmt.Sprintf("%s <-> %s", pair.Class1Name, pair.Class2Name),
				EntityType:  "class_pair",
				Description: fmt.Sprintf("Classes are too tightly coupled (%d calls each way)", min(pair.Calls1To2, pair.Calls2To1)),
				Metrics: map[string]any{
					"calls_1_to_2":        pair.Calls1To2,
					"calls_2_to_1":        pair.Calls2To1,
					"shared_field_access": pair.SharedFieldAccess,
					"class1":              pair.Class1Name,
					"class2":              pair.Class2Name,
				},
				Suggestion: "Extract shared logic into a new class or merge if appropriate",
			})
		}
	}

	return issues, nil
}

func (d *CouplingDetector) detectPrimitiveObsession(ctx context.Context) ([]model.DebtIssue, error) {
	classes, err := d.Metrics.GetAllClassMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var issues []model.DebtIssue

	for _, cls := range classes {
		if d.ShouldExclude(cls.FilePath, cls.Name, "") {
			continue
		}

		if cls.PrimitiveFieldCount > d.cfg.PrimitiveFieldThreshold {
			issues = append(issues, model.DebtIssue{
				Category:    model.CategoryCoupling,
				Subcategory: "primitive_obsession",
				Severity:    model.SeverityMedium,
				FilePath:    cls.FilePath,
				StartLine:   cls.StartLine,
				EndLine:     cls.EndLine,
				EntityName:  cls.Name,
				EntityType:  "class",
				Description: fmt.Sprintf("Class has %d primitive fields (threshold: %d)", cls.PrimitiveFieldCount, d.cfg.PrimitiveFieldThreshold),
				Metrics: map[string]any{
					"primitive_field_count": cls.PrimitiveFieldCount,
					"total_field_count":     cls.FieldCount,
					"threshold":             d.cfg.PrimitiveFieldThreshold,
				},
				Suggestion: "Consider creating value objects or domain types for related primitives",
			})
		}
	}

	return issues, nil
}
