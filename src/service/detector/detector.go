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
