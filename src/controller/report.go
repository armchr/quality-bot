package controller

import (
	"os"
	"path/filepath"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/service/report"
	"quality-bot/src/util"
)

// ReportController handles report generation
type ReportController struct {
	cfg *config.Config
}

// NewReportController creates a new report controller
func NewReportController(cfg *config.Config) *ReportController {
	return &ReportController{cfg: cfg}
}

// GenerateReports generates reports in all configured formats
func (c *ReportController) GenerateReports(analysisReport *model.AnalysisReport) ([]string, error) {
	util.Debug("Generating reports for %d formats: %v", len(c.cfg.Output.Formats), c.cfg.Output.Formats)
	reportGenerator := report.NewGenerator(c.cfg.Output)
	var outputPaths []string

	for _, format := range c.cfg.Output.Formats {
		util.Debug("Generating %s report", format)
		output, err := reportGenerator.Generate(analysisReport, format)
		if err != nil {
			util.Error("Failed to generate %s report: %v", format, err)
			return nil, err
		}

		// Determine output path
		outputPath := c.getOutputPath(analysisReport.RepoName, format)

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			util.Error("Failed to create output directory: %v", err)
			return nil, err
		}

		// Write file
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			util.Error("Failed to write report to %s: %v", outputPath, err)
			return nil, err
		}

		util.Info("Report written: %s", outputPath)
		outputPaths = append(outputPaths, outputPath)
	}

	return outputPaths, nil
}

// GenerateToString generates a report to a string
func (c *ReportController) GenerateToString(analysisReport *model.AnalysisReport, format string) (string, error) {
	reportGenerator := report.NewGenerator(c.cfg.Output)
	return reportGenerator.Generate(analysisReport, format)
}

func (c *ReportController) getOutputPath(repoName, format string) string {
	ext := format
	if format == "markdown" {
		ext = "md"
	}

	filename := repoName + "-debt-report." + ext
	return filepath.Join(c.cfg.Output.OutputDir, filename)
}
