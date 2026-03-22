package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/agent"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "(Re)create symlinks for all detected agents",
	Long:  `Creates or repairs symlinks from each agent's global skills folder to the central repository.`,
	RunE:  runLink,
}

func runLink(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	repoSkillsPath := skills.SkillsDir(cfg.RepoPath)
	agents := agent.KnownAgents()
	linked := 0

	for _, a := range agents {
		// Only link agents that are in config or whose parent dir exists
		inConfig := false
		for _, id := range cfg.Agents {
			if id == a.ID {
				inConfig = true
				break
			}
		}
		s := agent.GetStatus(a, repoSkillsPath)
		if !inConfig && !s.Exists {
			continue // agent not installed, skip
		}

		fmt.Printf("  Linking %-14s ", a.Name+"...")
		if err := agent.CreateSymlink(a.SkillsPath, repoSkillsPath); err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Println("done")
			linked++
			// Ensure agent is in config
			if !inConfig {
				cfg.Agents = append(cfg.Agents, a.ID)
			}
		}
	}

	if linked > 0 {
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	if linked == 0 {
		fmt.Println("No agents to link. Ensure agent apps are installed.")
	} else {
		fmt.Printf("\nLinked %d agent(s).\n", linked)
	}
	return nil
}
