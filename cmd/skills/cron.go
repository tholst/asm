package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tholst/asm/internal/config"
)

func init() {
	rootCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronStatusCmd)
	cronCmd.AddCommand(cronDoctorCmd)
	cronCmd.AddCommand(cronEnableCmd)
	cronCmd.AddCommand(cronDisableCmd)
	cronCmd.AddCommand(cronIntervalCmd)
	cronCmd.AddCommand(cronLogsCmd)
	cronLogsCmd.Flags().IntP("lines", "n", 20, "Number of log lines to show")
	cronLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output (tail -f)")
	cronDoctorCmd.Flags().Bool("fix", false, "Rewrite the cron entry to use the current skills binary path")
}

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage automatic sync schedule (cron)",
	Long:  `View, enable, disable, or configure the cron job that runs 'skills sync' automatically.`,
}

// --- status ---

var cronStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current cron job status",
	RunE:  runCronStatus,
}

func runCronStatus(_ *cobra.Command, _ []string) error {
	if err := requireCronPlatform(); err != nil {
		return err
	}

	crontab, _ := getCrontab()
	line, found := findSkillsCronLine(crontab)

	if !found {
		fmt.Println("Cron:      not installed")
		fmt.Println()
		fmt.Println("Enable with: skills cron enable")
		return nil
	}

	interval := parseIntervalFromCronLine(line)
	fmt.Printf("Cron:      enabled (every %d minutes)\n", interval)
	fmt.Printf("Schedule:  %s\n", cronScheduleFromLine(line))
	fmt.Printf("Log file:  %s\n", config.CollapseHome(config.LogFilePath()))
	printCronHealth(line)

	logPath := config.LogFilePath()
	if info, err := os.Stat(logPath); err == nil {
		modTime := info.ModTime()
		ago := time.Since(modTime).Truncate(time.Second)
		fmt.Printf("Last sync: %s (%s ago)\n", modTime.Format("2006-01-02 15:04:05"), ago)
		next := modTime.Add(time.Duration(interval) * time.Minute)
		if next.After(time.Now()) {
			until := time.Until(next).Truncate(time.Second)
			fmt.Printf("Next sync: ~%s (in ~%s)\n", next.Format("2006-01-02 15:04:05"), until)
		} else {
			fmt.Printf("Next sync: overdue (expected %s)\n", next.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("Last sync: no log file found")
	}

	return nil
}

// --- doctor ---

var cronDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check cron entry health and optionally repair it",
	RunE:  runCronDoctor,
}

func runCronDoctor(cmd *cobra.Command, _ []string) error {
	if err := requireCronPlatform(); err != nil {
		return err
	}

	crontab, _ := getCrontab()
	line, found := findSkillsCronLine(crontab)
	if !found {
		fmt.Println("Cron:      not installed")
		fmt.Println()
		fmt.Println("Enable with: skills cron enable")
		return nil
	}

	printCronHealth(line)

	fix, _ := cmd.Flags().GetBool("fix")
	health := inspectCronLine(line)
	if health.Healthy && !health.RepairRecommended {
		if fix {
			fmt.Println("Cron entry is already healthy. No changes made.")
		}
		return nil
	}

	if !fix {
		fmt.Println()
		if health.Healthy {
			fmt.Println("Rewrite with: skills cron doctor --fix")
		} else {
			fmt.Println("Repair with: skills cron doctor --fix")
		}
		return nil
	}

	newCrontab := removeCronLines(crontab)
	newCrontab = appendCronLine(newCrontab, parseIntervalFromCronLine(line))
	if err := writeCrontab(newCrontab); err != nil {
		return fmt.Errorf("writing crontab: %w", err)
	}

	fmt.Println("Cron entry repaired.")
	fmt.Printf("New entry:  %s\n", buildCronLine(parseIntervalFromCronLine(line)))
	return nil
}

// --- enable ---

var cronEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Install the cron entry for automatic sync",
	RunE:  runCronEnable,
}

