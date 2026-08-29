package theme

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/resources/font"
)

// ---------------------------------------------------------------------------
// 字体档位（字号）
// ---------------------------------------------------------------------------

const (
	// SizeCaption 是次级说明/单位等最小编号。
	SizeCaption = 13
	// SizeBody 是标签、数值等正文档位。
	SizeBody = 15
	// SizeSection 是分区标题/卡片标题档位。
	SizeSection = 16
	// SizeTitle 是面板主标题档位。
	SizeTitle = 20
)

// ---------------------------------------------------------------------------
// 几何常量
// ---------------------------------------------------------------------------

const (
	// CornerRadius 是卡片、按钮、进度条等控件的统一圆角半径（屏幕像素）。
	CornerRadius = 6
	// PanelPadding 是内容与面板边界之间的统一内边距。
	PanelPadding = 14
	// CardGap 是相邻卡片/分区之间的统一间距。
	CardGap = 12
	// PanelBorderWidth 是面板外框描边宽度。
	PanelBorderWidth = 2
	// CardBorderWidth 是卡片/分区描边宽度。
	CardBorderWidth = 1
)

// ---------------------------------------------------------------------------
// 调色板（右侧 sidebar 与底部 unitpanel 共用）
// ---------------------------------------------------------------------------

var (
	// PanelBackground 是两套面板共用的深海军蓝底色。
	// 统一右侧(A:222)与底部(A:232)的透明度差异。
	PanelBackground = color.RGBA{R: 8, G: 18, B: 23, A: 228}
	// PanelBorder 是面板外框描边。
	PanelBorder = color.RGBA{R: 85, G: 132, B: 143, A: 235}

	// CardBackground 是卡片/分区底色。
	// 统一右侧(A:214)与底部(A:220)的透明度差异。
	CardBackground = color.RGBA{R: 12, G: 31, B: 38, A: 218}
	// CardBorder 是卡片/分区描边。
	CardBorder = color.RGBA{R: 59, G: 97, B: 106, A: 220}

	// ButtonIdle 是按钮默认态填充。
	ButtonIdle = color.RGBA{R: 21, G: 48, B: 56, A: 245}
	// ButtonHover 是按钮悬停态填充。
	ButtonHover = color.RGBA{R: 31, G: 66, B: 75, A: 250}
	// ButtonPressed 是按钮按下态填充。
	ButtonPressed = color.RGBA{R: 12, G: 34, B: 41, A: 250}
	// ButtonDisabled 是按钮禁用态填充（偏暗红）。
	ButtonDisabled = color.RGBA{R: 72, G: 35, B: 35, A: 235}
	// ButtonMixed 是按钮混合态填充（黄褐色）。
	ButtonMixed = color.RGBA{R: 66, G: 59, B: 35, A: 235}

	// HandleFill 是展开/收起把手默认态填充。
	HandleFill = color.RGBA{R: 18, G: 52, B: 62, A: 252}
	// HandleHover 是把手悬停态填充。
	HandleHover = color.RGBA{R: 30, G: 76, B: 87, A: 255}
	// HandleBorder 是把手描边（低饱和青灰，弱化原先的金强调色）。
	HandleBorder = color.RGBA{R: 96, G: 140, B: 150, A: 150}
	// HandleArrow 是把手展开/收起箭头颜色（低饱和钢灰蓝，去掉黄色）。
	HandleArrow = color.RGBA{R: 150, G: 175, B: 185, A: 200}
	// HandleShadow 是把手投影。
	HandleShadow = color.RGBA{R: 0, G: 0, B: 0, A: 120}

	// ProgressTrack 是进度条轨道。
	ProgressTrack = color.RGBA{R: 28, G: 52, B: 59, A: 245}
	// ProgressFill 是进度条填充。
	ProgressFill = color.RGBA{R: 196, G: 171, B: 101, A: 245}
)

// ---------------------------------------------------------------------------
// 字体档位选择
// ---------------------------------------------------------------------------

// Title 返回面板主标题/分区标题所需的界面字体（语言相关）。
func Title() *text.GoTextFaceSource {
	return font.LocalizedTitle(font.Hang)
}

// Body 返回正文标签/数值所需的界面字体（语言相关）。
func Body() *text.GoTextFaceSource {
	return font.LocalizedUI(font.Kai)
}

// Numeric 返回纯数字/单位列所需的字体。数值只含拉丁数字与单位，四种语言共用
// 等宽字体即可清晰对齐；若值含 CJK/西里尔字形则由调用方回退到 Body()。
func Numeric() *text.GoTextFaceSource {
	return font.JetbrainsMono
}

