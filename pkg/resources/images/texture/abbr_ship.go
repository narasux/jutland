package texture

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/utils/colorx"
)

// abbrShipIconSize 舰种缩略图画布边长。主力舰体约 7x13，接近旧版 medium 图标。
const abbrShipIconSize = 18

const (
	iconOutlineW float32 = 1.15
	iconMarkW    float32 = 1.05
)

var iconOutline = color.RGBA{R: 18, G: 20, B: 24, A: 255}

// shipIconSilhouette 舰种缩略图的轮廓种类。
type shipIconSilhouette int

const (
	// silTriangle 细短三角（驱逐舰、鱼雷艇、护卫舰）
	silTriangle shipIconSilhouette = iota
	// silHull 尖艏五边形（巡洋舰、战列舰、辅助舰）
	silHull
	// silCarrier 尖三角 + 窄矩形 + 长方形舰体（航母）
	silCarrier
	// silSub 三角加舰艉横杠（潜艇）
	silSub
)

// shipIconMark 内部标记。巡洋 1 斜线、战列 2 斜线、航母横向分割，与参考图一致。
type shipIconMark int

const (
	markNone shipIconMark = iota
	markSlash
	markSlashHeavy
	markSlash2
	markCarrier
	markBars
	markDot
	markCross
	markX
	markC
)

// shipIconSpec 描述一种舰种缩略图。
type shipIconSpec struct {
	sil        shipIconSilhouette
	mark       shipIconMark
	glyphScale float32 // 内部文字缩放，0 表示 1
}

// shipIconSpecs 按舰种缩写（typeAbbr）定义缩略图。
var shipIconSpecs = map[string]shipIconSpec{
	"DD":        {sil: silTriangle},
	"TB":        {sil: silTriangle, mark: markDot},
	"FF":        {sil: silTriangle, mark: markBars},
	"CL":        {sil: silHull, mark: markSlash},
	"CLAA":      {sil: silHull, mark: markSlash},
	"CGN":       {sil: silHull, mark: markSlash},
	"CA":        {sil: silHull, mark: markSlashHeavy},
	"CB":        {sil: silHull, mark: markSlashHeavy},
	"BB":        {sil: silHull, mark: markSlash2},
	"BC":        {sil: silHull, mark: markSlash2},
	"CV":        {sil: silCarrier, mark: markCarrier},
	"CVL":       {sil: silCarrier, mark: markCarrier},
	"CVB":       {sil: silCarrier, mark: markCarrier},
	"CVE":       {sil: silCarrier, mark: markCarrier},
	"MAC":       {sil: silCarrier, mark: markCarrier},
	"SS":        {sil: silSub},
	"HS":        {sil: silHull, mark: markCross, glyphScale: 0.55},
	"Cargo":     {sil: silHull, mark: markC, glyphScale: 0.55},
	"IX":        {sil: silHull, mark: markX, glyphScale: 0.55},
	"Duck":      {sil: silHull, mark: markX},
	"WaterDrop": {sil: silHull, mark: markX},
	"Swordfish": {sil: silHull, mark: markX},
	"Molamola":  {sil: silHull, mark: markX},
	"default":   {sil: silHull, mark: markX, glyphScale: 0.55},
}

var (
	shipIconOnce   sync.Once
	shipIcons      map[string]*ebiten.Image
	enemyShipIcons map[string]*ebiten.Image
)

// GetAbbrShip 按舰种缩写（typeAbbr）获取缩略图，敌我使用不同配色。
func GetAbbrShip(typeAbbr string, isEnemy bool) *ebiten.Image {
	buildShipIcons()

	icons := shipIcons
	if isEnemy {
		icons = enemyShipIcons
	}
	if img, ok := icons[typeAbbr]; ok {
		return img
	}
	return icons["default"]
}

func buildShipIcons() {
	shipIconOnce.Do(func() {
		shipIcons = make(map[string]*ebiten.Image, len(shipIconSpecs))
		enemyShipIcons = make(map[string]*ebiten.Image, len(shipIconSpecs))
		for abbr, spec := range shipIconSpecs {
			shipIcons[abbr] = ebiten.NewImageFromImage(renderShipIcon(spec, colorx.Green))
			enemyShipIcons[abbr] = ebiten.NewImageFromImage(renderShipIcon(spec, colorx.Red))
		}
	})
}

func renderShipIcon(spec shipIconSpec, fill color.Color) *image.RGBA {
	return renderShipIconScaled(spec, fill, 1)
}

func renderShipIconScaled(spec shipIconSpec, fill color.Color, scale float32) *image.RGBA {
	if scale < 1 {
		scale = 1
	}
	size := int(math.Round(float64(abbrShipIconSize) * float64(scale)))
	r := newIconRaster(size, size)
	if spec.sil == silSub {
		drawSubmarine(r, toRGBA(fill), scale)
		return r.image()
	}
	pts := scalePts(silhouettePoints(spec.sil), scale)
	r.fillPoly(pts, toRGBA(fill))
	r.clip = pts
	drawMarks(r, spec, pts, scale)
	r.clip = nil
	r.strokePoly(pts, iconOutlineW*scale, iconOutline)
	return r.image()
}

func scalePts(pts [][2]float32, scale float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, p := range pts {
		out[i] = [2]float32{p[0] * scale, p[1] * scale}
	}
	return out
}

func silhouettePoints(sil shipIconSilhouette) [][2]float32 {
	const cx = float32(abbrShipIconSize) / 2
	switch sil {
	case silTriangle:
		return [][2]float32{{cx, 3.2}, {11.6, 13.4}, {4.4, 13.4}}
	case silCarrier:
		return carrierPoints(cx, 1.6, 14.5, 3.6, 2.2, 3.35)
	default:
		return hullPoints(cx, 1.6, 14.5, 3.4, 3.35)
	}
}

