package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "skills",
	Short: "Agent Skills Manager — sync AI agent skills across machines",
	Long: `skills manages a centralized Git repository of AI agent skills,
keeping them in sync across machines and coding agents (Claude Code, Cursor, Codex).

Get started:
  skills init     First-time setup on this machine
  skills sync     Pull latest skills and push local changes
  skills status   Show sync status and agent link state
  skills list     List all skills in the central repository
  skills cron     Manage automatic sync schedule`,
}

func main() {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