func runCronEnable(_ *cobra.Command, _ []string) error {
	if err := requireCronPlatform(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := installCron(cfg.EffectiveCronInterval()); err != nil {
		return err
	}
	fmt.Println("Cron entry installed.")
	return nil
}

// --- disable ---

var cronDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Remove the cron entry for automatic sync",
	RunE:  runCronDisable,
}

func runCronDisable(_ *cobra.Command, _ []string) error {
	if err := requireCronPlatform(); err != nil {
		return err
	}

	crontab, err := getCrontab()
	if err != nil {
		return fmt.Errorf("reading crontab: %w", err)
	}

	_, found := findSkillsCronLine(crontab)
	if !found {
		fmt.Println("Cron entry not installed, nothing to remove.")
		return nil
	}

	newCrontab := removeCronLines(crontab)
	if err := writeCrontab(newCrontab); err != nil {
		return fmt.Errorf("writing crontab: %w", err)
	}
	fmt.Println("Cron entry removed.")
	return nil
}

// --- interval ---

var cronIntervalCmd = &cobra.Command{
	Use:   "interval [minutes]",
	Short: "View or set the sync interval in minutes",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCronInterval,
}

func runCronInterval(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Printf("Current interval: %d minutes\n", cfg.EffectiveCronInterval())
		return nil
	}

	minutes, err := strconv.Atoi(args[0])
	if err != nil || minutes < 1 {
		return fmt.Errorf("interval must be a positive integer (minutes)")
	}
	if minutes < 5 {
		return fmt.Errorf("interval must be at least 5 minutes")
	}

	cfg.CronInterval = minutes
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Update cron if installed
	if err := requireCronPlatform(); err == nil {
		crontab, _ := getCrontab()
		if _, found := findSkillsCronLine(crontab); found {
			newCrontab := removeCronLines(crontab)
			newCrontab = appendCronLine(newCrontab, minutes)
			if err := writeCrontab(newCrontab); err != nil {
				return fmt.Errorf("updating crontab: %w", err)
			}
			fmt.Printf("Updated interval to every %d minutes.\n", minutes)
			return nil
		}
	}

	fmt.Printf("Interval saved (%d minutes). Run 'skills cron enable' to activate.\n", minutes)
	return nil
}

// --- logs ---

var cronLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show recent sync log entries",
	RunE:  runCronLogs,
}

func runCronLogs(cmd *cobra.Command, _ []string) error {
	logPath := config.LogFilePath()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Println("No sync logs found at", config.CollapseHome(logPath))
		return nil
	}

	follow, _ := cmd.Flags().GetBool("follow")
	if follow {
		tailCmd := exec.Command("tail", "-f", logPath)
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr
		return tailCmd.Run()
	}

	n, _ := cmd.Flags().GetInt("lines")
	lines, err := readLastLines(logPath, n)
	if err != nil {
		return fmt.Errorf("reading log: %w", err)
	}
	if len(lines) == 0 {
		fmt.Println("Log file is empty.")
		return nil
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

// --- shared helpers ---

func requireCronPlatform() error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("cron is not supported on %s", runtime.GOOS)
	}
	return nil
}

func getCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func findSkillsCronLine(crontab string) (string, bool) {
	for _, line := range strings.Split(crontab, "\n") {
		if strings.Contains(line, "skills sync") {
			return line, true
		}
	}
	return "", false
}

func parseIntervalFromCronLine(line string) int {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return config.DefaultCronInterval
	}
	minute := fields[0]
	if strings.HasPrefix(minute, "*/") {
		if n, err := strconv.Atoi(minute[2:]); err == nil {
			return n
		}
	}
	return config.DefaultCronInterval
}

func cronScheduleFromLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) >= 5 {
		return strings.Join(fields[:5], " ")
	}
	return line
}

type cronHealth struct {
	Healthy           bool
	RepairRecommended bool
	InstalledPath     string
	ExpectedPath      string
	Problems          []string
	Notes             []string
}

var resolveSkillsBinaryPath = skillsBinaryPath

