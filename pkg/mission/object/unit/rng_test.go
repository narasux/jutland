package unit

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestRandSourceIsSeededAndRestored 验证战斗随机源可注入、同 seed 复现、restore 后回到全局源。
// 场景：headless 每局必须用种子化 RNG；如果仍然读全局 rand，同 seed 的散布/暴击会漂移。
// 预期：两次 rand.NewSource(42) 的 Float64/Intn 序列完全一致；restore 后不再由该源驱动。
// 覆盖：SetRandSource(42) 取两个值，restore；再 SetRandSource(42) 取两个值并断言相等。
func TestRandSourceIsSeededAndRestored(t *testing.T) {
	restore := SetRandSource(rand.New(rand.NewSource(42)))
	firstFloat := RandFloat64()
	firstInt := RandIntn(1000)
	restore()

	restore = SetRandSource(rand.New(rand.NewSource(42)))
	secondFloat := RandFloat64()
	secondInt := RandIntn(1000)
	restore()

	require.Equal(t, firstFloat, secondFloat)
	require.Equal(t, firstInt, secondInt)
}

// TestGunFireUsesSeededSource 验证火炮散布走注入的 RNG，同 seed 落点一致、不同 seed 落点不同。
// 场景：observer/报告需要同 seed 复现同一次射击的弹着点；Gun.Fire 的散布是首个随机分支。
// 预期：seed=7 两次射击的 TargetPos 完全相同；seed=8 的 TargetPos 与 seed=7 不同。
// 覆盖：配置默认 GameSettings、注册测试弹药、19 格对舰炮与固定目标，fire(seed) 重置装填后调用 Gun.Fire。
func TestGunFireUsesSeededSource(t *testing.T) {
	old := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = old })

	const bulletName = "test/rng-shell"
	objBullet.Map[bulletName] = &objBullet.Bullet{
		Name: bulletName, Type: objBullet.TypeShell, Diameter: 127, Damage: 10,
	}
	t.Cleanup(func() { delete(objBullet.Map, bulletName) })

	gun := &Gun{
		Name: "RNGTest", BulletName: bulletName, BulletCount: 1, Range: 19,
		BulletSpread: 150, BulletSpeed: 0.2, AntiShip: true,
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
	fire := func(seed int64) *objBullet.Bullet {
		gun.ReloadStartTick = 0
		restore := SetRandSource(rand.New(rand.NewSource(seed)))
		defer restore()
		bullets := gun.Fire(shooter, target)
		require.Len(t, bullets, 1)
		return bullets[0]
	}
	a := fire(7)
	b := fire(7)
	c := fire(8)
	require.Equal(t, a.TargetPos, b.TargetPos)
	require.NotEqual(t, a.TargetPos, c.TargetPos)
}

// BenchmarkRandFloat64Hook 测量注入 RNG 的函数变量调用成本。
// 场景：火炮/暴击/AA 每个随机分支都走 RandFloat64，生产默认也走这个包装。
// 预期：输出 ns/op 与 allocs/op，用于和全局 rand 对比是否引入明显热路径回归。
// 覆盖：循环调用 RandFloat64，不设置 seed source（全局默认）。
func BenchmarkRandFloat64Hook(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = RandFloat64()
	}
}

// BenchmarkRandFloat64Global 作为包装层的对照基准。
// 场景：同机同进程直接调用 math/rand.Float64，量化函数变量多一层调用的成本。
// 预期：与 BenchmarkRandFloat64Hook 同量级；差值用于性能门禁判断。
// 覆盖：循环调用 rand.Float64，不产生分配。
func BenchmarkRandFloat64Global(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rand.Float64()
	}
}
