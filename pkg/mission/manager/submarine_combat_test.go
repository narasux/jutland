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

// TestGunsCannotFireAtSubmergedSubmarine 验证水面舰的普通舰炮与鱼雷不会朝下潜潜艇开火。
// 场景：驱逐舰 "gurkha" 在 (10,10)，U-96 在 (14,10) 且 DiveStateSubmerged；
// FireAt 把 Primary/Secondary/AntiAircraft 全部指向这艘潜艇，等于逼所有武器位都尝试攻击它。
// 预期：返回的每一发弹药都不是 TypeShell 也不是 TypeTorpedo，并且 CanHitSubmerged() 为真；
// 失败意味着普通炮弹/鱼雷能打进水下，潜艇的下潜保护形同虚设（深弹/反潜鱼雷仍应正常投放）。
// 覆盖：用真实 ShipMap 模板（gurkha、u_96）构造实例，逐发检查 Bullet.Type 与对潜命中属性。
func TestGunsCannotFireAtSubmergedSubmarine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	escort := newNamedShip(t, "gurkha", faction.HumanAlpha, 10, 10)
	sub := newNamedShip(t, "u_96", faction.ComputerAlpha, 14, 10)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged

	bullets := escort.FireAt(objUnit.FireTargets{Primary: sub, Secondary: sub, AntiAircraft: sub})
	for _, bt := range bullets {
		require.NotEqual(t, objBullet.TypeShell, bt.Type, "普通舰炮不应朝下潜潜艇开火")
		require.NotEqual(t, objBullet.TypeTorpedo, bt.Type, "水面舰普通鱼雷不应朝下潜潜艇发射")
		require.True(t, bt.CanHitSubmerged(), "对潜射击的弹药必须能命中水下目标")
	}
}

// TestSurfaceShipWithoutASWWeaponCannotEngageSubmerged 验证无任何反潜武器的水面舰对下潜目标射程为 0。
// 场景：战列舰 "lowa" 在 (10,10)，两艘 U-96 分别在 (12,10) 与 (11,10)，前者下潜、后者浮航，
// 比较 shipEngageRange 对二者的返回值。
// 预期：lowa 的 MaxToSubmergedRange 为 0，对下潜潜艇 shipEngageRange 也为 0；
// 对浮航潜艇则回退到 MaxToShipRange 并返回正数。失败意味着无深弹的战列舰能“点亮”并攻击水下潜艇，
// 或者反过来把浮航潜艇也当水下目标拒之门外。
// 覆盖：require.Zero/Positive 断言两个常量级射程与两种 DiveState 下的 shipEngageRange。
func TestSurfaceShipWithoutASWWeaponCannotEngageSubmerged(t *testing.T) {
	battleship := newNamedShip(t, "lowa", faction.HumanAlpha, 10, 10)
	sub := newNamedShip(t, "u_96", faction.ComputerAlpha, 12, 10)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged

	require.Zero(t, battleship.Weapon.MaxToSubmergedRange)
	require.Zero(t, shipEngageRange(battleship, sub), "没有反潜武器的战列舰点不亮也打不到水下潜艇")
	surfaceSub := newNamedShip(t, "u_96", faction.ComputerAlpha, 11, 10)
	surfaceSub.Submarine.DiveState = objUnit.DiveStateSurface
	require.Positive(t, shipEngageRange(battleship, surfaceSub),
		"同一条船打浮航潜艇时仍按对舰射程")
}

// TestSubmarineCarriesAntiSubmarineTorpedoes 验证潜艇自身鱼雷被计入对潜武器并带潜艇发射标记。
// 场景：直接用 u_96 模板在 (10,10) 生成一艘潜艇，检查其 ShipWeapon 的反潜相关字段，
// 这是潜艇对潜艇交战（以及潜射反潜）的前提条件。
// 预期：HasAntiSubWeapon 为 true，MaxToSubmergedRange 为正数，且每一条 Torpedoes 的
// SubmarineLaunched 都为 true；失败意味着潜艇没有可用的对潜射程，或者潜艇鱼雷按水面舰规则结算，
// 下潜的对手将无法被命中。
// 覆盖：newNamedShip(t,"u_96",HumanAlpha,10,10) 后遍历 Weapon.Torpedoes 逐条断言。
func TestSubmarineCarriesAntiSubmarineTorpedoes(t *testing.T) {
	sub := newNamedShip(t, "u_96", faction.HumanAlpha, 10, 10)
	require.True(t, sub.Weapon.HasAntiSubWeapon)
	require.Positive(t, sub.Weapon.MaxToSubmergedRange, "潜艇自身鱼雷应计入对潜射程")
	for _, torpedo := range sub.Weapon.Torpedoes {
		require.True(t, torpedo.SubmarineLaunched, "潜艇鱼雷应带 SubmarineLaunched 标记")
	}
}

// TestSubmarineTorpedoHitsDetectedSubmergedSubmarine 验证潜艇鱼雷能命中已被水听器点亮的下潜潜艇。
// 场景：两艘 u_96 分别在水下 (10,10) 与 (14,10)，目标被 MarkDetected(HumanAlpha,1000) 点亮，射手 Attack 后在 SimTick=1 的空白海域跑 200 拍。
// 预期：至少发射 1 根 TypeTorpedo 且目标 CurHP < TotalHP；回归前「水下一律免疫」把潜艇鱼雷也挡掉，所有鱼雷都打不到下潜潜艇。
// 覆盖：shipFireTargets/FireAt 发射、updateShotBullets 真实结算，SubmarineLaunched 与探测记忆共同放行。
func TestSubmarineTorpedoHitsDetectedSubmergedSubmarine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	shooter := newNamedShip(t, "u_96", faction.HumanAlpha, 10, 10)
	target := newNamedShip(t, "u_96", faction.ComputerAlpha, 14, 10)
	shooter.Submarine.DiveState = objUnit.DiveStateSubmerged
	target.Submarine.DiveState = objUnit.DiveStateSubmerged
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

// TestDestroyerWithDepthChargesHasAntiSubWeapon 验证驱逐舰的深弹挂载被识别为反潜武器且不参与炮战。
// 场景：用 "gurkha" 模板生成驱逐舰，读取 ShipWeapon 的 HasAntiSubWeapon 与 Rocket 列表；
// 英美驱逐舰配置了深弹/反潜火箭，这里检查它们在数据层的分类是否正确。
// 预期：HasAntiSubWeapon 为 true；至少存在一个 AntiSubmarine 的 Rocket，且该 Rocket 的
// AntiShip 为 false。失败意味着驱逐舰没有反潜能力，或者反潜火箭被同时算进水面炮战，
// 对舰伤害与弹药消耗会失控。
// 覆盖：newNamedShip 构造 gurkha，遍历 Weapon.Rockets 找 AntiSubmarine 项并断言 AntiShip 互斥。
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
