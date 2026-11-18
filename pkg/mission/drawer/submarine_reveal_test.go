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
		Uid: "sub", Name: "u_96", BelongPlayer: faction.ComputerAlpha,
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

// TestRevealSubmarinesDrawsRedBoxForUndetectedSub 验证调试开关只给未探测敌潜艇加红框，不改战斗可见性。
// 场景：敌潜艇 u_96 下潜于 (3,3)，玩家无声呐记忆；开启 Player.RevealSubmarines 后重绘一帧。
// 预期：关闭时 CanSeeShip=false；开启后 RevealsSubmarine=true 且仍不可见；失败意味着秘籍泄露视野。
// 覆盖：revealTestState 构造 SimTick=1 的潜航潜艇，两次 drawRevealScene 后断言两个可见性判断。
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

// TestRevealSubmarinesToggleDoesNotDrawForVisibleSub 验证已可见的水面潜艇不再叠加红框分支。
// 场景：潜艇上浮且 FogOfWar=false，玩家本来就能看见；此时秘籍开关仍开着。
// 预期：CanSeeShip=true 且 RevealsSubmarine=false；失败意味着可见潜艇被重复标注，红框盖住舰体。
// 覆盖：把 revealTestState 的潜艇改为 DiveStateSurface、关闭迷雾、打开 RevealSubmarines，断言两项判断。
func TestRevealSubmarinesToggleDoesNotDrawForVisibleSub(t *testing.T) {
	ms, sub := revealTestState()
	// 潜艇浮出水面：按普通迷雾规则可见，不再需要红框
	sub.DiveState = objUnit.DiveStateSurface
	ms.Core.FogOfWar = false
	ms.Player.RevealSubmarines = true

	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))
}

// TestOwnSubmarineDiveRenderingDoesNotPanic 验证己方潜艇的潜航表现与状态图标在常显开关两态下都能绘制。
// 场景：把 revealTestState 的 u_96 改为己方、位置 (5,5)，StartDive() 从水面进入 DiveStateDiving；200×200 屏幕、相机 10×10，ForceDisplayState 先 true 后 false 各画一帧。
// 预期：drawBattleShips 两次都正常返回，覆盖己方潜艇的潜航淡出、潜航状态牌，以及状态行在 ForceDisplayState 开/关下的两种分支；失败意味着下潜过渡表现让绘制崩溃。
// 覆盖：revealTestState + StartDive 断言成功，再调 drawRevealScene（Drawer.drawBattleShips 外包 require.NotPanics）两遍，不做像素校验。
func TestOwnSubmarineDiveRenderingDoesNotPanic(t *testing.T) {
	ms, sub := revealTestState()
	sub.BelongPlayer = faction.HumanAlpha
	sub.CurPos = objPos.NewR(5, 5)
	sub.DiveState = objUnit.DiveStateSurface
	require.True(t, sub.StartDive())

	ms.UI.GameOpts.ForceDisplayState = true
	drawRevealScene(t, ms)

	ms.UI.GameOpts.ForceDisplayState = false
	drawRevealScene(t, ms)
}
