package instruction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/grid"
)

// setupGameSettings 给指令层测试提供全局速度设置；MoveTo 依赖 config.G。
func setupGameSettings(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })
}

func TestShipMovePathDoesNotReturnToPositionAtCommandTime(t *testing.T) {
	cells := make(grid.Cells, 20)
	for y := range cells {
		cells[y] = make([]int, 40)
	}
	mapCfg := &mapcfg.MapCfg{Cells: cells, Width: 40, Height: 20}

	commandPos := objPos.New(10, 10)
	targetPos := objPos.New(30, 10)
	ship := &objUnit.BattleShip{Uid: "ship", CurPos: objPos.NewR(11, 10)}
	misState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: mapCfg},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}

	move := NewShipMovePath(ship.Uid, commandPos, targetPos, 0)
	move.genPath(misState)

	require.Equal(t, []objPos.MapPos{targetPos}, move.path)
}

// 起终点吸附到同一格（点击就在本格内）时，退化为对精确目标点的直线逼近。
func TestShipMovePathSinglePointPath(t *testing.T) {
	move := NewShipMovePath("ship", objPos.New(10, 10), objPos.NewR(10.4, 10.3), 0)
	move.applyPoints([]grid.Point{{X: 10, Y: 10}})
	require.Equal(t, []objPos.MapPos{objPos.NewR(10.4, 10.3)}, move.path)
	require.Equal(t, Ready, move.status)
}

// 直线移动被地形拦停时指令要正常收尾，不能永久滞留指令集。
func TestShipMoveBlockedFinishesInstruction(t *testing.T) {
	setupGameSettings(t)
	rows := mapcfg.MapData{
		"SSSSSSSS",
		"SSSCCSSS",
		"SSSSSSSS",
	}
	mapCfg := &mapcfg.MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 8, Height: 3}
	ship := &objUnit.BattleShip{
		Uid:          "ship",
		CurHP:        100,
		CurPos:       objPos.NewR(2.5, 1.5),
		CurRotation:  90,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
	}
	misState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: mapCfg}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}

	move := NewShipMove(ship.Uid, objPos.NewR(5.5, 1.5))
	for tick := 0; tick < 500 && !move.Executed(); tick++ {
		_ = move.Exec(misState)
	}
	require.True(t, move.Executed(), "被地形拦停后指令应完成")
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "拦停时中心点不应压进陆地格")
}

// 回归：中心点压在海岸格上的搁浅舰，重新下达移动指令后必须能退出搁浅并继续机动。
// 场景复现 2026-10-02 珍珠港存档：pennsylvania 卡死在 (51,88) 海岸格上。
func TestShipMovePathRecoversStrandedShipOnPearlHarbor(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)
	require.True(t, mapCfg.Map.IsLand(51, 88), "前提：(51,88) 应为海岸格")

	ship := &objUnit.BattleShip{
		Uid:          "HA/BB-8",
		CurHP:        100,
		CurPos:       objPos.NewR(51.52481545265086, 88.14369335431867),
		CurRotation:  295.525,
		CurSpeed:     0,
		MaxSpeed:     0.0175,
		Acceleration: 0.00015,
		RotateSpeed:  0.325,
	}
	misState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: mapCfg}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}

	move := NewShipMovePath(ship.Uid, ship.CurPos, objPos.NewR(40.5, 80.5), ship.CurSpeed)
	move.genPath(misState)
	require.Equal(t, Ready, move.status, "吸附后应能寻出退出路径")

	escapeTick := -1
	for tick := 0; tick < 6000; tick++ {
		_ = move.Exec(misState)
		if escapeTick < 0 && !mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY) {
			escapeTick = tick
		}
		if move.Executed() {
			break
		}
		// 寻路结果由后台线程送达；等待重寻路时按真实节奏让出主线程，
		// 否则紧凑循环会把 6000 拍压缩到几毫秒墙钟时间内跑完。
		if move.status == Preparing {
			time.Sleep(time.Millisecond)
		}
	}
	require.True(t, move.Executed(), "指令应完成")
	require.GreaterOrEqual(t, escapeTick, 0, "搁浅舰必须退出海岸格")
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "结束后中心点不应留在陆地格上")
	// 是真正到达目标附近，而不是触发放弃分支
	require.LessOrEqual(t, ship.CurPos.Distance(objPos.NewR(40.5, 80.5)), 1.0, "应抵达目标附近")
}

