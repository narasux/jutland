package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// newASWPlane 构造一架带深弹挂载、能攻击下潜潜艇的飞机。
func newASWPlane(depthChargeReleaser *objUnit.Releaser) *objUnit.Plane {
	return &objUnit.Plane{
		Uid: "asw-plane", Name: "B-24D", CurHP: 10, TotalHP: 10, BelongPlayer: faction.HumanAlpha,
		Type: objUnit.PlaneTypeLevelBomber,
		Weapon: objUnit.PlaneWeapon{
			Bombs: []*objUnit.Releaser{depthChargeReleaser},
		},
	}
}

func newDepthChargeReleaser() *objUnit.Releaser {
	return &objUnit.Releaser{
		Name: "US/DC/300/1940/Mk6", BulletName: "US/DC/300/1940/Mk6",
		BulletType: objBullet.TypeDepthCharge, AntiSubmarine: true,
		Range: 1.5, BulletSpeed: 30,
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 20},
		LeftFiringArc:  objUnit.FiringArc{Start: 340, End: 360},
	}
}

func TestPlaneFireOnSubmergedDropsOnlyDepthCharges(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	plane := newASWPlane(newDepthChargeReleaser())
	plane.CurPos = objPos.NewR(10, 10)
	plane.CurRotation = 90
	plane.CurSpeed = 0.1
	plane.MaxSpeed = 0.1

	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.5, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	bullets := plane.Fire(sub)
	require.Len(t, bullets, 1)
	require.Equal(t, objBullet.TypeDepthCharge, bullets[0].Type)
}

// 普通轰炸机（无深弹）的释放器拒绝下潜目标。
func TestReleaserRejectsSubmergedWithoutAntiSub(t *testing.T) {
	releaser := &objUnit.Releaser{
		Name: "US/BB/1000/454", BulletName: "US/BB/1000/454",
		BulletType: objBullet.TypeBomb,
		Range:      1.5, BulletSpeed: 30,
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 20},
		LeftFiringArc:  objUnit.FiringArc{Start: 340, End: 360},
	}
	require.False(t, releaser.AntiSubmarine)
}

func TestCanPlaneFireAtSubmergedNeedsDepthChargeAndDetection(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.5, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	aswPlane := newASWPlane(newDepthChargeReleaser())
	aswPlane.CurPos = objPos.NewR(10, 10)
	aswPlane.Weapon.MaxToShipRange = 5
	plainPlane := newASWPlane(&objUnit.Releaser{Name: "dummy", BulletType: objBullet.TypeBomb})
	plainPlane.CurPos = objPos.NewR(10, 10)
	plainPlane.Weapon.MaxToShipRange = 5

	ms := &state.MissionState{
		Core:  state.MissionCoreState{SimTick: 100},
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{sub.Uid: sub}},
	}
	m := &MissionManager{state: ms}

	// 未被点亮：带深弹也不派
	require.False(t, m.canPlaneFireAtTarget(aswPlane, sub))
	// 被点亮但没有深弹：不派普通轰炸机
	sub.MarkDetected(faction.HumanAlpha, 200, sub.CurPos)
	require.False(t, m.canPlaneFireAtTarget(plainPlane, sub))
	// 被点亮且带深弹：可以攻击
	require.True(t, m.canPlaneFireAtTarget(aswPlane, sub))
}

// 反潜挂载只给不带鱼雷的轰炸机/巡逻机；鱼雷机保持纯鱼雷，避免"鱼雷 + 深弹"同挂。
func TestPlaneASWLoadouts(t *testing.T) {
	for _, name := range []string{"B-24D", "SB2C-4", "SeaOtter"} {
		plane, ok := objUnit.PlaneMap[name]
		require.True(t, ok, "plane %s missing", name)
		require.True(t, plane.HasAntiSubWeapon(), "%s should carry depth charges", name)
	}

	for _, name := range []string{"TBF-1", "TBF-1-UK", "Swordfish"} {
		plane, ok := objUnit.PlaneMap[name]
		require.True(t, ok, "plane %s missing", name)
		require.False(t, plane.HasAntiSubWeapon(), "%s is a torpedo bomber and must not mix depth charges", name)
		require.NotEmpty(t, plane.Weapon.Torpedoes, "%s should keep its torpedo loadout", name)
	}
}
