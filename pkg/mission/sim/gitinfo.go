package sim

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/narasux/jutland/pkg/version"
)

// GitDirtyFile is one uncommitted path reported by git status.
type GitDirtyFile struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// GitInfo is the run's code provenance. It is intentionally best-effort:
// a missing git binary or a packaged binary must not block report generation.
type GitInfo struct {
	Available      bool           `json:"available"`
	Commit         string         `json:"commit"`
	CommitShort    string         `json:"commitShort"`
	Branch         string         `json:"branch"`
	Tag            string         `json:"tag"`
	Subject        string         `json:"subject"`
	Dirty          bool           `json:"dirty"`
	DirtyFiles     []GitDirtyFile `json:"dirtyFiles"`
	DirtyCount     int            `json:"dirtyCount"`
	DirtyTruncated bool           `json:"dirtyTruncated"`
	WorktreeHash   string         `json:"worktreeHash"`
	Error          string         `json:"error,omitempty"`
}

// CollectGitInfo reads git state under root. It never returns an error;
// failures are recorded in GitInfo.Error and fall back to build metadata.
func CollectGitInfo(root string) GitInfo {
	if root == "" {
		root, _ = os.Getwd()
	}
	info := GitInfo{}
	if _, err := gitOutput(root, "rev-parse", "--is-inside-work-tree"); err != nil {
		info.Error = "not a git worktree"
		info.Commit = version.GitCommit
		info.CommitShort = shortCommit(info.Commit)
		return info
	}
	info.Available = true
	info.Commit, _ = gitOutput(root, "rev-parse", "HEAD")
	info.CommitShort, _ = gitOutput(root, "rev-parse", "--short", "HEAD")
	info.Branch, _ = gitOutput(root, "branch", "--show-current")
	info.Tag, _ = gitOutput(root, "describe", "--tags", "--always")
	info.Subject, _ = gitOutput(root, "log", "-1", "--pretty=%s")

	status, err := gitOutputRaw(root, "status", "--porcelain=v1")
	if err != nil {
		info.Error = "git status failed: " + err.Error()
	} else {
		info.DirtyFiles, info.DirtyCount = parseGitStatus(status, 20)
		info.Dirty = info.DirtyCount > 0
		info.DirtyTruncated = info.DirtyCount > len(info.DirtyFiles)
	}
	info.WorktreeHash = worktreeHash(root, info.Dirty)
	return info
}

func parseGitStatus(status string, limit int) ([]GitDirtyFile, int) {
	if strings.TrimSpace(status) == "" {
		return nil, 0
	}
	var files []GitDirtyFile
	count := 0
	for _, line := range strings.Split(strings.TrimRight(status, "\n"), "\n") {
		if len(line) < 3 {
			continue
		}
		count++
		if len(files) >= limit {
			continue
		}
		files = append(files, GitDirtyFile{
			Status: strings.TrimSpace(line[:2]),
			Path:   strings.TrimSpace(line[3:]),
		})
	}
	return files, count
}

func worktreeHash(root string, dirty bool) string {
	if !dirty {
		return "clean"
	}
	h := sha256.New()
	diffPaths := []string{"pkg", "configs", "sim", "cmd", "go.mod", "go.sum", "main.go"}
	args := append([]string{"diff", "HEAD", "--"}, diffPaths...)
	if diff, err := gitOutput(root, args...); err == nil {
		_, _ = io.WriteString(h, diff)
	}
	untrackedArgs := append([]string{"ls-files", "--others", "--exclude-standard", "--"}, diffPaths...)
	untracked, err := gitOutput(root, untrackedArgs...)
	if err == nil {
		paths := strings.Split(strings.TrimSpace(untracked), "\n")
		sort.Strings(paths)
		for _, rel := range paths {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			_, _ = io.WriteString(h, rel)
			full := filepath.Join(root, filepath.FromSlash(rel))
			fileInfo, statErr := os.Stat(full)
			if statErr != nil || fileInfo.IsDir() {
				continue
			}
			if fileInfo.Size() > 1<<20 {
				_, _ = fmt.Fprintf(h, "\x00size:%d", fileInfo.Size())
				continue
			}
			file, openErr := os.Open(full)
			if openErr != nil {
				continue
			}
			_, _ = io.Copy(h, file)
			_ = file.Close()
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func gitOutputRaw(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmdArgs := append([]string{"-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func gitOutput(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmdArgs := append([]string{"-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
