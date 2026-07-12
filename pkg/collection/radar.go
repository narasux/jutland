package collection

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

type abilityDimension struct {
	// ID 用作缩放缓存的键，Label 用作图上展示，Value 决定这个维度从战力信息中取哪个字段。
	ID      string
	LabelID i18n.MessageID
	Value   func(objUnit.CombatPowerInfo) float64
}

// collectionAbilityDimensions 是图鉴雷达图唯一的维度定义入口。
// 调整顺序、增删维度或替换取值时，雷达布局会自动适配。
var collectionAbilityDimensions = []abilityDimension{
	{
		ID:      "anti_ship",
		LabelID: i18n.MsgRadarAntiShip,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.AntiShip) },
	},
	{
		ID:      "anti_air",
		LabelID: i18n.MsgRadarAntiAir,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.AntiAir) },
	},
	{
		ID:      "survival",
		LabelID: i18n.MsgRadarSurvival,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.Survival) },
	},
	{
		ID:      "mobility",
		LabelID: i18n.MsgRadarMobility,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.Mobility) },
	},
	{
		ID:      "projection",
		LabelID: i18n.MsgRadarProjection,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.Projection) },
	},
	{
		ID:      "burst",
		LabelID: i18n.MsgRadarBurst,
		Value:   func(power objUnit.CombatPowerInfo) float64 { return float64(power.Burst) },
	},
}

type abilityScales map[string]float64

type radarSubject struct {
	// Subject 代表当前绘制对象，包含展示名称、战力数值和是否为飞机页条目。
	Name    string
	Power   objUnit.CombatPowerInfo
	IsPlane bool
}

type radarHitArea struct {
	// radarHitArea 记录雷达轴标签的屏幕命中区域，用于 hover 时弹出具体数值说明。
	Point     image.Point
	LabelRect image.Rectangle
	Dimension abilityDimension
	Subject   radarSubject
	// Scale 是当前筛选结果在该维度上的 P95 基准，供 tooltip 展示相对位置。
	Scale float64
}

type combatPowerHitArea struct {
	// combatPowerHitArea 仅用于航母综合战力的 hover 拆分提示。
	Rect    image.Rectangle
	Subject radarSubject
}

func hoveredTruncatedTextArea(areas []truncatedTextHitArea) *truncatedTextHitArea {
	return truncatedTextAreaAt(areas, image.Pt(ebiten.CursorPosition()))
}

func truncatedTextAreaAt(areas []truncatedTextHitArea, point image.Point) *truncatedTextHitArea {
	for idx := range areas {
		if point.In(areas[idx].Rect) {
			return &areas[idx]
		}
	}
	return nil
}

// 当 hover 在 ... 之上时，tooltips 展示完整信息
func (d *Drawer) drawTruncatedTextTooltip(
	screen *ebiten.Image, hit *truncatedTextHitArea, fontSize float64,
) {
	if hit == nil || len(hit.Lines) == 0 {
		return
	}
	scale := fontSize / 17
	maxContentWidth := min(520*scale, float64(screen.Bounds().Dx())-52*scale)
	lines := make([]string, 0, len(hit.Lines))
	for _, line := range hit.Lines {
		lines = append(lines, wrapCollectionText(line, maxContentWidth, fontSize)...)
	}
	if len(lines) == 0 {
		return
	}
	contentWidth := 0.0
	for _, line := range lines {
		contentWidth = max(contentWidth, estimateCollectionTextWidth(line, fontSize))
	}
	width := contentWidth + 28*scale
	lineHeight := fontSize * 1.4
	height := float64(len(lines))*lineHeight + 22*scale
	cursorX, cursorY := ebiten.CursorPosition()
	x, y := float64(cursorX+16), float64(cursorY+18)
	if x+width > float64(screen.Bounds().Dx())-12 {
		x = float64(cursorX) - width - 16
	}
	if y+height > float64(screen.Bounds().Dy())-12 {
		y = float64(screen.Bounds().Dy()) - height - 12
	}
	x, y = max(12, x), max(12, y)

	vector.FillRect(
		screen, float32(x), float32(y), float32(width), float32(height),
		color.RGBA{18, 18, 18, 238}, false,
	)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1.5, colorx.Gold, false)
	for idx, line := range lines {
		d.drawText(
			screen, line, x+14*scale, y+11*scale+float64(idx)*lineHeight,
			fontSize, font.LocalizedUI(font.Kai), colorx.White,
		)
	}
}

