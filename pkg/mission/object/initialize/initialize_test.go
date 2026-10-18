package initialize

import (
	"fmt"
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// NewShip 通过 deepcopy 复制舰船模板；鱼雷扇形的分组与航迹必须随实例保留，
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
	seenTrack := map[string]bool{}
	for _, launcher := range shooter.Weapon.Torpedoes {
		// ooi 左舷 5 座 / 右舷 5 座，每舷 20 根管：同一射界的发射器共用一张扇形，
		// 组内交错分航迹，所以 FanTrack 在组内唯一、组内座数就是 FanStride。
		if launcher.FanTracks != 20 || launcher.FanStride != 5 {
			t.Fatalf("ooi fan group = %d tracks / %d stride, want 20 / 5",
				launcher.FanTracks, launcher.FanStride)
		}
		key := fmt.Sprintf("%v|%v|%d", launcher.LeftFiringArc, launcher.RightFiringArc, launcher.FanTrack)
		if seenTrack[key] {
			t.Fatalf("duplicate ooi fan track %s", key)
		}
		seenTrack[key] = true
		totalSlots += launcher.BulletCount
	}
	if totalSlots != 40 {
		t.Fatalf("ooi torpedo tubes = %d, want 40", totalSlots)
	}

	angles := map[float64]bool{}
	hasAimLine := false
	for _, bullet := range shooter.Fire(target) {
		if bullet.Type != objBullet.TypeTorpedo {
			continue
		}
		angles[bullet.Rotation] = true
		// 扇面是围绕出管位置旋转的，所以相对瞄准线的偏移也要从出管位置量。
		if math.Abs(torpedoOffsetFromAim(bullet, target)) < 1e-6 {
			hasAimLine = true
		}
	}
	if len(angles) != 5 {
		t.Fatalf("ooi starboard fan produced %d unique angles, want 5", len(angles))
	}
	if !hasAimLine {
		t.Fatal("ooi 首发齐射没有落在瞄准线上的那一根")
	}
}

// 每艘舰的首发鱼雷齐射都必须有一根压在瞄准线上，且每根都落在最内侧航迹范围内。
// 回归：旧实现按 (槽位 - (管数-1)/2) 排扇形，porter 2x4 的首发是 −12° / +1.71°，
// maury 右舷齐射是 −12.03° / +0.8°（有一根打到瞄准线反侧），正中那根永远空缺。
func TestFirstTorpedoVolleyCoversTheAimLine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	checked := 0
	for _, name := range objUnit.AllShipNames {
		ship := objUnit.ShipMap[name]
		if ship == nil || !ship.Weapon.HasTorpedo {
			continue
		}
		// 左右舷各试一次：只有射界覆盖该方位的发射器会开火。
		for _, bearing := range []float64{90, 270} {
			generator := objUnit.NewShipUidGenerator(faction.HumanAlpha)
			shooter := objUnit.NewShip(generator, name, objPos.NewR(0, 0), 0, faction.HumanAlpha)
			distance := 0.4 * shooter.Weapon.Torpedoes[0].Range
			rad := bearing * math.Pi / 180
			target := &objUnit.BattleShip{
				Uid: "target", Type: objUnit.ShipTypeDestroyer,
				CurHP: 100, TotalHP: 100, Length: 104, Width: 11,
				CurPos:       objPos.NewR(math.Sin(rad)*distance, -math.Cos(rad)*distance),
				BelongPlayer: faction.ComputerAlpha,
			}

			var offsets []float64
			for _, bullet := range shooter.Fire(target) {
				if bullet.Type != objBullet.TypeTorpedo {
					continue
				}
				offsets = append(offsets, torpedoOffsetFromAim(bullet, target))
			}
			if len(offsets) == 0 {
				continue
			}
			checked++

			hasCenter := false
			for _, offset := range offsets {
				if math.Abs(offset) < 1e-6 {
					hasCenter = true
				}
			}
			if !hasCenter {
				t.Fatalf("%s 朝 %.0f° 的首发齐射 %v 没有瞄准线上的那一根", name, bearing, offsets)
			}
			// 首发是每座的最内侧航迹：不会超出“基础角 x 齐射根数的一半”。
			limit := 4.0 * math.Ceil(float64(len(offsets))/2)
			for _, offset := range offsets {
				if math.Abs(offset) > limit+1e-6 {
					t.Fatalf("%s 朝 %.0f° 的首发偏移 %.2f 超出最内侧航迹 %.2f", name, bearing, offset, limit)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("没有检查到任何鱼雷齐射")
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

// torpedoOffsetFromAim 鱼雷相对瞄准线的角度偏移，归一到 (-180, 180]。
// 目标静止时提前量就是目标当前位置，而扇面是围绕出管位置旋转的，所以必须从
// 弹药的出管位置量，不能从舰体中心量（两者对目标的角度差可达几度）。
func torpedoOffsetFromAim(bullet *objBullet.Bullet, target *objUnit.BattleShip) float64 {
	aim := bullet.CurPos.Angle(target.CurPos)
	return math.Mod(bullet.CurPos.Angle(bullet.TargetPos)-aim+540, 360) - 180
}
