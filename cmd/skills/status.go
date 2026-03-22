package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/agent"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/git"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status and agent link state",
	RunE:  runStatus,
}

func runStatus(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("=== Skills Status ===")
	fmt.Println()

	// Repository info
	fmt.Printf("Repository:  %s\n", cfg.RepoPath)
	if cfg.RemoteURL != "" {
		fmt.Printf("Remote:      %s\n", cfg.RemoteURL)
	}
	fmt.Println()

	if !git.IsRepo(cfg.RepoPath) {
		fmt.Println("Repository not found. Run 'skills init' to set up.")
		return nil
	}

	// Git status
	branch := git.BranchName(cfg.RepoPath)
	fmt.Printf("Branch:      %s\n", branch)

	if git.HasRemote(cfg.RepoPath) {
		ahead, behind := git.AheadBehind(cfg.RepoPath)
		switch {
		case ahead > 0 && behind > 0:
			fmt.Printf("Sync:        %d ahead, %d behind (diverged — run 'skills sync')\n", ahead, behind)
		case ahead > 0:
			fmt.Printf("Sync:        %d commit(s) to push\n", ahead)
		case behind > 0:
			fmt.Printf("Sync:        %d commit(s) to pull\n", behind)
		default:
			fmt.Println("Sync:        up to date")
		}
	} else {
		fmt.Println("Sync:        no remote configured (local only)")
	}

	changed, _ := git.HasUncommittedChanges(cfg.RepoPath)
	if changed {
		fmt.Println("Changes:     uncommitted local changes present")
		statusOut, _ := git.Status(cfg.RepoPath)
		if statusOut != "" {
			for _, line := range splitLines(statusOut) {
				fmt.Printf("  %s\n", line)
			}
		}
	} else {
		fmt.Println("Changes:     none")
	}
	fmt.Println()

	// Skills count
	skillList, err := skills.List(cfg.RepoPath)
	if err != nil {
		fmt.Printf("Skills:      error reading skills: %v\n", err)
	} else {
		fmt.Printf("Skills:      %d\n", len(skillList))
	}
	fmt.Println()

	// Agent symlink status
	fmt.Println("Agent Links:")
	repoSkillsPath := skills.SkillsDir(cfg.RepoPath)
	anyAgent := false
	for _, a := range agent.KnownAgents() {
		s := agent.GetStatus(a, repoSkillsPath)
		if !s.Exists && !s.IsSymlink {
			// Check if parent dir exists to determine if agent is installed
			continue
		}
		anyAgent = true
		switch {
		case s.IsSymlink && s.SymlinkOK && s.Target == repoSkillsPath:
			fmt.Printf("  ✓ %-14s linked (%d skill(s))\n", a.Name, s.SkillCount)
		case s.IsSymlink && s.SymlinkOK:
			fmt.Printf("  ~ %-14s symlink points to wrong target: %s\n", a.Name, s.Target)
		case s.IsSymlink && !s.SymlinkOK:
			fmt.Printf("  ✗ %-14s broken symlink -> %s\n", a.Name, s.Target)
		case s.Exists:
			fmt.Printf("  - %-14s real directory (%d skill(s)) — not managed (run 'skills link')\n", a.Name, s.SkillCount)
		default:
			fmt.Printf("  - %-14s not linked\n", a.Name)
		}
	}
	if !anyAgent {
		fmt.Println("  No agents linked. Run 'skills link' to create symlinks.")
	}

	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
