package manager

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 计算下一帧任务状态
func (m *MissionManager) updateMissionStatus() {
	calcNextStatusByShips := func(curStatus state.MissionStatus) state.MissionStatus {
		// 还有战舰在沉没，游戏继续
		if len(m.state.Arena.DestroyedShips) != 0 {
			return curStatus
		}
		// 检查所有战舰，判定胜利 / 失败
		anySelfShip, anyEnemyShip := false, false
		for _, ship := range m.state.Arena.Ships {
			if ship.BelongPlayer == m.state.Player.CurPlayer {
				anySelfShip = true
			} else {
				anyEnemyShip = true
			}
		}
		// 自己的船都没了，失败
		if !anySelfShip && len(m.state.Arena.DestroyedShips) == 0 {
			return state.MissionFailed
		}
		// 敌人都不存在，胜利
		if !anyEnemyShip && len(m.state.Arena.DestroyedShips) == 0 {
			return state.MissionSuccess
		}
		return curStatus
	}

	switch m.state.Core.MissionStatus {
	case state.MissionRunning:
		// 暂停游戏
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.Core.MissionStatus = state.MissionPaused
			m.state.Core.ConfirmQuitMission = false
		}
		m.state.Core.MissionStatus = calcNextStatusByShips(m.state.Core.MissionStatus)
	case state.MissionPaused:
		if m.state.UI.DebugFlags.IsActive() {
			// debug 模式下跳过确认面板，Q 直接退出，Esc 直接继续
			if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
				m.state.Core.MissionStatus = state.MissionFailed
				m.state.Core.ConfirmQuitMission = true
				return
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				m.state.Core.MissionStatus = state.MissionRunning
				m.state.Core.ConfirmQuitMission = false
				return
			}
			return
		}

		input := state.PauseInputNone
		if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
			input = state.PauseInputQuit
		} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			input = state.PauseInputResume
		} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			sx, sy := ebiten.CursorPosition()
			ui := state.CalcPauseUILayout(m.state.View.Layout)
			if ui.PrimaryButton.Contains(sx, sy) {
				input = state.PauseInputResume
			} else if ui.DangerButton.Contains(sx, sy) {
				input = state.PauseInputQuit
			}
		}
		m.state.Core.MissionStatus, m.state.Core.ConfirmQuitMission = state.ApplyPauseInput(
			m.state.Core.MissionStatus,
			m.state.Core.ConfirmQuitMission,
			input,
		)
		return
	case state.MissionInMap:
		// 退出全屏地图模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.Core.MissionStatus = state.MissionRunning
		}
		m.state.Core.MissionStatus = calcNextStatusByShips(m.state.Core.MissionStatus)
	case state.MissionInTerminal:
		// 退出终端模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.Core.MissionStatus = state.MissionRunning
		}
		return
	case state.MissionInBuilding:
		// 退出建筑物交互模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.Core.MissionStatus = state.MissionRunning
		}
	default:
		m.state.Core.MissionStatus = state.MissionRunning
	}

	// 按下 m 键，切换地图展示模式
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		m.state.Core.MissionStatus = lo.Ternary(
			m.state.Core.MissionStatus != state.MissionInMap,
			state.MissionInMap,
			state.MissionRunning,
		)
	}

	// 按下 b 键，开启查看增援点模式
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		m.state.Core.MissionStatus = lo.Ternary(
			m.state.Core.MissionStatus != state.MissionInBuilding,
			state.MissionInBuilding,
			state.MissionRunning,
		)
	}

	// 按下 LeftCtrl，LeftShift 的同时按下 ` 键开启终端
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) &&
		ebiten.IsKeyPressed(ebiten.KeyShiftLeft) &&
		inpututil.IsKeyJustPressed(ebiten.KeyBackquote) {
		m.state.Core.MissionStatus = state.MissionInTerminal
		audio.PlayAudioToEnd(audioRes.NewCheating())
		// 进入终端会按下 ctrl，此时会导致进入编组模式，需要强制退出下
		m.state.Interaction.IsGrouping = false
	}
}