func inspectCronLine(line string) cronHealth {
	health := cronHealth{
		Healthy:      true,
		ExpectedPath: resolveSkillsBinaryPath(),
	}

	fields := strings.Fields(line)
	if len(fields) < 7 {
		health.Healthy = false
		health.Problems = append(health.Problems, "entry format is not recognized")
		return health
	}

	health.InstalledPath = fields[5]
	if fields[6] != "sync" {
		health.Healthy = false
		health.Problems = append(health.Problems, "entry does not invoke 'skills sync'")
	}

	if _, err := os.Stat(health.InstalledPath); err != nil {
		health.Healthy = false
		health.Problems = append(health.Problems, fmt.Sprintf("binary missing: %s", health.InstalledPath))
	}

	if health.InstalledPath != health.ExpectedPath && !isEphemeralGoRunBinary(health.ExpectedPath) {
		health.RepairRecommended = true
		health.Notes = append(health.Notes, fmt.Sprintf("entry points to %s, current binary is %s", health.InstalledPath, health.ExpectedPath))
	}

	return health
}

func printCronHealth(line string) {
	health := inspectCronLine(line)
	if health.Healthy {
		if health.RepairRecommended {
			fmt.Println("Health:    ok (rewrite recommended)")
		} else {
			fmt.Println("Health:    ok")
		}
		for _, note := range health.Notes {
			fmt.Printf("Note:      %s\n", note)
		}
		return
	}

	fmt.Println("Health:    needs repair")
	for _, problem := range health.Problems {
		fmt.Printf("Problem:   %s\n", problem)
	}
	for _, note := range health.Notes {
		fmt.Printf("Note:      %s\n", note)
	}
}

func buildCronLine(intervalMinutes int) string {
	binary := resolveSkillsBinaryPath()
	logPath := config.LogFilePath()
	return fmt.Sprintf("*/%d * * * * %s sync >> %s 2>&1", intervalMinutes, binary, logPath)
}

func skillsBinaryPath() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		if isEphemeralGoRunBinary(exe) {
			if path := findStableSkillsBinaryOnPath(); path != "" {
				return path
			}
		}
		return exe
	}
	if path := findStableSkillsBinaryOnPath(); path != "" {
		return path
	}
	return "/usr/local/bin/skills"
}

func isEphemeralGoRunBinary(path string) bool {
	base := filepath.Base(path)
	if base != "skills" {
		return false
	}
	if strings.Contains(path, string(os.PathSeparator)+"go-build") {
		return true
	}
	tempDir := os.TempDir()
	return strings.HasPrefix(path, tempDir+string(os.PathSeparator))
}

func findStableSkillsBinaryOnPath() string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "skills")
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0111 == 0 {
			continue
		}
		if isEphemeralGoRunBinary(candidate) {
			continue
		}
		return candidate
	}
	return ""
}

func removeCronLines(crontab string) string {
	var kept []string
	for _, line := range strings.Split(crontab, "\n") {
		if !strings.Contains(line, "skills sync") {
			kept = append(kept, line)
		}
	}
	result := strings.Join(kept, "\n")
	// Trim trailing blank lines but keep one newline
	result = strings.TrimRight(result, "\n")
	if result != "" {
		result += "\n"
	}
	return result
}

func appendCronLine(crontab string, intervalMinutes int) string {
	if len(crontab) > 0 && !strings.HasSuffix(crontab, "\n") {
		crontab += "\n"
	}
	return crontab + buildCronLine(intervalMinutes) + "\n"
}

func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// installCron adds the cron entry if not already present. Used by both init and cron enable.
func installCron(intervalMinutes int) error {
	crontab, _ := getCrontab()
	if line, found := findSkillsCronLine(crontab); found {
		if inspectCronLine(line).Healthy {
			fmt.Println("Cron entry already exists, skipping.")
			return nil
		}

		newCrontab := removeCronLines(crontab)
		newCrontab = appendCronLine(newCrontab, intervalMinutes)
		if err := writeCrontab(newCrontab); err != nil {
			return err
		}
		fmt.Println("Repaired existing cron entry.")
		return nil
	}
	newCrontab := appendCronLine(crontab, intervalMinutes)
	return writeCrontab(newCrontab)
}

func readLastLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}
