package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/git"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a local skill folder to the central repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func runAdd(_ *cobra.Command, args []string) error {
	sourcePath := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Resolve absolute path
	absSource, err := absolutePath(sourcePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absSource); err != nil {
		return fmt.Errorf("path not found: %s", absSource)
	}

	skillName, err := skills.Add(cfg.RepoPath, absSource)
	if err != nil {
		return err
	}

	// Commit the addition
	msg := fmt.Sprintf("feat: add skill '%s'", skillName)
	if err := git.CommitAll(cfg.RepoPath, msg); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	fmt.Printf("Added skill '%s' to repository.\n", skillName)
	fmt.Println("Run 'skills sync' to push to remote.")
	return nil
}

func absolutePath(p string) (string, error) {
	if len(p) > 0 && p[0] == '~' {
		return config.ExpandHome(p), nil
	}
	return p, nil
}
