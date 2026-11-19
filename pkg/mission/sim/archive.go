package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ArchiveOptions controls report-run retention.
type ArchiveOptions struct {
	Keep   int
	DryRun bool
}

// ArchiveResult describes what was (or would be) moved.
type ArchiveResult struct {
	ArchiveDir string   `json:"archiveDir"`
	Moved      []string `json:"moved"`
	Kept       []string `json:"kept"`
	DryRun     bool     `json:"dryRun"`
}

// ArchiveRuns moves all but the newest Keep run directories into
// reports/archive/<timestamp>/ and trims their history.jsonl entries.
// It never deletes a run; dry-run reports the plan without touching disk.
func ArchiveRuns(paths Paths, opts ArchiveOptions) (ArchiveResult, error) {
	if opts.Keep <= 0 {
		opts.Keep = 20
	}
	runsDir := filepath.Join(paths.Reports, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ArchiveResult{DryRun: opts.DryRun}, nil
		}
		return ArchiveResult{}, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	result := ArchiveResult{DryRun: opts.DryRun}
	if len(names) <= opts.Keep {
		result.Kept = names
		return result, nil
	}
	result.Kept = append(result.Kept, names[:opts.Keep]...)
	moved := append([]string(nil), names[opts.Keep:]...)
	result.Moved = moved
	archiveDir := filepath.Join(paths.Reports, "archive", time.Now().Format("20060102-150405"))
	for suffix := 2; exists(archiveDir); suffix++ {
		archiveDir = filepath.Join(
			paths.Reports,
			"archive",
			fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), suffix),
		)
	}
	result.ArchiveDir = archiveDir
	if opts.DryRun {
		return result, nil
	}
	if err := os.MkdirAll(filepath.Join(archiveDir, "runs"), 0o755); err != nil {
		return ArchiveResult{}, err
	}
	movedSet := map[string]bool{}
	for _, name := range moved {
		movedSet[name] = true
		if err := os.Rename(filepath.Join(runsDir, name), filepath.Join(archiveDir, "runs", name)); err != nil {
			return ArchiveResult{}, err
		}
	}
	history, err := LoadHistory(paths.Reports)
	if err != nil {
		return ArchiveResult{}, err
	}
	var keptHistory, archivedHistory []HistoryEntry
	for _, entry := range history {
		if movedSet[entry.RunID] {
			archivedHistory = append(archivedHistory, entry)
		} else {
			keptHistory = append(keptHistory, entry)
		}
	}
	if err := writeHistory(filepath.Join(archiveDir, "history.jsonl"), archivedHistory); err != nil {
		return ArchiveResult{}, err
	}
	if err := writeHistory(filepath.Join(paths.Reports, "history.jsonl"), keptHistory); err != nil {
		return ArchiveResult{}, err
	}
	manifest, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ArchiveResult{}, err
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "manifest.json"), append(manifest, '\n'), 0o644); err != nil {
		return ArchiveResult{}, err
	}
	if err := writeIndex(paths.Reports); err != nil {
		return ArchiveResult{}, err
	}
	return result, nil
}

func writeHistory(path string, entries []HistoryEntry) error {
	var b strings.Builder
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return writeFile(path, []byte(b.String()))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
