package manager

import (
	"fmt"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

const (
	// fireLayerMainShell 主 / 副炮用的大口径测试弹
	fireLayerMainShell = "test/fire-layer-main-shell"
	// fireLayerAAShell 防空炮用的小口径测试弹，便于在断言里区分是哪类炮开的火
	fireLayerAAShell = "test/fire-layer-aa-shell"
)

// registerFireLayerBullets 注册两种口径的测试弹药并登记清理。
func registerFireLayerBullets(t *testing.T) {
	t.Helper()
	for name, diameter := range map[string]int{fireLayerMainShell: 200, fireLayerAAShell: 40} {
		oldBullet, hadBullet := objBullet.Map[name]
		objBullet.Map[name] = &objBullet.Bullet{
			Name: name, Type: objBullet.TypeShell, Diameter: diameter,
			Damage: 100, CriticalRate: 0,
		}
		t.Cleanup(func() {
			if hadBullet {
				objBullet.Map[name] = oldBullet
			} else {
				delete(objBullet.Map, name)
			}
		})
	}
}

// newFireLayerGun 造一门全向射界、装填就绪的测试炮。
func newFireLayerGun(bulletName string, antiShip, antiAircraft bool) *objUnit.Gun {
	return &objUnit.Gun{
		Name: "test/fire-layer-gun", BulletName: bulletName,
		BulletCount: 1, ReloadTime: 0,
		Range: 30, BulletSpread: 0, BulletSpeed: 1.5,
		AntiShip: antiShip, AntiAircraft: antiAircraft,
		LeftFiringArc:  objUnit.FiringArc{Start: 0, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 360},
	}
}

// newFireLayerShooter 造一艘主炮 / 副炮 / 防空炮各一门的测试战舰，
// 主副炮与防空炮都是高平两用（与配置里 127mm 级火炮的标注一致）。
func newFireLayerShooter() *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: "shooter", BelongPlayer: faction.HumanAlpha,
		CurPos: objPos.NewR(50, 50), CurRotation: 0,
		TotalHP: 5000, CurHP: 5000,
		Weapon: objUnit.ShipWeapon{
			MainGuns:         []*objUnit.Gun{newFireLayerGun(fireLayerMainShell, true, true)},
			SecondaryGuns:    []*objUnit.Gun{newFireLayerGun(fireLayerMainShell, true, true)},
			AntiAircraftGuns: []*objUnit.Gun{newFireLayerGun(fireLayerAAShell, true, true)},
			MaxToShipRange:   30,
			MaxToPlaneRange:  30,
		},
	}
}

// fireLayerEnemyShip 造一艘指定舰种与血量的敌方战舰。
func fireLayerEnemyShip(uid string, shipType objUnit.ShipType, x float64, totalHP, curHP float64) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: uid, Type: shipType, BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(x, 50), TotalHP: totalHP, CurHP: curHP,
	}
}

// runShipFire 跑一帧战舰开火，返回本帧产生的全部弹药。
func runShipFire(
	t *testing.T,
	shooter *objUnit.BattleShip,
	ships map[string]*objUnit.BattleShip,
	planes map[string]*objUnit.Plane,
) []*objBullet.Bullet {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{Ships: ships, Planes: planes},
	}}
	manager.updateShipWeaponFire()
	return manager.state.Arena.ForwardingBullets
}

// countAimedAt 统计瞄准指定位置的弹药数，diameter 为 0 时不限口径。
// 目标速度为 0 时提前量落点就是目标当前位置，所以瞄准点可以直接比对坐标。
func countAimedAt(bullets []*objBullet.Bullet, pos objPos.MapPos, diameter int) int {
	count := 0
	for _, bt := range bullets {
		if bt.TargetPos.Near(pos, 1e-6) && (diameter == 0 || bt.Diameter == diameter) {
			count++
		}
	}
	return count
}

