package manager

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestMissionHeadlessSameSeedProducesSameTickFingerprint 验证完整任务在同步寻路下逐 tick 状态/指令一致。
// 场景：PearlHarbor1941、双方 AI、seed=1、固定时钟；此前异步寻路 worker 会让单舰航路在 tick 76 分叉。
// 预期：两次运行 80 个 tick 的舰船 HP/位置/旋转/目标与指令 String 指纹完全一致；失败说明还有异步或 map 顺序源。
// 覆盖：两个 fresh manager 顺序跑 80 tick，比较每 tick 的 OrderedShips + InstructionSet 指纹。
func TestMissionHeadlessSameSeedProducesSameTickFingerprint(t *testing.T) {
	a := runMissionTrace(t)
	b := runMissionTrace(t)
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			t.Fatalf("first divergence tick=%d\nA=%s\nB=%s", i, a[i], b[i])
		}
	}
}

// runMissionTrace 创建一次 headless 任务并返回每 tick 指纹。
// 副作用：替换并恢复 RNG/时钟/飞机 UID/SimTick；调用方负责单线程执行。
func runMissionTrace(t *testing.T) []string {
	config.G = config.NewDefaultGameSettings()
	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(1)))
	defer restoreRand()
	restoreClock := objUnit.SetNowMillis(func() int64 { return 1_700_000_000_000 })
	defer restoreClock()
	oldTick := objUnit.SimTick()
	objUnit.SetSimTick(0)
	defer objUnit.SetSimTick(oldTick)
	seq := 0
	restoreUID := objUnit.SetPlaneUIDGenerator(func() string { seq++; return fmt.Sprintf("p-%d", seq) })
	defer restoreUID()
	m := New("PearlHarbor1941", faction.SideP1)
	m.SetSimulationOptions(SimulationOptions{Headless: true, SyncTargeting: true, SkipCommands: false})
	out := make([]string, 0, 100)
	for tick := 0; tick < 80; tick++ {
		if _, err := m.UpdateHeadless(); err != nil {
			t.Fatal(err)
		}
		out = append(out, missionFingerprint(m.state, m.instructionSet))
	}
	return out
}

func missionFingerprint(ms *state.MissionState, is *InstructionSet) string {
	var b strings.Builder
	fmt.Fprintf(&b, "tick=%d ", ms.Core.SimTick)
	for _, s := range ms.Arena.OrderedShips() {
		fmt.Fprintf(
			&b,
			"S:%s,%.4f,%.4f,%.4f,%.4f,%.4f,%s;",
			s.Uid,
			s.CurHP,
			s.CurPos.RX,
			s.CurPos.RY,
			s.CurRotation,
			s.CurSpeed,
			s.AttackTarget,
		)
	}
	items := is.Items()
	uids := make([]string, 0, len(items))
	for uid := range items {
		uids = append(uids, uid)
	}
	sort.Strings(uids)
	for _, uid := range uids {
		fmt.Fprintf(&b, "I:%s=%s;", uid, items[uid].String())
	}
	return b.String()
}
