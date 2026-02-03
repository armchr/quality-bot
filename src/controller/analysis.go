package controller

import (
	"context"
	"time"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/service/codeapi"
	"quality-bot/src/service/detector"
	"quality-bot/src/service/metrics"
	"quality-bot/src/util"
)

// AnalysisController orchestrates the debt analysis process
type AnalysisController struct {
	cfg *config.Config
}

// NewAnalysisController creates a new analysis controller
func NewAnalysisController(cfg *config.Config) *AnalysisController {
	return &AnalysisController{cfg: cfg}
}

// AnalyzeRequest represents a request to analyze a repository
type AnalyzeRequest struct {
	RepoName  string
	Detectors []string // Optional: specific detectors to run (empty = all)
}

// Analyze runs the full analysis pipeline
func (c *AnalysisController) Analyze(ctx context.Context, req AnalyzeRequest) (*model.AnalysisReport, error) {
	startTime := time.Now()
	util.Info("Starting analysis for repository: %s", req.RepoName)

	// Create CodeAPI client
	codeapiClient := codeapi.NewClient(c.cfg.CodeAPI)
	util.Debug("CodeAPI client initialized (endpoint: %s)", c.cfg.CodeAPI.URL)

	// Create metrics provider
	metricsProvider := metrics.NewProvider(codeapiClient, req.RepoName, c.cfg.Cache)
	util.Debug("Metrics provider initialized (cache enabled: %v)", c.cfg.Cache.Enabled)

	// Create detector runner
	detectorRunner := detector.NewRunner(metricsProvider, codeapiClient, c.cfg)

	// Run all detectors
	util.Info("Running detectors")
	issues, err := detectorRunner.RunAll(ctx)
	if err != nil {
		util.Error("Detector run failed: %v", err)
		return nil, err
	}

	// Apply global filters
	preFilterCount := len(issues)
	issues = c.applyGlobalFilters(issues)
	if preFilterCount != len(issues) {
		util.Debug("Global filters reduced issues from %d to %d", preFilterCount, len(issues))
	}

	// Fetch code snippets if configured
	if c.cfg.Output.IncludeCodeSnippets {
		util.Debug("Fetching code snippets for %d issues", len(issues))
		issues = c.fetchCodeSnippets(ctx, codeapiClient, req.RepoName, issues)
	}

	// Generate report
	report := &model.AnalysisReport{
		RepoName:    req.RepoName,
		GeneratedAt: time.Now().UTC(),
		Issues:      issues,
		Summary:     c.generateSummary(issues),
	}

	util.Info("Analysis complete: %d issues found, debt score: %.1f (took %v)",
		len(issues), report.Summary.DebtScore, time.Since(startTime))

	return report, nil
}

// fetchCodeSnippets fetches code snippets for each issue from CodeAPI
func (c *AnalysisController) fetchCodeSnippets(ctx context.Context, client *codeapi.Client, repoName string, issues []model.DebtIssue) []model.DebtIssue {
	fetched := 0
	skipped := 0
	failed := 0

	for i := range issues {
		issue := &issues[i]

		// Skip file-level issues or issues without valid line ranges
		if issue.EntityType == "file" || issue.StartLine <= 0 || issue.EndLine <= 0 {
			skipped++
			continue
		}

		// Fetch snippet from CodeAPI
		resp, err := client.GetSnippet(ctx, repoName, issue.FilePath, issue.StartLine, issue.EndLine)
		if err != nil {
			util.Debug("Failed to fetch snippet for %s:%d-%d: %v", issue.FilePath, issue.StartLine, issue.EndLine, err)
			failed++
			continue
		}

		issue.CodeSnippet = resp.Code
		fetched++
	}

	util.Debug("Code snippets: %d fetched, %d skipped, %d failed", fetched, skipped, failed)
	return issues
}

func (c *AnalysisController) applyGlobalFilters(issues []model.DebtIssue) []model.DebtIssue {
	maxPerCategory := c.cfg.Output.MaxIssuesPerCategory
	if maxPerCategory <= 0 {
		return issues
	}

	byCategory := make(map[model.Category][]model.DebtIssue)
	for _, issue := range issues {
		if len(byCategory[issue.Category]) < maxPerCategory {
			byCategory[issue.Category] = append(byCategory[issue.Category], issue)
		}
	}

	var filtered []model.DebtIssue
	for _, catIssues := range byCategory {
		filtered = append(filtered, catIssues...)
	}

	return filtered
}

func (c *AnalysisController) generateSummary(issues []model.DebtIssue) model.ReportSummary {
	byCategory := make(map[model.Category]int)
	bySeverity := make(map[model.Severity]int)
	byFile := make(map[string]int)

	for _, issue := range issues {
		byCategory[issue.Category]++
		bySeverity[issue.Severity]++
		byFile[issue.FilePath]++
	}

	// Find hotspots
	type fileCount struct {
		path  string
		count int
	}
	var files []fileCount
	for path, count := range byFile {
		files = append(files, fileCount{path, count})
	}
	// Sort by count descending
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].count > files[i].count {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	topN := c.cfg.Output.HotspotsTopN
	if topN > len(files) {
		topN = len(files)
	}

	hotspots := make([]model.FileHotspot, topN)
	for i := 0; i < topN; i++ {
		hotspots[i] = model.FileHotspot{
			FilePath:   files[i].path,
			IssueCount: files[i].count,
		}
	}

	return model.ReportSummary{
		TotalIssues:  len(issues),
		ByCategory:   byCategory,
		BySeverity:   bySeverity,
		HotspotFiles: hotspots,
		DebtScore:    c.calculateDebtScore(issues),
	}
}

func (c *AnalysisController) calculateDebtScore(issues []model.DebtIssue) float64 {
	if len(issues) == 0 {
		return 0
	}

	weights := map[model.Severity]int{
		model.SeverityLow:      1,
		model.SeverityMedium:   3,
		model.SeverityHigh:     7,
		model.SeverityCritical: 15,
	}

	var total int
	for _, issue := range issues {
		total += weights[issue.Severity]
	}

	score := float64(total) / 10.0
	if score > 100 {
		score = 100
	}

	return score
}
