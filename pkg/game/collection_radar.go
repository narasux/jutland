package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

type abilityDimension struct {
	ID    string
	Label string
	Value func(objUnit.CombatPowerInfo) float64
}

// collectionAbilityDimensions 是图鉴雷达图唯一的维度定义入口。
// 调整顺序、增删维度或替换取值时，雷达布局会自动适配。
var collectionAbilityDimensions = []abilityDimension{
	{ID: "anti_ship", Label: "对舰", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.AntiShip) }},
	{ID: "anti_air", Label: "对空", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.AntiAir) }},
	{ID: "survival", Label: "生存", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.Survival) }},
	{ID: "mobility", Label: "机动", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.Mobility) }},
	{ID: "range", Label: "射程", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.Range) }},
	{ID: "burst", Label: "爆发", Value: func(power objUnit.CombatPowerInfo) float64 { return float64(power.Burst) }},
}

type abilityScales map[string]float64

type radarSubject struct {
	Name    string
	Power   objUnit.CombatPowerInfo
	IsPlane bool
}

type radarHitArea struct {
	Point     image.Point
	LabelRect image.Rectangle
	Dimension abilityDimension
	Subject   radarSubject
}

func calculateAbilityScales(powers []objUnit.CombatPowerInfo) abilityScales {
	scales := abilityScales{}
	for _, dimension := range collectionAbilityDimensions {
		values := make([]float64, 0, len(powers))
		for _, power := range powers {
			values = append(values, max(0, dimension.Value(power)))
		}
		sort.Float64s(values)
		if len(values) == 0 {
			scales[dimension.ID] = 1
			continue
		}
		idx := int(math.Ceil(float64(len(values))*0.95)) - 1
		idx = max(0, min(idx, len(values)-1))
		scales[dimension.ID] = max(1, values[idx])
	}
	return scales
}

func (d *Drawer) drawAbilityRadar(
	screen *ebiten.Image,
	centerX, centerY, radius float64,
	subject radarSubject,
	scales abilityScales,
	screenOffset image.Point,
	labelFontSize float64,
) []radarHitArea {
	count := len(collectionAbilityDimensions)
	if count < 3 || radius <= 0 {
		return nil
	}

	gridColor := color.RGBA{R: 133, G: 119, B: 96, A: 150}
	axisColor := color.RGBA{R: 175, G: 157, B: 126, A: 180}
	fillColor := color.RGBA{R: 75, G: 155, B: 190, A: 90}
	strokeColor := color.RGBA{R: 110, G: 205, B: 225, A: 230}

	pointAt := func(index int, distance float64) (float64, float64) {
		angle := -math.Pi/2 + 2*math.Pi*float64(index)/float64(count)
		return centerX + math.Cos(angle)*distance, centerY + math.Sin(angle)*distance
	}

	for ring := 1; ring <= 4; ring++ {
		path := radarPolygonPath(count, func(index int) (float64, float64) {
			return pointAt(index, radius*float64(ring)/4)
		})
		vector.StrokePath(screen, path, &vector.StrokeOptions{Width: 1}, colorDrawOptions(gridColor))
	}
	for idx := range count {
		x, y := pointAt(idx, radius)
		vector.StrokeLine(screen, float32(centerX), float32(centerY), float32(x), float32(y), 1, axisColor, false)
	}

	valuePath := radarPolygonPath(count, func(index int) (float64, float64) {
		dimension := collectionAbilityDimensions[index]
		value := dimension.Value(subject.Power)
		normalized := min(1, value/max(1, scales[dimension.ID]))
		return pointAt(index, radius*normalized)
	})
	vector.FillPath(screen, valuePath, &vector.FillOptions{}, colorDrawOptions(fillColor))
	vector.StrokePath(screen, valuePath, &vector.StrokeOptions{Width: 2}, colorDrawOptions(strokeColor))

	hits := make([]radarHitArea, 0, count)
	for idx, dimension := range collectionAbilityDimensions {
		labelOffset := 18 * labelFontSize / 16
		x, y := pointAt(idx, radius+labelOffset)
		labelWidth := estimateCollectionTextWidth(dimension.Label, labelFontSize)
		labelX, labelY := x-labelWidth/2, y-labelFontSize/2
		d.drawText(screen, dimension.Label, labelX, labelY, labelFontSize, font.Kai, colorx.White)
		hits = append(hits, radarHitArea{
			Point: image.Pt(int(x)+screenOffset.X, int(y)+screenOffset.Y),
			LabelRect: image.Rect(
				int(labelX)+screenOffset.X-6, int(labelY)+screenOffset.Y-6,
				int(labelX+labelWidth)+screenOffset.X+6,
				int(labelY+labelFontSize*1.25)+screenOffset.Y+6,
			),
			Dimension: dimension,
			Subject:   subject,
		})
	}
	return hits
}

