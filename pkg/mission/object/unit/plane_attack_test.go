package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
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

// TestPlaneFireReleasesConfiguredSalvo 验证齐射配置一次投放多枚挂载，默认仍每次一枚。
// 场景：A/B 这类小型攻击机只携带少量挂载，单枚释放会迫使飞机反复通场，暴露在防空火力中。
// 预期：SalvoSize=2 时第一次 Fire 释放 2 枚炸弹、第二次释放剩余 2 枚；SalvoSize=0 时每次只释放 1 枚。
// 覆盖：注册测试炸弹与 4 个同参数释放器，目标在正面 1 格内且射界全开，统计 Released 数量。
func TestPlaneFireReleasesConfiguredSalvo(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-salvo-bomb"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeBomb, Damage: 100}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
	})

	newPlane := func(salvo int) *Plane {
		weapon := PlaneWeapon{SalvoSize: salvo, ReleaseInterval: 0}
		for range 4 {
			weapon.Bombs = append(weapon.Bombs, &Releaser{
				BulletName: bulletName, BulletType: objBullet.TypeBomb,
				Range: 2, BulletSpeed: 0.1,
				RightFiringArc: FiringArc{Start: 0, End: 360},
			})
		}
		return &Plane{
			Uid: "salvo-bomber", CurHP: 100,
			CurPos: objPos.NewR(0, 0), CurRotation: 0, BelongPlayer: faction.HumanAlpha,
			Weapon: weapon,
		}
	}
	target := &BattleShip{
		Uid: "target", Type: ShipTypeCruiser, CurHP: 1000,
		CurPos: objPos.NewR(1, 0), BelongPlayer: faction.ComputerAlpha,
	}
	released := func(p *Plane) int {
		count := 0
		for _, bomb := range p.Weapon.Bombs {
			if bomb.Released {
				count++
			}
		}
		return count
	}

	SetSimTick(10)
	salvo := newPlane(2)
	salvo.Fire(target)
	if got := released(salvo); got != 2 {
		t.Fatalf("first salvo released %d bombs, want 2", got)
	}
	salvo.Fire(target)
	if got := released(salvo); got != 4 {
		t.Fatalf("second salvo released %d bombs total, want 4", got)
	}

	single := newPlane(0)
	single.Fire(target)
	if got := released(single); got != 1 {
		t.Fatalf("default release count = %d, want 1", got)
	}
}

// TestHasUsableOrdnanceForMixedLoadout 验证混合挂载按目标筛选可投放弹药。
// 场景：飞机同时挂普通炸弹与反潜深弹；对水面舰只有普通炸弹能投，对下潜潜艇只有深弹能投。
// 预期：普通炸弹未投时对两者都为 true；普通炸弹投完后对水面舰为 false、对下潜潜艇仍为 true。
// 覆盖：同一架飞机依次查询水面舰与 DiveStateSubmerged 潜艇，断言反潜门槛不会阻塞返航。
func TestHasUsableOrdnanceForMixedLoadout(t *testing.T) {
	bomb := &Releaser{}
	depthCharge := &Releaser{AntiSubmarine: true}
	plane := &Plane{Weapon: PlaneWeapon{Bombs: []*Releaser{bomb, depthCharge}}}

	surface := &BattleShip{Type: ShipTypeCruiser}
	submerged := &BattleShip{
		Type:      ShipTypeSubmarine,
		Submarine: SubmarineState{DiveState: DiveStateSubmerged},
	}
	if !plane.HasUsableOrdnanceFor(surface) {
		t.Fatal("surface target should be attackable while the normal bomb is unreleased")
	}
	if !plane.HasUsableOrdnanceFor(submerged) {
		t.Fatal("submerged target should be attackable by the depth charge")
	}

	bomb.Released = true
	if plane.HasUsableOrdnanceFor(surface) {
		t.Fatal("surface target must not count the inapplicable depth charge as usable ordnance")
	}
	if !plane.HasUsableOrdnanceFor(submerged) {
		t.Fatal("submerged target should still be attackable by the depth charge")
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
