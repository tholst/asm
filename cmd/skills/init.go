package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/agent"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/git"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "First-time setup on this machine",
	Long:  `Interactive wizard that sets up the central skills repository and links all detected agents.`,
	RunE:  runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	fmt.Println("=== Agent Skills Manager — Setup ===")
	fmt.Println()

	// Check git is installed
	if !git.IsInstalled() {
		return fmt.Errorf("git is not installed or not in PATH; please install git and try again")
	}

	// Step 1: Detect existing agent skill folders
	fmt.Println("Scanning for installed agents...")
	knownAgents := agent.KnownAgents()
	type agentInfo struct {
		a      agent.Agent
		status agent.Status
	}
	var detected []agentInfo
	for _, a := range knownAgents {
		s := agent.GetStatus(a, "")
		parent := filepath.Dir(a.SkillsPath)
		if _, err := os.Stat(parent); err == nil {
			detected = append(detected, agentInfo{a, s})
			managed := ""
			if s.IsSymlink {
				managed = " (already managed)"
			}
			fmt.Printf("  ✓ %-14s %s%s\n", a.Name, a.SkillsPath, managed)
		}
	}
	if len(detected) == 0 {
		fmt.Println("  No supported agents detected. Continuing anyway...")
	}
	fmt.Println()

	// Step 2: Ask for repository source
	choice := promptChoice(
		"How would you like to set up the central skills repository?",
		[]string{
			"Use an existing local Git repository",
			"Clone from a remote Git repository (GitHub, etc.)",
			"Create a new repository",
		},
	)

	repoPath := ""
	remoteURL := ""

	switch choice {
	case 0: // existing local repo
		repoPath = promptDefault("Path to existing repository", config.DefaultRepoPath())
		repoPath = config.ExpandHome(repoPath)
		if !git.IsRepo(repoPath) {
			return fmt.Errorf("%s is not a git repository", repoPath)
		}
		remoteURL = git.RemoteURL(repoPath)
		fmt.Printf("Using existing repository at %s\n", repoPath)

	case 1: // clone from remote
		remoteURL = prompt("Remote repository URL (e.g. git@github.com:you/skills.git): ")
		if remoteURL == "" {
			return fmt.Errorf("remote URL cannot be empty")
		}
		repoPath = promptDefault("Clone to", config.DefaultRepoPath())
		repoPath = config.ExpandHome(repoPath)
		if _, err := os.Stat(repoPath); err == nil {
			if git.IsRepo(repoPath) {
				fmt.Printf("Repository already exists at %s, skipping clone.\n", repoPath)
			} else {
				return fmt.Errorf("%s already exists and is not a git repository", repoPath)
			}
		} else {
			fmt.Printf("Cloning %s -> %s\n", remoteURL, repoPath)
			if err := git.Clone(remoteURL, repoPath); err != nil {
				return err
			}
		}

	case 2: // create new repo
		repoPath = promptDefault("Create repository at", config.DefaultRepoPath())
		repoPath = config.ExpandHome(repoPath)
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return fmt.Errorf("creating repo dir: %w", err)
		}
		if !git.IsRepo(repoPath) {
			if err := git.Init(repoPath); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
		}
		if err := scaffoldRepo(repoPath); err != nil {
			return err
		}
		fmt.Printf("Created new repository at %s\n", repoPath)
		fmt.Println()
		fmt.Println("To push to GitHub, run:")
		fmt.Printf("  git -C %s remote add origin git@github.com:YOU/skills.git\n", repoPath)
		fmt.Printf("  git -C %s push -u origin main\n", repoPath)
	}

	// Ensure skills directory exists
	if err := skills.EnsureSkillsDir(repoPath); err != nil {
		return fmt.Errorf("ensuring skills dir: %w", err)
	}
	repoSkillsPath := skills.SkillsDir(repoPath)
	fmt.Println()

	// Step 3: Handle existing skills for each detected agent
	var agentIDs []string
	for _, info := range detected {
		a := info.a
		s := info.status
		agentIDs = append(agentIDs, a.ID)

		if s.IsSymlink {
			continue // already managed
		}
		if s.Exists && s.SkillCount > 0 {
			fmt.Printf("%s has %d existing skill(s) at %s\n", a.Name, s.SkillCount, a.SkillsPath)
			if promptYN("Migrate these skills into the central repository?", true) {
				migrated, err := skills.MigrateFrom(repoPath, a.SkillsPath)
				if err != nil {
					fmt.Printf("Warning: migration error: %v\n", err)
				}
				if len(migrated) > 0 {
					hostname, _ := os.Hostname()
					msg := fmt.Sprintf("feat: migrate skills from %s on %s", a.ID, hostname)
					if err := git.CommitAll(repoPath, msg); err != nil {
						fmt.Printf("Warning: commit failed: %v\n", err)
					} else {
						fmt.Printf("  Migrated %d skill(s) and committed.\n", len(migrated))
					}
				}
			} else {
				fmt.Println("  Existing skills will be backed up when the symlink is created.")
			}
		}
	}
	fmt.Println()

	// Step 4: Create symlinks
	fmt.Println("Creating symlinks...")
	for _, info := range detected {
		a := info.a
		fmt.Printf("  Linking %-14s ", a.Name+"...")
		if err := agent.CreateSymlink(a.SkillsPath, repoSkillsPath); err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Println("done")
		}
	}

	// Step 5: Write config
	cfg := &config.Config{
		RepoPath:    repoPath,
		RemoteURL:   remoteURL,
		Agents:      agentIDs,
		SyncOnStart: false,
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("\nConfiguration saved to %s\n", config.ConfigFilePath())

	// Step 6: Offer cron setup (macOS / Linux)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		fmt.Println()
		if promptYN("Set up automatic sync every 30 minutes (via cron)?", true) {
			if err := installCron(cfg.EffectiveCronInterval()); err != nil {
				fmt.Printf("Warning: could not install cron entry: %v\n", err)
				fmt.Println("You can set it up later with: skills cron enable")
			} else {
				fmt.Println("Cron entry installed. Manage with: skills cron status")
			}
		}
	}

	fmt.Println()
	fmt.Println("Setup complete! Run 'skills status' to verify everything is working.")
	return nil
}

func scaffoldRepo(repoPath string) error {
	if err := skills.EnsureSkillsDir(repoPath); err != nil {
		return fmt.Errorf("creating skills dir: %w", err)
	}
	readme := "# Skills Repository\n\nCentral repository for AI agent skills managed by [Agent Skills Manager](https://github.com/tholst/asm).\n"
	readmePath := filepath.Join(repoPath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return fmt.Errorf("writing README: %w", err)
		}
	}
	if err := ensureGitignore(repoPath); err != nil {
		return err
	}
	if err := git.CommitAll(repoPath, "feat: initialize skills repository"); err != nil {
		fmt.Printf("Warning: initial commit failed (repo may be empty): %v\n", err)
	}
	return nil
}

