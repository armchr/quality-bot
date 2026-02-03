package detector

import (
	"context"
	"fmt"
	"sync"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/service/codeapi"
	"quality-bot/src/service/metrics"
	"quality-bot/src/util"
)

// DuplicationDetector detects similar or duplicate code
type DuplicationDetector struct {
	BaseDetector
	cfg             config.DuplicationDetectorConfig
	metricsProvider *metrics.Provider
	codeapiClient   *codeapi.Client
}

// NewDuplicationDetector creates a new duplication detector
func NewDuplicationDetector(
	base BaseDetector,
	cfg config.DuplicationDetectorConfig,
	metricsProvider *metrics.Provider,
	codeapiClient *codeapi.Client,
) *DuplicationDetector {
	return &DuplicationDetector{
		BaseDetector:    base,
		cfg:             cfg,
		metricsProvider: metricsProvider,
		codeapiClient:   codeapiClient,
	}
}

// Name returns the detector name
func (d *DuplicationDetector) Name() string {
	return "duplication"
}

// IsEnabled returns whether the detector is enabled
func (d *DuplicationDetector) IsEnabled() bool {
	return d.cfg.Enabled
}

// Detect runs duplication detection
func (d *DuplicationDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
	util.Debug("Duplication detector: fetching function metrics")
	functions, err := d.metricsProvider.GetAllFunctionMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Filter candidates: functions above minimum line count
	var candidates []model.FunctionMetrics
	excluded := 0
	trivialSkipped := 0
	tooSmall := 0

	for _, fn := range functions {
		if d.ShouldExclude(fn.FilePath, fn.ClassName, fn.Name) {
			excluded++
			continue
		}

		if fn.LineCount < d.cfg.MinLines {
			tooSmall++
			continue
		}

		// Skip trivial functions if configured
		if d.cfg.SkipTrivial && d.isTrivialFunction(fn) {
			trivialSkipped++
			continue
		}

		candidates = append(candidates, fn)
	}

	util.Debug("Duplication detector: %d candidates from %d functions (excluded: %d, too small: %d, trivial: %d)",
		len(candidates), len(functions), excluded, tooSmall, trivialSkipped)

	// Limit candidates for performance
	if len(candidates) > d.cfg.MaxFunctionsToCheck {
		util.Debug("Duplication detector: limiting to %d candidates (from %d)", d.cfg.MaxFunctionsToCheck, len(candidates))
		candidates = candidates[:d.cfg.MaxFunctionsToCheck]
	}

	// Track reported pairs to avoid duplicates
	var (
		issues   []model.DebtIssue
		mu       sync.Mutex
		reported = make(map[string]bool)
	)

	// Use concurrency for similarity search
	sem := make(chan struct{}, d.Cfg.Concurrency.SimilaritySearchWorkers)
	var wg sync.WaitGroup

	util.Debug("Duplication detector: searching for similar code with %d workers", d.Cfg.Concurrency.SimilaritySearchWorkers)

	for _, fn := range candidates {
		wg.Add(1)
		go func(fn model.FunctionMetrics) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			matches, err := d.findSimilarFunctions(ctx, fn)
			if err != nil {
				util.Debug("Duplication detector: similarity search failed for %s: %v", fn.Name, err)
				return
			}

			mu.Lock()
			defer mu.Unlock()

			for _, match := range matches {
				// Create unique key for the pair
				key := d.pairKey(fn.ID, match.FunctionID)
				if reported[key] {
					continue
				}
				reported[key] = true

				issues = append(issues, d.createDuplicationIssue(fn, match))
			}
		}(fn)
	}

	wg.Wait()

	util.Debug("Duplication detector: found %d duplicate pairs", len(issues))
	return d.FilterBySeverity(issues), nil
}

func (d *DuplicationDetector) isTrivialFunction(fn model.FunctionMetrics) bool {
	trivialNames := []string{
		"get", "set", "is", "has",
		"Get", "Set", "Is", "Has",
		"__init__", "__str__", "__repr__",
		"toString", "hashCode", "equals",
		"String", "Hash", "Equals",
	}

	for _, prefix := range trivialNames {
		if len(fn.Name) >= len(prefix) && fn.Name[:len(prefix)] == prefix {
			return true
		}
	}

	// Very small functions are likely trivial
	if fn.LineCount <= 5 && fn.CyclomaticComplexity <= 1 {
		return true
	}

	return false
}

func (d *DuplicationDetector) findSimilarFunctions(ctx context.Context, fn model.FunctionMetrics) ([]codeapi.SimilarCodeMatch, error) {
	req := codeapi.SimilarCodeRequest{
		FunctionID: fn.ID,
		TopK:       10,
		MinScore:   d.cfg.SimilarityThreshold,
	}

	resp, err := d.codeapiClient.SearchSimilarCode(ctx, req)
	if err != nil {
		return nil, err
	}

	// Filter out the function itself and overlapping functions
	var matches []codeapi.SimilarCodeMatch
	for _, match := range resp.Matches {
		// Skip self
		if match.FunctionID == fn.ID {
			continue
		}

		// Skip functions in the same file with overlapping lines
		if match.FilePath == fn.FilePath &&
			d.linesOverlap(fn.StartLine, fn.EndLine, match.StartLine, match.EndLine) {
			continue
		}

		// Only include matches above threshold
		if match.Score >= d.cfg.SimilarityThreshold {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

func (d *DuplicationDetector) linesOverlap(start1, end1, start2, end2 int) bool {
	return start1 <= end2 && start2 <= end1
}

func (d *DuplicationDetector) pairKey(id1, id2 string) string {
	if id1 < id2 {
		return id1 + ":" + id2
	}
	return id2 + ":" + id1
}

func (d *DuplicationDetector) createDuplicationIssue(fn model.FunctionMetrics, match codeapi.SimilarCodeMatch) model.DebtIssue {
	severity := model.SeverityMedium
	if match.Score > 0.95 {
		severity = model.SeverityHigh
	}

	return model.DebtIssue{
		Category:    model.CategoryDuplication,
		Subcategory: "similar_code",
		Severity:    severity,
		FilePath:    fn.FilePath,
		StartLine:   fn.StartLine,
		EndLine:     fn.EndLine,
		EntityName:  fn.Name,
		EntityType:  "function",
		Description: fmt.Sprintf("Function is %.0f%% similar to %s in %s:%d",
			match.Score*100, match.FunctionName, match.FilePath, match.StartLine),
		Metrics: map[string]any{
			"similarity_score":   match.Score,
			"duplicate_file":     match.FilePath,
			"duplicate_function": match.FunctionName,
			"duplicate_line":     match.StartLine,
		},
		Suggestion: "Extract common logic into a shared function",
	}
}