// describeBullets 汇总本帧弹药，失败信息里能直接看出打到了谁。
func describeBullets(bullets []*objBullet.Bullet) string {
	parts := make([]string, 0, len(bullets))
	for _, bt := range bullets {
		parts = append(parts, fmt.Sprintf("%v dia=%d -> %s", bt.TargetObjType, bt.Diameter, bt.TargetPos.String()))
	}
	return strings.Join(parts, "; ")
}

// TestShipFireTargetsLayerByWeaponClass 验证开火目标按武器类别分层，不再共用一个随机目标。
// 场景：测试舰在 (50,50) 带 200mm 主炮/副炮与 40mm 防空炮各一门、射程 30；敌舰为满血战列舰 big（评分 0）、残血巡洋舰 hurt（0.6×0.9=0.54）、满血驱逐舰 near（最近），另有敌机在 (52,50)，跑一帧 updateShipWeaponFire。
// 预期：主炮 1 发 200mm 打 hurt，副炮 1 发 200mm 打 near，防空炮 1 发 40mm 打敌机；big 不被任何炮选中。
// 覆盖：shipFireTargets 的 primaryTargetScore/betterPrimaryTarget 集火评分、副炮最近敌舰、防空炮最近敌机三层择敌。
func TestShipFireTargetsLayerByWeaponClass(t *testing.T) {
	registerFireLayerBullets(t)
	shooter := newFireLayerShooter()
	// 满血战列舰（评分 0）、残血巡洋舰（0.6 × 0.9 = 0.54）、满血驱逐舰（0，最近）
	big := fireLayerEnemyShip("big", objUnit.ShipTypeBattleShip, 60, 9000, 9000)
	hurt := fireLayerEnemyShip("hurt", objUnit.ShipTypeCruiser, 70, 3000, 300)
	near := fireLayerEnemyShip("near", objUnit.ShipTypeDestroyer, 55, 2000, 2000)
	enemyPlane := &objUnit.Plane{
		Uid: "enemy-plane", BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(52, 50), TotalHP: 100, CurHP: 100,
	}

	bullets := runShipFire(t, shooter, map[string]*objUnit.BattleShip{
		shooter.Uid: shooter, big.Uid: big, hurt.Uid: hurt, near.Uid: near,
	}, map[string]*objUnit.Plane{enemyPlane.Uid: enemyPlane})

	if got := countAimedAt(bullets, hurt.CurPos, 200); got != 1 {
		t.Fatalf("main gun should concentrate on the damaged cruiser: %s", describeBullets(bullets))
	}
	if got := countAimedAt(bullets, near.CurPos, 200); got != 1 {
		t.Fatalf("secondary gun should fire at the nearest destroyer: %s", describeBullets(bullets))
	}
	if got := countAimedAt(bullets, enemyPlane.CurPos, 40); got != 1 {
		t.Fatalf("AA gun should fire at the nearest enemy plane: %s", describeBullets(bullets))
	}
	if got := countAimedAt(bullets, big.CurPos, 0); got != 0 {
		t.Fatalf("no gun should pick the full-health battleship: %s", describeBullets(bullets))
	}
}

