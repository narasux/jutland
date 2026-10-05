package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// 构造 8x3 测试地图：中间一行 x=3、x=4 是海岸格，其余为浅海。
func newCoastMapCfg() *mapcfg.MapCfg {
	rows := mapcfg.MapData{
		"SSSSSSSS",
		"SSSCCSSS",
		"SSSSSSSS",
	}
	return &mapcfg.MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 8, Height: 3}
}

// 从水里朝陆地格直线开：必须被拦停（blocked），位置和速度都不能变。
func TestMoveToBlocksEnteringLand(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := newCoastMapCfg()
	// 舰在 (2.5,1.5) 即 (2,1) 浅海格，正东是 (3,1) 海岸格
	ship := &BattleShip{
		CurHP:        100,
		CurPos:       objPos.NewR(2.5, 1.5),
		CurRotation:  90,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
	}

	var blocked bool
	for i := 0; i < 200; i++ {
		arrive, stepBlocked := ship.MoveTo(mapCfg, objPos.NewR(4.5, 1.5), false)
		if stepBlocked {
			blocked = true
			break
		}
		require.False(t, arrive)
	}
	require.True(t, blocked, "朝陆地格移动必须被拦停")
	// 拦停时中心点仍留在水里（紧贴岸边），速度清零
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "拦停时中心点不应压进陆地格")
	require.Equal(t, 0.0, ship.CurSpeed)
}

// 已经搁浅（中心点在海岸格上）的舰船允许移动，用于退出搁浅；换 WaterDrop 特例不受影响。
func TestMoveToAllowsStrandedEscape(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := newCoastMapCfg()
	// 中心点压在 (3,1) 海岸格上，朝正西的水面退
	ship := &BattleShip{
		CurHP:        100,
		CurPos:       objPos.NewR(3.5, 1.5),
		CurRotation:  270,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
	}
	arrive, blocked := ship.MoveTo(mapCfg, objPos.NewR(2.5, 1.5), true)
	require.False(t, blocked, "搁浅舰向水面后退不应被拦")
	require.False(t, arrive)
	// 持续后退，中心点应能离开海岸格回到水中
	for i := 0; i < 100 && mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY); i++ {
		ship.MoveTo(mapCfg, objPos.NewR(2.5, 1.5), true)
	}
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "搁浅舰应能退回水中")
	require.False(t, blocked)
}

func TestMoveToWaterDropCanCrossLand(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := newCoastMapCfg()
	ship := &BattleShip{
		TypeAbbr:     "WaterDrop",
		CurHP:        100,
		CurPos:       objPos.NewR(2.5, 1.5),
		CurRotation:  90,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
	}
	_, blocked := ship.MoveTo(mapCfg, objPos.NewR(4.5, 1.5), false)
	require.False(t, blocked, "可上陆的特殊船不受地形限制")
}

// 搁浅舰只能朝目标点靠近：朝着远离目标的方向开必须被拦停，否则舰体会在陆地上
// 横向漂移（实测会沿珍珠港岸线一路漂进另一片不连通的港湾，之后寻路必然失败）。
// 转向不受限制：原地转到目标方向后必须能退出来。
func TestMoveToStrandedShipOnlyApproachesTarget(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := newCoastMapCfg()
	// 中心点压在 (3,1) 海岸格上，目标在正西的水面
	ship := &BattleShip{
		CurHP:        100,
		CurPos:       objPos.NewR(3.5, 1.5),
		CurRotation:  90,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
	}
	target := objPos.NewR(2.5, 1.5)

	// 舰艏朝正东（远离目标）：必须被拦停，位置不变
	_, blocked := ship.MoveTo(mapCfg, target, true)
	require.True(t, blocked, "搁浅舰朝远离目标的方向移动必须被拦停")
	require.Equal(t, 3.5, ship.CurPos.RX)

	// 原地转向朝西后，必须能朝目标退出去
	ship.CurRotation = 270
	_, blocked = ship.MoveTo(mapCfg, target, true)
	require.False(t, blocked, "搁浅舰朝目标移动不应被拦")
	require.Less(t, ship.CurPos.RX, 3.5, "应真正向水面移动")
}

// 舰体已经压上岸的舰船（系泊位、擦岸）不允许把压岸程度进一步加深，
// 但仍必须能平移与退出来，否则系泊中的战舰会被永久锁死在岸边。
func TestMoveToMooredShipCannotPushDeeperIntoLand(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	rows := mapcfg.MapData{
		"SSSSSSSSSS",
		"SSSSSSSSSS",
		"CCCCCCCCCC",
		"SSSSSSSSSS",
		"SSSSSSSSSS",
		"SSSSSSSSSS",
	}
	mapCfg := &mapcfg.MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 10, Height: 6}
	// 中心点在水里 (5,3)，舰艏（正北）已经压进 (5,2) 海岸格
	ship := &BattleShip{
		CurHP:        100,
		CurPos:       objPos.NewR(5.5, 3.206),
		CurRotation:  0,
		CurSpeed:     0.02,
		MaxSpeed:     0.02,
		Acceleration: 0.001,
		RotateSpeed:  10,
		Length:       210,
		Width:        33,
	}
	require.False(t, mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY), "前提：中心点应在水面")
	overlap := ship.hullLandOverlap(mapCfg, ship.CurPos, ship.CurRotation)
	require.Greater(t, overlap, 0, "前提：舰艏应已压上岸")

	// 继续朝岸里挤：压岸采样点增加，必须被拦停且位置不变
	_, blocked := ship.MoveTo(mapCfg, objPos.NewR(5.5, 1.0), true)
	require.True(t, blocked, "不允许把压岸程度进一步加深")
	require.Equal(t, 3.206, ship.CurPos.RY)

	// 掉头朝外海退：必须能走
	ship.CurRotation = 180
	_, blocked = ship.MoveTo(mapCfg, objPos.NewR(5.5, 5.5), true)
	require.False(t, blocked, "系泊中的舰船必须能退出来")
	require.Greater(t, ship.CurPos.RY, 3.206)
}

// HullLandOverlapSamples 是启动期配置校验与运行期阻挡判定共用的舰体压岸采样：
// 完全在水面时为 0，舰艏压上岸时必须 > 0（按半长采样，不能把整舰长当半长）。
func TestHullLandOverlapSamples(t *testing.T) {
	cfg := newCoastMapCfg()
	// 8x3 地图，x=3、x=4 是海岸格；半长 1 格 = 1 个 MapBlockSize
	const block = constants.MapBlockSize

	// 舰体在开阔水面（y=0 行，横向），完全不压岸
	require.Equal(t, 0, HullLandOverlapSamples(cfg, objPos.NewR(1, 0.5), 90, block*2))
	// 舰艏越过海岸格（从 (2.3,1.5) 朝东，半长 1 格 → 舰艏到 x=3.3）
	require.Greater(t, HullLandOverlapSamples(cfg, objPos.NewR(2.3, 1.5), 90, block*2), 0)
	// 中心点在水面、但舰艉压在岸上（从 (4.8,1.5) 朝东，舰艉回到 x=3.8 海岸格）
	require.Greater(t, HullLandOverlapSamples(cfg, objPos.NewR(4.8, 1.5), 90, block*2), 0)
	// 传入的是整舰长：同一位置下，舰体够长才会压上岸
	require.Greater(t, HullLandOverlapSamples(cfg, objPos.NewR(2.3, 1.5), 90, block*2), 0)
	require.Equal(t, 0, HullLandOverlapSamples(cfg, objPos.NewR(2.3, 1.5), 90, block/2))
}
