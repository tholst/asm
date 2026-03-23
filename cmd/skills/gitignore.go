package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// requiredGitignoreEntries lists patterns that should be in every skills repo .gitignore.
var requiredGitignoreEntries = []string{
	".DS_Store",
	".system/",
}

// ensureGitignore creates .gitignore if missing, or appends any missing required entries.
func ensureGitignore(repoPath string) error {
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	content := string(existing)
	var toAdd []string
	for _, entry := range requiredGitignoreEntries {
		if !containsLine(content, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	// Ensure existing content ends with a newline before appending
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += strings.Join(toAdd, "\n") + "\n"

	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}
	return nil
}

// containsLine checks whether s contains the given line (exact match on a whole line).
func containsLine(s, line string) bool {
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}
