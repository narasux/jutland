package cliutil

import "testing"

// TestSplitAllowsFlagsAfterPositional 验证位置参数后的 flag 不会被 flag 包截断。
// 场景：用户常写 `macro scenario --ticks 6000 --seed 7`；Go flag.Parse 在首个位置参数处停止。
// 预期：Split 把 --ticks/6000、--seed/7 放进 flags，scenario 放进 positionals；--open 不吞下一个 token。
// 覆盖：表驱动混合顺序、值 flag、bool flag，断言两个切片。
func TestSplitAllowsFlagsAfterPositional(t *testing.T) {
	flags, pos := Split([]string{"scenario", "--ticks", "6000", "--seed=7", "--open"}, "open")
	if len(pos) != 1 || pos[0] != "scenario" {
		t.Fatalf("positionals = %v", pos)
	}
	want := []string{"--ticks", "6000", "--seed=7", "--open"}
	if len(flags) != len(want) {
		t.Fatalf("flags = %v, want %v", flags, want)
	}
	for i := range want {
		if flags[i] != want[i] {
			t.Fatalf("flags = %v, want %v", flags, want)
		}
	}
}
