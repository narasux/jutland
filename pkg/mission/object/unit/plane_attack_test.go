package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestAttackLeadSpeedUsesWeaponBallistics 验证对空提前量使用机炮初速，没有匹配武器时回退飞机速度。
// 场景：战斗机用 0.1875 格/帧的机炮打飞机，玩家期望提前量按弹速算；对舰没有对舰武器时只能按机身速度。
// 预期：TypePlane 返回 0.1875；TypeShip 返回 CurSpeed。
// 覆盖：构造带 AntiAircraft 机炮的战斗机，分别查询两种目标类型。
func TestAttackLeadSpeedUsesWeaponBallistics(t *testing.T) {
	plane := &Plane{
		Type:     PlaneTypeFighter,
		CurSpeed: 0.1,
		Weapon: PlaneWeapon{
			Guns: []*Gun{{
				BulletSpeed:  0.1875,
				AntiAircraft: true,
			}},
		},
	}
	if got := plane.AttackLeadSpeed(object.TypePlane); math.Abs(got-0.1875) > 1e-9 {
		t.Fatalf("attack lead speed = %v, want weapon speed 0.1875", got)
	}
	if got := plane.AttackLeadSpeed(object.TypeShip); math.Abs(got-plane.CurSpeed) > 1e-9 {
		t.Fatalf("attack lead speed without matching weapon = %v, want plane speed %v", got, plane.CurSpeed)
	}
}

// TestAttackLeadSpeedPrefersShipReleaseWeapon 验证对舰攻击优先使用投放武器（鱼雷/炸弹）的弹速。
// 场景：鱼雷机同时有机炮和鱼雷，对舰提前量应按鱼雷速度而不是机炮速度，否则投雷点会偏。
// 预期：TypeShip 返回鱼雷 BulletSpeed 0.1，而不是机炮 0.2。
// 覆盖：构造带 AntiShip 机炮和鱼雷的飞机，调用 AttackLeadSpeed(TypeShip)。
func TestAttackLeadSpeedPrefersShipReleaseWeapon(t *testing.T) {
	plane := &Plane{
		Type: PlaneTypeTorpedoBomber,
		Weapon: PlaneWeapon{
			Guns: []*Gun{{
				BulletSpeed: 0.2,
				AntiShip:    true,
			}},
			Torpedoes: []*Releaser{{
				BulletSpeed: 0.1,
			}},
		},
	}
	if got := plane.AttackLeadSpeed(object.TypeShip); math.Abs(got-0.1) > 1e-9 {
		t.Fatalf("ship attack lead speed = %v, want release weapon speed 0.1", got)
	}
}

// TestFighterPursuitAlignsNoseWithWeaponLeadPoint 验证战斗机追击时把机头对准武器提前点，保持在射界内。
// 场景：敌机在正前方略偏上，战斗机需要用机炮提前量追踪；若只朝敌机当前位置转，会飞出机炮射界。
// 预期：追击一帧后，敌机方位与机头夹角不超过 20°。
// 覆盖：构造战斗机/敌机/提前点，调用 FighterPursuitStrategy.MoveTo，计算相对方位。
func TestFighterPursuitAlignsNoseWithWeaponLeadPoint(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	enemyPos := objPos.NewR(10, 9.7)
	leadPos := objPos.NewR(10, 9)
	plane := &Plane{
		Uid:          "fighter",
		CurHP:        100,
		CurPos:       objPos.NewR(10, 10),
		CurRotation:  0,
		CurSpeed:     0.12,
		MaxSpeed:     0.12,
		RotateSpeed:  30,
		RemainRange:  100,
		BelongPlayer: faction.HumanAlpha,
	}

	(&FighterPursuitStrategy{}).MoveTo(plane, nil, leadPos, enemyPos, 0.06)

	bearing := plane.CurPos.Angle(enemyPos)
	relativeBearing := math.Mod(bearing-plane.CurRotation+540, 360) - 180
	if math.Abs(relativeBearing) > 20 {
		t.Fatalf(
			"fighter drifted outside its forward firing arc: bearing=%.2f rotation=%.2f",
			bearing,
			plane.CurRotation,
		)
	}
}
