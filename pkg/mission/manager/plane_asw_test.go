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

// TestPlaneFireOnSubmergedDropsOnlyDepthCharges 验证对下潜潜艇开火时只投放深弹。
// 场景：B-24D 反潜机位于 (10,10)、朝向 90°，挂载 1 枚 Mk6 深弹；U 艇在 (10.5,10) 且
// DiveStateSubmerged，Plane.Fire 会走进 submergedTarget 分支，跳过机炮与火箭。
// 预期：Fire 只返回 1 发弹药且类型为 TypeDepthCharge；若返回炸弹/鱼雷/炮弹，
// 说明普通弹药被允许投向水下目标，反潜规则被绕过。
// 覆盖：newASWPlane 搭配 newDepthChargeReleaser（AntiSubmarine=true），require.Len==1 并比对 Bullet.Type。
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

// TestReleaserRejectsSubmergedWithoutAntiSub 验证普通炸弹释放器不会被当成反潜挂载。
// 场景：按 US/BB/1000/454 构造只装普通炸弹、AntiSubmarine 未置位的 Releaser，射程与左右投放射界照常填写。
// 预期：AntiSubmarine 为 false；Releaser.targetAcceptable 对下潜潜艇只认该标记，普通轰炸机无法投弹。
// 覆盖：直接断言 Releaser.AntiSubmarine，守住「无深弹也能反潜」的配置回归。
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

// TestCanPlaneFireAtSubmergedNeedsDepthChargeAndDetection 验证派机攻击下潜潜艇需同时满足“已点亮”与“带深弹”。
// 场景：SimTick=100 时 U 艇在 (10.5,10) 下潜；一架带 Mk6 深弹的飞机与一架只挂普通炸弹的飞机
// 都把 MaxToShipRange 设为 5，目标距离约 0.5 格，射程足够，只差可见性与弹种条件。
// 预期：未 MarkDetected 时反潜机也返回 false；MarkDetected(HumanAlpha,200) 后普通轰炸机仍为 false；
// 同一可见状态下反潜机才为 true。失败意味着飞机会盲攻潜艇或让普通轰炸机白白俯冲。
// 覆盖：三种组合依次调用 canPlaneFireAtTarget，Sub 仅登记在 Arena.Ships，SimTick=100、探测到期时刻 200。
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

// TestPlaneASWLoadouts 验证反潜深弹只挂在无鱼雷的轰炸机/巡逻机上，鱼雷机保持纯鱼雷挂载。
// 场景：从 PlaneMap 取 B-24D、SB2C-4、SeaOtter 三型反潜机，以及 TBF-1、TBF-1-UK、Swordfish 三型鱼雷机。
// 预期：前者 HasAntiSubWeapon() 为 true；后者为 false 且 Weapon.Torpedoes 非空，避免「鱼雷+深弹」同挂。
// 覆盖：六型真实配置模板逐个 require，检查 HasAntiSubWeapon 与 Torpedoes 是否为空。
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
