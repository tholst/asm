package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// prompt displays a question and returns the trimmed user input.
func prompt(question string) string {
	fmt.Print(question)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptDefault displays a question with a default value and returns input or default.
func promptDefault(question, defaultVal string) string {
	fmt.Printf("%s [%s]: ", question, defaultVal)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

// promptYN asks a yes/no question and returns true for yes.
func promptYN(question string, defaultYes bool) bool {
	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}
	fmt.Print(question + suffix)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

// promptChoice presents a numbered menu and returns the selected index (0-based).
func promptChoice(question string, choices []string) int {
	fmt.Println(question)
	for i, c := range choices {
		fmt.Printf("  %d) %s\n", i+1, c)
	}
	for {
		fmt.Printf("Choose [1-%d]: ", len(choices))
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		var n int
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil {
			if n >= 1 && n <= len(choices) {
				return n - 1
			}
		}
		fmt.Printf("Please enter a number between 1 and %d.\n", len(choices))
	}
}
