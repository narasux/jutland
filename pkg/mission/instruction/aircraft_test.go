package instruction

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestAircraftToggleInstructions 验证 EnableAircraft/DisableAircraft 直接翻转起降开关。
// 场景：航母 "carrier" 的 Aircraft.Disable 默认 false，先执行禁飞再执行放飞，
// 两条都是即时指令，Exec 后应立刻生效并标记 Executed。
// 预期：禁飞后 Disable=true 且 disable.Executed()；放飞后 Disable=false 且
// enable.Executed()。开关没翻转会让舰载机菜单与实际起飞许可不一致。
// 覆盖：只构造含 Ships 的 MissionState，按顺序执行两条指令并检查字段。
func TestAircraftToggleInstructions(t *testing.T) {
	ship := &objUnit.BattleShip{Uid: "carrier"}
	missionState := &state.MissionState{
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{ship.Uid: ship}},
	}

	disable := NewDisableAircraft(ship.Uid)
	if err := disable.Exec(missionState); err != nil {
		t.Fatal(err)
	}
	if !ship.Aircraft.Disable || !disable.Executed() {
		t.Fatal("disable instruction did not disable takeoff")
	}

	enable := NewEnableAircraft(ship.Uid)
	if err := enable.Exec(missionState); err != nil {
		t.Fatal(err)
	}
	if ship.Aircraft.Disable || !enable.Executed() {
		t.Fatal("enable instruction did not enable takeoff")
	}
}
