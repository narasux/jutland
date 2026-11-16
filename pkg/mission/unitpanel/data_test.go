package unitpanel

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestWeaponRowsHideUnequippedAndReportMixedState 验证武器行只为实际装备的类型生成，并按启用/禁用给出混合态。
// 场景：两舰都装主炮但 b 关闭主炮；只有 a 装鱼雷且已关闭；未装备的类型不应出现在控制列表。
// 预期：仅主炮、鱼雷 2 行，Toggle 分别为 mixed、disabled；失败意味着未装备类型混入或状态判断错。
// 覆盖：panelTestState 构造 a（主炮+禁用鱼雷）、b（主炮+禁用主炮），weaponRows(ms,10_000) 断言行与状态。
func TestWeaponRowsHideUnequippedAndReportMixedState(t *testing.T) {
	first := &objUnit.BattleShip{
		Uid: "a",
		Weapon: objUnit.ShipWeapon{
			MainGuns:        []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
			Torpedoes:       []*objUnit.TorpedoLauncher{{ReloadStartAt: 0, ReloadTime: 1}},
			TorpedoDisabled: true,
		},
		CurHP: 1,
	}
	second := &objUnit.BattleShip{
		Uid: "b",
		Weapon: objUnit.ShipWeapon{
			MainGuns:        []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
			MainGunDisabled: true,
		},
		CurHP: 1,
	}
	missionState := panelTestState(first, second)

	rows := weaponRows(missionState, 10_000)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want only main guns and torpedoes", len(rows))
	}
	if rows[0].Type != objUnit.WeaponTypeMainGun || rows[0].Toggle != toggleMixed {
		t.Fatalf("main gun row = %+v", rows[0])
	}
	if rows[1].Type != objUnit.WeaponTypeTorpedo || rows[1].Toggle != toggleDisabled {
		t.Fatalf("torpedo row = %+v", rows[1])
	}
}

// TestWeaponActionEnablesMixedSelectionAndIgnoresUnequipped 验证批量武器动作只作用于装备该武器的舰。
// 场景：a 主炮开、b 主炮关、c 无武器；点击主炮开关时存在禁用，应整体转为启用。
// 预期：Enable=true，ShipUids 为 [a,b] 共 2 个（不含 c）；失败意味着给无武器舰发指令或开关方向反了。
// 覆盖：panelTestState(a,b,c)，New().weaponAction(ms,WeaponTypeMainGun)，断言 Enable 与 ShipUids 长度。
func TestWeaponActionEnablesMixedSelectionAndIgnoresUnequipped(t *testing.T) {
	first := &objUnit.BattleShip{Uid: "a", CurHP: 1, Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{{}}}}
	second := &objUnit.BattleShip{
		Uid:    "b",
		CurHP:  1,
		Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{{}}, MainGunDisabled: true},
	}
	third := &objUnit.BattleShip{Uid: "c", CurHP: 1}
	missionState := panelTestState(first, second, third)

	action := New().weaponAction(missionState, objUnit.WeaponTypeMainGun)
	if !action.Enable || len(action.ShipUids) != 2 {
		t.Fatalf("action = %+v, want enable for two equipped ships", action)
	}
}

func panelTestState(ships ...*objUnit.BattleShip) *state.MissionState {
	byUID := make(map[string]*objUnit.BattleShip, len(ships))
	selected := make([]string, 0, len(ships))
	for _, ship := range ships {
		byUID[ship.Uid] = ship
		selected = append(selected, ship.Uid)
	}
	return &state.MissionState{
		Interaction: state.MissionInteractionState{SelectedShips: selected},
		Arena:       state.MissionArenaState{Ships: byUID, Planes: map[string]*objUnit.Plane{}},
	}
}

// TestShipSpeedRowUsesCurrentDiveStateLimit 验证单位面板航速行的上限跟随当前深度状态。
// 场景：config.G.SpeedMultiplier=2；潜艇 U-2323 水面 9.7 节、SubmergedSpeedRatio=1.29，CurSpeed 是世界系速度（已乘全局倍率），旧实现会让水下更快的潜艇显示「当前航速 > 最大航速」。
// 预期：潜航时航速行显示 "12.5 / 12.5"（9.7×1.29 四舍五入），切回水面显示 "9.7 / 9.7"；失败意味着上限仍按水面 MaxSpeed 计算，读数自相矛盾。
// 覆盖：临时设置 config.G 并 Cleanup 恢复；panelTestState + New().shipGroups 取 speedRowValue，两态比较，CurSpeedKnots 会抵消全局倍率。
func TestShipSpeedRowUsesCurrentDiveStateLimit(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 2}

	// U-2323：水面 9.7 节，水下倍率 1.29（约 12.5 节）。
	ship := &objUnit.BattleShip{
		Uid:                 "sub",
		Type:                objUnit.ShipTypeSubmarine,
		TotalHP:             100,
		CurHP:               100,
		MaxSpeed:            9.7 / objUnit.ShipSpeedScale,
		SubmergedSpeedRatio: 1.29,
		DiveState:           objUnit.DiveStateSubmerged,
		// CurSpeed 是世界系速度，已乘全局倍率。
		CurSpeed: 9.7 * 1.29 / objUnit.ShipSpeedScale * 2,
	}
	ms := panelTestState(ship)
	if got := speedRowValue(t, New().shipGroups(ms, ship)); got != "12.5 / 12.5" {
		t.Fatalf("submerged speed row = %q, want submerged cruise limit", got)
	}

	ship.DiveState = objUnit.DiveStateSurface
	ship.CurSpeed = 9.7 / objUnit.ShipSpeedScale * 2
	if got := speedRowValue(t, New().shipGroups(ms, ship)); got != "9.7 / 9.7" {
		t.Fatalf("surface speed row = %q, want surface speed", got)
	}
}

func speedRowValue(t *testing.T, groups []infoGroup) string {
	t.Helper()
	label := i18n.Text(i18n.MsgUnitPanelSpeed)
	for _, group := range groups {
		for _, item := range group.items {
			if item.label == label {
				return item.value
			}
		}
	}
	t.Fatalf("speed row %q not found", label)
	return ""
}
