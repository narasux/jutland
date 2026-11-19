// Package sim implements the headless simulation workspace, scenario loading and report rendering.
package sim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvHome is the required root of the private simulation lab.
	EnvHome = "JUTLAND_SIM_HOME"
	// EnvReportDir optionally overrides the report directory.
	EnvReportDir = "JUTLAND_SIM_REPORT_DIR"
)

// Paths contains the resolved lab layout used by the CLI.
type Paths struct {
	Home      string
	Scenarios string
	Golden    string
	Baselines string
	Reports   string
	Notes     string
}

// ResolveHome returns the absolute JUTLAND_SIM_HOME path.
// It intentionally has no fallback: an unset, empty or whitespace-only value is an error.
func ResolveHome() (string, error) {
	raw, ok := os.LookupEnv(EnvHome)
	if !ok || strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf(
			"%s is required and must not be empty.\n"+
				"Set it to your private simulation lab, e.g.:\n"+
				"  export %s=\"$HOME/jutland-sim-lab\"\n"+
				"Then run: jutland-sim init",
			EnvHome, EnvHome,
		)
	}
	expanded, err := expandHome(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", EnvHome, err)
	}
	return filepath.Clean(abs), nil
}

// ResolvePaths resolves and validates an existing lab root plus report directory.
// outOverride is the CLI --out value; an empty string means "not set".
func ResolvePaths(outOverride string) (Paths, error) {
	home, err := ResolveHome()
	if err != nil {
		return Paths{}, err
	}
	info, err := os.Stat(home)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Paths{}, fmt.Errorf(
				"%s points to %q, which does not exist; run `jutland-sim init` first",
				EnvHome, home,
			)
		}
		return Paths{}, fmt.Errorf("stat %s: %w", EnvHome, err)
	}
	if !info.IsDir() {
		return Paths{}, fmt.Errorf("%s points to %q, which is not a directory", EnvHome, home)
	}
	return resolvePaths(home, outOverride)
}

// ResolvePathsForInit resolves the lab root without requiring it to exist yet.
func ResolvePathsForInit() (Paths, error) {
	home, err := ResolveHome()
	if err != nil {
		return Paths{}, err
	}
	if info, statErr := os.Stat(home); statErr == nil && !info.IsDir() {
		return Paths{}, fmt.Errorf("%s points to %q, which is not a directory", EnvHome, home)
	}
	return resolvePaths(home, "")
}

func resolvePaths(home, outOverride string) (Paths, error) {
	reports, err := resolveReportDir(home, outOverride)
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		Home:      home,
		Scenarios: filepath.Join(home, "scenarios"),
		Golden:    filepath.Join(home, "golden"),
		Baselines: filepath.Join(home, "baselines"),
		Reports:   reports,
		Notes:     filepath.Join(home, "notes"),
	}, nil
}

func resolveReportDir(home, outOverride string) (string, error) {
	raw := strings.TrimSpace(outOverride)
	if raw == "" {
		value, ok := os.LookupEnv(EnvReportDir)
		if ok {
			if strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("%s is set but empty; unset it or provide a directory", EnvReportDir)
			}
			raw = strings.TrimSpace(value)
		}
	}
	if raw == "" {
		return filepath.Join(home, "reports"), nil
	}
	expanded, err := expandHome(raw)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve report directory: %w", err)
	}
	return filepath.Clean(abs), nil
}

// EnsureLayout creates the lab skeleton. It is used by `jutland-sim init`.
func EnsureLayout(paths Paths) error {
	dirs := []string{
		paths.Home,
		paths.Scenarios,
		paths.Golden,
		paths.Baselines,
		paths.Notes,
		paths.Reports,
		filepath.Join(paths.Reports, "assets"),
		filepath.Join(paths.Reports, "runs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

// ValidateLayout checks that the directories needed by a run exist.
func ValidateLayout(paths Paths) error {
	for _, dir := range []string{paths.Home, paths.Scenarios, paths.Golden, paths.Reports} {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("lab is incomplete: %s is missing; run `jutland-sim init` first", dir)
		}
	}
	return nil
}

func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
