package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func newNamedShip(t *testing.T, name string, player faction.Player, x, y float64) *objUnit.BattleShip {
	t.Helper()
	_, ok := objUnit.ShipMap[name]
	require.True(t, ok, "ship template %s missing", name)
	uid := objUnit.NewShipUidGenerator(player)
	return objUnit.NewShip(uid, name, objPos.NewR(x, y), 90, player)
}

func TestGunsCannotFireAtSubmergedSubmarine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	escort := newNamedShip(t, "gurkha", faction.HumanAlpha, 10, 10)
	sub := newNamedShip(t, "u_boat_type_viic", faction.ComputerAlpha, 14, 10)
	sub.DiveState = objUnit.DiveStateSubmerged

	bullets := escort.FireAt(objUnit.FireTargets{Primary: sub, Secondary: sub, AntiAircraft: sub})
	for _, bt := range bullets {
		require.NotEqual(t, objBullet.TypeShell, bt.Type, "普通舰炮不应朝下潜潜艇开火")
		require.NotEqual(t, objBullet.TypeTorpedo, bt.Type, "水面舰普通鱼雷不应朝下潜潜艇发射")
		require.True(t, bt.CanHitSubmerged(), "对潜射击的弹药必须能命中水下目标")
	}
}

func TestSurfaceShipWithoutASWWeaponCannotEngageSubmerged(t *testing.T) {
	battleship := newNamedShip(t, "lowa", faction.HumanAlpha, 10, 10)
	sub := newNamedShip(t, "u_boat_type_viic", faction.ComputerAlpha, 12, 10)
	sub.DiveState = objUnit.DiveStateSubmerged

	require.Zero(t, battleship.Weapon.MaxToSubmergedRange)
	require.Zero(t, shipEngageRange(battleship, sub), "没有反潜武器的战列舰点不亮也打不到水下潜艇")
	surfaceSub := newNamedShip(t, "u_boat_type_viic", faction.ComputerAlpha, 11, 10)
	surfaceSub.DiveState = objUnit.DiveStateSurface
	require.Positive(t, shipEngageRange(battleship, surfaceSub),
		"同一条船打浮航潜艇时仍按对舰射程")
}

func TestSubmarineCarriesAntiSubmarineTorpedoes(t *testing.T) {
	sub := newNamedShip(t, "u_boat_type_viic", faction.HumanAlpha, 10, 10)
	require.True(t, sub.Weapon.HasAntiSubWeapon)
	require.Positive(t, sub.Weapon.MaxToSubmergedRange, "潜艇自身鱼雷应计入对潜射程")
	for _, torpedo := range sub.Weapon.Torpedoes {
		require.True(t, torpedo.SubmarineLaunched, "潜艇鱼雷应带 SubmarineLaunched 标记")
	}
}

// 潜艇对潜艇的鱼雷必须真的能造成伤害：先由水听器点亮目标，再走真实的
// 发射与命中结算链路。回归：命中结算里多出一道「水下一律免疫」的判断，
// 把潜艇鱼雷 / 反潜鱼雷也一起挡掉，结果任何鱼雷都打不到下潜潜艇。
func TestSubmarineTorpedoHitsDetectedSubmergedSubmarine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	shooter := newNamedShip(t, "u_boat_type_viic", faction.HumanAlpha, 10, 10)
	target := newNamedShip(t, "u_boat_type_viic", faction.ComputerAlpha, 14, 10)
	shooter.DiveState = objUnit.DiveStateSubmerged
	target.DiveState = objUnit.DiveStateSubmerged
	// 模拟水听器已经点亮目标，否则下潜潜艇按规则不可见、也不该被盲射命中。
	target.MarkDetected(faction.HumanAlpha, 1000, target.CurPos)
	shooter.Attack(target.Uid)

	mm := &MissionManager{state: &state.MissionState{
		Core: state.MissionCoreState{
			SimTick:   1,
			MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{}},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{shooter.Uid: shooter, target.Uid: target},
		},
	}}

	fired := false
	for tick := 0; tick < 200; tick++ {
		for _, bt := range shooter.FireAt(mm.shipFireTargets(shooter)) {
			if bt.Type == objBullet.TypeTorpedo {
				fired = true
			}
			mm.state.Arena.ForwardingBullets = append(mm.state.Arena.ForwardingBullets, bt)
		}
		mm.updateShotBullets()
	}

	require.True(t, fired, "潜艇应对被点亮的下潜潜艇发射鱼雷")
	require.Less(t, target.CurHP, target.TotalHP, "潜艇鱼雷命中下潜潜艇后应造成伤害")
}

func TestDestroyerWithDepthChargesHasAntiSubWeapon(t *testing.T) {
	escort := newNamedShip(t, "gurkha", faction.HumanAlpha, 10, 10)
	require.True(t, escort.Weapon.HasAntiSubWeapon, "英/美驱逐舰应挂深弹架")

	found := false
	for _, rocket := range escort.Weapon.Rockets {
		if rocket.AntiSubmarine {
			found = true
			require.False(t, rocket.AntiShip, "纯反潜火箭不应参与水面炮战")
		}
	}
	require.True(t, found)
}
