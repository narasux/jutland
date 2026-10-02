package instruction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/metadata"
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
