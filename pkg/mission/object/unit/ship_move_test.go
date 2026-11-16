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

// TestMoveToBlocksEnteringLand 验证从水里朝陆地格直线开必须被拦停，位置和速度都不能变。
// 场景：8×3 海岸地图（中间行 (3,1)/(4,1) 为海岸格 C，其余浅海 S）；舰船中心在 (2.5,1.5)
// 即 (2,1) 水面格，航向 90（正东）、CurSpeed=MaxSpeed=0.02，目标 (4.5,1.5) 正在海岸格另一侧。
// 预期：200 帧内某一帧 MoveTo 返回 blocked=true；拦停时中心点仍在水中（Map.IsLand(MX,MY)=false，
// 紧贴岸边），CurSpeed 归零；未拦停的帧 arrive 均为 false。
// 覆盖：循环 200 次 MoveTo(mapCfg, (4.5,1.5), false)，记录 blocked 并断言中心点未压陆、速度清零。
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

// TestMoveToAllowsStrandedEscape 验证已经搁浅（中心点在海岸格上）的舰船允许移动，用于退出搁浅。
// 场景：舰船中心压在 (3.5,1.5) 即 (3,1) 海岸格上，航向 270（正西）朝 (2.5,1.5) 水面后退，
// 调用时传入 stranded=true；WaterDrop 特例走另一条分支（见 TestMoveToWaterDropCanCrossLand）。
// 预期：首帧 MoveTo 返回 blocked=false、arrive=false；持续后退 100 帧内中心点离开海岸格回到水面。
// 覆盖：调用 MoveTo(mapCfg, (2.5,1.5), true) 断言未拦停，再循环直到 Map.IsLand 为 false。
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

// TestMoveToWaterDropCanCrossLand 验证 WaterDrop 特例可以穿越陆地。
// 场景：小黄鸭/水滴这类特殊单位不遵守舰船地形限制；若统一按陆地阻挡，它们会被卡在岸上。
// 预期：从 (2.5,1.5) 朝东越过海岸格移动，blocked 为 false。
// 覆盖：构造 WaterDrop 舰船和 8x3 海岸地图，调用 MoveTo 并断言未被拦。
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

// TestMoveToStrandedShipOnlyApproachesTarget 验证搁浅舰只能朝目标点靠近，朝远离目标的方向开必须被拦停。
// 场景：中心点压在 (3,1) 海岸格、目标在正西 (2.5,1.5)；若允许随意移动，舰体会在陆地上横向漂移
// （实测会沿珍珠港岸线一路漂进另一片不连通的港湾，之后寻路必然失败）。转向不受限制：
// 原地转到目标方向后必须能退出来。
// 预期：航向 90（正东、远离目标）时 blocked=true 且 CurPos.RX 保持 3.5 不变；
// 原地转向 270 后 blocked=false，RX < 3.5，真正向水面移动。
// 覆盖：同一艘搁浅舰两次 MoveTo(mapCfg, target, true)，断言 blocked 与位置变化。
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

// TestMoveToMooredShipCannotPushDeeperIntoLand 验证舰体已经压上岸的舰船不允许把压岸程度进一步加深，
// 但仍必须能平移与退出来。
// 场景：10×6 地图 y=2 整行海岸；舰船中心 (5.5,3.206) 在水面，Length=210、Width=33，
// 舰艏朝正北已经压进 (5,2) 海岸格（hullLandOverlap > 0），像系泊位或擦岸状态。
// 预期：朝岸里 (5.5,1.0) 移动 blocked=true 且 CurPos.RY 保持 3.206；
// 掉头航向 180 朝外海 (5.5,5.5) 移动 blocked=false 且 RY > 3.206；否则系泊中的战舰会被永久锁死在岸边。
// 覆盖：先断言前提（中心点在水面、压岸采样 > 0），再两次 MoveTo(stranded=true) 断言拦停与可退出。
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

// TestHullLandOverlapSamples 验证 HullLandOverlapSamples 的舰体压岸采样：
// 完全在水面时为 0，舰艏或舰艉压上岸时必须 > 0，且按传入的半长采样。
// 场景：该函数是启动期配置校验与运行期阻挡判定共用的采样；8×3 地图 x=3、x=4 为海岸格，
// 全长参数分别取 block×2（256px → 半长 1 格）与 block/2（64px → 半长 0.25 格）。
// 预期：开阔水面 (1,0.5) 朝东 → 0；从 (2.3,1.5) 朝东（半长 1 格 → 舰艏到 x=3.3）→ > 0；
// 中心在水面但舰艉压岸 (4.8,1.5) → > 0；同一位置改传 block/2 的短半长 → 0。
// 覆盖：四次调用 HullLandOverlapSamples(cfg, pos, rotation, halfLength)，
// 断言全水面为 0、压岸 > 0、短半长不压岸为 0。
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
