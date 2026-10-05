package initialize

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// NewShip 通过 deepcopy 复制舰船模板；鱼雷扇形的槽位必须随实例保留，
// 否则 ooi 等多发射器舰船进入战斗后又会把所有鱼雷打向同一点。
func TestOoiTorpedoFanSurvivesNewShipCopy(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	generator := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	shooter := objUnit.NewShip(
		generator, "ooi", objPos.NewR(10, 10), 0, faction.HumanAlpha,
	)
	target := &objUnit.BattleShip{
		Uid:          "target",
		Type:         objUnit.ShipTypeDestroyer,
		CurHP:        100,
		TotalHP:      100,
		CurPos:       objPos.NewR(14, 10),
		BelongPlayer: faction.ComputerAlpha,
	}

	totalSlots := 0
	seenSlots := map[int]bool{}
	for _, launcher := range shooter.Weapon.Torpedoes {
		if launcher.FanSlotCount != 40 {
			t.Fatalf("ooi fan slot count = %d, want 40", launcher.FanSlotCount)
		}
		if seenSlots[launcher.FanSlot] {
			t.Fatalf("duplicate ooi fan start slot %d", launcher.FanSlot)
		}
		seenSlots[launcher.FanSlot] = true
		totalSlots += launcher.BulletCount
	}
	if totalSlots != 40 {
		t.Fatalf("ooi torpedo tubes = %d, want 40", totalSlots)
	}

	angles := map[float64]bool{}
	for _, bullet := range shooter.Fire(target) {
		if bullet.Type != objBullet.TypeTorpedo {
			continue
		}
		angles[bullet.CurPos.Angle(bullet.TargetPos)] = true
	}
	if len(angles) != 5 {
		t.Fatalf("ooi starboard fan produced %d unique angles, want 5", len(angles))
	}
}

// 回归：所有任务的初始舰位中心点必须在可航行水面上（启动期校验会直接报错），
// 舰体压岸只允许出现在有意的系泊位，并且数量、舰名要可追溯。
func TestMissionInitShipsStayOnWater(t *testing.T) {
	moored := map[string][]string{}
	for _, mission := range metadata.AllMissions() {
		md := metadata.Get(mission)
		if md.MapCfg == nil {
			t.Fatalf("任务 %q 缺少地图配置", mission)
		}
		for _, initShip := range md.InitShips {
			ship, ok := objUnit.ShipMap[initShip.ShipName]
			if !ok {
				t.Fatalf("任务 %q 初始舰 %q 没有对应模板", mission, initShip.ShipName)
			}
			if md.MapCfg.Map.IsLand(initShip.Pos.MX, initShip.Pos.MY) {
				t.Fatalf(
					"任务 %q 初始舰 %q 中心点落在陆格 (%d,%d)",
					mission, initShip.ShipName, initShip.Pos.MX, initShip.Pos.MY,
				)
			}
			if objUnit.HullLandOverlapSamples(
				md.MapCfg, initShip.Pos, initShip.Rotation, ship.Length,
			) > 0 {
				moored[mission] = append(moored[mission], initShip.ShipName)
			}
		}
	}
	// 珍珠港 1941 的码头泊位是有意贴着岸的，其余任务不应出现舰体压岸
	for mission, ships := range moored {
		if mission != "PearlHarbor1941" {
			t.Fatalf("任务 %q 出现非预期舰体压岸：%v", mission, ships)
		}
	}
	t.Logf("舰体压岸（系泊位）：%v", moored)
}
