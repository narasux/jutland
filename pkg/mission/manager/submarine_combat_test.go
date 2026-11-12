package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
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
