package state

import "github.com/narasux/jutland/pkg/utils/layout"

// PauseUIRect 表示暂停面板中可绘制、可点击的矩形区域。
type PauseUIRect struct {
	X, Y, W, H float64
}

// Contains 判断屏幕坐标是否落在暂停 UI 矩形内。
func (r PauseUIRect) Contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

// PauseUILayout 保存暂停面板和按钮的共享布局，避免绘制区域和点击区域不一致。
type PauseUILayout struct {
	Panel         PauseUIRect
	PrimaryButton PauseUIRect
	DangerButton  PauseUIRect
}

// CalcPauseUILayout 根据当前屏幕尺寸计算暂停面板布局。
func CalcPauseUILayout(screen layout.ScreenLayout) PauseUILayout {
	panelW, panelH := 420.0, 220.0
	buttonW, buttonH := 112.0, 38.0
	gap := 34.0

	panelX := (float64(screen.Width) - panelW) / 2
	panelY := (float64(screen.Height) - panelH) / 2
	buttonY := panelY + 126
	primaryX := panelX + (panelW-buttonW*2-gap)/2

	return PauseUILayout{
		Panel:         PauseUIRect{X: panelX, Y: panelY, W: panelW, H: panelH},
		PrimaryButton: PauseUIRect{X: primaryX, Y: buttonY, W: buttonW, H: buttonH},
		DangerButton:  PauseUIRect{X: primaryX + buttonW + gap, Y: buttonY, W: buttonW, H: buttonH},
	}
}
