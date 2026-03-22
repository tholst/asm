package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tholst/asm/internal/agent"
)

// Skill represents a skill in the central repository.
type Skill struct {
	Name        string
	Description string
	Path        string // absolute path to the skill directory
}

// SkillsDir returns the path to the skills directory within the repo.
func SkillsDir(repoPath string) string {
	return filepath.Join(repoPath, "skills")
}

// EnsureSkillsDir creates the skills directory if it doesn't exist.
func EnsureSkillsDir(repoPath string) error {
	return os.MkdirAll(SkillsDir(repoPath), 0755)
}

// List returns all skills found in the repository.
func List(repoPath string) ([]Skill, error) {
	skillsDir := SkillsDir(repoPath)
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills dir: %w", err)
	}
	var result []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, e.Name())
		skillMD := filepath.Join(skillPath, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			continue // not a valid skill directory
		}
		result = append(result, Skill{
			Name:        e.Name(),
			Path:        skillPath,
			Description: readDescription(skillMD),
		})
	}
	return result, nil
}

// Add copies a skill directory into the central repository.
// Returns the skill name on success.
func Add(repoPath, sourcePath string) (string, error) {
	skillMD := filepath.Join(sourcePath, "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		return "", fmt.Errorf("no SKILL.md found in %s", sourcePath)
	}
	skillName := filepath.Base(sourcePath)
	dest := filepath.Join(SkillsDir(repoPath), skillName)
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("skill '%s' already exists in repository", skillName)
	}
	if err := agent.CopyDir(sourcePath, dest); err != nil {
		return "", fmt.Errorf("copying skill: %w", err)
	}
	return skillName, nil
}

// Remove deletes a skill from the central repository.
func Remove(repoPath, name string) error {
	skillPath := filepath.Join(SkillsDir(repoPath), name)
	if _, err := os.Stat(skillPath); err != nil {
		return fmt.Errorf("skill '%s' not found in repository", name)
	}
	return os.RemoveAll(skillPath)
}

// Exists reports whether a skill with the given name exists.
func Exists(repoPath, name string) bool {
	_, err := os.Stat(filepath.Join(SkillsDir(repoPath), name))
	return err == nil
}

// MigrateFrom copies all valid skills from srcPath into the repo's skills dir.
// Returns the list of skill names that were migrated.
func MigrateFrom(repoPath, srcPath string) ([]string, error) {
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", srcPath, err)
	}
	var migrated []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillSrc := filepath.Join(srcPath, e.Name())
		if _, err := os.Stat(filepath.Join(skillSrc, "SKILL.md")); err != nil {
			continue // not a valid skill
		}
		dest := filepath.Join(SkillsDir(repoPath), e.Name())
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("  Skipping '%s' (already exists in repo)\n", e.Name())
			continue
		}
		if err := agent.CopyDir(skillSrc, dest); err != nil {
			return migrated, fmt.Errorf("copying '%s': %w", e.Name(), err)
		}
		migrated = append(migrated, e.Name())
	}
	return migrated, nil
}

// readDescription parses the description field from a SKILL.md YAML frontmatter.
func readDescription(skillMDPath string) string {
	f, err := os.Open(skillMDPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // end of frontmatter
		}
		if inFrontmatter && strings.HasPrefix(line, "description:") {
			val := strings.TrimPrefix(line, "description:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return ""
}
