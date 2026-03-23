package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCronInterval is the default sync interval in minutes.
const DefaultCronInterval = 30

// Config holds the settings for the skills manager.
type Config struct {
	RepoPath     string   `json:"repo_path"`
	RemoteURL    string   `json:"remote_url"`
	Agents       []string `json:"agents"`
	SyncOnStart  bool     `json:"sync_on_start"`
	CronInterval int      `json:"cron_interval,omitempty"`
}

// EffectiveCronInterval returns the configured interval or the default.
func (c *Config) EffectiveCronInterval() int {
	if c.CronInterval <= 0 {
		return DefaultCronInterval
	}
	return c.CronInterval
}

// DefaultRepoPath returns the default local repo path (~/.skills-repo).
func DefaultRepoPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".skills-repo")
}

// ConfigFilePath returns the path to the config file.
func ConfigFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "skills", "config.json")
}

// LogFilePath returns the path to the sync log file.
func LogFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "skills", "sync.log")
}

// Load reads and returns the config from disk.
func Load() (*Config, error) {
	path := ConfigFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not initialized: run 'skills init' first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.RepoPath = ExpandHome(cfg.RepoPath)
	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path := ConfigFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	// Store with ~ for portability across machines
	stored := *cfg
	stored.RepoPath = CollapseHome(cfg.RepoPath)
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// ExpandHome expands a leading ~ in a path.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~/") && path != "~" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// CollapseHome replaces the home directory prefix with ~.
func CollapseHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	if path == home {
		return "~"
	}
	return path
}
