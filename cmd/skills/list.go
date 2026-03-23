package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/config"
	"github.com/tholst/asm/internal/skills"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills in the central repository",
	RunE:  runList,
}

const nameColWidth = 30

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	skillList, err := skills.List(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	if len(skillList) == 0 {
		fmt.Println("No skills found. Use 'skills add <path>' to add a skill.")
		return nil
	}

	descWidth := terminalWidth() - nameColWidth - 1
	if descWidth < 40 {
		descWidth = 40
	}

	fmt.Printf("%-*s %s\n", nameColWidth, "Name", "Description")
	fmt.Printf("%-*s %s\n", nameColWidth, "----", "-----------")
	indent := strings.Repeat(" ", nameColWidth+1)
	for _, s := range skillList {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		lines := wrapWords(desc, descWidth)
		fmt.Printf("%-*s %s\n", nameColWidth, s.Name, lines[0])
		for _, line := range lines[1:] {
			fmt.Printf("%s%s\n", indent, line)
		}
	}
	fmt.Printf("\n%d skill(s) total\n", len(skillList))
	return nil
}

// terminalWidth returns the terminal width from $COLUMNS or defaults to 120.
func terminalWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		var w int
		if _, err := fmt.Sscanf(cols, "%d", &w); err == nil && w > 0 {
			return w
		}
	}
	return 120
}

// wrapWords splits text into lines of at most width characters, breaking on word boundaries.
func wrapWords(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= width {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	lines = append(lines, current)
	return lines
}
