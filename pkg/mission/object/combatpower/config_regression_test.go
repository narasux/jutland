package combatpower_test

import (
	"testing"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	// 注册初始化顺序，用真实 configs 校验战力结果。
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

const (
	// autocannonDiameter 视为“机炮”的最大口径。
	autocannonDiameter = 45
	// minCheckedShips 保证回归用例真的覆盖到成规模的舰船样本。
	minCheckedShips = 100
)

// TestAntiShipTopContributionIsNotAutocannon 验证机炮不会再主导对舰输出。
// 场景：遍历 objUnit.AllShipNames/ShipMap，只看仍有非机炮对舰手段（主炮、副炮、鱼雷、舰载机）的舰船；
// 口径 ≤45mm（autocannonDiameter）且能在 GunMap/BulletMap 中解析到弹丸的武器视为机炮。
// 预期：每艘入选舰的 AntiShipContributions[0] 都不是机炮；入选样本不得少于 100 艘（minCheckedShips）。
// 覆盖：按真实 configs 计算出的贡献排序第一项 + 样本量下限，防止机炮重新占据榜首。
func TestAntiShipTopContributionIsNotAutocannon(t *testing.T) {
	checked := 0
	for _, name := range objUnit.AllShipNames {
		ship := objUnit.ShipMap[name]
		if ship == nil {
			continue
		}
		contributions := ship.CombatPower.Details.AntiShipContributions
		if len(contributions) == 0 || !hasNonAutocannonContribution(contributions) {
			continue
		}
		checked++
		if top := contributions[0]; isAutocannon(top.Name) {
			t.Fatalf(
				"ship %s top anti-ship contribution is %s (%.1f): autocannon must not lead anti-ship output",
				name, top.Name, top.Value,
			)
		}
	}
	if checked < minCheckedShips {
		t.Fatalf("only %d ships checked, want at least %d", checked, minCheckedShips)
	}
}

// TestReportedCarrierAntiShipOutputIsNotGunDriven 验证航母对舰主要贡献不是机炮，并固定加贺 41% 的回归场景。
// 场景：加贺曾出现 25mm/60 机炮占对舰输出 41%、舰载机只排第二；用例直接读取真实 configs 的 kaga，
// 按 Aircraft.Groups 区分舰载机贡献与 ≤45mm 机炮贡献（autocannonShareLimit=15%）。
// 预期：舰载机 DPS 必须严格大于机炮 DPS，且机炮占 AntiShipContributions 总和的比例 <15%。
// 覆盖：kaga 的 AntiShipContributions 分类求和、空输出保护与份额阈值断言。
func TestReportedCarrierAntiShipOutputIsNotGunDriven(t *testing.T) {
	const autocannonShareLimit = 15.0

	ship := objUnit.ShipMap["kaga"]
	if ship == nil {
		t.Fatal("ship kaga not found")
	}
	planeNames := map[string]bool{}
	for _, group := range ship.Aircraft.Groups {
		planeNames[group.Name] = true
	}

	total, airDPS, autocannonDPS := 0.0, 0.0, 0.0
	for _, c := range ship.CombatPower.Details.AntiShipContributions {
		total += c.Value
		switch {
		case planeNames[c.Name]:
			airDPS += c.Value
		case isAutocannon(c.Name):
			autocannonDPS += c.Value
		}
	}
	if total <= 0 {
		t.Fatal("kaga anti-ship output is empty")
	}
	if airDPS <= autocannonDPS {
		t.Fatalf("kaga air group %.1f should outweigh autocannons %.1f", airDPS, autocannonDPS)
	}
	if share := autocannonDPS / total * 100; share >= autocannonShareLimit {
		t.Fatalf("kaga autocannon anti-ship share = %.1f%%, want below %.0f%%", share, autocannonShareLimit)
	}
}

// TestJapanese25mmRateMatchesHistory 验证九六式 25mm 机炮每管射速符合史实。
// 场景：该炮史实循环射速约 220 发/分（≈3.7 发/秒），是二战主要参战国最慢的 25mm 级防空炮；
// 配置曾出现单装 750 发/分、双联/三联 1200 发/分，比同级高 3~5 倍。
// 预期：JP/25/60/Type96、JP/25/60/2/Type96、JP/25/60/3/Type96 的 1/ReloadTime 均在 3.0~4.2 发/秒，
// 且双联/三联每管射速与单装完全相等（当前 reloadTime 均为 0.27s，即 3.70 发/秒）。
// 覆盖：GunMap 三型炮的射速区间 + 单装/多联每管射速一致性。
func TestJapanese25mmRateMatchesHistory(t *testing.T) {
	const minRate, maxRate = 3.0, 4.2

	perBarrel := map[string]float64{}
	for _, name := range []string{"JP/25/60/Type96", "JP/25/60/2/Type96", "JP/25/60/3/Type96"} {
		gun := objUnit.GunMap[name]
		if gun == nil {
			t.Fatalf("gun %s not found", name)
		}
		perBarrel[name] = 1 / gun.ReloadTime
		if rate := perBarrel[name]; rate < minRate || rate > maxRate {
			t.Fatalf(
				"%s fires %.2f rounds/s per barrel, want %.1f~%.1f (historical 220 rpm)",
				name, rate, minRate, maxRate,
			)
		}
	}
	for _, name := range []string{"JP/25/60/2/Type96", "JP/25/60/3/Type96"} {
		if perBarrel[name] != perBarrel["JP/25/60/Type96"] {
			t.Fatalf(
				"%s per-barrel rate %.2f differs from single mount %.2f",
				name, perBarrel[name], perBarrel["JP/25/60/Type96"],
			)
		}
	}
}

// TestWaterDropAndSwordfishRamWeaponsDifferInAntiAir 验证“水滴可对空、巨型剑鱼只能对舰”的撞击武器区分。
// 场景：waterdrop（1HP、水平/垂直免伤 100%）与 swordfish（100000HP、无减伤）使用同一撞击炮弹
// （SE/GB/1/2048、3 发、0.001s、1.5 射程），但 Impact 对空、SwordfishImpact 仅对舰。
// 预期：Impact 的 AntiShip/AntiAircraft 均为 true，SwordfishImpact 仅 AntiShip；水滴对舰对空战力为正，
// 剑鱼 AntiAir=0 且对舰为正；两舰 Details.AntiShipDPS 相等；水滴 Survival/Total 均高于剑鱼。
// 覆盖：GunMap 两门撞击炮能力位、CombatPower 对舰/对空轴、有效对舰 DPS 相等与免伤舰体生存排序。
func TestWaterDropAndSwordfishRamWeaponsDifferInAntiAir(t *testing.T) {
	waterdrop, swordfish := objUnit.ShipMap["waterdrop"], objUnit.ShipMap["swordfish"]
	if waterdrop == nil || swordfish == nil {
		t.Fatal("waterdrop or swordfish not found")
	}

	impact, swordfishImpact := objUnit.GunMap["Impact"], objUnit.GunMap["SwordfishImpact"]
	if impact == nil || swordfishImpact == nil {
		t.Fatal("Impact or SwordfishImpact gun not found")
	}
	if !impact.AntiShip || !impact.AntiAircraft {
		t.Fatalf("waterdrop Impact must be anti-ship and anti-air: %+v", impact)
	}
	if !swordfishImpact.AntiShip || swordfishImpact.AntiAircraft {
		t.Fatalf("SwordfishImpact must be anti-ship only: %+v", swordfishImpact)
	}

	if waterdrop.CombatPower.AntiShip <= 0 || waterdrop.CombatPower.AntiAir <= 0 {
		t.Fatalf("waterdrop must keep positive anti-ship/anti-air: %+v", waterdrop.CombatPower)
	}
	if swordfish.CombatPower.AntiShip <= 0 || swordfish.CombatPower.AntiAir != 0 {
		t.Fatalf("swordfish must be anti-ship only: %+v", swordfish.CombatPower)
	}
	if waterdrop.CombatPower.Details.AntiShipDPS != swordfish.CombatPower.Details.AntiShipDPS {
		t.Fatalf(
			"effective anti-ship DPS differs: %v vs %v",
			waterdrop.CombatPower.Details.AntiShipDPS, swordfish.CombatPower.Details.AntiShipDPS,
		)
	}
	// 生存与综合战力仍要体现舰体差距：水滴 100% 免伤，生存应高于只有高 HP 的剑鱼；
	// 综合战力随有效生存一起上升，不能再被 1 点 HP 拉低。
	if waterdrop.CombatPower.Survival <= swordfish.CombatPower.Survival ||
		waterdrop.CombatPower.Total <= swordfish.CombatPower.Total {
		t.Fatalf(
			"invulnerable hull must outrank pure high-HP hull: waterdrop = %+v, swordfish = %+v",
			waterdrop.CombatPower, swordfish.CombatPower,
		)
	}
}

func hasNonAutocannonContribution(contributions []objUnit.CombatPowerContribution) bool {
	for _, c := range contributions {
		if diameter := gunBulletDiameter(c.Name); diameter == 0 || diameter > autocannonDiameter {
			return true
		}
	}
	return false
}

func isAutocannon(gunName string) bool {
	diameter := gunBulletDiameter(gunName)
	return diameter > 0 && diameter <= autocannonDiameter
}

func gunBulletDiameter(gunName string) int {
	gun, ok := objUnit.GunMap[gunName]
	if !ok || gun == nil {
		return 0
	}
	bullet, ok := objBullet.Map[gun.BulletName]
	if !ok || bullet == nil {
		return 0
	}
	return bullet.Diameter
}
