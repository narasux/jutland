package tutorial

// Rect 教学卡片的屏幕矩形。
type Rect struct {
	X, Y, W, H float64
}

// Contains 判断屏幕坐标是否落在矩形内。
func (r Rect) Contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

// UILayout 步骤卡片的固定布局；绘制与命中判定共用，避免两套坐标。
type UILayout struct {
	Card   Rect
	Header Rect
	Title  Rect
	Body   Rect
	KeyRow Rect
	Next   Rect
	Skip   Rect
	Exit   Rect
	// Toggle 是收起 / 展开把手的可点击区域。
	Toggle Rect
	// Collapsed 表示本次布局是收起态：Card 只剩底部窄条，其余区域为零值。
	Collapsed bool
}

const (
	// tutorialToggleW / tutorialToggleH 是收起/展开把手的胶囊尺寸。
	tutorialToggleW = 132.0
	tutorialToggleH = 22.0
	// tutorialCollapsedH 是收起后底部窄条的高度。
	tutorialCollapsedH = 34.0
)

func clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// CalcLayout 按屏幕逻辑尺寸计算步骤卡布局；collapsed 为 true 时只给底部窄条和把手。
func CalcLayout(screenW, screenH float64, collapsed bool) UILayout {
	if collapsed {
		barW := clamp(screenW*0.4, 360, 640)
		barX := (screenW - barW) / 2
		barY := screenH - tutorialCollapsedH
		return UILayout{
			Card:      Rect{X: barX, Y: barY, W: barW, H: tutorialCollapsedH},
			Toggle:    Rect{X: barX + barW - 36, Y: barY + 5, W: 26, H: tutorialCollapsedH - 10},
			Collapsed: true,
		}
	}

	cardW := clamp(screenW*0.56, 600, 960)
	cardH := 196.0
	if screenH < 720 {
		cardH = 180
	}
	cardX := (screenW - cardW) / 2
	cardY := screenH - cardH - 24

	pad := 18.0
	exit := Rect{X: cardX + cardW - pad - 80, Y: cardY + 10, W: 80, H: 26}
	next := Rect{X: cardX + cardW - pad - 130, Y: cardY + cardH - pad - 32, W: 130, H: 32}
	skip := Rect{X: next.X - 10 - 90, Y: next.Y, W: 90, H: 32}

	buttonsLeft := skip.X
	if next.X < buttonsLeft {
		buttonsLeft = next.X
	}
	keyRow := Rect{
		X: cardX + pad,
		Y: cardY + cardH - pad - 32,
		W: buttonsLeft - (cardX + pad) - 12,
		H: 32,
	}
	if keyRow.W < 80 {
		keyRow.W = 80
	}

	return UILayout{
		Card:   Rect{X: cardX, Y: cardY, W: cardW, H: cardH},
		Header: Rect{X: cardX + pad, Y: cardY + 10, W: cardW - pad*2 - 86, H: 20},
		Title:  Rect{X: cardX + pad, Y: cardY + 32, W: cardW - pad*2 - 86, H: 28},
		Body:   Rect{X: cardX + pad, Y: cardY + 64, W: cardW - pad*2, H: cardH - 64 - 58},
		KeyRow: keyRow,
		Next:   next,
		Skip:   skip,
		Exit:   exit,
		// 把手骑在卡片上缘中央，像抽屉标签一样提示「可以收起」。
		Toggle: Rect{
			X: cardX + cardW/2 - tutorialToggleW/2,
			Y: cardY - tutorialToggleH/2,
			W: tutorialToggleW,
			H: tutorialToggleH,
		},
	}
}
