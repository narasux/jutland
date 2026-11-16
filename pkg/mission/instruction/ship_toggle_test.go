package instruction

import (
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestWeaponToggleInstructionsUpdateShip 验证启用/禁用武器指令直接修改战舰分组与单体开关。
// 场景：单位面板点击“启用/禁用主炮”或 AI 下发同类指令；若只改分组标记而漏单门炮，面板与实战会不一致。
// 预期：EnableWeapon 后 MainGunDisabled=false 且每门炮 Disable=false；DisableWeapon 后两者都为 true；
// 两个指令执行后 Executed=true，Uid 使用舰船 UID。
// 覆盖：构造主炮已禁用的战舰，依次执行 NewEnableWeapon/NewDisableWeapon 并逐项断言。
func TestWeaponToggleInstructionsUpdateShip(t *testing.T) {
	ship := &objUnit.BattleShip{
		Uid: "bb",
		Weapon: objUnit.ShipWeapon{
			MainGuns:        []*objUnit.Gun{{Disable: true}},
			MainGunDisabled: true,
		},
	}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
	}}

	enable := NewEnableWeapon(ship.Uid, objUnit.WeaponTypeMainGun)
	require.NoError(t, enable.Exec(ms))
	require.True(t, enable.Executed())
	require.False(t, ship.Weapon.MainGunDisabled)
	require.False(t, ship.Weapon.MainGuns[0].Disable)
	require.Equal(t, GenInstrUid(NameEnableWeapon, ship.Uid), enable.Uid())
	require.Contains(t, enable.String(), "mainGun")

	disable := NewDisableWeapon(ship.Uid, objUnit.WeaponTypeMainGun)
	require.NoError(t, disable.Exec(ms))
	require.True(t, disable.Executed())
	require.True(t, ship.Weapon.MainGunDisabled)
	require.True(t, ship.Weapon.MainGuns[0].Disable)
	require.Equal(t, GenInstrUid(NameDisableWeapon, ship.Uid), disable.Uid())
}

// TestAircraftToggleInstructionsUpdateDeck 验证启用/禁用舰载机指令修改机库 Disable 开关。
// 场景：航母暂停放飞/恢复放飞；若指令只改 UI 状态而未写回 ShipAircraft.Disable，
// 自动警戒和手动起飞仍会继续派机。
// 预期：EnableAircraft 后 Disable=false；DisableAircraft 后 Disable=true；两者 Executed=true。
// 覆盖：构造已禁用机库的航母，依次执行两条指令并断言开关与 UID。
func TestAircraftToggleInstructionsUpdateDeck(t *testing.T) {
	ship := &objUnit.BattleShip{
		Uid:      "cv",
		Aircraft: objUnit.ShipAircraft{Disable: true},
	}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
	}}

	enable := NewEnableAircraft(ship.Uid)
	require.NoError(t, enable.Exec(ms))
	require.True(t, enable.Executed())
	require.False(t, ship.Aircraft.Disable)
	require.Equal(t, GenInstrUid(NameEnableAircraft, ship.Uid), enable.Uid())

	disable := NewDisableAircraft(ship.Uid)
	require.NoError(t, disable.Exec(ms))
	require.True(t, disable.Executed())
	require.True(t, ship.Aircraft.Disable)
	require.Equal(t, GenInstrUid(NameDisableAircraft, ship.Uid), disable.Uid())
}

// TestToggleInstructionsIgnoreMissingShip 验证目标舰船已不存在时四条切换指令仍安全完成。
// 场景：指令入队后战舰被击沉并从 Arena.Ships 移除；队列不能因找不到目标而报错或卡住。
// 预期：四条 Exec 都返回 nil、Executed=true，不 panic。
// 覆盖：空 MissionState 上依次执行 Enable/DisableWeapon 与 Enable/DisableAircraft。
func TestToggleInstructionsIgnoreMissingShip(t *testing.T) {
	ms := &state.MissionState{}
	instructions := []Instruction{
		NewEnableWeapon("missing", objUnit.WeaponTypeAll),
		NewDisableWeapon("missing", objUnit.WeaponTypeAll),
		NewEnableAircraft("missing"),
		NewDisableAircraft("missing"),
	}
	for _, instr := range instructions {
		require.NoError(t, instr.Exec(ms))
		require.True(t, instr.Executed())
	}
}
