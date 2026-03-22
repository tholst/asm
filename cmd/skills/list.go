package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills in the central repository",
	RunE:  runList,
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	skillList, err := skills.List(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	if len(skillList) == 0 {
		fmt.Println("No skills found. Use 'skills add <path>' to add a skill.")
		return nil
	}

	fmt.Printf("%-30s %s\n", "Name", "Description")
	fmt.Printf("%-30s %s\n", "----", "-----------")
	for _, s := range skillList {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Printf("%-30s %s\n", s.Name, desc)
	}
	fmt.Printf("\n%d skill(s) total\n", len(skillList))
	return nil
}
