package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/agent"
	"github.com/tholst/asm/internal/config"
)

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Remove symlinks and restore original skill folders",
	Long:  `Removes the symlinks created by 'skills link' and restores any backed-up folders.`,
	RunE:  runUnlink,
}

func runUnlink(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("This will remove symlinks and restore original skill folders (if backups exist).")
	if !promptYN("Continue?", false) {
		fmt.Println("Cancelled.")
		return nil
	}

	unlinked := 0
	for _, agentID := range cfg.Agents {
		a, ok := agent.AgentByID(agentID)
		if !ok {
			continue
		}
		fmt.Printf("  Unlinking %-14s ", a.Name+"...")
		if err := agent.RemoveSymlink(a.SkillsPath); err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Println("done")
			unlinked++
		}
	}

	if unlinked > 0 {
		fmt.Printf("\nUnlinked %d agent(s). Run 'skills link' to re-create symlinks.\n", unlinked)
	} else {
		fmt.Println("No symlinks to remove.")
	}
	return nil
}
