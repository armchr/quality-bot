package model

import "time"

// Severity represents the severity level of a debt issue
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Category represents the category of technical debt
type Category string

const (
	CategoryComplexity  Category = "complexity"
	CategorySize        Category = "size"
	CategoryCoupling    Category = "coupling"
	CategoryDuplication Category = "duplication"
	CategoryDeadCode    Category = "dead_code"
)

// DebtIssue represents a single detected technical debt issue
type DebtIssue struct {
	Category    Category       `json:"category"`
	Subcategory string         `json:"subcategory"`
	Severity    Severity       `json:"severity"`
	FilePath    string         `json:"file_path"`
	StartLine   int            `json:"start_line"`
	EndLine     int            `json:"end_line"`
	EntityName  string         `json:"entity_name"`
	EntityType  string         `json:"entity_type"` // function, class, file
	Description string         `json:"description"`
	Metrics     map[string]any `json:"metrics"`
	Suggestion  string         `json:"suggestion"`
	CodeSnippet string         `json:"code_snippet,omitempty"` // Optional: actual code
}

// AnalysisReport represents the complete analysis output
type AnalysisReport struct {
	RepoName    string        `json:"repo_name"`
	GeneratedAt time.Time     `json:"generated_at"`
	Summary     ReportSummary `json:"summary"`
	Issues      []DebtIssue   `json:"issues"`
}

// ReportSummary contains aggregated statistics
type ReportSummary struct {
	TotalIssues  int              `json:"total_issues"`
	ByCategory   map[Category]int `json:"by_category"`
	BySeverity   map[Severity]int `json:"by_severity"`
	HotspotFiles []FileHotspot    `json:"hotspot_files"`
	DebtScore    float64          `json:"debt_score"`
}

// FileHotspot represents a file with many issues
type FileHotspot struct {
	FilePath   string `json:"file_path"`
	IssueCount int    `json:"issue_count"`
}
