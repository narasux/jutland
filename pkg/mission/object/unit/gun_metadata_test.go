package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestGunFireRecordsWeaponAndTargetMetadata 验证炮弹在发射瞬间写入武器名、目标 UID 与射程比例。
// 场景：observer 需要按武器/目标归因；炮弹飞行数拍后目标可能已改名或离场，事后回查不可靠。
// 预期：TestGun 发射的炮弹 WeaponName="TestGun"、TargetUID="target"、FireDistance≈19、
// FireRangeRatio≈1.0；失败说明 observer 会丢失武器或目标归因。
// 覆盖：构造 19 格射程对舰炮与 19 格外目标，设置全局 config 和弹药模板后调用 Gun.Fire，
// 断言返回炮弹的四个元数据字段；不进入命中结算。
func TestGunFireRecordsWeaponAndTargetMetadata(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	oldTick := SimTick()
	SetSimTick(1)
	t.Cleanup(func() {
		config.G = oldSettings
		SetSimTick(oldTick)
	})

	const bulletName = "test/gun-metadata-shell"
	objBullet.Map[bulletName] = &objBullet.Bullet{
		Name: bulletName, Type: objBullet.TypeShell, Diameter: 127, Damage: 10,
	}
	t.Cleanup(func() { delete(objBullet.Map, bulletName) })

	gun := &Gun{
		Name: "TestGun", BulletName: bulletName, BulletCount: 1, Range: 19,
		BulletSpeed: 0.2, AntiShip: true, PosPercent: 0,
		LeftFiringArc:  FiringArc{Start: 180, End: 360},
		RightFiringArc: FiringArc{Start: 0, End: 180},
	}
	shooter := &BattleShip{
		Uid: "shooter", Length: 100, Width: 10,
		CurPos: objPos.NewR(5, 24), CurRotation: 90,
		BelongPlayer: faction.HumanAlpha,
	}
	target := &BattleShip{
		Uid: "target", Length: 100, Width: 10,
		CurPos: objPos.NewR(24, 24), CurRotation: 0,
		BelongPlayer: faction.ComputerAlpha,
	}
	bullets := gun.Fire(shooter, target)
	require.Len(t, bullets, 1)
	require.Equal(t, "TestGun", bullets[0].WeaponName)
	require.Equal(t, "target", bullets[0].TargetUID)
	require.InDelta(t, 19, bullets[0].FireDistance, 1e-6)
	require.InDelta(t, 1, bullets[0].FireRangeRatio, 1e-6)
}
