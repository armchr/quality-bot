package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"quality-bot/src/config"
	"quality-bot/src/util"
)

// Handler handles CLI commands
type Handler struct {
	cfg        *config.Config
	configPath string
	rootCmd    *cobra.Command
}

// New creates a new CLI handler
func New() *Handler {
	h := &Handler{}
	h.setupCommands()
	return h
}

func (h *Handler) setupCommands() {
	h.rootCmd = &cobra.Command{
		Use:   "quality-bot",
		Short: "Technical debt detection agent",
		Long:  "Analyzes codebases to detect technical debt using CodeAPI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return h.loadConfig()
		},
	}

	// Global flags
	h.rootCmd.PersistentFlags().StringVarP(&h.configPath, "config", "c", "",
		"Path to configuration file")

	// Add subcommands
	h.rootCmd.AddCommand(h.analyzeCmd())
	h.rootCmd.AddCommand(h.versionCmd())
	h.rootCmd.AddCommand(h.detectorsCmd())
}

func (h *Handler) loadConfig() error {
	loader := config.NewLoader()
	cfg, err := loader.Load(h.configPath)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	h.cfg = cfg

	// Initialize logger from config
	util.SetDefaultLogger(cfg.Logging)
	util.Debug("Configuration loaded successfully")
	util.Debug("Log level set to: %s", cfg.Logging.Level)

	return nil
}

// Execute runs the CLI
func (h *Handler) Execute() error {
	return h.rootCmd.Execute()
}

// Run is the main entry point
func Run() {
	handler := New()
	if err := handler.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