func hullPoints(cx, bowY, stern, bowLen, halfW float32) [][2]float32 {
	shoulder := bowY + bowLen
	return [][2]float32{
		{cx, bowY},
		{cx + halfW, shoulder},
		{cx + halfW, stern},
		{cx - halfW, stern},
		{cx - halfW, shoulder},
	}
}

// carrierPoints 尖三角 + 很窄的长方形，其后才是主舰体。
func carrierPoints(cx, bowY, stern, triLen, neckLen, halfW float32) [][2]float32 {
	triBase := bowY + triLen
	neck := triBase + neckLen
	return [][2]float32{
		{cx, bowY},
		{cx + halfW, triBase},
		{cx + halfW, neck},
		{cx + halfW, stern},
		{cx - halfW, stern},
		{cx - halfW, neck},
		{cx - halfW, triBase},
	}
}

func drawSubmarine(r *iconRaster, fill color.RGBA, scale float32) {
	tri := scalePts([][2]float32{{8, 2.8}, {11.2, 10.6}, {4.8, 10.6}}, scale)
	bar := scalePts([][2]float32{{4.6, 12.2}, {11.4, 12.2}, {11.4, 14.4}, {4.6, 14.4}}, scale)
	r.fillPoly(tri, fill)
	r.fillPoly(bar, fill)
	r.strokePoly(tri, iconOutlineW*scale, iconOutline)
	r.strokePoly(bar, iconOutlineW*scale, iconOutline)
}

func drawMarks(r *iconRaster, spec shipIconSpec, pts [][2]float32, scale float32) {
	markW := iconMarkW * scale
	switch spec.mark {
	case markSlash:
		drawSlashes(r, pts, []float32{0.42}, markW)
	case markSlashHeavy:
		drawSlashes(r, pts, []float32{0.42}, markW+0.85*scale)
	case markSlash2:
		drawSlashes(r, pts, []float32{0.28, 0.58}, markW)
	case markCarrier:
		drawCarrierDeck(r, pts, markW)
	case markBars:
		r.strokeLine(6.6*scale, 8.2*scale, 9.4*scale, 8.2*scale, markW, iconOutline)
		r.strokeLine(5.8*scale, 12.0*scale, 10.2*scale, 12.0*scale, markW, iconOutline)
	case markDot:
		r.fillCircle(8*scale, 8.6*scale, 0.95*scale, iconOutline)
	case markCross:
		drawBodyPlus(r, pts, markW, spec.glyphScale)
	case markX:
		drawBodyX(r, pts, markW, spec.glyphScale)
	case markC:
		drawBodyC(r, pts, markW, spec.glyphScale)
	}
}

func drawSlashes(r *iconRaster, pts [][2]float32, stops []float32, width float32) {
	if len(pts) < 5 {
		return
	}
	leftA, leftB := pts[4], pts[3]
	rightA, rightB := pts[1], pts[2]
	for _, t := range stops {
		x1, y1 := lerp2(leftA, leftB, t)
		x2, y2 := lerp2(rightA, rightB, t+0.20)
		r.strokeLine(x1, y1, x2, y2, width, iconOutline)
	}
}

// drawCarrierDeck 窄矩形之后画竖向分割，主舰体内再画纵向中线。
func drawCarrierDeck(r *iconRaster, pts [][2]float32, width float32) {
	if len(pts) < 7 {
		return
	}
	r.strokeLine(pts[5][0], pts[5][1], pts[2][0], pts[2][1], width, iconOutline)
	sx, sy := (pts[5][0]+pts[2][0])/2, (pts[5][1]+pts[2][1])/2
	ex, ey := (pts[4][0]+pts[3][0])/2, (pts[4][1]+pts[3][1])/2
	r.strokeLine(sx, sy, ex, ey, width, iconOutline)
}

func hullBodyCenter(pts [][2]float32) (cx, cy, halfW, halfH float32) {
	left, right := pts[4][0], pts[1][0]
	top, bot := pts[4][1], pts[3][1]
	return (left + right) / 2, (top + bot) / 2, (right - left) / 2, (bot - top) / 2
}

func glyphSize(v float32) float32 {
	if v <= 0 {
		return 1
	}
	return v
}

func drawBodyPlus(r *iconRaster, pts [][2]float32, width, size float32) {
	cx, cy, hw, hh := hullBodyCenter(pts)
	s := glyphSize(size)
	r.strokeLine(cx-hw*0.72*s, cy, cx+hw*0.72*s, cy, width, iconOutline)
	r.strokeLine(cx, cy-hh*0.78*s, cx, cy+hh*0.78*s, width, iconOutline)
}

func drawBodyX(r *iconRaster, pts [][2]float32, width, size float32) {
	cx, cy, hw, hh := hullBodyCenter(pts)
	s := glyphSize(size)
	r.strokeLine(cx-hw*0.72*s, cy-hh*0.78*s, cx+hw*0.72*s, cy+hh*0.78*s, width, iconOutline)
	r.strokeLine(cx+hw*0.72*s, cy-hh*0.78*s, cx-hw*0.72*s, cy+hh*0.78*s, width, iconOutline)
}

func drawBodyC(r *iconRaster, pts [][2]float32, width, size float32) {
	cx, cy, hw, hh := hullBodyCenter(pts)
	s := glyphSize(size)
	rad := min32(hw, hh) * 0.62 * s
	r.strokeArc(cx, cy, rad, width, deg(130), deg(410), iconOutline)
}

func deg(v float32) float32 {
	return v * float32(math.Pi) / 180
}

func lerp2(a, b [2]float32, t float32) (float32, float32) {
	return a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t
}
