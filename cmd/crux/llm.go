package main

import (
	"github.com/ideamans/go-llm-cli-kit/llmcmd"
	"github.com/spf13/cobra"

	"github.com/ideamans/crux-cli/internal/llmdocs"
)

// llmConfig describes the `crux llm` subcommand.
func llmConfig() llmcmd.Config {
	return llmcmd.Config{Docs: llmdocs.Docs()}
}

func init() {
	llmcmd.AddTo(rootCmd, llmConfig())

	// Bare `crux` (no subcommand) prints standard help instead of erroring.
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}
}
