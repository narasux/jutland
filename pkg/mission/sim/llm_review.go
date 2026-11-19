package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// deepSeekEndpoint is intentionally fixed; override only in tests.
var deepSeekEndpoint = "https://api.deepseek.com/chat/completions"

const deepSeekModel = "deepseek-chat"

// LLMReviewRequest is the OpenAI-compatible chat request.
type LLMReviewRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream"`
}

// LLMMessage is one chat message.
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMReview is the persisted review artifact.
type LLMReview struct {
	SchemaVersion int       `json:"schemaVersion"`
	RunID         string    `json:"runId"`
	Model         string    `json:"model"`
	GeneratedAt   time.Time `json:"generatedAt"`
	Content       string    `json:"content"`
	PromptTokens  int       `json:"promptTokens,omitempty"`
	OutputTokens  int       `json:"outputTokens,omitempty"`
	TotalTokens   int       `json:"totalTokens,omitempty"`
}

type deepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ReviewRun sends the run's AI artifacts to DeepSeek and writes llm-review.md/json.
// The API key is read from DEEPSEEK_API_KEY; the endpoint is fixed in code.
func ReviewRun(paths Paths, runArg string, dryRun bool) (string, error) {
	runDir, err := resolveRunDir(paths.Reports, runArg)
	if err != nil {
		return "", err
	}
	runID := filepath.Base(runDir)
	context, err := buildLLMContext(runDir)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if key == "" && !dryRun {
		return "", fmt.Errorf("DEEPSEEK_API_KEY is required")
	}
	reqBody := LLMReviewRequest{
		Model: deepSeekModel,
		Messages: []LLMMessage{
			{Role: "system", Content: "你是 Jutland 的战斗模拟与平衡分析助手。只基于提供的 run 数据回答，引用具体 runId/metric；区分 bug、平衡、样本不足。"},
			{Role: "user", Content: context},
		},
		Temperature: 0.2,
		MaxTokens:   2400,
		Stream:      false,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if dryRun {
		return "", writeLLMDryRun(runDir, payload)
	}
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest(http.MethodPost, deepSeekEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("deepseek HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed deepSeekResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse deepseek response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("deepseek: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("deepseek response has no choices")
	}
	review := LLMReview{
		SchemaVersion: 1, RunID: runID, Model: deepSeekModel,
		GeneratedAt: time.Now(), Content: parsed.Choices[0].Message.Content,
		PromptTokens: parsed.Usage.PromptTokens, OutputTokens: parsed.Usage.CompletionTokens,
		TotalTokens: parsed.Usage.TotalTokens,
	}
	dir := filepath.Join(paths.Reports, "llm", runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	mdPath := filepath.Join(dir, "llm-review.md")
	if err := os.WriteFile(mdPath, []byte(review.Content), 0o644); err != nil {
		return "", err
	}
	jsonPath := filepath.Join(dir, "llm-review.json")
	encoded, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(jsonPath, append(encoded, '\n'), 0o644); err != nil {
		return "", err
	}
	return mdPath, nil
}

func buildLLMContext(runDir string) (string, error) {
	names := []string{
		"ai/analysis-context.md", "ai/findings.json", "ai/metrics.json",
		"ai/diagnostics.json", "ai/changes.json", "ai/llm-prompt.md",
	}
	var b strings.Builder
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(runDir, filepath.FromSlash(name)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		b.WriteString("\n\n===== ")
		b.WriteString(name)
		b.WriteString(" =====\n")
		b.Write(data)
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("run has no AI artifacts")
	}
	return b.String(), nil
}

func writeLLMDryRun(runDir string, payload []byte) error {
	// Dry-run is written next to the run's AI files, not the report root.
	dir := filepath.Join(runDir, "ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "llm-request.json"), append(payload, '\n'), 0o644)
}
