package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude-feats",
	Short: "RPG-style feats for your Claude Code sessions",
	Long: `claude-feats hooks into Claude Code's Stop event, analyzes your session
transcripts, and unlocks achievements stored in ~/.claude-feats/.

Run 'claude-feats hook install' to wire it up automatically.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(manaCmd)
	rootCmd.AddCommand(bioCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(hookCmd)
}
