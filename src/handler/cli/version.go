package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (h *Handler) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("quality-bot %s\n", h.cfg.Agent.Version)
		},
	}
}

func (h *Handler) detectorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detectors",
		Short: "List available detectors",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available detectors:")
			fmt.Println("  - complexity     : Cyclomatic complexity and nesting depth")
			fmt.Println("  - size_structure : Long methods, large classes/files, parameter lists")
			fmt.Println("  - coupling       : Feature envy, inappropriate intimacy, dependencies")
			fmt.Println("  - duplication    : Similar code detection")
			fmt.Println("")
			fmt.Println("Planned (future):")
			fmt.Println("  - dead_code      : Unused functions and classes")
		},
	}
}
