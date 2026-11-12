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

// 本期落地的两款制导鱼雷必须在配置里显式开启制导；普通鱼雷保持无制导。
func TestGuidedTorpedoConfig(t *testing.T) {
	guided := map[string]float64{
		"JP/TB/Kaiten/1944/Type1": 120,
		"US/TB/324/1957/Mk44":     150,
	}
	for name, wantTurnRate := range guided {
		bt := objBullet.Map[name]
		if bt == nil {
			t.Fatalf("bullet %q not found", name)
		}
		if !bt.IsGuided() {
			t.Fatalf("bullet %q is not guided", name)
		}
		if bt.Guidance.Mode != objBullet.GuidanceModePursuit {
			t.Fatalf("bullet %q mode = %q, want pursuit", name, bt.Guidance.Mode)
		}
		if bt.Guidance.TurnRate != wantTurnRate {
			t.Fatalf("bullet %q turnRate = %v, want %v", name, bt.Guidance.TurnRate, wantTurnRate)
		}
	}

	// 普通鱼雷不能被误加制导。
	for _, name := range []string{"US/TB/533/1935/Mk15", "JP/TB/610/1933/Type93Mod1"} {
		if bt := objBullet.Map[name]; bt != nil && bt.IsGuided() {
			t.Fatalf("bullet %q should stay unguided", name)
		}
	}
}

// 反潜射程是平衡拍板值（2026-11-10）：舰载深弹架 2 格、RBU 4 格、Mk44 8 格。
// 配置里的 range 是公里数，初始化时统一 ÷2；这里锁住最终格数，避免改配置时算错。
func TestASWWeaponRangesAfterScaling(t *testing.T) {
	cases := []struct {
		ship string
		want float64
	}{
		{"gurkha", 2},     // 英 Mk VII 深弹架：配置 4 km
		{"porter", 2},     // 美 Mk 6 深弹架：配置 4 km
		{"hammann", 2},    // 同上
		{"jiujiang", 4},   // RBU 反潜火箭：配置 8 km
		{"long_beach", 8}, // Mk32 反潜鱼雷：配置 16 km
	}
	for _, tc := range cases {
		ship, ok := objUnit.ShipMap[tc.ship]
		if !ok {
			t.Fatalf("ship template %s missing", tc.ship)
		}
		if math.Abs(ship.Weapon.MaxToSubmergedRange-tc.want) > 1e-9 {
			t.Fatalf("%s MaxToSubmergedRange = %v, want %v", tc.ship, ship.Weapon.MaxToSubmergedRange, tc.want)
		}
		if !ship.Weapon.HasAntiSubWeapon {
			t.Fatalf("%s should be flagged as anti-submarine capable", tc.ship)
		}
	}
}

// 三艘潜艇的水下航速按 Wikipedia 的实测值配置（2026-11-10）：
// VIIC 7.6 kn、I-168 8.2 kn、Nautilus 8.0 kn；配置用“水面航速 × 倍率”表达。
func TestSubmarineSubmergedSpeedMatchesHistory(t *testing.T) {
	cases := []struct {
		ship string
		want float64
	}{
		{"u_boat_type_viic", 7.6},
		{"i_168", 8.2},
		{"uss_nautilus", 8.0},
	}
	for _, tc := range cases {
		ship, ok := objUnit.ShipMap[tc.ship]
		if !ok {
			t.Fatalf("ship template %s missing", tc.ship)
		}
		// MaxSpeed 初始化时已除以 ShipSpeedScale，这里还原成配置节数。
		kn := ship.MaxSpeed * objUnit.ShipSpeedScale * ship.SubmergedSpeedRatio
		if math.Abs(kn-tc.want) > 0.1 {
			t.Fatalf("%s submerged speed = %.2f kn, want %.1f kn", tc.ship, kn, tc.want)
		}
	}
}

// 所有潜艇模板默认入场即水下；需要水面开局时必须显式配置 surface。
func TestSubmarineTemplatesDefaultSubmerged(t *testing.T) {
	count := 0
	for name, ship := range objUnit.ShipMap {
		if ship.Type != objUnit.ShipTypeSubmarine {
			continue
		}
		count++
		if ship.DiveState != objUnit.DiveStateSubmerged {
			t.Fatalf("submarine %s default dive state = %v, want submerged", name, ship.DiveState)
		}
	}
	if count == 0 {
		t.Fatal("no submarine templates loaded")
	}
}
