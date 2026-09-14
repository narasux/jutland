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

// TestAntiShipTopContributionIsNotAutocannon 防止机炮再次主导对舰输出。
// 只要这艘船还有非机炮的对舰手段（主炮、副炮、鱼雷、舰载机），
// 对舰第一贡献就不应该是口径 ≤45mm 的机炮。
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

// TestReportedCarrierAntiShipOutputIsNotGunDriven 固定“航母对舰主要贡献是机炮”的回归用例。
// 加贺曾出现 25mm/60 机炮占对舰输出 41%、舰载机只排第二的情况。
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

// TestJapanese25mmRateMatchesHistory 固定九六式 25mm 机炮的每管射速。
// 该炮史实循环射速约 220 发/分（≈3.7 发/秒），是二战主要参战国里最慢的 25mm 级
// 防空炮；配置里曾出现单装 750 发/分、双联/三联 1200 发/分的数值，比同级高 3~5 倍。
// 单装、双联、三联共用同一门炮，每管射速必须一致。
func TestJapanese25mmRateMatchesHistory(t *testing.T) {
	const minRate, maxRate = 3.0, 4.2

	perBarrel := map[string]float64{}
	for _, name := range []string{"JP/25/60", "JP/25/60/2", "JP/25/60/3"} {
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
	for _, name := range []string{"JP/25/60/2", "JP/25/60/3"} {
		if perBarrel[name] != perBarrel["JP/25/60"] {
			t.Fatalf(
				"%s per-barrel rate %.2f differs from single mount %.2f",
				name, perBarrel[name], perBarrel["JP/25/60"],
			)
		}
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