func calculateAbilityScales(powers []objUnit.CombatPowerInfo) abilityScales {
	// 雷达轴的缩放不是线性取最大值，而是取 95 分位，避免极端值把普通单位压扁。
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
	// 雷达图既负责展示整体能力轮廓，也负责为每个轴生成 hover 命中区。
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
		label := i18n.Text(dimension.LabelID)
		labelFont := font.LocalizedUI(font.Kai)
		labelWidth := layout.CalcTextWidth(label, labelFontSize, labelFont)
		labelX, labelY := x-labelWidth/2, y-labelFontSize/2
		d.drawText(screen, label, labelX, labelY, labelFontSize, labelFont, colorx.White)
		hits = append(hits, radarHitArea{
			Point: image.Pt(int(x)+screenOffset.X, int(y)+screenOffset.Y),
			LabelRect: image.Rect(
				int(labelX)+screenOffset.X-6, int(labelY)+screenOffset.Y-6,
				int(labelX+labelWidth)+screenOffset.X+6,
				int(labelY+labelFontSize*1.25)+screenOffset.Y+6,
			),
			Dimension: dimension,
			Subject:   subject,
			Scale:     max(1, scales[dimension.ID]),
		})
	}
	return hits
}

func radarPolygonPath(count int, pointAt func(index int) (float64, float64)) *vector.Path {
	// 按固定顶点数生成闭合多边形路径，供网格、填充和描边复用。
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
	// 统一给 vector.Path 生成带抗锯齿的颜色绘制参数。
	result := &vector.DrawPathOptions{AntiAlias: true}
	result.ColorScale.ScaleWithColor(clr)
	return result
}

func hoveredRadarArea(areas []radarHitArea) *radarHitArea {
	// hover 判定同时支持标签框和轴点附近点击，降低玩家找命中的成本。
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
	// 这里只比较平方距离，避免每帧做开方。
	dx, dy := left.X-right.X, left.Y-right.Y
	return dx*dx + dy*dy
}

func (d *Drawer) drawRadarTooltip(screen *ebiten.Image, hit *radarHitArea, fontSize float64) {
	// tooltip 不走普通布局系统，因为它需要根据鼠标位置动态避让屏幕边缘。
	if hit == nil {
		return
	}
	lines := radarTooltipLines(hit.Dimension, hit.Subject, hit.Scale)
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

	vector.FillRect(
		screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		color.RGBA{18, 18, 18, 238},
		false,
	)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1.5, colorx.Gold, false)
	for idx, line := range lines {
		clr := colorx.White
		if idx == 0 {
			clr = colorx.Gold
		}
		d.drawText(
			screen, line, x+14*scale, y+12*scale+float64(idx)*lineHeight,
			fontSize, font.LocalizedUI(font.Kai), clr,
		)
	}
}

