package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// registerCombatTestPlane 注册测试机型，供 range/ASW 查询使用。
func registerCombatTestPlane(t *testing.T, name string, plane *Plane) {
	t.Helper()
	old, had := PlaneMap[name]
	PlaneMap[name] = plane
	t.Cleanup(func() {
		if had {
			PlaneMap[name] = old
		} else {
			delete(PlaneMap, name)
		}
	})
}

// TestCanTakeOffWithinRangeChecksDeckGroupAndPlaneRange 验证基地能否按目标类型、甲板起飞点与航程派出飞机。
// 场景：AI 规划增援/空袭时会问“这艘航母能不能打到 50 格外”；禁用、无甲板或航程不足都应回答否。
// 预期：航程 100 的舰攻对 50 格返回 true、150 格 false；Disable/无 deck 返回 false。
// 覆盖：构造单起飞点+单编组，调用 CanTakeOffWithinRange 四个边界；同时验证 TakeoffPointCount。
func TestCanTakeOffWithinRangeChecksDeckGroupAndPlaneRange(t *testing.T) {
	const planeName = "test-range-plane"
	registerCombatTestPlane(t, planeName, &Plane{Name: planeName, Type: PlaneTypeTorpedoBomber, Range: 100})
	sa := &ShipAircraft{
		Groups: []PlaneGroup{{Name: planeName, TargetType: object.TypeShip, CurCount: 1}},
		deck:   &CarrierDeck{TakeoffPoints: []TakeoffPoint{{}}},
	}

	require.True(t, sa.CanTakeOffWithinRange(object.TypeShip, 50))
	require.False(t, sa.CanTakeOffWithinRange(object.TypeShip, 150))
	require.False(t, sa.CanTakeOffWithinRange(object.TypePlane, 50))

	sa.Disable = true
	require.False(t, sa.CanTakeOffWithinRange(object.TypeShip, 50))
	sa.Disable = false
	sa.deck = nil
	require.False(t, sa.CanTakeOffWithinRange(object.TypeShip, 50))

	require.Equal(t, 1, (&ShipAircraft{deck: &CarrierDeck{TakeoffPoints: []TakeoffPoint{{}}}}).TakeoffPointCount())
	require.Zero(t, (&ShipAircraft{}).TakeoffPointCount())
}

// TestHasAntiSubAircraftDetectsLoadedASWOrdnance 验证基地反潜能力取决于编组内飞机是否挂载反潜弹。
// 场景：AI 选择反潜巡逻队时查询 HasAntiSubAircraft；普通轰炸机/无挂载不能误判为可反潜。
// 预期：挂载 AntiSubmarine 炸弹的机型返回 true；普通炸弹或空编组返回 false。
// 覆盖：注册两种机型并构造 Groups，分别调用 HasAntiSubAircraft。
func TestHasAntiSubAircraftDetectsLoadedASWOrdnance(t *testing.T) {
	const aswName = "test-asw-plane"
	const normalName = "test-normal-plane"
	registerCombatTestPlane(t, aswName, &Plane{
		Name:   aswName,
		Weapon: PlaneWeapon{Bombs: []*Releaser{{AntiSubmarine: true}}},
	})
	registerCombatTestPlane(t, normalName, &Plane{
		Name:   normalName,
		Weapon: PlaneWeapon{Bombs: []*Releaser{{AntiSubmarine: false}}},
	})

	require.True(t, (&ShipAircraft{Groups: []PlaneGroup{{Name: aswName}}}).HasAntiSubAircraft())
	require.False(t, (&ShipAircraft{Groups: []PlaneGroup{{Name: normalName}}}).HasAntiSubAircraft())
	require.False(t, (&ShipAircraft{}).HasAntiSubAircraft())
}

// TestGunInShotRangeAndAntiType 验证火炮的射程/射界判定与目标类型过滤。
// 场景：炮塔左射界 45°~135°、右射界 225°~315°，正东/正西目标分别落在左右射界，正北在盲区；
// 超出 10 格也不可打。
// 预期：InShotRange 对 (5,0)/(−5,0) true，(20,0)/(0,5) false；IsAvailableAntiType 区分飞机/舰船。
// 覆盖：构造单门炮，逐点调用两个方法。
func TestGunInShotRangeAndAntiType(t *testing.T) {
	gun := &Gun{
		Range:          10,
		AntiAircraft:   true,
		AntiShip:       true,
		LeftFiringArc:  FiringArc{Start: 45, End: 135},
		RightFiringArc: FiringArc{Start: 225, End: 315},
	}
	origin := objPos.New(0, 0)
	require.True(t, gun.InShotRange(0, origin, objPos.New(5, 0)))
	require.False(t, gun.InShotRange(0, origin, objPos.New(20, 0)))
	require.True(t, gun.InShotRange(0, origin, objPos.New(-5, 0)))
	require.False(t, gun.InShotRange(0, origin, objPos.New(0, 5)))

	require.True(t, gun.IsAvailableAntiType(object.TypePlane))
	require.True(t, gun.IsAvailableAntiType(object.TypeShip))
	require.False(t, gun.IsAvailableAntiType(object.TypeLand))
}
