package initialize

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestI400SeiranWiring 验证伊-400 与晴岚的配置引用链完整。
// 场景：伊-400 按史实搭载三架晴岚，从艇艏弹射器放飞、返航在艇侧水面回收；
// 舰船配置、飞机配置、甲板模板三者任一改名或漏配，初始化阶段不会报错，
// 但游戏里要么没有飞机要么无法起降。
// 预期：ShipMap[i_400] 有 3 架 M6A1 库存、甲板模板解析为舷侧着水回收、
// 弹射点存在；PlaneMap[M6A1] 是日本鱼雷轰炸机并带一枚航空鱼雷。
// 覆盖：直接读初始化后的全局模板表，断言库存数、HasPlane、deck 模式、
// 机种/国籍与鱼雷挂载数量。
func TestI400SeiranWiring(t *testing.T) {
	ship, ok := objUnit.ShipMap["i_400"]
	if !ok {
		t.Fatalf("i_400 missing from ShipMap")
	}
	if !ship.Aircraft.HasPlane {
		t.Fatalf("i_400 has no aircraft group")
	}
	if len(ship.Aircraft.Groups) != 1 || ship.Aircraft.Groups[0].Name != "M6A1" {
		t.Fatalf("i_400 aircraft groups = %+v, want one M6A1 group", ship.Aircraft.Groups)
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 3 {
		t.Fatalf("i_400 seiran stock = %d, want 3", got)
	}
	deck, ok := objUnit.DeckMap[ship.Aircraft.Deck]
	if !ok {
		t.Fatalf("i_400 deck %q missing from DeckMap", ship.Aircraft.Deck)
	}
	if deck.Landing.Mode != objUnit.LandingModeSea {
		t.Fatalf("i_400 landing mode = %q, want sea recovery", deck.Landing.Mode)
	}
	if got := ship.Aircraft.TakeoffPointCount(); got != 1 {
		t.Fatalf("i_400 takeoff points = %d, want 1 catapult", got)
	}

	plane, ok := objUnit.PlaneMap["M6A1"]
	if !ok {
		t.Fatalf("M6A1 missing from PlaneMap")
	}
	if plane.Type != objUnit.PlaneTypeTorpedoBomber {
		t.Fatalf("M6A1 type = %q, want torpedo_bomber", plane.Type)
	}
	if plane.Nation != objUnit.NationJP {
		t.Fatalf("M6A1 nation = %q, want jp", plane.Nation)
	}
	if len(plane.Weapon.Torpedoes) != 1 {
		t.Fatalf("M6A1 torpedoes = %d, want 1", len(plane.Weapon.Torpedoes))
	}
}
