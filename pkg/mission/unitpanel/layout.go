package unitpanel

import (
	"math"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/theme"
)

// 单位内容区各分区的固定尺寸（局部坐标系，相对滚动视口原点）。
const (
	// headerHeight 是内容区顶部「舰名 + 资金 + 己方舰数」标题带的高度。
	headerHeight = 30.0
	// sectionGap 是内容区相邻分区之间的屏幕像素间距。
	sectionGap = 12.0
	// infoLineH 是基础信息分区单行的屏幕像素高度。
	infoLineH = 17.0
	// infoTitleH 是基础信息分区「基础信息」标题带的高度。
	infoTitleH = 26.0
	// weaponRowH 是武器控制分区单行的屏幕像素高度（两行式：信息行 + 进度行）。
	weaponRowH = 44.0
	// aircraftRowH 是航空状态表单行的屏幕像素高度。
	aircraftRowH = 24.0
	// systemTabH 是武器/航空页签条的高度。
	systemTabH = 30.0
	// focusRowH 是多选舰船列表单行的屏幕像素高度。
	focusRowH = 30.0
	// maxFocusRows 是多选焦点舰列表单次展示的最大行数，超出部分通过上一艘/下一艘切换。
	maxFocusRows = 6
	// contentPad 是内容区左右以及顶部相对滚动视口的内边距。
	contentPad = float64(theme.PanelPadding)
)

type Rect struct {
	X, Y, W, H float64
}

func (r Rect) contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

// panelLayout 描述单位内容在滚动视口中的纵向分区几何。
// 所有坐标均为局部坐标：原点 (0,0) 是滚动视口左上角，Y 已叠加 scrollY 偏移。
type panelLayout struct {
	Region        Rect
	Header        Rect
	Visual        Rect
	Info          Rect
	Systems       Rect
	contentHeight float64
}

// sectionHeights 计算各分区在给定内容宽度下的自然高度。
// 内容超出视口高度时由外层滚动条处理，而不是压缩行高。
func (p *Panel) sectionHeights(ms *state.MissionState, width float64) (visualH, infoH, systemsH float64) {
	ships := selectedShips(ms)
	if len(ships) == 0 {
		return 0, 0, 0
	}
	if len(ships) == 1 {
		// 单舰侧/俯视图占一个近似正方形区域。
		visualH = math.Min(width, 180)
	} else {
		// 多选展示焦点舰列表：只显示以焦点为中心的一组行，避免选中大量舰船时列表过高。
		rows := min(len(ships), maxFocusRows)
		visualH = 28 + float64(rows)*focusRowH + 8
	}
	if groups := p.buildInfoGroups(ms, ships); len(groups) > 0 {
		infoH = infoTitleH + float64(infoLineCount(groups))*infoLineH
	}
	if p.tab == TabAircraft {
		if rows, _ := aircraftRows(ms); len(rows) > 0 {
			// 页签条 + 起飞总开关 + 表头 + 各行 + 合计行。
			systemsH = systemTabH + 46 + float64(len(rows)+2)*aircraftRowH
		} else {
			systemsH = systemTabH + 42
		}
	} else {
		if rows := weaponRows(ms, nowMillis()); len(rows) > 0 {
			// 页签条 + 全部武器开关 + 各行。
			systemsH = systemTabH + 46 + float64(len(rows))*weaponRowH
		} else {
			systemsH = systemTabH + 42
		}
	}
	return
}

// calcLayout 计算滚动视口内各分区的纵向布局。
// region 为外层传入的视口（屏幕坐标），本函数将其归一化到以自身左上角为原点的局部坐标，
// 并按 scrollY 上移，便于在离屏缓冲里做裁剪。
func (p *Panel) calcLayout(ms *state.MissionState, region Rect, scrollY float64) panelLayout {
	visualH, infoH, systemsH := p.sectionHeights(ms, region.W)
	innerX := contentPad
	innerW := region.W - contentPad*2
	cardTop := contentPad

	layout := panelLayout{Region: Rect{X: 0, Y: 0, W: region.W, H: region.H}}
	header := Rect{X: innerX, Y: cardTop, W: innerW, H: headerHeight}
	layout.Header = header
	cursor := header.Y + headerHeight + sectionGap

	place := func(h float64) Rect {
		r := Rect{X: innerX, Y: cursor, W: innerW, H: h}
		cursor = cursor + h + sectionGap
		return r
	}
	if visualH > 0 {
		layout.Visual = place(visualH)
	}
	if infoH > 0 {
		layout.Info = place(infoH)
	}
	if systemsH > 0 {
		layout.Systems = place(systemsH)
	}
	// 自然内容高度：从视口顶部到最后一个分区底部的距离。
	bottom := header.Y + headerHeight
	for _, section := range []Rect{layout.Visual, layout.Info, layout.Systems} {
		if section.H > 0 {
			bottom = section.Y + section.H
		}
	}
	layout.contentHeight = bottom

	// 应用滚动偏移：所有分区上移 scrollY。
	offset := -scrollY
	layout.Region.Y += offset
	layout.Header.Y += offset
	layout.Visual.Y += offset
	layout.Info.Y += offset
	layout.Systems.Y += offset
	return layout
}

// infoLineCount 累计分组节占用的行数（节标题行 + 字段行）。
func infoLineCount(groups []infoGroup) int {
	lines := 0
	for _, group := range groups {
		if group.title != "" {
			lines++
		}
		lines += len(group.items)
	}
	return lines
}
