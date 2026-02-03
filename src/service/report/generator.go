package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/util"
)

// Generator generates reports in various formats
type Generator struct {
	cfg config.OutputConfig
}

// NewGenerator creates a new report generator
func NewGenerator(cfg config.OutputConfig) *Generator {
	return &Generator{cfg: cfg}
}

// Generate generates a report in the specified format
func (g *Generator) Generate(report *model.AnalysisReport, format string) (string, error) {
	util.Debug("Generating report in %s format (%d issues)", format, len(report.Issues))
	switch format {
	case "json":
		return g.generateJSON(report)
	case "markdown", "md":
		return g.generateMarkdown(report)
	case "sarif":
		return g.generateSARIF(report)
	default:
		util.Warn("Unsupported report format requested: %s", format)
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (g *Generator) generateJSON(report *model.AnalysisReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (g *Generator) generateMarkdown(report *model.AnalysisReport) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("# Technical Debt Analysis Report\n\n")
	sb.WriteString(fmt.Sprintf("**Repository:** %s\n", report.RepoName))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05 UTC")))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Issues:** %d\n", report.Summary.TotalIssues))
	sb.WriteString(fmt.Sprintf("- **Debt Score:** %.1f/100\n\n", report.Summary.DebtScore))

	// By Severity
	sb.WriteString("### Issues by Severity\n\n")
	sb.WriteString("| Severity | Count |\n")
	sb.WriteString("|----------|-------|\n")
	for _, sev := range []model.Severity{model.SeverityCritical, model.SeverityHigh, model.SeverityMedium, model.SeverityLow} {
		count := report.Summary.BySeverity[sev]
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", sev, count))
	}
	sb.WriteString("\n")

	// By Category
	sb.WriteString("### Issues by Category\n\n")
	sb.WriteString("| Category | Count |\n")
	sb.WriteString("|----------|-------|\n")
	for _, cat := range []model.Category{model.CategoryComplexity, model.CategorySize, model.CategoryCoupling, model.CategoryDuplication} {
		count := report.Summary.ByCategory[cat]
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", cat, count))
	}
	sb.WriteString("\n")

	// Hotspots
	if len(report.Summary.HotspotFiles) > 0 {
		sb.WriteString("### Hotspot Files\n\n")
		sb.WriteString("| File | Issue Count |\n")
		sb.WriteString("|------|-------------|\n")
		for _, hs := range report.Summary.HotspotFiles {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", hs.FilePath, hs.IssueCount))
		}
		sb.WriteString("\n")
	}

	// Issues by Category
	sb.WriteString("## Issues\n\n")

	issuesByCategory := make(map[model.Category][]model.DebtIssue)
	for _, issue := range report.Issues {
		issuesByCategory[issue.Category] = append(issuesByCategory[issue.Category], issue)
	}

	for _, cat := range []model.Category{model.CategoryComplexity, model.CategorySize, model.CategoryCoupling, model.CategoryDuplication} {
		issues := issuesByCategory[cat]
		if len(issues) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s (%d issues)\n\n", strings.Title(string(cat)), len(issues)))

		for _, issue := range issues {
			sb.WriteString(fmt.Sprintf("#### %s `%s`\n\n", severityEmoji(issue.Severity), issue.EntityName))
			sb.WriteString(fmt.Sprintf("- **File:** `%s:%d-%d`\n", issue.FilePath, issue.StartLine, issue.EndLine))
			sb.WriteString(fmt.Sprintf("- **Type:** %s\n", issue.Subcategory))
			sb.WriteString(fmt.Sprintf("- **Severity:** %s\n", issue.Severity))
			sb.WriteString(fmt.Sprintf("- **Description:** %s\n", issue.Description))

			if g.cfg.IncludeSuggestions && issue.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("- **Suggestion:** %s\n", issue.Suggestion))
			}

			if g.cfg.IncludeMetrics && len(issue.Metrics) > 0 {
				sb.WriteString("- **Metrics:**\n")
				for k, v := range issue.Metrics {
					sb.WriteString(fmt.Sprintf("  - %s: %v\n", k, v))
				}
			}

			if g.cfg.IncludeCodeSnippets && issue.CodeSnippet != "" {
				sb.WriteString("\n**Code:**\n```\n")
				sb.WriteString(issue.CodeSnippet)
				sb.WriteString("\n```\n")
			}

			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

func (g *Generator) generateSARIF(report *model.AnalysisReport) (string, error) {
	sarif := map[string]any{
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]any{
			{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":           "quality-bot",
						"version":        "1.0.0",
						"informationUri": "https://github.com/example/quality-bot",
						"rules":          g.buildSARIFRules(report.Issues),
					},
				},
				"results": g.buildSARIFResults(report.Issues),
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (g *Generator) buildSARIFRules(issues []model.DebtIssue) []map[string]any {
	ruleMap := make(map[string]bool)
	var rules []map[string]any

	for _, issue := range issues {
		ruleID := string(issue.Category) + "/" + issue.Subcategory
		if ruleMap[ruleID] {
			continue
		}
		ruleMap[ruleID] = true

		rules = append(rules, map[string]any{
			"id":   ruleID,
			"name": issue.Subcategory,
			"shortDescription": map[string]any{
				"text": issue.Description,
			},
			"defaultConfiguration": map[string]any{
				"level": sarifLevel(issue.Severity),
			},
		})
	}

	return rules
}

func (g *Generator) buildSARIFResults(issues []model.DebtIssue) []map[string]any {
	var results []map[string]any

	for _, issue := range issues {
		result := map[string]any{
			"ruleId":  string(issue.Category) + "/" + issue.Subcategory,
			"level":   sarifLevel(issue.Severity),
			"message": map[string]any{"text": issue.Description},
			"locations": []map[string]any{
				{
					"physicalLocation": map[string]any{
						"artifactLocation": map[string]any{
							"uri": issue.FilePath,
						},
						"region": map[string]any{
							"startLine": issue.StartLine,
							"endLine":   issue.EndLine,
						},
					},
				},
			},
		}

		if issue.Suggestion != "" {
			result["fixes"] = []map[string]any{
				{
					"description": map[string]any{"text": issue.Suggestion},
				},
			}
		}

		results = append(results, result)
	}

	return results
}

func severityEmoji(s model.Severity) string {
	switch s {
	case model.SeverityCritical:
		return "[CRITICAL]"
	case model.SeverityHigh:
		return "[HIGH]"
	case model.SeverityMedium:
		return "[MEDIUM]"
	default:
		return "[LOW]"
	}
}

func sarifLevel(s model.Severity) string {
	switch s {
	case model.SeverityCritical, model.SeverityHigh:
		return "error"
	case model.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}
