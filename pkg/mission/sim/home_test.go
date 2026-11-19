package sim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setEnvForTest sets or unsets an environment variable and restores it after the test.
func setEnvForTest(t *testing.T, key, value string, set bool) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if set {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("set env %s: %v", key, err)
		}
	} else {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// TestResolveHomeRequiresNonEmpty 验证 JUTLAND_SIM_HOME 缺失、空、全空白都必须报错且不创建目录。
// 场景：P0 决策要求 lab 根必填、无默认值；命令行若静默回退会把报告写进游戏仓库。
// 预期：三种输入都返回包含 JUTLAND_SIM_HOME 的错误；传入的候选目录始终不存在。
// 覆盖：表驱动 unset/空字符串/空格，调用 ResolveHome 并检查错误文本与 os.Stat。
func TestResolveHomeRequiresNonEmpty(t *testing.T) {
	cases := []struct {
		name string
		set  bool
		val  string
	}{
		{name: "unset", set: false},
		{name: "empty", set: true, val: ""},
		{name: "spaces", set: true, val: "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnvForTest(t, EnvHome, tc.val, tc.set)
			got, err := ResolveHome()
			if err == nil {
				t.Fatalf("ResolveHome() = %q, want error", got)
			}
			if !strings.Contains(err.Error(), EnvHome) {
				t.Fatalf("error = %v, want it to mention %s", err, EnvHome)
			}
			if _, statErr := os.Stat(got); !os.IsNotExist(statErr) {
				t.Fatalf("ResolveHome created %q or returned a path with unexpected stat: %v", got, statErr)
			}
		})
	}
}

// TestResolvePathsUsesHomeAndOverrides 验证 home、report dir 环境变量和 --out 的优先级。
// 场景：lab 根存在；JUTLAND_SIM_REPORT_DIR 指向自定义目录；--out 再覆盖报告目录。
// 预期：Paths.Scenarios/Golden/Baselines 挂在 home 下；报告目录依次被环境变量、--out 改写。
// 覆盖：t.Setenv 设置 home/report dir，调用 ResolvePaths("") 与 ResolvePaths(customOut) 断言路径。
func TestResolvePathsUsesHomeAndOverrides(t *testing.T) {
	home := t.TempDir()
	setEnvForTest(t, EnvHome, home, true)
	reportOverride := filepath.Join(home, "custom-reports")
	setEnvForTest(t, EnvReportDir, reportOverride, true)

	paths, err := ResolvePaths("")
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	if paths.Home != home {
		t.Fatalf("Home = %q, want %q", paths.Home, home)
	}
	if paths.Scenarios != filepath.Join(home, "scenarios") {
		t.Fatalf("Scenarios = %q", paths.Scenarios)
	}
	if paths.Reports != reportOverride {
		t.Fatalf("Reports = %q, want %q", paths.Reports, reportOverride)
	}

	out := filepath.Join(home, "out")
	paths, err = ResolvePaths(out)
	if err != nil {
		t.Fatalf("ResolvePaths(--out) error = %v", err)
	}
	if paths.Reports != out {
		t.Fatalf("Reports = %q, want %q", paths.Reports, out)
	}
}

// TestResolvePathsRejectsEmptyReportDir 验证显式设置但为空的 JUTLAND_SIM_REPORT_DIR 必须报错。
// 场景：用户误把环境变量设成空串，若回退到默认目录会写错位置。
// 预期：ResolvePaths 返回包含变量名的错误，不返回任何路径。
// 覆盖：home 合法、report dir 设为 "  "，调用 ResolvePaths 断言 err 与空 Paths。
func TestResolvePathsRejectsEmptyReportDir(t *testing.T) {
	home := t.TempDir()
	setEnvForTest(t, EnvHome, home, true)
	setEnvForTest(t, EnvReportDir, "  ", true)

	paths, err := ResolvePaths("")
	if err == nil {
		t.Fatalf("ResolvePaths() = %+v, want error", paths)
	}
	if !strings.Contains(err.Error(), EnvReportDir) {
		t.Fatalf("error = %v, want it to mention %s", err, EnvReportDir)
	}
}

// TestResolvePathsRejectsMissingOrFileHome 验证 home 不存在或是普通文件时拒绝运行。
// 场景：env 指向尚未 init 的目录，或误指向文件；两者都不能被当成合法 lab。
// 预期：错误分别提示 init 与 not a directory；不自动创建目录。
// 覆盖：先 ResolvePaths(missing) 再 os.Stat，随后创建文件并 ResolvePaths(file)。
func TestResolvePathsRejectsMissingOrFileHome(t *testing.T) {
	parent := t.TempDir()
	missing := filepath.Join(parent, "missing")
	setEnvForTest(t, EnvHome, missing, true)
	setEnvForTest(t, EnvReportDir, "", false)

	if _, err := ResolvePaths(""); err == nil || !strings.Contains(err.Error(), "init") {
		t.Fatalf("missing home error = %v, want init hint", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("missing home was created: %v", err)
	}

	file := filepath.Join(parent, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setEnvForTest(t, EnvHome, file, true)
	if _, err := ResolvePaths(""); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("file home error = %v, want not a directory", err)
	}
}

// TestInitAndValidateLayout 验证 init 能创建完整骨架，且 ValidateLayout 在骨架缺失时报错。
// 场景：home 不存在时先 ResolvePathsForInit，再 EnsureLayout；随后删除 golden 模拟不完整 lab。
// 预期：所有目录存在且 ValidateLayout 通过；删除 golden 后 ValidateLayout 返回错误。
// 覆盖：ResolvePathsForInit -> EnsureLayout -> os.Stat 五个目录 -> ValidateLayout -> RemoveAll(golden) -> 再验证。
func TestInitAndValidateLayout(t *testing.T) {
	home := filepath.Join(t.TempDir(), "lab")
	setEnvForTest(t, EnvHome, home, true)

	paths, err := ResolvePathsForInit()
	if err != nil {
		t.Fatalf("ResolvePathsForInit() error = %v", err)
	}
	if err := EnsureLayout(paths); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}
	for _, dir := range []string{paths.Home, paths.Scenarios, paths.Golden, paths.Baselines, paths.Notes, paths.Reports} {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Fatalf("expected dir %s, err=%v", dir, err)
		}
	}
	if err := ValidateLayout(paths); err != nil {
		t.Fatalf("ValidateLayout() error = %v", err)
	}
	if err := os.RemoveAll(paths.Golden); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLayout(paths); err == nil {
		t.Fatal("ValidateLayout() = nil, want error after removing golden")
	}
}
