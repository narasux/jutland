package instruction

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// hullLandOverlapAt 采样舰体中心线，返回压进陆格的采样点数。
func hullLandOverlapAt(mapCfg *mapcfg.MapCfg, ship *objUnit.BattleShip) int {
	halfLength := ship.Length / 128 / 2
	sinVal := math.Sin(ship.CurRotation * math.Pi / 180)
	cosVal := math.Cos(ship.CurRotation * math.Pi / 180)
	overlap := 0
	for i := -4; i <= 4; i++ {
		t := float64(i) / 4
		px := ship.CurPos.RX + sinVal*halfLength*t
		py := ship.CurPos.RY - cosVal*halfLength*t
		if mapCfg.Map.IsLand(int(math.Floor(px)), int(math.Floor(py))) {
			overlap++
		}
	}
	return overlap
}

// TestShipMovePathPearlHarborBerthsCanEscape 验证珍珠港开局系泊舰都能一次性脱离泊位。
// 场景：metadata.Get("PearlHarbor1941") 的 InitShips 逐艘按 md.Pos/md.Rotation 造舰，
// 用 hullLandOverlapAt 采样中心线筛出舰体压岸的系泊舰（要求 ≥4 艘）；统一命令
// 驶向港外主海域 (59.5,130.5)，每艘最多 14000 拍并让出后台寻路。
// 预期：所有系泊舰都 Executed 且 !Failed，即“点了没反应 / 在岸边磨”不再出现；
// 旧实现会因未吸附起终点或只校验中心线而卡住。
// 覆盖：mapcfg.GetByName("pearl_harbor") + newBattleShip + NewShipMovePath +
// runShipMovePath，统计 moored/escaped 并断言相等。
func TestShipMovePathPearlHarborBerthsCanEscape(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)

	mission := metadata.Get("PearlHarbor1941")
	require.NotEmpty(t, mission.InitShips)

	// 港外开阔水域：珍珠港主海域
	openSea := objPos.NewR(59.5, 130.5)
	moored, escaped := 0, 0
	for _, md := range mission.InitShips {
		ship := newBattleShip("berth/"+md.ShipName, md.Pos, md.Rotation)
		if hullLandOverlapAt(mapCfg, ship) == 0 {
			continue
		}
		moored++
		misState := newShipMoveState(mapCfg, ship)
		move := NewShipMovePath(ship.Uid, ship.CurPos, openSea, 0)
		runShipMovePath(t, misState, move, 14000)
		if move.Executed() && !move.Failed() {
			escaped++
			continue
		}
		t.Logf("系泊舰 %s (%d,%d) rot=%.0f 未能脱离：failed=%v 距目标=%.2f pos=%s",
			md.ShipName, md.Pos.MX, md.Pos.MY, md.Rotation, move.Failed(),
			ship.CurPos.Distance(openSea), ship.CurPos.String())
	}
	require.GreaterOrEqual(t, moored, 4, "前提：珍珠港应有若干系泊中（舰体压岸）的舰船")
	require.Equal(t, moored, escaped, "系泊舰必须都能一次性脱离泊位")
}

// TestShipMovePathCoastalTransitKeepsHullOffLand 验证沿岸长距离机动全程舰体不压岸。
// 场景：珍珠港战列舰 HA/BB-14 在 (45.5,126.5)、航向 90°，目标 (90.5,126.5)；
// 最多 9000 拍逐帧 Exec，每拍用 hullLandOverlapAt 采样中心线统计压岸拍数。
// 预期：指令完成且不判 Failed，抵达目标 1 格内，landTicks=0。旧实现合并航线时
// 只校验中心线，舰体偏离航线半个格就会切进陆格、顶着岸边磨。
// 覆盖：mapcfg.GetByName("pearl_harbor") + NewShipMovePath + 手动推进循环，
// 断言 Executed/!Failed/距离/landTicks。
func TestShipMovePathCoastalTransitKeepsHullOffLand(t *testing.T) {
	setupGameSettings(t)
	mapCfg := mapcfg.GetByName("pearl_harbor")
	require.NotNil(t, mapCfg)

	ship := newBattleShip("HA/BB-14", objPos.NewR(45.5, 126.5), 90)
	misState := newShipMoveState(mapCfg, ship)
	target := objPos.NewR(90.5, 126.5)
	move := NewShipMovePath(ship.Uid, ship.CurPos, target, 0)

	landTicks, ticks := 0, 0
	for tick := 0; tick < 9000; tick++ {
		ticks = tick
		_ = move.Exec(misState)
		if hullLandOverlapAt(mapCfg, ship) > 0 {
			landTicks++
		}
		if move.Executed() {
			break
		}
		if move.status == Preparing {
			time.Sleep(time.Millisecond)
		}
	}

	require.True(t, move.Executed(), "沿岸长机动应正常收尾")
	require.False(t, move.Failed(), "沿岸长机动不应判为不可达")
	require.LessOrEqual(t, ship.CurPos.Distance(target), 1.0, "应抵达目标附近")
	require.Zero(t, landTicks, "沿岸航线不应让舰体压上岸（%d/%d 拍压岸）", landTicks, ticks)
}
