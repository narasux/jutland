package unitpanel

import (
	"math"

	"github.com/narasux/jutland/pkg/utils/layout"
)

const (
	// handleWidth 与 handleHeight 是底部展开把手的屏幕像素尺寸。
	// 尺寸需要同时容纳明确的动作提示，并在全屏海面上保持足够可见性。
	handleWidth  = 240.0
	handleHeight = 48.0
	// handleBottomMargin 是折叠把手与屏幕底边之间的安全距离，避免入口贴边融入 HUD。
	handleBottomMargin = 12.0
	// headerHeight 是展开面板标题栏的屏幕像素高度。
	headerHeight = 38.0
	// panelHeightRatio 是面板高度相对屏幕高度的比例。
	panelHeightRatio = 0.30
	// panelMinHeight 与 panelMaxHeight 限制不同分辨率下的面板高度。
	panelMinHeight = 240.0
	panelMaxHeight = 340.0
	// panelPadding 是内容与面板边界之间的屏幕像素间距。
	panelPadding = 14.0
	// columnGap 是三栏内容之间的屏幕像素间距。
	columnGap = 12.0
)

type rect struct {
	X, Y, W, H float64
}

func (r rect) contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

type panelLayout struct {
	Screen  layout.ScreenLayout
	Panel   rect
	Handle  rect
	Header  rect
	Visual  rect
	Info    rect
	Systems rect
}

// expandedPanelHeight 返回当前分辨率下展开面板的屏幕像素高度。
func expandedPanelHeight(screen layout.ScreenLayout) float64 {
	return math.Max(panelMinHeight, math.Min(float64(screen.Height)*panelHeightRatio, panelMaxHeight))
}

// calcLayout 计算底部面板的稳定三栏布局；rightInset 为展开的右侧栏占用宽度。
func calcLayout(screen layout.ScreenLayout, expanded bool, rightInset float64) panelLayout {
	availableW := math.Max(480, float64(screen.Width)-rightInset)
	panelH := expandedPanelHeight(screen)
	panelY := float64(screen.Height)
	if expanded {
		panelY -= panelH
	}
	handleY := float64(screen.Height) - handleHeight - handleBottomMargin
	if expanded {
		handleY = panelY - handleHeight
	}
	handleX := math.Max(0, (availableW-handleWidth)/2)
	panel := rect{X: 0, Y: panelY, W: availableW, H: panelH}
	header := rect{X: panel.X, Y: panel.Y, W: panel.W, H: headerHeight}

	contentX := panel.X + panelPadding
	contentY := panel.Y + headerHeight + panelPadding
	contentW := panel.W - panelPadding*2
	contentH := panel.H - headerHeight - panelPadding*2
	visualW := math.Max(150, contentW*0.23)
	infoW := math.Max(190, contentW*0.27)
	systemsW := math.Max(220, contentW-visualW-infoW-columnGap*2)
	if visualW+infoW+systemsW+columnGap*2 > contentW {
		scale := contentW / (visualW + infoW + systemsW + columnGap*2)
		visualW *= scale
		infoW *= scale
		systemsW *= scale
	}

	visual := rect{X: contentX, Y: contentY, W: visualW, H: contentH}
	info := rect{X: visual.X + visual.W + columnGap, Y: contentY, W: infoW, H: contentH}
	systems := rect{X: info.X + info.W + columnGap, Y: contentY, W: systemsW, H: contentH}
	return panelLayout{
		Screen:  screen,
		Panel:   panel,
		Handle:  rect{X: handleX, Y: handleY, W: handleWidth, H: handleHeight},
		Header:  header,
		Visual:  visual,
		Info:    info,
		Systems: systems,
	}
}
