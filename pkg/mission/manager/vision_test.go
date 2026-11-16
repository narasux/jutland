package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestUpdateVisionLightsInsideRadiusOnly 验证只有存活友军单位会点亮视野格，沉船不发光、停用机场仍发光。
// 场景：开启 FogOfWar 的 20x20 视野，友方 "alive" 在 (5.2,5.2) 视距 2；"dead" 的 CurHP=0、
// 视距 8 却停在 (15,15)；机场 "field" 被 Disabled 但位置在 (1,1)，仍应提供固定半径视野。
// 预期：(5,5) 可见且被记为已探索；(15,15) 不可见（沉船不能照亮地图）；(1,1) 可见。
// 失败意味着沉船或残骸继续揭图，或者停用机场凭空失去视野，迷雾边缘会与实际观察不符。
// 覆盖：updateVision 后分别断言 VisibleAt(5,5)、VisibleAt(15,15)、VisibleAt(1,1) 与 ExploredAt(5,5)。
func TestUpdateVisionLightsInsideRadiusOnly(t *testing.T) {
	vision := state.NewFactionVision(20, 20)
	ms := &state.MissionState{
		Core: state.MissionCoreState{FogOfWar: true},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			Visions: map[faction.Player]*state.FactionVision{
				faction.HumanAlpha: vision,
			},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"alive": {
					Uid: "alive", CurHP: 10, SightRange: 2,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.NewR(5.2, 5.2),
				},
				"dead": {
					Uid: "dead", CurHP: 0, SightRange: 8,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.New(15, 15),
				},
			},
			Airfields: map[string]*objBuilding.Airfield{
				"field": {
					Uid: "field", Disabled: true, BelongPlayer: faction.HumanAlpha,
					Pos: objPos.New(1, 1),
				},
			},
		},
	}

	(&MissionManager{state: ms}).updateVision()

	if !vision.VisibleAt(5, 5) {
		t.Fatal("cell under the ship should be visible")
	}
	if vision.VisibleAt(15, 15) {
		t.Fatal("dead ship should not light the map")
	}
	if !vision.VisibleAt(1, 1) {
		t.Fatal("disabled airfield should still light its cell")
	}
	if !vision.ExploredAt(5, 5) {
		t.Fatal("visible cells should become explored")
	}
}

// TestUpdateVisionSkipsWhenFogDisabled 验证关闭迷雾时 updateVision 直接返回、完全不创建视野结构。
// 场景：MissionState 未设置 Core.FogOfWar（默认 false），Arena 里仍有视距 4、位于 (2,2) 的友舰。
// 预期：调用 updateVision 后 Player.Visions 仍为 nil；失败意味着关闭迷雾仍会分配并扫描视野网格，
// 白白消耗每帧性能，还可能让绘制层误以为存在迷雾蒙层。
// 覆盖：只构造舰船与默认迷雾状态，调用 updateVision 后检查 Visions 映射为空。
func TestUpdateVisionSkipsWhenFogDisabled(t *testing.T) {
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"s": {
					Uid: "s", CurHP: 10, SightRange: 4,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.New(2, 2),
				},
			},
		},
	}
	(&MissionManager{state: ms}).updateVision()
	if ms.Player.Visions != nil {
		t.Fatal("disabled fog should not create vision maps")
	}
}
