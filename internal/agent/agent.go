package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Agent represents a supported coding agent.
type Agent struct {
	ID         string // e.g. "claude-code"
	Name       string // e.g. "Claude Code"
	SkillsPath string // absolute path to global skills directory
}

// KnownAgents returns all supported agents with expanded paths.
func KnownAgents() []Agent {
	home, _ := os.UserHomeDir()
	return []Agent{
		{
			ID:         "claude-code",
			Name:       "Claude Code",
			SkillsPath: filepath.Join(home, ".claude", "skills"),
		},
		{
			ID:         "cursor",
			Name:       "Cursor",
			SkillsPath: filepath.Join(home, ".cursor", "skills"),
		},
		{
			ID:         "codex",
			Name:       "Codex",
			SkillsPath: filepath.Join(home, ".codex", "skills"),
		},
	}
}

// AgentByID returns the agent with the given ID.
func AgentByID(id string) (Agent, bool) {
	for _, a := range KnownAgents() {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

// DetectedAgents returns only agents whose parent directories exist.
func DetectedAgents() []Agent {
	var detected []Agent
	for _, a := range KnownAgents() {
		parent := filepath.Dir(a.SkillsPath)
		if _, err := os.Stat(parent); err == nil {
			detected = append(detected, a)
		}
	}
	return detected
}

// Status describes the current state of an agent's skills folder.
type Status struct {
	Agent     Agent
	Exists    bool
	IsSymlink bool
	SymlinkOK bool   // symlink target exists and is reachable
	Target    string // if symlink, where it points
	SkillCount int
}

// GetStatus returns the symlink/folder status for an agent.
func GetStatus(a Agent, repoSkillsPath string) Status {
	s := Status{Agent: a}
	fi, err := os.Lstat(a.SkillsPath)
	if err != nil {
		return s // path doesn't exist
	}
	s.Exists = true

	if fi.Mode()&os.ModeSymlink != 0 {
		s.IsSymlink = true
		target, err := os.Readlink(a.SkillsPath)
		if err == nil {
			s.Target = target
			if _, err := os.Stat(a.SkillsPath); err == nil {
				s.SymlinkOK = true
			}
		}
	}

	// Count skills (subdirs containing SKILL.md)
	if s.SymlinkOK || (s.Exists && !s.IsSymlink) {
		entries, err := os.ReadDir(a.SkillsPath)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					if _, err := os.Stat(filepath.Join(a.SkillsPath, e.Name(), "SKILL.md")); err == nil {
						s.SkillCount++
					}
				}
			}
		}
	}
	return s
}

// CreateSymlink creates a symlink from agentSkillsPath -> repoSkillsPath.
// Backs up any existing non-symlink directory to <path>.backup.
func CreateSymlink(agentSkillsPath, repoSkillsPath string) error {
	fi, err := os.Lstat(agentSkillsPath)
	if err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			// Already a symlink — check if it points to the right place
			target, err := os.Readlink(agentSkillsPath)
			if err != nil {
				return fmt.Errorf("reading symlink: %w", err)
			}
			if target == repoSkillsPath {
				return nil // already correct, nothing to do
			}
			// Wrong target — remove and re-create
			if err := os.Remove(agentSkillsPath); err != nil {
				return fmt.Errorf("removing old symlink: %w", err)
			}
		} else {
			// Real directory — back it up before replacing
			backup := agentSkillsPath + ".backup"
			if err := os.Rename(agentSkillsPath, backup); err != nil {
				return fmt.Errorf("backing up %s to %s: %w", agentSkillsPath, backup, err)
			}
			fmt.Printf("  Backed up %s -> %s\n", agentSkillsPath, backup)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", agentSkillsPath, err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(agentSkillsPath), 0755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	if err := os.Symlink(repoSkillsPath, agentSkillsPath); err != nil {
		return fmt.Errorf("creating symlink %s -> %s: %w", agentSkillsPath, repoSkillsPath, err)
	}
	return nil
}

// RemoveSymlink removes the symlink at agentSkillsPath and restores a backup if present.
func RemoveSymlink(agentSkillsPath string) error {
	fi, err := os.Lstat(agentSkillsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to do
		}
		return fmt.Errorf("stat %s: %w", agentSkillsPath, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink (not managed by skills)", agentSkillsPath)
	}
	if err := os.Remove(agentSkillsPath); err != nil {
		return fmt.Errorf("removing symlink: %w", err)
	}
	// Restore backup if one exists
	backup := agentSkillsPath + ".backup"
	if _, err := os.Stat(backup); err == nil {
		if err := os.Rename(backup, agentSkillsPath); err != nil {
			return fmt.Errorf("restoring backup from %s: %w", backup, err)
		}
		fmt.Printf("  Restored %s from backup\n", agentSkillsPath)
	}
	return nil
}

// CopyDir recursively copies src to dst.
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
