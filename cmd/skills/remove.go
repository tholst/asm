package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/git"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a skill from the central repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func runRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !skills.Exists(cfg.RepoPath, name) {
		return fmt.Errorf("skill '%s' not found in repository", name)
	}

	if !promptYN(fmt.Sprintf("Remove skill '%s'? This cannot be undone (Git history preserved).", name), false) {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := skills.Remove(cfg.RepoPath, name); err != nil {
		return err
	}

	msg := fmt.Sprintf("feat: remove skill '%s'", name)
	if err := git.CommitAll(cfg.RepoPath, msg); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	fmt.Printf("Removed skill '%s'. Run 'skills sync' to push to remote.\n", name)
	return nil
}
