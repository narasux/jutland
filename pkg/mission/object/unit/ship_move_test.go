package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

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
