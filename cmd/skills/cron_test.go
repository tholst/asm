package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectCronLineHealthy(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	binary := makeExecutableFile(t, filepath.Join(t.TempDir(), "skills"))
	restore := stubResolveSkillsBinaryPath(binary)
	defer restore()

	line := "*/30 * * * * " + binary + " sync >> " + filepath.Join(t.TempDir(), "sync.log") + " 2>&1"
	health := inspectCronLine(line)

	if !health.Healthy {
		t.Fatalf("expected cron line to be healthy, got %+v", health)
	}
	if health.RepairRecommended {
		t.Fatalf("expected no repair recommendation, got %+v", health)
	}
	if len(health.Problems) != 0 {
		t.Fatalf("expected no problems, got %v", health.Problems)
	}
}

func TestInspectCronLineMissingBinaryNeedsRepair(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	missingBinary := filepath.Join(t.TempDir(), "missing-skills")
	restore := stubResolveSkillsBinaryPath(missingBinary)
	defer restore()

	line := "*/30 * * * * " + missingBinary + " sync >> " + filepath.Join(t.TempDir(), "sync.log") + " 2>&1"
	health := inspectCronLine(line)

	if health.Healthy {
		t.Fatalf("expected unhealthy cron line, got %+v", health)
	}
	if !containsSubstring(health.Problems, "binary missing") {
		t.Fatalf("expected missing binary problem, got %v", health.Problems)
	}
}

func TestInspectCronLineDifferentExistingBinaryRecommendsRewrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	installedBinary := makeExecutableFile(t, filepath.Join(t.TempDir(), "installed", "skills"))
	expectedBinary := makeExecutableFile(t, filepath.Join(t.TempDir(), "expected", "skills"))
	restore := stubResolveSkillsBinaryPath(expectedBinary)
	defer restore()

	line := "*/30 * * * * " + installedBinary + " sync >> " + filepath.Join(t.TempDir(), "sync.log") + " 2>&1"
	health := inspectCronLine(line)

	if !health.Healthy {
		t.Fatalf("expected healthy cron line with rewrite recommendation, got %+v", health)
	}
	if !health.RepairRecommended {
		t.Fatalf("expected rewrite recommendation, got %+v", health)
	}
	if !containsSubstring(health.Notes, "entry points to") {
		t.Fatalf("expected rewrite note, got %v", health.Notes)
	}
}

func TestIsEphemeralGoRunBinary(t *testing.T) {
	t.Run("temp exe path", func(t *testing.T) {
		path := filepath.Join(string(os.PathSeparator), "var", "folders", "tmp", "go-build123", "b001", "exe", "skills")
		if !isEphemeralGoRunBinary(path) {
			t.Fatalf("expected %q to be treated as ephemeral", path)
		}
	})

	t.Run("go build cache path", func(t *testing.T) {
		path := filepath.Join(string(os.PathSeparator), "Users", "th", "Library", "Caches", "go-build", "83", "skills")
		if !isEphemeralGoRunBinary(path) {
			t.Fatalf("expected %q to be treated as ephemeral", path)
		}
	})

	t.Run("stable install path", func(t *testing.T) {
		path := filepath.Join(string(os.PathSeparator), "Users", "th", "go", "bin", "skills")
		if isEphemeralGoRunBinary(path) {
			t.Fatalf("expected %q to be treated as stable", path)
		}
	})
}

func TestFindStableSkillsBinaryOnPathSkipsEphemeralEntries(t *testing.T) {
	root := t.TempDir()
	ephemeralDir := filepath.Join(root, "go-build-cache", "bin")
	stableDir := filepath.Join(root, "stable", "bin")

	makeExecutableFile(t, filepath.Join(ephemeralDir, "skills"))
	stableBinary := makeExecutableFile(t, filepath.Join(stableDir, "skills"))

	t.Setenv("PATH", strings.Join([]string{ephemeralDir, stableDir}, string(os.PathListSeparator)))

	got := findStableSkillsBinaryOnPath()
	if got != stableBinary {
		t.Fatalf("expected stable binary %q, got %q", stableBinary, got)
	}
}

func stubResolveSkillsBinaryPath(path string) func() {
	previous := resolveSkillsBinaryPath
	resolveSkillsBinaryPath = func() string { return path }
	return func() {
		resolveSkillsBinaryPath = previous
	}
}

func makeExecutableFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
