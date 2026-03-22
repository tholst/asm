package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// run executes a git command in dir and returns stdout.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// runInteractive runs git with output forwarded to the terminal.
func runInteractive(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsInstalled reports whether git is available in PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsRepo reports whether dir is inside a git repository.
func IsRepo(dir string) bool {
	_, err := run(dir, "rev-parse", "--git-dir")
	return err == nil
}

// Clone clones remoteURL to localPath, streaming output to the terminal.
func Clone(remoteURL, localPath string) error {
	cmd := exec.Command("git", "clone", remoteURL, localPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloning %s: %w", remoteURL, err)
	}
	return nil
}

// Init initializes a new git repository at dir.
func Init(dir string) error {
	_, err := run(dir, "init")
	return err
}

// HasRemote reports whether the repo has a remote named "origin".
func HasRemote(dir string) bool {
	out, err := run(dir, "remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "origin" {
			return true
		}
	}
	return false
}

// SetRemote sets or updates the origin remote URL.
func SetRemote(dir, url string) error {
	if HasRemote(dir) {
		_, err := run(dir, "remote", "set-url", "origin", url)
		return err
	}
	_, err := run(dir, "remote", "add", "origin", url)
	return err
}

// RemoteURL returns the URL of the origin remote, or "" if none.
func RemoteURL(dir string) string {
	out, _ := run(dir, "remote", "get-url", "origin")
	return out
}

// Fetch fetches from origin. No-op if no remote.
func Fetch(dir string) error {
	if !HasRemote(dir) {
		return nil
	}
	_, err := run(dir, "fetch", "origin")
	return err
}

// HasUncommittedChanges reports whether there are staged or unstaged changes.
func HasUncommittedChanges(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// CommitAll stages all changes and commits with the given message.
// Placeholders: <hostname>, <timestamp>.
// Returns nil without error if there is nothing to commit.
func CommitAll(dir, message string) error {
	if _, err := run(dir, "add", "-A"); err != nil {
		return err
	}
	// Check if anything was staged
	_, err := run(dir, "diff", "--cached", "--quiet")
	if err == nil {
		return nil // nothing staged, exit cleanly
	}
	hostname, _ := os.Hostname()
	msg := strings.ReplaceAll(message, "<hostname>", hostname)
	msg = strings.ReplaceAll(msg, "<timestamp>", time.Now().Format("2006-01-02T15:04:05"))
	_, err = run(dir, "commit", "-m", msg)
	return err
}

// Add stages a specific path.
func Add(dir, path string) error {
	_, err := run(dir, "add", path)
	return err
}

// Commit creates a commit with the given message (supports <hostname>, <timestamp>).
func Commit(dir, message string) error {
	hostname, _ := os.Hostname()
	msg := strings.ReplaceAll(message, "<hostname>", hostname)
	msg = strings.ReplaceAll(msg, "<timestamp>", time.Now().Format("2006-01-02T15:04:05"))
	_, err := run(dir, "commit", "-m", msg)
	return err
}

// Pull runs git pull --rebase from origin.
func Pull(dir string) error {
	if !HasRemote(dir) {
		return nil
	}
	return runInteractive(dir, "pull", "--rebase")
}

// AbortRebase aborts an in-progress rebase (best-effort).
func AbortRebase(dir string) {
	run(dir, "rebase", "--abort") //nolint:errcheck
}

// Push pushes to origin. No-op if no remote.
func Push(dir string) error {
	if !HasRemote(dir) {
		return nil
	}
	return runInteractive(dir, "push")
}

// Status returns the short git status output.
func Status(dir string) (string, error) {
	return run(dir, "status", "--short")
}

// Log returns recent log entries (oneline format).
func Log(dir string, n int) (string, error) {
	return run(dir, "log", fmt.Sprintf("--max-count=%d", n), "--oneline")
}

// BranchName returns the current branch name.
func BranchName(dir string) string {
	out, _ := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	return out
}

// AheadBehind returns how many commits the local branch is ahead/behind origin.
// Returns (0, 0) if no remote or on error.
func AheadBehind(dir string) (ahead, behind int) {
	out, err := run(dir, "rev-list", "--count", "--left-right", "HEAD...@{u}")
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(out)
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &ahead)
		fmt.Sscanf(parts[1], "%d", &behind)
	}
	return ahead, behind
}
