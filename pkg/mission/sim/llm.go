package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LLMPackage is the small, prompt-ready bundle for one run.
type LLMPackage struct {
	SchemaVersion   int      `json:"schemaVersion"`
	RunID           string   `json:"runId"`
	Scenario        string   `json:"scenario"`
	Fidelity        string   `json:"fidelity"`
	Files           []string `json:"files"`
	TotalBytes      int64    `json:"totalBytes"`
	EstimatedTokens int      `json:"estimatedTokens"`
}

// WriteLLMPackage copies only analysis artifacts (never HTML/CSV) into
// reports/llm/<runId>/ and writes package.json.
func WriteLLMPackage(paths Paths, runArg string) (string, error) {
	runDir, err := resolveRunDir(paths.Reports, runArg)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(runDir, "result.json"))
	if err != nil {
		return "", err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	runID := filepath.Base(runDir)
	outDir := filepath.Join(paths.Reports, "llm", runID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	names := []string{
		"ai/analysis-context.md", "ai/findings.json", "ai/metrics.json",
		"ai/diagnostics.json", "ai/changes.json", "ai/llm-prompt.md",
	}
	pkg := LLMPackage{
		SchemaVersion: 1, RunID: runID, Scenario: result.Scenario.Name, Fidelity: result.Fidelity,
	}
	for _, name := range names {
		src := filepath.Join(runDir, filepath.FromSlash(name))
		payload, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		dst := filepath.Join(outDir, filepath.Base(name))
		if err := os.WriteFile(dst, payload, 0o644); err != nil {
			return "", err
		}
		pkg.Files = append(pkg.Files, filepath.Base(name))
		pkg.TotalBytes += int64(len(payload))
	}
	if len(pkg.Files) == 0 {
		return "", fmt.Errorf("run %s has no AI artifacts", runID)
	}
	if _, err := os.Stat(filepath.Join(outDir, "llm-prompt.md")); os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(outDir, "llm-prompt.md"), []byte(llmPrompt), 0o644); err != nil {
			return "", err
		}
		pkg.Files = append(pkg.Files, "llm-prompt.md")
		pkg.TotalBytes += int64(len(llmPrompt))
	}
	sort.Strings(pkg.Files)
	pkg.EstimatedTokens = int(pkg.TotalBytes / 4)
	payload, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(outDir, "package.json"), append(payload, '\n'), 0o644); err != nil {
		return "", err
	}
	readme := fmt.Sprintf(
		"# LLM package\n\n- run: `%s`\n- scenario: %s\n- fidelity: %s\n- files: %s\n- estimated tokens: %d\n\nUse `analysis-context.md`, `findings.json`, `metrics.json`, `diagnostics.json`, `changes.json` and `llm-prompt.md`. Do not read HTML/CSV; they are intentionally excluded.\n",
		runID,
		result.Scenario.Name,
		result.Fidelity,
		strings.Join(pkg.Files, ", "),
		pkg.EstimatedTokens,
	)
	if err := os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readme), 0o644); err != nil {
		return "", err
	}
	return outDir, nil
}
