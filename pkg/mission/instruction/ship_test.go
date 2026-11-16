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

// TestShipMovePathDoesNotReturnToPositionAtCommandTime 验证寻路不再把下令时位置当首航点。
// 场景：40x20 全水面地图，玩家在 (10,10) 下令让位于 (11,10) 的舰开往 (30,10)，
// genPath 后直接检查生成的航点列表。
// 预期：path 只有终点 (30,10) 一个点。若把 commandPos 也塞进航路，舰船会先掉头
// 开回 1 格外的旧位置，表现为点击后先倒退再前进。
// 覆盖：make(grid.Cells) 全 0 海图，调用 NewShipMovePath 与 genPath，Equal 比较整条 path。
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

// TestShipMovePathSinglePointPath 验证起终点吸附到同一格时退化为精确点直线逼近。
// 场景：NewShipMovePath("ship", (10,10), (10.4,10.3)) 后直接 applyPoints 传入
// 单个格点 (10,10)——模拟寻路起终点吸附到同一格。
// 预期：path 只含精确点击点 (10.4,10.3)，status 变为 Ready；不能改写成格心
// (10.5,10.5)，否则近距离微调会先开向格心再折返。
// 覆盖：直接调用 applyPoints([]grid.Point{{X:10, Y:10}})，Equal 比较 path 与
// objPos.NewR(10.4,10.3)，并检查 status==Ready。
func TestShipMovePathSinglePointPath(t *testing.T) {
	move := NewShipMovePath("ship", objPos.New(10, 10), objPos.NewR(10.4, 10.3), 0)
	move.applyPoints([]grid.Point{{X: 10, Y: 10}})
	require.Equal(t, []objPos.MapPos{objPos.NewR(10.4, 10.3)}, move.path)
	require.Equal(t, Ready, move.status)
}

// TestShipMoveBlockedFinishesInstruction 验证被地形拦停时移动指令正常收尾。
// 场景：8x3 地图 "SSSSSSSS"/"SSSCCSSS"/"SSSSSSSS"，(3,1)、(4,1) 是海岸格；
// 舰船在 (2.5,1.5) 朝正东、速度 0.02，命令开往 (5.5,1.5)，逐拍 Exec 最多 500 拍。
// 预期：move.Executed()=true，且结束时中心点不在陆格；旧行为可能永久滞留指令集
// 或把舰船中心压进陆地。
// 覆盖：调用 NewShipMove 后循环 Exec，断言 Executed 与 mapCfg.Map.IsLand 为 false。
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

// TestShipMovePathRecoversStrandedShipOnPearlHarbor 验证珍珠港搁浅舰能退出海岸格。
// 场景：复现 2026-10-02 珍珠港存档——pennsylvania（Uid "HA/BB-8"）中心点卡在
// 海岸格 (51,88)，实坐标 (51.5248,88.1437)、航向 295.525°、MaxSpeed 0.0175、
// 转向 0.325；命令开往 (40.5,80.5)，先 genPath 再逐拍 Exec，最多 6000 拍。
// 预期：genPath 后 status=Ready（吸附起终点后可寻路）；执行完成、至少有一拍
// 中心点离开陆格，最终距目标 ≤1.0 格且不是触发放弃分支；寻路 Preparing 时
// 每次让出 1ms 等后台结果。
// 覆盖：mapcfg.GetByName("pearl_harbor") 前提断言 IsLand(51,88)，NewShipMovePath
// + genPath + 手动推进 Exec，断言 Ready/Executed/escapeTick/距离。
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

// TestShipMovePathStrandedShipEscapesWithAsyncSearch 验证真实异步寻路下搁浅舰也能脱困。
// 场景：珍珠港海岸格 (59,122) 上的战列舰 HA/BB-9 位于 (59.5,122.5)，目标
// (59.5,140.5)；不预先 genPath，由 NewShipMovePath 提交后台寻路，runShipMovePath
// 推进最多 6000 拍、Preparing 时每拍睡 1ms。
// 预期：指令完成且不判 Failed，中心点离开陆格并抵达目标 1 格内。旧实现把未吸附的
// 起终点交给 Grid.Search，起点是陆格 → 判无效 → 第一拍就结束、舰船一动不动。
// 覆盖：newBattleShip + newShipMoveState + runShipMovePath，断言 Executed、
// !Failed、!IsLand、距离 ≤1.0，并 t.Logf 记录耗时。
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

// TestShipMovePathGoalOnCoastStopsOnWater 验证右键点在海岸格时目标吸附到同域水面。
// 场景：珍珠港战列舰 HA/BB-10 在 (59.5,124.5) 航向 90°，点击 (59.5,122.5)，
// 前提断言 (59,122) 是海岸格；异步推进最多 6000 拍。
// 预期：目标经 prepareTarget 吸附后，指令完成且不判 Failed，舰船停在非陆格，
// 最终距原始点击点 ≤2.0 格。旧实现把原始陆格当末段航点，舰船会主动往岸上开、
// 撞停后放弃。
// 覆盖：mapcfg.GetByName("pearl_harbor") + NewShipMovePath + runShipMovePath，
// 断言 Executed/!Failed/!IsLand/距离，并记录拍数。
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