// newBattleShip 造一艘战列舰量级的测试舰（长度 210m、21 节、转向 0.375°/帧）。
func newBattleShip(uid string, pos objPos.MapPos, rotation float64) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid:          uid,
		CurHP:        100,
		TotalHP:      100,
		CurPos:       pos,
		CurRotation:  rotation,
		MaxSpeed:     21.0 / 1200.0,
		Acceleration: 0.00020833333333333335,
		RotateSpeed:  0.375,
		Length:       210,
		Width:        33,
	}
}

// newShipMoveState 构造只含一艘舰的任务状态（含 UI 标记表，用于验证失败提示）。
func newShipMoveState(mapCfg *mapcfg.MapCfg, ship *objUnit.BattleShip) *state.MissionState {
	return &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: mapCfg}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
		UI: state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
}

// newCoastMapCfg 构造 8x3 测试地图：中间一行 x=3、x=4 是海岸格，其余为浅海。
func newCoastMapCfg() *mapcfg.MapCfg {
	rows := mapcfg.MapData{
		"SSSSSSSS",
		"SSSCCSSS",
		"SSSSSSSS",
	}
	return &mapcfg.MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 8, Height: 3}
}

// runShipMovePath 按真实对局节奏推进指令：寻路结果由后台线程送达。
func runShipMovePath(t *testing.T, misState *state.MissionState, move *ShipMovePath, maxTicks int) int {
	t.Helper()
	for tick := 0; tick < maxTicks; tick++ {
		_ = move.Exec(misState)
		if move.Executed() {
			return tick
		}
		if move.status == Preparing {
			time.Sleep(time.Millisecond)
		}
	}
	return maxTicks
}

// 回归：中心点压在海岸格上的搁浅舰，走真实异步寻路（不预先 genPath）也必须能退回水中。
// 旧实现把未吸附的起终点直接交给 Grid.Search，起点是陆格 → 寻路判为无效 →
// 指令第一拍就结束、舰船一动不动（玩家点击毫无反应）。
func TestShipMovePathStrandedShipEscapesWithAsyncSearch(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)
	require.True(t, mapCfg.Map.IsLand(59, 122), "前提：(59,122) 应为海岸格")

	ship := newBattleShip("HA/BB-9", objPos.NewR(59.5, 122.5), 0)
	misState := newShipMoveState(mapCfg, ship)
	target := objPos.NewR(59.5, 140.5)

	move := NewShipMovePath(ship.Uid, ship.CurPos, target, ship.CurSpeed)
	ticks := runShipMovePath(t, misState, move, 6000)

	require.True(t, move.Executed(), "指令应完成")
	require.False(t, move.Failed(), "搁浅舰退回外海不应判定失败")
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "舰体中心必须离开陆格")
	require.LessOrEqual(t, ship.CurPos.Distance(target), 1.0, "应抵达目标附近")
	t.Logf("搁浅恢复耗时 %d 拍（%.1fs）", ticks, float64(ticks)/60)
}

// 右键点在海岸格上：目标必须吸附到同域水面，舰船停在岸边水面并正常完成。
// 旧实现把原始陆格点当末段航点，舰船会主动往岸上开、撞停后放弃。
func TestShipMovePathGoalOnCoastStopsOnWater(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)
	require.True(t, mapCfg.Map.IsLand(59, 122), "前提：(59,122) 应为海岸格")

	ship := newBattleShip("HA/BB-10", objPos.NewR(59.5, 124.5), 90)
	misState := newShipMoveState(mapCfg, ship)
	clicked := objPos.NewR(59.5, 122.5)

	move := NewShipMovePath(ship.Uid, ship.CurPos, clicked, ship.CurSpeed)
	ticks := runShipMovePath(t, misState, move, 6000)

	require.True(t, move.Executed(), "指令应完成")
	require.False(t, move.Failed(), "点在岸上不应判为不可达")
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "舰船不能停在陆格上")
	require.LessOrEqual(t, ship.CurPos.Distance(clicked), 2.0, "应停在点击点附近的岸边水面")
	t.Logf("岸边点击停靠耗时 %d 拍", ticks)
}