// ---------------------------------------------------------------------------
// 通用绘制帮手
// ---------------------------------------------------------------------------

// FillRoundedRect 绘制带统一圆角的填充矩形，可选描边。
// 右侧 sidebar 与底部 unitpanel 共用，保证两套面板的圆角、描边语言一致。
func FillRoundedRect(
	screen *ebiten.Image,
	x, y, w, h, radius float64,
	fill, stroke color.Color,
	strokeWidth float32,
) {
	radius = min(radius, min(w, h)/2)
	fx, fy, fw, fh, r := float32(x), float32(y), float32(w), float32(h), float32(radius)
	var path vector.Path
	path.MoveTo(fx+r, fy)
	path.LineTo(fx+fw-r, fy)
	path.QuadTo(fx+fw, fy, fx+fw, fy+r)
	path.LineTo(fx+fw, fy+fh-r)
	path.QuadTo(fx+fw, fy+fh, fx+fw-r, fy+fh)
	path.LineTo(fx+r, fy+fh)
	path.QuadTo(fx, fy+fh, fx, fy+fh-r)
	path.LineTo(fx, fy+r)
	path.QuadTo(fx, fy, fx+r, fy)
	path.Close()

	fillOpts := &vector.DrawPathOptions{AntiAlias: true}
	fillOpts.ColorScale.ScaleWithColor(fill)
	vector.FillPath(screen, &path, &vector.FillOptions{}, fillOpts)
	if strokeWidth > 0 && stroke != nil {
		strokeOpts := &vector.DrawPathOptions{AntiAlias: true}
		strokeOpts.ColorScale.ScaleWithColor(stroke)
		vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: strokeWidth}, strokeOpts)
	}
}

// TriangleDir 表示 V 形折线箭头指向的方向。
type TriangleDir int

const (
	// TriangleUp 箭头朝上。
	TriangleUp TriangleDir = iota
	// TriangleDown 箭头朝下。
	TriangleDown
	// TriangleLeft 箭头朝左。
	TriangleLeft
	// TriangleRight 箭头朝右。
	TriangleRight
)

// chevronPoints 返回以 (centerX, centerY) 为中心的 V 形折线的起点、顶点、终点。
// 上下方向时 size 为开口半宽、深度取其一半；左右方向旋转 90 度（开口与深度互换），
// 线宽取深度值，保证四个方向下的折线形状一致。供 DrawChevron 与测试共用。
func chevronPoints(centerX, centerY, size float64, dir TriangleDir) ([2]float64, [2]float64, [2]float64) {
	opening, depth := size, size/2
	switch dir {
	case TriangleUp:
		return [2]float64{centerX - opening, centerY + depth},
			[2]float64{centerX, centerY - depth},
			[2]float64{centerX + opening, centerY + depth}
	case TriangleLeft:
		return [2]float64{centerX + depth, centerY - opening},
			[2]float64{centerX - depth, centerY},
			[2]float64{centerX + depth, centerY + opening}
	case TriangleRight:
		return [2]float64{centerX - depth, centerY - opening},
			[2]float64{centerX + depth, centerY},
			[2]float64{centerX - depth, centerY + opening}
	default: // TriangleDown
		return [2]float64{centerX - opening, centerY - depth},
			[2]float64{centerX, centerY + depth},
			[2]float64{centerX + opening, centerY - depth}
	}
}

// DrawChevron 绘制一个以 (centerX, centerY) 为中心、size 为开口半宽的 V 形折线箭头，
// 使用圆头圆角描边，视觉上比实心三角形更轻盈。右侧 sidebar 与底部 unitpanel 共用，
// 保证两套把手的箭头语言一致。
func DrawChevron(screen *ebiten.Image, centerX, centerY, size float64, dir TriangleDir, clr color.Color) {
	start, apex, end := chevronPoints(centerX, centerY, size, dir)

	var path vector.Path
	path.MoveTo(float32(start[0]), float32(start[1]))
	path.LineTo(float32(apex[0]), float32(apex[1]))
	path.LineTo(float32(end[0]), float32(end[1]))
	strokeOp := &vector.StrokeOptions{
		Width:    float32(size / 2),
		LineCap:  vector.LineCapRound,
		LineJoin: vector.LineJoinRound,
	}
	opts := &vector.DrawPathOptions{AntiAlias: true}
	opts.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(screen, &path, strokeOp, opts)
}