// TestShipMovePathBeachedShipTurnsAroundInOneOrder 验证顶岸舰单次指令内完成掉头脱离。
// 场景：珍珠港战列舰 HA/BB-11 在 (59.5,123.5) 朝正北；先朝岸线 (59.5,100.5)
// MoveTo 直到 blocked（最多 2000 拍），确认 CurSpeed=0 后改令开往 (59.5,133.5)，
// 最多推进 6000 拍。
// 预期：指令完成且不判 Failed，最终距目标 ≤1.0，且离开起点 >8 格。旧实现累计
// 受阻 80 拍（约 1.3s）就放弃，而战列舰掉头需要 400 拍以上，玩家实测要下 3 次
// 指令、耗 19 秒才能挪出；现有 blockedNoProgressMaxTicks=240 只惩罚“既无位移
// 也无转向”。
// 覆盖：mapcfg.GetByName("pearl_harbor") + 预顶岸 + NewShipMovePath +
// runShipMovePath，断言 Executed/!Failed/距离/离岸距离。
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

// TestShipMovePathUnreachableTargetMarksFailed 验证封闭水域目标快速失败且不再弹提示。
// 场景：珍珠港内湾 (58,121) 是水面，但与主海域 (59,140) 不在同一 WaterRegion；
// 战列舰 HA/BB-12 在主海域 (59.5,140.5)，目标 (58.5,121.5)，最多推进 600 拍。
// 预期：指令正常 Executed，Failed=true，且在 120 拍内结束（寻路结果返回即失败），
// UI.GameMarks 为空——不再生成“无法抵达”提示。
// 覆盖：mapcfg.GetByName("pearl_harbor") 断言 IsSea(58,121) 与水域不连通，
// NewShipMovePath + runShipMovePath，断言 Executed/Failed/拍数/GameMarks。
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
	require.Empty(t, misState.UI.GameMarks, "机动失败不再弹“无法抵达”提示")
}

// TestShipMovePathNoProgressMarksFailed 验证长时间无位移无转向的机动标记失败。
// 场景：内联构造的 8x4 地图（避免 EnsureBorder 夹 RY）中 (3,1)、(4,1) 是海岸；
// 战列舰 HA/BB-13 在 (2.9,1.5) 朝正东、MaxSpeed 0.02；目标 (5.5,1.5) 本身是
// 水面，但手工把 path 设为陆格 (4.5,1.5)、status=Ready、initSpeed=-1，舰船航向
// 已对准正东、既不会转向也过不去。
// 预期：最多 1000 拍内 Executed 且 Failed=true，耗时 ≥ blockedNoProgressMaxTicks-1
// （240 拍），全程中心点不压进陆格。该规则替代旧的“累计受阻 80 拍放弃”，专门
// 兜住真正不可达的机动。
// 覆盖：rows.ToGridCells() 手工地图 + newShipMoveState + runShipMovePath，
// 断言 Executed/Failed/ticks/IsLand。
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

// TestShipMovePathRestoresSpeedAtCruiseSpeed 验证换路径首帧按倍率后的巡航速度恢复航速。
// 场景：config.G.SpeedMultiplier=2，舰船 MaxSpeed=0.03，CruiseSpeed()=0.06；
// 以巡航速度 0.06 创建 NewShipMovePath，genPath 后把 CurSpeed 设为 0.06，
// 执行首帧。
// 预期：CurSpeed 仍为 0.06（InDelta 1e-9），不被未乘倍率的 MaxSpeed=0.03 夹半。
// 旧实现用 MaxSpeed 做上限，在“快”倍速下把舰船砍回半速再慢慢加速，导致航速长期
// 低于巡航速度、慢速鱼雷提前量全错。
// 覆盖：40x20 全水面网格 + NewShipMovePath/genPath + 单次 Exec，断言
// CruiseSpeed 与 CurSpeed。
func TestShipMovePathRestoresSpeedAtCruiseSpeed(t *testing.T) {
	setupGameSettings(t)
	config.G.SpeedMultiplier = 2

	cells := make(grid.Cells, 20)
	for y := range cells {
		cells[y] = make([]int, 40)
	}
	mapCfg := &mapcfg.MapCfg{Cells: cells, Width: 40, Height: 20}

	ship := &objUnit.BattleShip{
		Uid: "ship", CurPos: objPos.NewR(10, 10),
		MaxSpeed: 0.03, Acceleration: 0.001, TotalHP: 100, CurHP: 100,
	}
	misState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: mapCfg},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}

	cruise := ship.CruiseSpeed()
	require.InDelta(t, 0.06, cruise, 1e-9)

	move := NewShipMovePath(ship.Uid, ship.CurPos, objPos.New(30, 10), cruise)
	move.genPath(misState)
	ship.CurSpeed = cruise
	require.NoError(t, move.Exec(misState))
	require.InDelta(t, cruise, ship.CurSpeed, 1e-9, "换路径后不该被砍到未乘倍率的速度")
}
