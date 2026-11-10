package drawer

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// revealTestState 构造一艘未探测到的敌方潜艇，并开启调试暴露。
func revealTestState() (*state.MissionState, *objUnit.BattleShip) {
	sub := &objUnit.BattleShip{
		Uid: "sub", Name: "u_boat_type_viic", BelongPlayer: faction.ComputerAlpha,
		CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
		CurPos:    objPos.NewR(3, 3), CurRotation: 90,
		Length: 67.2, Width: 6.2,
	}
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 1},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 200, Height: 200},
			Camera: state.Camera{Width: 10, Height: 10},
		},
		UI:     state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena:  state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{sub.Uid: sub}},
	}
	return ms, sub
}

// drawRevealScene 绘制一帧，确认新的红框分支能正常跑通（不校验像素：
// ebiten 在测试进程里不允许 ReadPixels，属于运行期限制）。
func drawRevealScene(t *testing.T, ms *state.MissionState) {
	t.Helper()
	screen := ebiten.NewImage(200, 200)
	require.NotPanics(t, func() {
		(&Drawer{}).drawBattleShips(screen, ms)
	})
}

func TestRevealSubmarinesDrawsRedBoxForUndetectedSub(t *testing.T) {
	ms, sub := revealTestState()
	drawRevealScene(t, ms)
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub), "秘籍关闭时潜艇不可见")

	ms.Player.RevealSubmarines = true
	drawRevealScene(t, ms)
	require.True(t, ms.RevealsSubmarine(faction.HumanAlpha, sub), "秘籍开启后应走红框分支")
	// 只做表现：战斗可见性不变
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))
}

func TestRevealSubmarinesToggleDoesNotDrawForVisibleSub(t *testing.T) {
	ms, sub := revealTestState()
	// 潜艇浮出水面：按普通迷雾规则可见，不再需要红框
	sub.DiveState = objUnit.DiveStateSurface
	ms.Core.FogOfWar = false
	ms.Player.RevealSubmarines = true

	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))
}