// TestShipFireTargetsRespectExplicitAttackTarget 验证明确 AttackTarget 指令优先于评分与距离。
// 场景：测试舰 AttackTarget="big"，场上仍有评分更高的残血巡洋舰 hurt 与更近的驱逐舰 near，跑一帧开火。
// 预期：主炮与副炮各 1 发 200mm 全部打 big（共 2 发）；hurt 与 near 均无弹药瞄准。
// 覆盖：shipFireTargets 对有效 AttackTarget 的优先采用，覆盖 betterPrimaryTarget 与最近目标规则。
func TestShipFireTargetsRespectExplicitAttackTarget(t *testing.T) {
	registerFireLayerBullets(t)
	shooter := newFireLayerShooter()
	shooter.AttackTarget = "big"
	big := fireLayerEnemyShip("big", objUnit.ShipTypeBattleShip, 60, 9000, 9000)
	hurt := fireLayerEnemyShip("hurt", objUnit.ShipTypeCruiser, 70, 3000, 300)
	near := fireLayerEnemyShip("near", objUnit.ShipTypeDestroyer, 55, 2000, 2000)

	bullets := runShipFire(t, shooter, map[string]*objUnit.BattleShip{
		shooter.Uid: shooter, big.Uid: big, hurt.Uid: hurt, near.Uid: near,
	}, nil)

	// 主炮与副炮各一发，都打点名的 big
	if got := countAimedAt(bullets, big.CurPos, 200); got != 2 {
		t.Fatalf("both main and secondary guns should fire at the named target: %s", describeBullets(bullets))
	}
	if got := countAimedAt(bullets, hurt.CurPos, 0); got != 0 {
		t.Fatalf("named target must override the damage-based preference: %s", describeBullets(bullets))
	}
	if got := countAimedAt(bullets, near.CurPos, 0); got != 0 {
		t.Fatalf("named target must override the nearest-target rule: %s", describeBullets(bullets))
	}
}

// TestShipFireFallsBackToAirTargetsWithoutShips 验证高平两用炮在无敌舰时跨类别回退打敌机，防空能力不丢。
// 场景：测试舰主/副炮/防空炮均为 antiShip+antiAircraft，场上只有一架敌机在 (52,50)，跑一帧 updateShipWeaponFire。
// 预期：主炮与副炮各 1 发 200mm 打敌机（共 2 发），防空炮 1 发 40mm 也打敌机。
// 覆盖：BattleShip.FireAt 中 Primary/Secondary 为空时回退到 AntiAircraft 的高平两用逻辑。
func TestShipFireFallsBackToAirTargetsWithoutShips(t *testing.T) {
	registerFireLayerBullets(t)
	shooter := newFireLayerShooter()
	enemyPlane := &objUnit.Plane{
		Uid: "enemy-plane", BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(52, 50), TotalHP: 100, CurHP: 100,
	}

	bullets := runShipFire(t, shooter, map[string]*objUnit.BattleShip{shooter.Uid: shooter},
		map[string]*objUnit.Plane{enemyPlane.Uid: enemyPlane})

	// 主炮与副炮都是高平两用，没有敌舰时一并转打敌机
	if got := countAimedAt(bullets, enemyPlane.CurPos, 200); got != 2 {
		t.Fatalf(
			"dual-purpose main and secondary guns should engage the plane when no ship is in range: %s",
			describeBullets(bullets),
		)
	}
	if got := countAimedAt(bullets, enemyPlane.CurPos, 40); got != 1 {
		t.Fatalf("AA gun should engage the plane: %s", describeBullets(bullets))
	}
}

// TestShipFireFallsBackToSurfaceTargetsWithoutPlanes 验证无敌机时防空炮反向回退打敌舰，不空转。
// 场景：测试舰在 (50,50)，唯一敌舰驱逐舰在 (60,50)，场上无飞机，跑一帧开火。
// 预期：防空炮 1 发 40mm 瞄准敌舰。
// 覆盖：BattleShip.FireAt 中 AntiAircraft 为空时回退到 Primary（主/副炮目标）的分支。
func TestShipFireFallsBackToSurfaceTargetsWithoutPlanes(t *testing.T) {
	registerFireLayerBullets(t)
	shooter := newFireLayerShooter()
	enemy := fireLayerEnemyShip("enemy", objUnit.ShipTypeDestroyer, 60, 2000, 2000)

	bullets := runShipFire(t, shooter, map[string]*objUnit.BattleShip{
		shooter.Uid: shooter, enemy.Uid: enemy,
	}, nil)

	if got := countAimedAt(bullets, enemy.CurPos, 40); got != 1 {
		t.Fatalf("dual-purpose AA gun should engage ships when no plane is in range: %s", describeBullets(bullets))
	}
}
