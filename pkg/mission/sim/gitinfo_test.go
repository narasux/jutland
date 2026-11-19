package sim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseGitStatusPreservesFirstPath 验证 git status 首行的前导空格不会吞掉文件名的第一个字符。
// 场景：工作区首个修改是 " M .gitignore"；若先 TrimSpace 整段输出，会解析成 "gitignore"。
// 预期：状态为 M，路径为 .gitignore；第二行 ?? cmd/ 也保留。
// 覆盖：直接调用 parseGitStatus，输入带前导空格的两行，断言数量、状态与路径。
func TestParseGitStatusPreservesFirstPath(t *testing.T) {
	files, count := parseGitStatus(" M .gitignore\n?? cmd/\n", 20)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(files) != 2 {
		t.Fatalf("files = %+v, want 2", files)
	}
	if files[0].Status != "M" || files[0].Path != ".gitignore" {
		t.Fatalf("first = %+v, want M .gitignore", files[0])
	}
	if files[1].Status != "??" || files[1].Path != "cmd/" {
		t.Fatalf("second = %+v", files[1])
	}
}

// TestParseGitStatusTruncatesList 验证脏文件只保留前 limit 个，但总数仍完整。
// 场景：25 个未提交文件，报告只展示前 2 个，需知道还有 23 个。
// 预期：files 长度 2，count=25，DirtyTruncated 由上层按 count>len(files) 计算。
// 覆盖：构造 25 行 ?? 路径，调用 parseGitStatus(status, 2)。
func TestParseGitStatusTruncatesList(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 25; i++ {
		b.WriteString("?? file")
		b.WriteByte(byte('a' + i))
		b.WriteByte('\n')
	}
	files, count := parseGitStatus(b.String(), 2)
	if count != 25 {
		t.Fatalf("count = %d, want 25", count)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
}

// TestCollectGitInfoCleanAndDirty 验证在真实临时 git 仓库中能识别 clean/dirty 与 worktree hash。
// 场景：git init + 提交一个文件；首次 clean，修改后 dirty；同一提交下 dirty hash 稳定。
// 预期：clean 时 Available/Commit 非空、Dirty=false、WorktreeHash=clean；
// 修改后 Dirty=true、DirtyCount=1、WorktreeHash 非 clean 且两次采集一致。
// 覆盖：跳过无 git 环境；exec git 建仓提交，CollectGitInfo 两次比较。
func TestCollectGitInfoCleanAndDirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "initial")

	clean := CollectGitInfo(dir)
	if !clean.Available || clean.Commit == "" {
		t.Fatalf("clean info = %+v", clean)
	}
	if clean.Dirty {
		t.Fatalf("clean.Dirty = true, files=%+v", clean.DirtyFiles)
	}
	if clean.WorktreeHash != "clean" {
		t.Fatalf("clean.WorktreeHash = %q", clean.WorktreeHash)
	}

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty := CollectGitInfo(dir)
	if !dirty.Dirty || dirty.DirtyCount != 1 {
		t.Fatalf("dirty info = %+v", dirty)
	}
	if dirty.WorktreeHash == "clean" {
		t.Fatal("dirty.WorktreeHash should not be clean")
	}
	again := CollectGitInfo(dir)
	if again.WorktreeHash != dirty.WorktreeHash {
		t.Fatalf("worktree hash unstable: %q vs %q", dirty.WorktreeHash, again.WorktreeHash)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
