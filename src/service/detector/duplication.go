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
				// Create unique key for the pair using file:line
				matchKey := fmt.Sprintf("%s:%d", match.Chunk.FilePath, match.Chunk.StartLine)
				key := d.pairKey(fn.ID, matchKey)
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

func (d *DuplicationDetector) findSimilarFunctions(ctx context.Context, fn model.FunctionMetrics) ([]codeapi.SimilarCodeResult, error) {
	// Fetch code snippet for this function
	repoName := d.metricsProvider.RepoName()
	snippet, err := d.codeapiClient.GetSnippet(ctx, repoName, fn.FilePath, fn.StartLine, fn.EndLine)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch code snippet: %w", err)
	}

	if snippet.Code == "" {
		return nil, nil // No code to search for
	}

	// Determine language from file extension
	language := d.detectLanguage(fn.FilePath)
	if language == "" {
		return nil, nil // Unknown language
	}

	req := codeapi.SimilarCodeRequest{
		RepoName:    repoName,
		CodeSnippet: snippet.Code,
		Language:    language,
		Limit:       10,
		IncludeCode: false,
	}

	resp, err := d.codeapiClient.SearchSimilarCode(ctx, req)
	if err != nil {
		return nil, err
	}

	// Filter out non-function matches and self-matches
	var matches []codeapi.SimilarCodeResult
	for _, result := range resp.Results {
		// Skip non-function matches (while, switch, for, etc.)
		// We only want function-to-function duplicates
		if !d.isFunctionChunk(result.Chunk) {
			continue
		}

		// Skip matches in the same file with overlapping lines (self-matches)
		if result.Chunk.FilePath == fn.FilePath &&
			d.linesOverlap(fn.StartLine, fn.EndLine, result.Chunk.StartLine, result.Chunk.EndLine) {
			continue
		}

		// Only include matches above threshold
		if result.Score >= d.cfg.SimilarityThreshold {
			matches = append(matches, result)
		}
	}

	// Deduplicate overlapping matches (e.g., function vs inner switch/loop)
	matches = d.deduplicateOverlappingMatches(matches)

	return matches, nil
}

// deduplicateOverlappingMatches removes overlapping matches in the same file,
// keeping only the most relevant one (highest score, or largest if scores are similar)
func (d *DuplicationDetector) deduplicateOverlappingMatches(matches []codeapi.SimilarCodeResult) []codeapi.SimilarCodeResult {
	if len(matches) <= 1 {
		return matches
	}

	// Group matches by file
	byFile := make(map[string][]codeapi.SimilarCodeResult)
	for _, m := range matches {
		byFile[m.Chunk.FilePath] = append(byFile[m.Chunk.FilePath], m)
	}

	var result []codeapi.SimilarCodeResult
	for _, fileMatches := range byFile {
		if len(fileMatches) == 1 {
			result = append(result, fileMatches[0])
			continue
		}

		// For each file, filter out matches that overlap with a better match
		kept := make([]bool, len(fileMatches))
		for i := range kept {
			kept[i] = true
		}

		for i := 0; i < len(fileMatches); i++ {
			if !kept[i] {
				continue
			}
			for j := i + 1; j < len(fileMatches); j++ {
				if !kept[j] {
					continue
				}

				// Check if these two matches overlap
				if d.linesOverlap(
					fileMatches[i].Chunk.StartLine, fileMatches[i].Chunk.EndLine,
					fileMatches[j].Chunk.StartLine, fileMatches[j].Chunk.EndLine,
				) {
					// Keep the better one: higher score wins, or larger span if scores are close
					better := d.betterMatch(fileMatches[i], fileMatches[j])
					if better == i {
						kept[j] = false
					} else {
						kept[i] = false
						break // i is no longer kept, move on
					}
				}
			}
		}

		for i, m := range fileMatches {
			if kept[i] {
				result = append(result, m)
			}
		}
	}

	return result
}

// betterMatch returns the index (0 or 1) of the better match
// Prefers higher score, or larger span if scores are within 5%
func (d *DuplicationDetector) betterMatch(a, b codeapi.SimilarCodeResult) int {
	scoreDiff := a.Score - b.Score
	if scoreDiff > 0.05 {
		return 0 // a has significantly higher score
	}
	if scoreDiff < -0.05 {
		return 1 // b has significantly higher score
	}

	// Scores are similar, prefer larger span (more likely to be the function, not inner block)
	spanA := a.Chunk.EndLine - a.Chunk.StartLine
	spanB := b.Chunk.EndLine - b.Chunk.StartLine
	if spanA >= spanB {
		return 0
	}
	return 1
}

func (d *DuplicationDetector) detectLanguage(filePath string) string {
	ext := ""
	if idx := len(filePath) - 1; idx >= 0 {
		for i := len(filePath) - 1; i >= 0; i-- {
			if filePath[i] == '.' {
				ext = filePath[i+1:]
				break
			}
		}
	}

	switch ext {
	case "go":
		return "go"
	case "py":
		return "python"
	case "java":
		return "java"
	case "ts":
		return "typescript"
	case "js":
		return "javascript"
	case "cs":
		return "csharp"
	default:
		return ""
	}
}

func (d *DuplicationDetector) linesOverlap(start1, end1, start2, end2 int) bool {
	return start1 <= end2 && start2 <= end1
}

// isFunctionChunk returns true if the chunk represents a function, not an inner block
func (d *DuplicationDetector) isFunctionChunk(chunk codeapi.SimilarCodeChunk) bool {
	// Check chunk type if available
	if chunk.ChunkType == "function" {
		return true
	}

	// If chunk type is explicitly something else, it's not a function
	if chunk.ChunkType != "" && chunk.ChunkType != "function" {
		return false
	}

	// Fallback: check if name looks like a statement keyword
	statementKeywords := map[string]bool{
		"while": true, "for": true, "if": true, "else": true,
		"switch": true, "case": true, "try": true, "catch": true,
		"finally": true, "do": true, "foreach": true, "with": true,
		"match": true, "select": true, "loop": true,
	}

	return !statementKeywords[chunk.Name]
}

func (d *DuplicationDetector) pairKey(id1, id2 string) string {
	if id1 < id2 {
		return id1 + ":" + id2
	}
	return id2 + ":" + id1
}

func (d *DuplicationDetector) createDuplicationIssue(fn model.FunctionMetrics, match codeapi.SimilarCodeResult) model.DebtIssue {
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
			match.Score*100, match.Chunk.Name, match.Chunk.FilePath, match.Chunk.StartLine),
		Metrics: map[string]any{
			"similarity_score":   match.Score,
			"duplicate_file":     match.Chunk.FilePath,
			"duplicate_function": match.Chunk.Name,
			"duplicate_line":     match.Chunk.StartLine,
		},
		Suggestion: "Extract common logic into a shared function",
	}
}
