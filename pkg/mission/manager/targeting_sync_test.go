package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

// TestSyncTargetingBuildsInline 验证同步规划在当前 tick 完成，不启动后台 goroutine。
// 场景：headless 同 seed 复现要求 plan 的 ComputedTick/Revision 固定，后台 goroutine 的完成时机不能参与结果。
// 预期：SetSyncTargeting(true) 后 updateTargetPlanning 立即写入 Revision=simTick、Busy=false、Dirty=false。
// 覆盖：最小 MissionState + simTick=100，markTargetingDirty 后调用 updateTargetPlanning，断言 plan 字段与标志。
func TestSyncTargetingBuildsInline(t *testing.T) {
	m := &MissionManager{state: &state.MissionState{}}
	m.simTick = 100
	m.SetSyncTargeting(true)
	m.markTargetingDirty()

	m.updateTargetPlanning()

	require.False(t, m.targetingBusy)
	require.False(t, m.targetingDirty)
	require.Equal(t, int64(100), m.targetingPlan.Revision)
	require.Equal(t, int64(100), m.targetingPlan.ComputedTick)
	require.Greater(t, m.targetingPlan.ExpiresTick, m.targetingPlan.ComputedTick)
}
