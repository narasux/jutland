package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
)

type orderTestInstr struct {
	uid   string
	order *[]string
}

func (i *orderTestInstr) Exec(*state.MissionState) error {
	*i.order = append(*i.order, i.uid)
	return nil
}
func (i *orderTestInstr) Executed() bool { return false }
func (i *orderTestInstr) Uid() string    { return i.uid }
func (i *orderTestInstr) String() string { return i.uid }

// TestInstructionSetExecAllUsesStableOrder 验证指令集按 UID 字典序执行，而不是 map 随机序。
// 场景：同一 tick 内多艘舰/机同时有移动/攻击指令；map 随机序会让同 seed 的事件顺序漂移。
// 预期：依次 Add c/a/b 后 ExecAll 记录 a,b,c；Add aa 后再执行追加 aa 到 a,b,c 之后（重建排序为 a,aa,b,c）。
// 覆盖：fake Instruction 追加自身 UID，断言两次 ExecAll 的顺序；不依赖真实指令/单位。
func TestInstructionSetExecAllUsesStableOrder(t *testing.T) {
	var got []string
	set := NewInstructionSet()
	set.Add(&orderTestInstr{uid: "c", order: &got})
	set.Add(&orderTestInstr{uid: "a", order: &got})
	set.Add(&orderTestInstr{uid: "b", order: &got})

	set.ExecAll(&state.MissionState{})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	got = got[:0]
	set.Add(&orderTestInstr{uid: "aa", order: &got})
	set.ExecAll(&state.MissionState{})
	want = []string{"a", "aa", "b", "c"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("order after Add = %v, want %v", got, want)
		}
	}
}
