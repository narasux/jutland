package sim

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReviewRunDeepSeekMock 验证 DeepSeek 请求/响应落盘与 dry-run。
// 场景：临时 run 含 ai/analysis-context.md；httptest 模拟 DeepSeek 返回 content=ok。
// 预期：请求带 Authorization Bearer test-key、model=deepseek-chat；llm-review.md=ok，json usage 写入。
// 覆盖：ReviewRun 正常分支 + dry-run 请求文件；测试后恢复 endpoint/env。
func TestReviewRunDeepSeekMock(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1", "ai")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "analysis-context.md"), []byte("run context"), 0o644))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var req LLMReviewRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, deepSeekModel, req.Model)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	defer server.Close()
	oldEndpoint := deepSeekEndpoint
	deepSeekEndpoint = server.URL
	defer func() { deepSeekEndpoint = oldEndpoint }()
	t.Setenv("DEEPSEEK_API_KEY", "test-key")

	out, err := ReviewRun(Paths{Reports: dir}, "r1", false)
	require.NoError(t, err)
	require.FileExists(t, out)
	content, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "ok", string(content))
	var review LLMReview
	payload, err := os.ReadFile(filepath.Join(dir, "llm", "r1", "llm-review.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, &review))
	require.Equal(t, 15, review.TotalTokens)

	t.Setenv("DEEPSEEK_API_KEY", "")
	require.NoError(t, ReviewRunDry(Paths{Reports: dir}, "r1"))
}

func ReviewRunDry(paths Paths, runArg string) error {
	_, err := ReviewRun(paths, runArg, true)
	return err
}
