package metadata

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestASWTestInitialCameraShowsOpeningZoneShip 验证 ASWTest 的 P1 初始镜头覆盖北区开场舰 johnston。
// 场景：ASWTest 三个试验区相距较远，无法同屏显示；镜头聚焦北区 johnston (18,18)，其余区按 M 查看。
// 预期：johnston 落在 [cam.RX, cam.RX+10]×[cam.RY, cam.RY+5.625] 视口内，且 P1 初始舰数为 4（三区各舰）；
// 失败意味着开场镜头没有对准正在操作的北区主力，玩家会先看到空海。
// 覆盖：Get("ASWTest") 后调用 CameraPosForPlayerSide(SideP1)，遍历 InitShips 筛出 johnston 并断言四边界，
// 同时统计 P1 舰数。
func TestASWTestInitialCameraShowsOpeningZoneShip(t *testing.T) {
	const (
		// 默认缩放 1x 时，1280x720 屏幕大约显示 10x5.6 格。
		viewWidth  = 1280.0 / 128.0
		viewHeight = 720.0 / 128.0
	)

	md := Get("ASWTest")
	cam := md.CameraPosForPlayerSide(faction.SideP1)

	playerShips := 0
	openingFound := false
	for _, init := range md.InitShips {
		if init.BelongSide != faction.SideP1 {
			continue
		}
		playerShips++
		if init.ShipName != "johnston" {
			continue
		}
		openingFound = true
		require.GreaterOrEqual(t, init.Pos.RX, cam.RX, "johnston is left of the initial camera")
		require.LessOrEqual(t, init.Pos.RX, cam.RX+viewWidth, "johnston is right of the initial camera")
		require.GreaterOrEqual(t, init.Pos.RY, cam.RY, "johnston is above the initial camera")
		require.LessOrEqual(t, init.Pos.RY, cam.RY+viewHeight, "johnston is below the initial camera")
	}
	require.True(t, openingFound, "ASWTest 缺少北区开场舰 johnston")
	require.Equal(t, 4, playerShips, "ASWTest 应有三区共四艘 P1 初始舰")
}

// TestASWTestZonesStaySeparated 验证 ASWTest 三个试验区之间不会跨区接敌。
// 场景：ASWTest 为快速测试把区内双方拉到 32～37 格；三个区按北/西南/东分开，若区间过近，AI 会跨区追目标。
// 预期：不同区的敌我最近距离 ≥50 格（约 6.4km）；区内 32 格是刻意例外。失败说明需要整体平移某个试验区。
// 覆盖：按 x>90 / y>90 / 其余 划分东/西南/北三个区，收集 P1/P2 后只比较不同区之间的敌我距离，取最小值断言。
func TestASWTestZonesStaySeparated(t *testing.T) {
	const minCrossZone = 50.0

	md := Get("ASWTest")
	type zonePoint struct {
		zone string
		pos  objPos.MapPos
	}
	zoneOf := func(p objPos.MapPos) string {
		switch {
		case p.RX > 90:
			return "east"
		case p.RY > 90:
			return "southwest"
		default:
			return "north"
		}
	}

	var player, enemy []zonePoint
	for _, init := range md.InitShips {
		switch init.BelongSide {
		case faction.SideP1:
			player = append(player, zonePoint{zoneOf(init.Pos), init.Pos})
		case faction.SideP2:
			enemy = append(enemy, zonePoint{zoneOf(init.Pos), init.Pos})
		}
	}
	require.NotEmpty(t, player)
	require.NotEmpty(t, enemy)

	nearest := math.Inf(1)
	for _, mine := range player {
		for _, foe := range enemy {
			if mine.zone == foe.zone {
				continue
			}
			nearest = math.Min(nearest, mine.pos.Distance(foe.pos))
		}
	}
	require.GreaterOrEqual(
		t, nearest, minCrossZone,
		"ASWTest 跨区最近敌我舰仅相距 %.1f 格", nearest,
	)
}

// TestSubmarineTestKeepsOpeningSeparation 验证潜艇试验场双方开场不贴脸，留出展开、下潜和观察时间。
// 场景：SubmarineTest 四个试验区仍按历史/演练节奏布置，区内双方开场约 60 格（约 7.7km）。
// 预期：最近敌我舰距离 ≥50 格（约 6.4km）；低于阈值说明布阵把双方放进开局即交火的距离。
// 覆盖：Get("SubmarineTest") 收集 P1/P2 全部初始舰位，双重循环取 math.Min，再用 require.GreaterOrEqual 断言。
func TestSubmarineTestKeepsOpeningSeparation(t *testing.T) {
	const minSeparation = 50.0

	md := Get("SubmarineTest")
	var player, enemy []objPos.MapPos
	for _, init := range md.InitShips {
		switch init.BelongSide {
		case faction.SideP1:
			player = append(player, init.Pos)
		case faction.SideP2:
			enemy = append(enemy, init.Pos)
		}
	}
	require.NotEmpty(t, player)
	require.NotEmpty(t, enemy)

	nearest := math.Inf(1)
	for _, mine := range player {
		for _, foe := range enemy {
			nearest = math.Min(nearest, mine.Distance(foe))
		}
	}
	require.GreaterOrEqual(
		t, nearest, minSeparation,
		"SubmarineTest 最近敌我舰仅相距 %.1f 格", nearest,
	)
}