// 回归：舰艏顶岸被拦停后，单次指令内必须完成掉头脱离。
// 旧实现累计受阻 80 拍（1.3s）就放弃，而战列舰掉头需要 400 拍以上，
// 玩家必须反复右键下单（实测 3 次、19 秒）才能把船挪出来。
func TestShipMovePathBeachedShipTurnsAroundInOneOrder(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)

	ship := newBattleShip("HA/BB-11", objPos.NewR(59.5, 123.5), 0)
	misState := newShipMoveState(mapCfg, ship)
	// 先朝正北的岸线开，直到被地形拦停（模拟玩家把船顶到岸边）。
	for i := 0; i < 2000; i++ {
		if _, blocked := ship.MoveTo(mapCfg, objPos.NewR(59.5, 100.5), false); blocked {
			break
		}
	}
	require.Equal(t, 0.0, ship.CurSpeed, "前提：顶岸后应被拦停")
	startPos := ship.CurPos.Copy()

	target := objPos.NewR(59.5, 133.5)
	move := NewShipMovePath(ship.Uid, ship.CurPos, target, ship.CurSpeed)
	ticks := runShipMovePath(t, misState, move, 6000)

	require.True(t, move.Executed(), "指令应完成")
	require.False(t, move.Failed(), "贴岸掉头是可完成机动，不应判定失败")
	require.LessOrEqual(t, ship.CurPos.Distance(target), 1.0, "应抵达目标附近")
	require.Greater(t, ship.CurPos.Distance(startPos), 8.0, "应真正离开岸边")
	t.Logf("贴岸掉头耗时 %d 拍（%.1fs）", ticks, float64(ticks)/60)
}

// 目标在封闭水域（珍珠港内湾）时，寻路不可达必须快速结束并标记失败，
// 供 UI 提示，而不是让舰船在岸边无限磨。
func TestShipMovePathUnreachableTargetMarksFailed(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)
	require.True(t, mapCfg.Map.IsSea(58, 121), "前提：(58,121) 是内湾水面")
	require.NotEqual(
		t,
		mapCfg.WaterRegionAt(58, 121),
		mapCfg.WaterRegionAt(59, 140),
		"前提：内湾与主海域不连通",
	)

	ship := newBattleShip("HA/BB-12", objPos.NewR(59.5, 140.5), 0)
	misState := newShipMoveState(mapCfg, ship)

	move := NewShipMovePath(ship.Uid, ship.CurPos, objPos.NewR(58.5, 121.5), ship.CurSpeed)
	ticks := runShipMovePath(t, misState, move, 600)

	require.True(t, move.Executed(), "不可达目标也要正常收尾，不能永久滞留指令")
	require.True(t, move.Failed(), "不可达目标必须标记失败")
	require.Less(t, ticks, 120, "应在寻路结果回来后立刻结束，而不是原地磨")
	require.NotEmpty(t, misState.UI.GameMarks, "失败时必须在舰船位置弹出提示，不能静默结束")
}

// 长时间既无位移也无转向（航点被地形彻底挡死、航向又已经对准）时必须结束机动
// 并标记失败：这条规则替代了旧的“累计受阻 80 拍放弃”，用来兜住真正不可达的机动。
func TestShipMovePathNoProgressMarksFailed(t *testing.T) {
	setupGameSettings(t)
	// 四行地图：避免 EnsureBorder 把 RY 夹到边界，干扰“位置是否变化”的判断。
	rows := mapcfg.MapData{
		"SSSSSSSS",
		"SSSCCSSS",
		"SSSSSSSS",
		"SSSSSSSS",
	}
	mapCfg := &mapcfg.MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 8, Height: 4}
	ship := newBattleShip("HA/BB-13", objPos.NewR(2.9, 1.5), 90)
	ship.MaxSpeed = 0.02
	ship.Acceleration = 0.001
	misState := newShipMoveState(mapCfg, ship)

	// 目标本身在水面（不会被吸附改写），但手工指定的航点在海岸格上：
	// 航向已经对准正东，舰船既不会转向也走不过去，位置几乎不变 → 触发无进展放弃。
	move := NewShipMovePath(ship.Uid, ship.CurPos, objPos.NewR(5.5, 1.5), 0)
	move.path = []objPos.MapPos{objPos.NewR(4.5, 1.5)}
	move.status = Ready
	move.initSpeed = -1

	ticks := runShipMovePath(t, misState, move, 1000)

	require.True(t, move.Executed(), "无进展的机动必须结束")
	require.True(t, move.Failed(), "无进展必须标记失败")
	require.GreaterOrEqual(t, ticks, blockedNoProgressMaxTicks-1)
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "全程不应压进陆格")
}
