package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"quality-bot/src/controller"
	"quality-bot/src/util"
)

func (h *Handler) analyzeCmd() *cobra.Command {
	var (
		repoName   string
		outputFile string
		format     string
		timeout    time.Duration
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze a repository for technical debt",
		Long:  "Runs all enabled detectors against a repository and generates a report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoName == "" {
				return fmt.Errorf("--repo is required")
			}

			util.Info("Analyzing repository: %s (timeout: %v)", repoName, timeout)

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// Run analysis
			analysisCtrl := controller.NewAnalysisController(h.cfg)
			report, err := analysisCtrl.Analyze(ctx, controller.AnalyzeRequest{
				RepoName: repoName,
			})
			if err != nil {
				util.Error("Analysis failed: %v", err)
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Output results
			if outputFile != "" {
				// Set output directory from flag
				h.cfg.Output.OutputDir = outputFile
				if format != "" {
					h.cfg.Output.Formats = []string{format}
				}

				// Generate report files
				reportCtrl := controller.NewReportController(h.cfg)
				paths, err := reportCtrl.GenerateReports(report)
				if err != nil {
					return fmt.Errorf("generating reports: %w", err)
				}
				for _, path := range paths {
					fmt.Printf("Report written to %s\n", path)
				}
			} else {
				// Output to stdout
				reportCtrl := controller.NewReportController(h.cfg)
				outputFormat := format
				if outputFormat == "" {
					outputFormat = "json"
				}

				output, err := reportCtrl.GenerateToString(report, outputFormat)
				if err != nil {
					// Fallback to raw JSON
					data, _ := json.MarshalIndent(report, "", "  ")
					fmt.Println(string(data))
				} else {
					fmt.Println(output)
				}
			}

			// Print summary to stderr
			fmt.Fprintf(os.Stderr, "\nAnalysis complete:\n")
			fmt.Fprintf(os.Stderr, "  Total issues: %d\n", report.Summary.TotalIssues)
			fmt.Fprintf(os.Stderr, "  Debt score: %.1f/100\n", report.Summary.DebtScore)

			return nil
		},
	}

	cmd.Flags().StringVarP(&repoName, "repo", "r", "", "Repository name (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output directory path")
	cmd.Flags().StringVarP(&format, "format", "f", "", "Output format (json, markdown, sarif)")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Minute, "Analysis timeout")

	cmd.MarkFlagRequired("repo")

	return cmd
}
