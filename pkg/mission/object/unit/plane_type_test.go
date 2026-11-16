package unit

import (
	"testing"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/mission/object"
)

// TestPlaneWeaponReleaseIntervalUnmarshalsFractionalSeconds 验证 releaseInterval 支持 JSON5 小数秒。
// 场景：飞机配置写 releaseInterval: 0.3（秒），解析成 float64 后用于投放间隔；若按整数解析会变成 0。
// 预期：Unmarshal 后 ReleaseInterval 等于 0.3。
// 覆盖：用 json5.Unmarshal 解析内联配置，断言字段值。
func TestPlaneWeaponReleaseIntervalUnmarshalsFractionalSeconds(t *testing.T) {
	var weapon PlaneWeapon
	if err := json5.Unmarshal([]byte(`{releaseInterval: 0.3}`), &weapon); err != nil {
		t.Fatalf("unmarshal releaseInterval: %v", err)
	}
	if weapon.ReleaseInterval != 0.3 {
		t.Fatalf("releaseInterval = %v, want 0.3", weapon.ReleaseInterval)
	}
}

// TestPlaneTypeAttacksShips 验证哪些机种参与对舰攻击。
// 场景：AI 分配对舰任务/图鉴筛选时按机种判断；战斗机只制空，俯冲/水平/鱼雷/攻击机应对舰。
// 预期：Fighter 返回 false，DiveBomber/LevelBomber/TorpedoBomber/Attacker 返回 true。
// 覆盖：逐个机种调用 AttacksShips 并断言。
func TestPlaneTypeAttacksShips(t *testing.T) {
	if PlaneTypeFighter.AttacksShips() {
		t.Fatal("fighter should not attack ships")
	}
	for _, planeType := range []PlaneType{
		PlaneTypeDiveBomber, PlaneTypeLevelBomber, PlaneTypeTorpedoBomber, PlaneTypeAttacker,
	} {
		if !planeType.AttacksShips() {
			t.Fatalf("%s should attack ships", planeType)
		}
	}
}

// TestGetPlaneTargetObjTypeAttacker 验证攻击机默认目标类型为舰船，且视距按轰炸机档。
// 场景：Il-2 这类攻击机在配置里没有显式 targetType，AI 需要推断它打船。
// 预期：GetPlaneTargetObjType 返回 TypeShip；ResolvePlaneSight 返回 SightRangeBomber。
// 覆盖：临时注册 Il-2 模板，调用两个函数断言。
func TestGetPlaneTargetObjTypeAttacker(t *testing.T) {
	previous := PlaneMap
	t.Cleanup(func() { PlaneMap = previous })

	PlaneMap = map[string]*Plane{
		"Il-2": {
			Name:   "Il-2",
			Type:   PlaneTypeAttacker,
			Weapon: PlaneWeapon{Guns: []*Gun{{}}, Rockets: []*PlaneRocketLauncher{{}}},
		},
	}
	if got := GetPlaneTargetObjType("Il-2"); got != object.TypeShip {
		t.Fatalf("attacker target = %v, want ship", got)
	}
	if got := ResolvePlaneSight(PlaneTypeAttacker, 0); got != SightRangeBomber {
		t.Fatalf("attacker sight = %v, want %v", got, SightRangeBomber)
	}
}

// TestGetPlaneTargetObjTypeLevelBomber 验证水平轰炸机默认目标类型为舰船。
// 场景：SeaOtter 等水平轰炸机配置只挂炸弹，没有显式 targetType；AI 应把它派去打船。
// 预期：GetPlaneTargetObjType 返回 TypeShip。
// 覆盖：临时注册 SeaOtter 模板，调用函数断言。
func TestGetPlaneTargetObjTypeLevelBomber(t *testing.T) {
	previous := PlaneMap
	t.Cleanup(func() { PlaneMap = previous })

	PlaneMap = map[string]*Plane{
		"SeaOtter": {
			Name:   "SeaOtter",
			Type:   PlaneTypeLevelBomber,
			Weapon: PlaneWeapon{Bombs: []*Releaser{{}}},
		},
	}
	if got := GetPlaneTargetObjType("SeaOtter"); got != object.TypeShip {
		t.Fatalf("level bomber target = %v, want ship", got)
	}
}
