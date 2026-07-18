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
