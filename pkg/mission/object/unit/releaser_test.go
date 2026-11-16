package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestAerialTorpedoCanAttackShorelineTargetWithoutRunningFarPastIt
// 验证空投鱼雷打贴岸目标时不会因目标身后的陆地而拒投，
// 也不会把鱼雷寿命算得过长而冲上岸。
// 场景：目标在 (1.5,0.5)，身后 (2.0,0.5) 是陆地；若用整条路径检测陆地会误判，若寿命按射程算会撞岸。
// 预期：pathCrossesLand 返回 false；Fire 生成 1 枚鱼雷；Life 按到目标距离+1 帧，最大行程小于到岸距离。
// 覆盖：构造 5 格海岸地图、贴岸目标，调用 pathCrossesLand/Fire 并断言弹道参数。
func TestAerialTorpedoCanAttackShorelineTargetWithoutRunningFarPastIt(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-aerial-torpedo"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeTorpedo}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
	})

	torpedo := &Releaser{
		BulletName:     bulletName,
		Range:          3,
		BulletSpeed:    0.25,
		RightFiringArc: FiringArc{Start: 0, End: 180},
	}
	plane := &Plane{
		Uid:          "torpedo-bomber",
		CurHP:        100,
		CurPos:       objPos.NewR(0.5, 0.5),
		CurRotation:  0,
		BelongPlayer: faction.HumanAlpha,
	}
	target := &BattleShip{
		Uid:          "shoreline-target",
		CurHP:        100,
		CurPos:       objPos.NewR(1.5, 0.5),
		BelongPlayer: faction.ComputerAlpha,
	}
	terrain := mapcfg.MapData{"SSCSS"}

	if torpedo.pathCrossesLand(plane, target, &terrain) {
		t.Fatal("land behind target incorrectly blocked torpedo release")
	}
	bullets := torpedo.Fire(plane, target)
	if len(bullets) != 1 {
		t.Fatalf("released torpedoes = %d, want 1", len(bullets))
	}

	bullet := bullets[0]
	wantLife := int(math.Ceil(plane.CurPos.Distance(target.CurPos)/bullet.Speed)) + 1
	if bullet.Life != wantLife {
		t.Fatalf("torpedo life = %d, want %d", bullet.Life, wantLife)
	}
	landDistance := 2.0 - plane.CurPos.RX
	if maxTravel := float64(bullet.Life) * bullet.Speed; maxTravel >= landDistance {
		t.Fatalf("torpedo max travel = %.2f, reaches land at %.2f", maxTravel, landDistance)
	}
}