func radarTooltipLines(dimension abilityDimension, subject radarSubject, scale float64) []string {
	// tooltip 先给出维度总值，再列出这一项的主要来源，便于看出战力从哪里来。
	power := subject.Power
	value := int(math.Round(dimension.Value(power)))
	percent := int(math.Round(min(1, max(0, dimension.Value(power))/max(1, scale)) * 100))
	lines := []string{i18n.Format(i18n.MsgRadarSubjectDimension, map[string]any{
		"Name": subject.Name, "Dimension": i18n.Text(dimension.LabelID),
	})}
	if subject.IsPlane && power.FormationSize > 1 {
		lines = append(lines, i18n.Format(i18n.MsgRadarFormationScope, map[string]any{"Count": power.FormationSize}))
	}
	lines = append(
		lines,
		i18n.Format(i18n.MsgRadarAbilityValue, map[string]any{"Value": value}),
		i18n.Format(i18n.MsgRadarRelativePosition, map[string]any{"Percent": percent}),
	)
	var contributions []objUnit.CombatPowerContribution
	switch dimension.ID {
	case "anti_ship":
		lines = append(
			lines,
			i18n.Format(
				i18n.MsgRadarAntiShipDPS,
				map[string]any{"Value": fmt.Sprintf("%.1f", power.Details.AntiShipDPS)},
			),
		)
		contributions = power.Details.AntiShipContributions
	case "anti_air":
		lines = append(
			lines,
			i18n.Format(
				i18n.MsgRadarAntiAirDPS,
				map[string]any{"Value": fmt.Sprintf("%.1f", power.Details.AntiAirDPS)},
			),
		)
		if subject.IsPlane {
			lines = append(lines, i18n.Text(i18n.MsgRadarTargetingNote))
		}
		contributions = power.Details.AntiAirContributions
	case "survival":
		lines = append(
			lines,
			i18n.Format(
				i18n.MsgRadarEffectiveHP,
				map[string]any{"Value": fmt.Sprintf("%.0f", power.Details.EffectiveHP)},
			),
		)
	case "mobility":
		lines = append(lines, i18n.Text(i18n.MsgRadarMobilityNote))
	case "projection":
		if subject.IsPlane {
			lines = append(
				lines,
				i18n.Format(
					i18n.MsgRadarCombatRadius,
					map[string]any{"Value": fmt.Sprintf("%.0f", power.Details.MaxProjectionDistanceKM)},
				),
			)
		} else {
			lines = append(
				lines,
				i18n.Format(
					i18n.MsgRadarProjectionDistance,
					map[string]any{"Value": fmt.Sprintf("%.1f", power.Details.MaxProjectionDistanceKM)},
				),
			)
		}
	case "burst":
		lines = append(
			lines,
			i18n.Format(
				i18n.MsgRadarBurstDamage,
				map[string]any{"Value": fmt.Sprintf("%.0f", power.Details.BurstDamage)},
			),
		)
		contributions = power.Details.BurstContributions
	}
	if len(contributions) > 0 {
		lines = append(lines, i18n.Text(i18n.MsgRadarMainContributions))
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
			name := contribution.Name
			if ref := objRef.GetReference(name); ref != nil && ref.DisplayName != "" {
				name = ref.DisplayName
			}
			lines = append(lines, i18n.Format(i18n.MsgRadarContribution, map[string]any{
				"Name": name, "Percent": fmt.Sprintf("%.0f", percent),
			}))
		}
	}
	return lines
}

func hoveredCombatPowerArea(areas []combatPowerHitArea) *combatPowerHitArea {
	point := image.Pt(ebiten.CursorPosition())
	for idx := range areas {
		if point.In(areas[idx].Rect) {
			return &areas[idx]
		}
	}
	return nil
}

func (d *Drawer) drawCombatPowerTooltip(
	screen *ebiten.Image, hit *combatPowerHitArea, fontSize float64,
) {
	// 航母的舰体与航空战力只在需要时显示，避免长期占用雷达图下方空间。
	if hit == nil {
		return
	}
	scale := fontSize / 17
	width := 260.0 * scale
	lineHeight := fontSize * 1.45
	height := lineHeight*4 + 22*scale
	cursorX, cursorY := ebiten.CursorPosition()
	x, y := float64(cursorX+16), float64(cursorY+18)
	if x+width > float64(screen.Bounds().Dx())-12 {
		x = float64(cursorX) - width - 16
	}
	if y+height > float64(screen.Bounds().Dy())-12 {
		y = float64(screen.Bounds().Dy()) - height - 12
	}
	x, y = max(12, x), max(12, y)

	vector.FillRect(
		screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		color.RGBA{18, 18, 18, 238},
		false,
	)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1.5, colorx.Gold, false)
	bodyFont := font.LocalizedUI(font.Kai)
	d.drawText(
		screen,
		i18n.Format(i18n.MsgRadarSubjectPower, map[string]any{"Name": hit.Subject.Name}),
		x+14*scale,
		y+11*scale,
		fontSize,
		bodyFont,
		colorx.Gold,
	)
	labels := []string{
		i18n.Text(i18n.MsgRadarOverall),
		i18n.Text(i18n.MsgRadarHull),
		i18n.Text(i18n.MsgRadarAviation),
	}
	values := []int{hit.Subject.Power.Total, hit.Subject.Power.Hull, hit.Subject.Power.Aviation}
	for idx, label := range labels {
		lineY := y + 11*scale + float64(idx+1)*lineHeight
		d.drawText(screen, label, x+14*scale, lineY, fontSize, bodyFont, colorx.White)
		value := fmt.Sprintf("%d", values[idx])
		valueX := x + width - 14*scale - estimateCollectionTextWidth(value, fontSize)
		d.drawText(screen, value, valueX, lineY, fontSize, font.JetbrainsMono, colorx.White)
	}
}
