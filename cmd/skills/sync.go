package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/agent"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/git"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull from Git, push local changes, and verify symlinks",
	Long: `Syncs the central skills repository with the remote:
  1. Fetches from remote
  2. Commits any local uncommitted changes
  3. Pulls with rebase
  4. Pushes to remote
  5. Verifies all agent symlinks are intact`,
	RunE: runSync,
}

func runSync(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !git.IsInstalled() {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	if !git.IsRepo(cfg.RepoPath) {
		return fmt.Errorf("repository not found at %s; run 'skills init' to set up", cfg.RepoPath)
	}

	hasRemote := git.HasRemote(cfg.RepoPath)
	verbose := os.Getenv("SKILLS_VERBOSE") != ""

	// Step 1: Fetch
	if hasRemote {
		if verbose {
			fmt.Println("Fetching from remote...")
		}
		if err := git.Fetch(cfg.RepoPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: fetch failed (offline?): %v\n", err)
			// Continue with local-only operations
			hasRemote = false
		}
	}

	// Step 2: Commit local changes
	changed, err := git.HasUncommittedChanges(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("checking for changes: %w", err)
	}
	if changed {
		msg := "sync: update skills on <hostname> at <timestamp>"
		if err := git.CommitAll(cfg.RepoPath, msg); err != nil {
			return fmt.Errorf("committing local changes: %w", err)
		}
		fmt.Println("Committed local changes.")
	}

	// Step 3: Pull --rebase
	if hasRemote {
		if verbose {
			fmt.Println("Pulling with rebase...")
		}
		if err := git.Pull(cfg.RepoPath); err != nil {
			// Conflict or pull failure
			git.AbortRebase(cfg.RepoPath)
			notifyConflict(cfg)
			writeLog(cfg.RepoPath, fmt.Sprintf("sync conflict: %v", err))
			return fmt.Errorf("pull failed (conflict?): %v\nManually resolve and run 'skills sync' again", err)
		}
	}

	// Step 4: Push
	if hasRemote {
		if verbose {
			fmt.Println("Pushing...")
		}
		if err := git.Push(cfg.RepoPath); err != nil {
			return fmt.Errorf("push failed: %w", err)
		}
	}

	// Step 5: Verify symlinks
	repoSkillsPath := skills.SkillsDir(cfg.RepoPath)
	brokenLinks := 0
	for _, agentID := range cfg.Agents {
		a, ok := agent.AgentByID(agentID)
		if !ok {
			continue
		}
		s := agent.GetStatus(a, repoSkillsPath)
		if s.Exists && s.IsSymlink && !s.SymlinkOK {
			// Broken symlink — try to repair
			fmt.Printf("Repairing broken symlink for %s...\n", a.Name)
			if err := agent.CreateSymlink(a.SkillsPath, repoSkillsPath); err != nil {
				fmt.Fprintf(os.Stderr, "  Failed to repair %s: %v\n", a.Name, err)
				brokenLinks++
			}
		} else if !s.Exists {
			// Missing symlink — re-create
			parent := repoSkillsPath // use first dir as existence check
			if _, err := os.Stat(parent); err == nil {
				if err := agent.CreateSymlink(a.SkillsPath, repoSkillsPath); err != nil {
					fmt.Fprintf(os.Stderr, "  Failed to create symlink for %s: %v\n", a.Name, err)
					brokenLinks++
				}
			}
		}
	}

	// Only print output if something happened (safe for cron quiet mode)
	if changed || brokenLinks > 0 || verbose {
		skillList, _ := skills.List(cfg.RepoPath)
		fmt.Printf("Sync complete: %d skill(s) in repository.\n", len(skillList))
	}
	return nil
}

func notifyConflict(cfg *config.Config) {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "Skills sync conflict — manual resolution needed. Run: skills sync")
		return
	}
	_ = cfg
	script := `display notification "Skills sync conflict — manual resolution needed" with title "Agent Skills Manager" subtitle "Run: skills sync"`
	exec.Command("osascript", "-e", script).Run() //nolint:errcheck
}

func writeLog(repoPath, msg string) {
	logPath := config.LogFilePath()
	if err := os.MkdirAll(logPath[:len(logPath)-len("/sync.log")], 0755); err != nil {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", msg)
}