func radarPolygonPath(count int, pointAt func(index int) (float64, float64)) *vector.Path {
	path := &vector.Path{}
	for idx := range count {
		x, y := pointAt(idx)
		if idx == 0 {
			path.MoveTo(float32(x), float32(y))
		} else {
			path.LineTo(float32(x), float32(y))
		}
	}
	path.Close()
	return path
}

func colorDrawOptions(clr color.Color) *vector.DrawPathOptions {
	result := &vector.DrawPathOptions{AntiAlias: true}
	result.ColorScale.ScaleWithColor(clr)
	return result
}

func hoveredRadarArea(areas []radarHitArea) *radarHitArea {
	cursorX, cursorY := ebiten.CursorPosition()
	point := image.Pt(cursorX, cursorY)
	for idx := range areas {
		area := &areas[idx]
		if point.In(area.LabelRect) || distanceSquared(point, area.Point) <= 18*18 {
			return area
		}
	}
	return nil
}

func distanceSquared(left, right image.Point) int {
	dx, dy := left.X-right.X, left.Y-right.Y
	return dx*dx + dy*dy
}

func (d *Drawer) drawRadarTooltip(screen *ebiten.Image, hit *radarHitArea, fontSize float64) {
	if hit == nil {
		return
	}
	lines := radarTooltipLines(hit.Dimension, hit.Subject)
	if len(lines) == 0 {
		return
	}

	scale := fontSize / 17
	width := 240.0 * scale
	for _, line := range lines {
		width = max(width, estimateCollectionTextWidth(line, fontSize)+28*scale)
	}
	lineHeight := fontSize * 1.40
	height := float64(len(lines))*lineHeight + 24*scale
	cursorX, cursorY := ebiten.CursorPosition()
	x, y := float64(cursorX+16), float64(cursorY+18)
	if x+width > float64(screen.Bounds().Dx())-12 {
		x = float64(cursorX) - width - 16
	}
	if y+height > float64(screen.Bounds().Dy())-12 {
		y = float64(screen.Bounds().Dy()) - height - 12
	}
	x, y = max(12, x), max(12, y)

	vector.FillRect(screen, float32(x), float32(y), float32(width), float32(height), color.RGBA{18, 18, 18, 238}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1.5, colorx.Gold, false)
	for idx, line := range lines {
		clr := colorx.White
		if idx == 0 {
			clr = colorx.Gold
		}
		d.drawText(
			screen, line, x+14*scale, y+12*scale+float64(idx)*lineHeight,
			fontSize, font.Kai, clr,
		)
	}
}

func radarTooltipLines(dimension abilityDimension, subject radarSubject) []string {
	power := subject.Power
	value := int(math.Round(dimension.Value(power)))
	lines := []string{fmt.Sprintf("%s · %s", subject.Name, dimension.Label), fmt.Sprintf("能力值：%d", value)}
	var contributions []objUnit.CombatPowerContribution
	switch dimension.ID {
	case "anti_ship":
		lines = append(lines, fmt.Sprintf("有效对舰输出：%.1f /s", power.Details.AntiShipDPS))
		contributions = power.Details.AntiShipContributions
	case "anti_air":
		lines = append(lines, fmt.Sprintf("有效对空输出：%.1f /s", power.Details.AntiAirDPS))
		contributions = power.Details.AntiAirContributions
	case "survival":
		lines = append(lines, fmt.Sprintf("有效耐久：%.0f", power.Details.EffectiveHP))
	case "mobility":
		lines = append(lines, "由速度、转向、加速度综合计算")
	case "range":
		if subject.IsPlane {
			lines = append(lines, fmt.Sprintf("作战航程：%.0f km", power.Details.MaxRange*14.4))
		} else {
			lines = append(lines, fmt.Sprintf("最大打击距离：%.1f 格", power.Details.MaxRange))
		}
	case "burst":
		lines = append(lines, fmt.Sprintf("首轮期望伤害：%.0f", power.Details.BurstDamage))
		contributions = power.Details.BurstContributions
	}
	if len(contributions) > 0 {
		lines = append(lines, "主要贡献：")
		total := 0.0
		for _, contribution := range contributions {
			total += contribution.Value
		}
		for idx, contribution := range contributions {
			if idx >= 3 {
				break
			}
			percent := 0.0
			if total > 0 {
				percent = contribution.Value / total * 100
			}
			lines = append(lines, fmt.Sprintf("  %s  %.0f%%", contribution.Name, percent))
		}
	}
	return lines
}
