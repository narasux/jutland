package collection

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/i18n"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	planeImg "github.com/narasux/jutland/pkg/resources/images/plane"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/stretchr/testify/require"
)

// TestLocalizedContributionNameUsesCurrentLanguage 验证战力贡献名称跟随当前界面语言取译名。
// 场景：战力提示里的贡献项只存配置键名，注册中/日译名后切换语言应即时显示对应名称。
// 预期：中文得到「测试飞机 ×12」，日文得到「テスト機 ×12」；失败说明语言切换或数量后缀失效。
// 覆盖：临时注册同一机名的 zh-Hans / ja 参考，构造 Count=12 的贡献项，切语言两次并还原现场。
func TestLocalizedContributionNameUsesCurrentLanguage(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	const planeName = "test-localized-contribution-plane"
	objRef.SetReference(i18n.LanguageZhHans, planeName, &objRef.Reference{DisplayName: "测试飞机"})
	objRef.SetReference(i18n.LanguageJapanese, planeName, &objRef.Reference{DisplayName: "テスト機"})
	contribution := objUnit.CombatPowerContribution{Name: planeName, Count: 12}

	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Equal(t, "测试飞机 ×12", localizedContributionName(contribution))
	i18n.SetLanguage(string(i18n.LanguageJapanese))
	require.Equal(t, "テスト機 ×12", localizedContributionName(contribution))
}

// TestPlaneArmamentItemsUseLocalizedReferenceNames 验证飞机武装按类型合并同类武器并取译名。
// 场景：机炮槽位重复填入同一机炮时，卡片应压缩为「名称 x 数量」，且中英文标签随语言切换。
// 预期：英文为 Gun/Test Cannon x 2 与 Bomb/Test Bomb x 1；中文为 机炮/测试机炮 x 2、炸弹/测试炸弹 x 1。
// 覆盖：构造 2 门同名机炮和 1 枚炸弹，注册中英 reference，分别断言完整 InfoItem 切片。
func TestPlaneArmamentItemsUseLocalizedReferenceNames(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	const gunName = "test-localized-plane-gun"
	const bombName = "test-localized-plane-bomb"
	objRef.SetReference(i18n.LanguageZhHans, gunName, &objRef.Reference{DisplayName: "测试机炮"})
	objRef.SetReference(i18n.LanguageEnglish, gunName, &objRef.Reference{DisplayName: "Test Cannon"})
	objRef.SetReference(i18n.LanguageZhHans, bombName, &objRef.Reference{DisplayName: "测试炸弹"})
	objRef.SetReference(i18n.LanguageEnglish, bombName, &objRef.Reference{DisplayName: "Test Bomb"})
	plane := &objUnit.Plane{
		Weapon: objUnit.PlaneWeapon{
			Guns:  []*objUnit.Gun{{Name: gunName}, {Name: gunName}},
			Bombs: []*objUnit.Releaser{{Name: bombName}},
		},
	}

	i18n.SetLanguage(string(i18n.LanguageEnglish))
	require.Equal(t, []objRef.InfoItem{
		{Label: "Gun", Value: "Test Cannon x 2"},
		{Label: "Bomb", Value: "Test Bomb x 1"},
	}, planeArmamentItems(plane))

	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Equal(t, []objRef.InfoItem{
		{Label: "机炮", Value: "测试机炮 x 2"},
		{Label: "炸弹", Value: "测试炸弹 x 1"},
	}, planeArmamentItems(plane))
}

// TestTruncateCollectionTextKeepsEllipsisWithinWidth 验证超宽文本截断后仍带省略号且不越界。
// 场景：18px 字号下武装值远宽于卡片列宽，需要截断；短词 Torpedoes 则必须原样保留。
// 预期：长文本 truncated=true、以「…」结尾且估算宽度 ≤ maxWidth；Torpedoes 不被改动。
// 覆盖：maxWidth 取 "Main Battery 3x3" 的宽度，分别截断 406mm/50 Mk.7 长串与短词。
func TestTruncateCollectionTextKeepsEllipsisWithinWidth(t *testing.T) {
	fontSize := 18.0
	maxWidth := estimateCollectionTextWidth("Main Battery 3x3", fontSize)
	textFont := font.LocalizedUI(font.Kai)
	display, truncated := truncateCollectionText(
		"Main Battery 3x3 406mm/50 Mk.7", maxWidth, fontSize, textFont,
	)

	require.True(t, truncated)
	require.True(t, strings.HasSuffix(display, "…"))
	require.LessOrEqual(t, estimateCollectionTextWidth(display, fontSize), maxWidth)

	display, truncated = truncateCollectionText("Torpedoes", maxWidth, fontSize, textFont)
	require.False(t, truncated)
	require.Equal(t, "Torpedoes", display)
}

// TestTruncateRussianShipNameKeepsEllipsisWithinArchiveCard 验证俄文长舰名在档案卡宽度内截断。
// 场景：30px Hang 字体下「Лонг-Бич (вариант D)」超宽，卡片只放得下到「(вариант」的宽度。
// 预期：返回 truncated=true、以「…」结尾且估算宽度 ≤ maxWidth；失败意味着俄文标题溢出卡片。
// 覆盖：切到 ru 语言并注册清理，maxWidth 精确取半截名称宽度，再断言三点。
func TestTruncateRussianShipNameKeepsEllipsisWithinArchiveCard(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageRussian))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	fontSize := 30.0
	textFont := font.LocalizedUI(font.Hang)
	maxWidth := estimateCollectionTextWidth("Лонг-Бич (вариант", fontSize)
	display, truncated := truncateCollectionText(
		"Лонг-Бич (вариант D)", maxWidth, fontSize, textFont,
	)

	require.True(t, truncated)
	require.True(t, strings.HasSuffix(display, "…"))
	require.LessOrEqual(t, estimateCollectionTextWidth(display, fontSize), maxWidth)
}

// TestDrawFittedRussianShipNameRegistersFullNameTooltip 验证被截断的舰名会登记完整名称悬浮提示。
// 场景：俄文舰名在 800×120 画布上以 30px 绘制仍超宽，玩家需要 hover 才能看到全名。
// 预期：textHits 恰好 1 条，Lines 保存完整原名，命中矩形宽度为正；失败意味着提示丢失或错位。
// 覆盖：CollectionUI 配 NewDrawer，maxWidth 取半截名称宽度，调用 drawFittedText 后检查 textHits。
func TestDrawFittedRussianShipNameRegistersFullNameTooltip(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageRussian))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	const fullName = "Лонг-Бич (вариант D)"
	ui := &CollectionUI{drawer: NewDrawer()}
	fontSize := 30.0
	textFont := font.LocalizedUI(font.Hang)
	maxWidth := estimateCollectionTextWidth("Лонг-Бич (вариант", fontSize)
	screen := ebiten.NewImage(800, 120)

	ui.drawFittedText(
		screen, fullName, 20, 20, maxWidth, fontSize, textFont, color.White,
		image.Point{}, []string{fullName},
	)

	require.Len(t, ui.textHits, 1)
	require.Equal(t, []string{fullName}, ui.textHits[0].Lines)
	require.Positive(t, ui.textHits[0].Rect.Dx())
}

// TestArmamentItemsUseTwoLineLayoutForWordWrappingLanguages 验证分词语言用两行键值排版。
// 场景：卡片宽度只够两行排版容纳完整数值；英文/俄文按分词换行，日文仍走单行键值。
// 预期：英/俄无截断提示；日文产生 1 条提示，内容为「両用砲  6x2 127mm/38 Mk12」。
// 覆盖：表驱动 3 种语言，card.W = 数值宽 + 标签宽/2 + 54，正文 18px，检查 textHits 数量与行内容。
func TestArmamentItemsUseTwoLineLayoutForWordWrappingLanguages(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	fontSize := 18.0
	cases := []struct {
		lang        i18n.Language
		label       string
		value       string
		wantStacked bool
	}{
		{i18n.LanguageEnglish, "Dual-purpose Guns", "6x2 127mm/38 Mk12", true},
		{i18n.LanguageRussian, "Универсальные орудия", "6x2 127mm/38 Mk12", true},
		{i18n.LanguageJapanese, "両用砲", "6x2 127mm/38 Mk12", false},
	}
	for _, tc := range cases {
		t.Run(string(tc.lang), func(t *testing.T) {
			i18n.SetLanguage(string(tc.lang))
			labelWidth := estimateCollectionTextWidth(tc.label, fontSize)
			valueWidth := estimateCollectionTextWidth(tc.value, fontSize)
			// 卡片宽度让数值在单行键值里放不下，但只有两行排版能完整容纳它。
			card := collectionCard{W: valueWidth + labelWidth/2 + 54, H: 300}
			ui := &CollectionUI{drawer: NewDrawer(), metrics: collectionMetrics{Scale: 1, Body: fontSize}}
			screen := ebiten.NewImage(400, 320)

			ui.drawArmamentItems(
				screen, []objRef.InfoItem{{Label: tc.label, Value: tc.value}}, card, 58,
			)

			if tc.wantStacked {
				require.Empty(t, ui.textHits)
			} else {
				require.Len(t, ui.textHits, 1)
				require.Equal(t, []string{tc.label + "  " + tc.value}, ui.textHits[0].Lines)
			}
		})
	}
}

// TestArmamentItemsPlaceValueOnSecondLine 验证两行排版把数值放到第二行，且两行不重叠、不越卡片。
// 场景：英文两项武装在 120px 宽卡片里放不下单行键值，drawArmamentItems 改用堆叠布局。
// 预期：两条 textHit 的 Lines 分别为标签/数值；左缘对齐；第一行整体在第二行之上；最底部不越卡片高度。
// 覆盖：startY=58、正文 18px、卡片 120×300，调用 drawArmamentItems 后断言 Lines/左缘/上下边界。
func TestArmamentItemsPlaceValueOnSecondLine(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageEnglish))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	const (
		fontSize = 18.0
		startY   = 58.0
	)
	ui := &CollectionUI{drawer: NewDrawer(), metrics: collectionMetrics{Scale: 1, Body: fontSize}}
	screen := ebiten.NewImage(200, 320)

	ui.drawArmamentItems(
		screen,
		[]objRef.InfoItem{
			{Label: "Dual-purpose Guns", Value: "6x2 127mm/38 Mk12"},
			{Label: "Anti-aircraft Guns", Value: "12x4 40mm/60 Bofors"},
		},
		collectionCard{W: 120, H: 300},
		startY,
	)

	require.Len(t, ui.textHits, 2)
	first, second := ui.textHits[0], ui.textHits[1]
	require.Equal(t, []string{"Dual-purpose Guns", "6x2 127mm/38 Mk12"}, first.Lines)
	require.Equal(t, []string{"Anti-aircraft Guns", "12x4 40mm/60 Bofors"}, second.Lines)
	require.Equal(t, first.Rect.Min.X, second.Rect.Min.X, "stacked rows share the same left edge")
	require.GreaterOrEqual(t, first.Rect.Min.Y, int(math.Floor(startY)))
	require.Less(t, first.Rect.Max.Y, second.Rect.Min.Y, "stacked rows must not overlap")
	require.LessOrEqual(t, second.Rect.Max.Y, 300, "stacked rows must stay inside the card")
}

// TestFitCollectionLinesAddsEllipsisWhenRowsOverflow 验证行数超限时最后一行补省略号。
// 场景：三行文本只允许显示两行（maxLines=2），第三行必须折叠且不能静默丢失信息。
// 预期：truncated=true、返回 2 行、末行以「…」结尾且宽度不超过 maxWidth。
// 覆盖：三行短句、18px Kai 字体，maxWidth 取 "Second line" 宽度，断言四项。
func TestFitCollectionLinesAddsEllipsisWhenRowsOverflow(t *testing.T) {
	fontSize := 18.0
	textFont := font.LocalizedUI(font.Kai)
	maxWidth := estimateCollectionTextWidth("Second line", fontSize)

	lines, truncated := fitCollectionLines(
		[]string{"First line", "Second line", "Third line"},
		maxWidth,
		fontSize,
		textFont,
		2,
	)

	require.True(t, truncated)
	require.Len(t, lines, 2)
	require.True(t, strings.HasSuffix(lines[1], "…"))
	require.LessOrEqual(t, estimateCollectionTextWidth(lines[1], fontSize), maxWidth)
}

// TestFitCollectionLinesKeepsCompleteRowsUnchanged 验证未超限时不截断、不改写原文。
// 场景：两行文本恰好等于 maxLines=2，且每行都在 maxWidth 内，属于正常展示路径。
// 预期：truncated=false，返回切片与输入完全一致；失败意味着正常文本被误加省略号。
// 覆盖：18px Kai 字体，maxWidth 取 "Second line" 宽度，比对 input 与 lines。
func TestFitCollectionLinesKeepsCompleteRowsUnchanged(t *testing.T) {
	fontSize := 18.0
	textFont := font.LocalizedUI(font.Kai)
	input := []string{"First line", "Second line"}

	lines, truncated := fitCollectionLines(
		input,
		estimateCollectionTextWidth("Second line", fontSize),
		fontSize,
		textFont,
		2,
	)

	require.False(t, truncated)
	require.Equal(t, input, lines)
}

// TestTruncatedTextAreaAt 验证截断文本的命中区按矩形半开区间判定。
// 场景：鼠标在提示区内部 (50,30) 应显示 tooltip，在右边界 (110,30) 已不属于该矩形。
// 预期：内部点返回同一 *truncatedTextHitArea，右边界点返回 nil；失败意味着提示越界触发。
// 覆盖：单个矩形 (10,20)-(110,45)，两次调用并用 require.Same / Nil 断言。
func TestTruncatedTextAreaAt(t *testing.T) {
	areas := []truncatedTextHitArea{
		{Rect: image.Rect(10, 20, 110, 45), Lines: []string{"full armament"}},
	}

	require.Same(t, &areas[0], truncatedTextAreaAt(areas, image.Pt(50, 30)))
	require.Nil(t, truncatedTextAreaAt(areas, image.Pt(110, 30)))
}

// TestDropdownListWidthFitsLongestLabelAndScreen 验证下拉宽度贴合最长标签并被屏幕宽度夹住。
// 场景：标签 All / Aircraft Carrier 在 20px、scale=1 下，长标签需要额外 36px 内边距。
// 预期：可用宽度 500 时等于 ceil(最长标签宽+36)；可用宽度 120 时收窄到 120。
// 覆盖：由 estimateCollectionTextWidth 独立算出 contentWidth，再断言两次 dropdownListWidth。
func TestDropdownListWidthFitsLongestLabelAndScreen(t *testing.T) {
	fontSize := 20.0
	scale := 1.0
	labels := []string{"All", "Aircraft Carrier"}
	contentWidth := int(math.Ceil(estimateCollectionTextWidth(labels[1], fontSize) + 36*scale))

	require.Equal(t, contentWidth, dropdownListWidth(100, 500, scale, fontSize, labels))
	require.Equal(t, 120, dropdownListWidth(100, 120, scale, fontSize, labels))
}

// TestDropdownChevronGeometryStaysSymmetricAndInsideRect 验证下拉箭头左右对称、开口向下且不越出文本框。
// 场景：下拉框在 100..260 的矩形内绘制，箭头需避开右侧内边距；语言/缩放变化只改变尺寸，不应破坏对称性。
// 预期：左右端点到中心 X 距离相等、两端同高且低于中心点；三个关键点都落在 rect 内。
// 覆盖：rect=(100,20)-(260,60)、scale=1 调用 dropdownChevronGeometry，断言对称/方向/边界。
func TestDropdownChevronGeometryStaysSymmetricAndInsideRect(t *testing.T) {
	rect := image.Rect(100, 20, 260, 60)
	chevron := dropdownChevronGeometry(rect, 1)

	require.InDelta(t, float64(chevron.CenterX-chevron.LeftX), float64(chevron.RightX-chevron.CenterX), 1e-6)
	require.Equal(t, chevron.LeftY, chevron.RightY)
	require.Less(t, chevron.LeftY, chevron.CenterY, "chevron must open downward")

	for _, x := range []float32{chevron.LeftX, chevron.CenterX, chevron.RightX} {
		require.GreaterOrEqual(t, x, float32(rect.Min.X))
		require.LessOrEqual(t, x, float32(rect.Max.X))
	}
	for _, y := range []float32{chevron.LeftY, chevron.CenterY, chevron.RightY} {
		require.GreaterOrEqual(t, y, float32(rect.Min.Y))
		require.LessOrEqual(t, y, float32(rect.Max.Y))
	}
}

// TestCombatPowerHeaderPositionsKeepLabelAndValueSeparated 验证总战力标题与右侧数值不重叠。
// 场景：1.3 倍缩放下卡片 x=440、宽 960，英文标签 Total Power 与数值 278 同排右对齐。
// 预期：labelX+labelWidth ≤ valueX，且 valueX+valueWidth ≤ 卡片右缘-20×scale。
// 覆盖：分别用 18×scale 与 24×scale 估算宽度，再断言 combatPowerHeaderPositions 两个边界。
func TestCombatPowerHeaderPositionsKeepLabelAndValueSeparated(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageEnglish))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	card := collectionCard{X: 440, W: 960}
	scale := 1.3
	label := "Total Power"
	value := "278"
	labelX, valueX := combatPowerHeaderPositions(card, scale, label, value)

	labelWidth := estimateCollectionTextWidth(label, 18*scale)
	valueWidth := layout.CalcTextWidth(value, 24*scale, font.JetbrainsMono)
	require.LessOrEqual(t, labelX+labelWidth, valueX)
	require.LessOrEqual(t, valueX+valueWidth, card.X+card.W-20*scale)
}

// TestShipBlueprintUsesSharedScaleAndKeepsCarrierLengthOrder 验证统一蓝图倍率保留航母相对舰长。
// 场景：临时放入 saratoga/yorktown/essex 三艘美航母，1100×360 蓝图区共用同一像素比例。
// 预期：yorktown 短于 essex 且比值≈247/266，saratoga/essex≈275/266（容差 0.02）。
// 覆盖：替换 ShipMap / AllShipNames 并在 Cleanup 还原，按 GetTop 图像高度乘共享倍率断言。
func TestShipBlueprintUsesSharedScaleAndKeepsCarrierLengthOrder(t *testing.T) {
	previousShipMap := objUnit.ShipMap
	previousShipNames := objUnit.AllShipNames
	t.Cleanup(func() {
		objUnit.ShipMap = previousShipMap
		objUnit.AllShipNames = previousShipNames
	})

	objUnit.ShipMap = map[string]*objUnit.BattleShip{
		"saratoga": {Name: "saratoga", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
		"yorktown": {Name: "yorktown", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
		"essex":    {Name: "essex", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
	}
	objUnit.AllShipNames = []string{"saratoga", "yorktown", "essex"}

	scale := collectionShipBlueprintScale(image.Rect(0, 0, 1100, 360))
	require.Positive(t, scale)

	saratogaLength := float64(shipImg.GetTop("saratoga", 4).Bounds().Dy()) * scale
	yorktownLength := float64(shipImg.GetTop("yorktown", 4).Bounds().Dy()) * scale
	essexLength := float64(shipImg.GetTop("essex", 4).Bounds().Dy()) * scale

	require.Less(t, yorktownLength, essexLength)
	require.InDelta(t, 247.0/266.0, yorktownLength/essexLength, 0.02)
	require.InDelta(t, 275.0/266.0, saratogaLength/essexLength, 0.02)
}

// TestShipBlueprintWideTopViewDoesNotShrinkSharedScale 验证超宽俯视图只缩小自身、不拖低全局倍率。
// 场景：America 俯视图比半高更宽；旧算法按半高适配会得到 wideFit，新算法按整块 innerH 与 gap 打包。
// 预期：packedFit > wideFit、sharedScale > wideFit×1.5；midway 用 sharedScale，America 用 packedFit 且更小。
// 覆盖：1100×360 蓝图算 innerW/innerH/gap 与 halfHeight，比较四种倍率及 midway 新长度 > 旧长度×1.5。
func TestShipBlueprintWideTopViewDoesNotShrinkSharedScale(t *testing.T) {
	previousShipMap := objUnit.ShipMap
	previousShipNames := objUnit.AllShipNames
	t.Cleanup(func() {
		objUnit.ShipMap = previousShipMap
		objUnit.AllShipNames = previousShipNames
	})

	objUnit.ShipMap = map[string]*objUnit.BattleShip{
		"midway":  {Name: "midway", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
		"essex":   {Name: "essex", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
		"america": {Name: "america", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
	}
	objUnit.AllShipNames = []string{"midway", "essex", "america"}

	rect := image.Rect(0, 0, 1100, 360)
	innerW, innerH, gap := collectionBlueprintInnerSize(rect)
	halfHeight := collectionBlueprintHalfHeight(innerH, gap)
	sharedScale := collectionShipBlueprintScale(rect)
	require.Positive(t, sharedScale)

	cvxTop := shipImg.GetTop("america", 4)
	cvxSide := shipImg.GetSide("america", 4)
	midwayTop := shipImg.GetTop("midway", 4)
	midwaySide := shipImg.GetSide("midway", 4)
	require.NotNil(t, cvxTop)
	require.NotNil(t, midwayTop)

	wideFit := collectionImageFitScale(cvxTop, innerW, halfHeight, 90, false)
	packedFit := collectionShipBlueprintFitScale(cvxSide, cvxTop, innerW, innerH, gap)
	require.Positive(t, wideFit)
	require.Positive(t, packedFit)
	require.Greater(t, packedFit, wideFit)
	require.Greater(t, sharedScale, wideFit*1.5)

	midwayShip := objUnit.ShipMap["midway"]
	cvxShip := objUnit.ShipMap["america"]
	midwayScale := collectionShipBlueprintDrawScale(
		sharedScale, midwayShip, midwaySide, midwayTop, innerW, innerH, gap,
	)
	cvxScale := collectionShipBlueprintDrawScale(
		sharedScale, cvxShip, cvxSide, cvxTop, innerW, innerH, gap,
	)

	require.InDelta(t, sharedScale, midwayScale, 1e-9)
	require.InDelta(t, packedFit, cvxScale, 1e-9)
	require.Less(t, cvxScale, sharedScale)

	midwayLength := float64(midwayTop.Bounds().Dy()) * midwayScale
	legacyMidwayLength := float64(midwayTop.Bounds().Dy()) * wideFit
	require.Greater(t, midwayLength, legacyMidwayLength*1.5)
}

// TestShipArchiveInfoItemsIncludeLengthAndWidth 验证档案卡包含按配置长度/宽度格式化的尺寸行。
// 场景：构造 2030 年、长 493、宽 54 的航母且无 reference，尺寸仍应只来自 BattleShip 字段。
// 预期：MsgCollectionDimensions 行的值等于 MsgValueLengthWidth 以 Length=493、Width=54 格式化的结果。
// 覆盖：把 InfoItem 转成 label→value map，按当前语言的维度标签取值并精确比较。
func TestShipArchiveInfoItemsIncludeLengthAndWidth(t *testing.T) {
	items := shipArchiveInfoItems(&objUnit.BattleShip{
		Nation: objUnit.NationUS,
		Type:   objUnit.ShipTypeAircraftCarrier,
		Year:   2030,
		Length: 493,
		Width:  54,
	}, nil)
	got := map[string]string{}
	for _, item := range items {
		got[item.Label] = item.Value
	}
	require.Equal(
		t,
		i18n.Format(i18n.MsgValueLengthWidth, map[string]any{"Length": "493", "Width": "54"}),
		got[i18n.Text(i18n.MsgCollectionDimensions)],
	)
}

// TestShipArchiveSpeedShowsConfiguredKnots 验证档案航速显示配置节数而非内部折算值。
// 场景：lowa 在 ships.json5 配置 33 节，初始化时按 ShipSpeedScale 折算存储，档案卡必须还原。
// 预期：MsgCollectionSpeed 行为「33 节」；失败意味着直接展示折算后的一半数值。
// 覆盖：切到 zh-Hans，从 ShipMap 取 lowa，遍历 shipArchiveInfoItems 建 map 后断言该行。
func TestShipArchiveSpeedShowsConfiguredKnots(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	// 衣阿华在 ships.json5 中配置的最高航速为 33 节，档案卡必须显示 33，
	// 而不是初始化折算（除以 ShipSpeedScale）后未还原的一半数值。
	ship := objUnit.ShipMap["lowa"]
	require.NotNil(t, ship)
	got := map[string]string{}
	for _, item := range shipArchiveInfoItems(ship, nil) {
		got[item.Label] = item.Value
	}
	require.Equal(t, "33 节", got[i18n.Text(i18n.MsgCollectionSpeed)])
}

// TestFormatCollectionYearAnnotatesOnlyNonReal 验证年份文本只对图纸与架空追加括号备注。
// 场景：史实舰不标注，未建成方案标“图纸”，架空单位标“架空”，缺年份则用 -- 占位。
// 预期：1945/real → “1945”；0/real → “--”；1939/incomplete → “1939（未完工）”；1945/prototype → “1945（仅试验机）”；1943/limited → “1943（少量生产）”；1939/paper → “1939（图纸）”；2030/fictional → “2030（架空）”。
// 覆盖：表驱动 real、incomplete、prototype、limited、paper、fictional 与 0 年七组输入，切 zh-Hans 后比较 formatCollectionYear 返回值。
func TestFormatCollectionYearAnnotatesOnlyNonReal(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	cases := []struct {
		year   int
		status objUnit.HistoricalStatus
		want   string
	}{
		{1945, objUnit.HistoricalStatusReal, "1945"},
		{0, objUnit.HistoricalStatusReal, "--"},
		{1939, objUnit.HistoricalStatusIncomplete, "1939（未完工）"},
		{1945, objUnit.HistoricalStatusPrototype, "1945（仅试验机）"},
		{1943, objUnit.HistoricalStatusLimited, "1943（少量生产）"},
		{1939, objUnit.HistoricalStatusPaper, "1939（图纸）"},
		{2030, objUnit.HistoricalStatusFictional, "2030（架空）"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, formatCollectionYear(tc.year, tc.status))
	}
}

// TestShipArchiveYearAnnotatesIncompletePaperAndFictional 验证舰船档案的年份行使用括号备注。
// 场景：格拉夫·齐柏林已下水但未完工，斯大林格勒 X 是图纸方案，江户是架空战列舰，衣阿华是史实战列舰。
// 预期：年份行分别为 1943（未完工）、1948（图纸）、1951（架空）、1945；失败意味着状态又变成独立徽标或漏标。
// 覆盖：切 zh-Hans，遍历四艘真实模板的 shipArchiveInfoItems，按年份标签取 Value 后断言。
func TestShipArchiveYearAnnotatesIncompletePaperAndFictional(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	yearValue := func(name string) string {
		ship := objUnit.ShipMap[name]
		require.NotNil(t, ship)
		for _, item := range shipArchiveInfoItems(ship, nil) {
			if item.Label == i18n.Text(i18n.MsgCollectionYear) {
				return item.Value
			}
		}
		t.Fatalf("%s has no year item", name)
		return ""
	}
	require.Equal(t, "1943（未完工）", yearValue("graf_zeppelin"))
	require.Equal(t, "1948（图纸）", yearValue("stalingrad_x"))
	require.Equal(t, "1951（架空）", yearValue("edo"))
	require.Equal(t, "1945", yearValue("lowa"))
}

// TestSubmarineArchiveShowsSurfaceAndSubmergedSpeed 验证潜艇档案分列水面与水下航速。
// 场景：XXI 型 u_2540 水下 17.16 节快于水面 15.6 节，只给一档「速度」会误导玩家。
// 预期：水面行「15.6 节」、水下行「17.2 节」，且不再出现通用 MsgCollectionSpeed 行。
// 覆盖：切到 zh-Hans，取真实 u_2540，遍历档案项建 map，断言两档数值并排除通用速度键。
func TestSubmarineArchiveShowsSurfaceAndSubmergedSpeed(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	// XXI 型（U-2540）水下 17.16 节快于水面 15.6 节，图鉴必须分别给出两档航速，
	// 不能只显示一档“速度”。
	ship := objUnit.ShipMap["u_2540"]
	require.NotNil(t, ship)
	got := map[string]string{}
	for _, item := range shipArchiveInfoItems(ship, nil) {
		got[item.Label] = item.Value
	}
	require.Equal(t, "15.6 节", got[i18n.Text(i18n.MsgCollectionSpeedSurface)])
	require.Equal(t, "17.2 节", got[i18n.Text(i18n.MsgCollectionSpeedSubmerged)])
	require.NotContains(t, got, i18n.Text(i18n.MsgCollectionSpeed))
}

// TestCollectionShipLengthPixelsIgnoresPlaceholderSide 验证 1×1 占位侧视图不参与舰长计算。
// 场景：彩蛋 waterdrop 只有有效俯视图，侧视图是 1×1 占位图，必须被判为不可用而非舰长来源。
// 预期：可用性检查中 top 非 nil、side 为 nil；舰长等于俯视旋转 90° 后的宽度且 >8px。
// 覆盖：取 waterdrop 侧/俯图，调用 collectionUsableBlueprintImage 与 collectionShipLengthPixels 比较。
func TestCollectionShipLengthPixelsIgnoresPlaceholderSide(t *testing.T) {
	side := shipImg.GetSide("waterdrop", 4)
	top := shipImg.GetTop("waterdrop", 4)
	require.NotNil(t, top)
	require.NotNil(t, collectionUsableBlueprintImage(top))
	require.Nil(t, collectionUsableBlueprintImage(side))

	topW, _ := collectionImageRotatedSize(top, 90)
	require.InDelta(t, topW, collectionShipLengthPixels(side, top, 1), 1e-9)
	require.Greater(t, topW, 8.0)
}

// TestShipBlueprintViewGapGrowsWhenSpaceAllows 验证侧视/俯视间距取最小值并随留白增长但有上限。
// 场景：1100×360 蓝图 innerH=324、minGap=max(40,38.88)=40；留白多时 innerH=400、两图高 40/36。
// 预期：minViewGap=minGap；紧凑时间隙=40；宽松时落在 (40, 324) 内且不超过 innerH×0.24=96。
// 覆盖：先断言 minGap≥40，再用紧/松两组参数调用 viewGap，比较上下界。
func TestShipBlueprintViewGapGrowsWhenSpaceAllows(t *testing.T) {
	_, innerH, minGap := collectionBlueprintInnerSize(image.Rect(0, 0, 1100, 360))
	require.GreaterOrEqual(t, minGap, collectionBlueprintMinGap)
	require.InDelta(t, minGap, collectionBlueprintMinViewGap(innerH), 1e-9)

	require.InDelta(t, minGap, collectionBlueprintViewGap(innerH, 100, innerH-100-minGap, minGap), 1e-9)

	roomy := collectionBlueprintViewGap(400, 40, 36, minGap)
	require.Greater(t, roomy, minGap)
	require.Less(t, roomy, 400.0-40.0-36.0)
	require.LessOrEqual(t, roomy, 400.0*0.24)
}

// TestMovePlaneTypeSkipsEmptyTypesAndResetsPosition 验证切机种跳过当前国籍无飞机的类型并复位滚动。
// 场景：仅 US-Fighter、US-Torpedo、JP-Dive 时，从 fighter 前进应跳过 JP 的 dive 和空的 level。
// 预期：落到 torpedo，planeFirstIndex 归零，pendingRebuild=true；失败意味着出现空列表或旧滚动位置。
// 覆盖：替换 PlaneMap/AllPlaneNames 并 Cleanup 还原，初始 index=3，调用 movePlaneType(1) 后断言。
func TestMovePlaneTypeSkipsEmptyTypesAndResetsPosition(t *testing.T) {
	previousPlaneMap := objUnit.PlaneMap
	previousPlaneNames := objUnit.AllPlaneNames
	t.Cleanup(func() {
		objUnit.PlaneMap = previousPlaneMap
		objUnit.AllPlaneNames = previousPlaneNames
	})

	objUnit.PlaneMap = map[string]*objUnit.Plane{
		"US-Fighter": {
			Name:   "US-Fighter",
			Nation: objUnit.NationUS,
			Type:   objUnit.PlaneTypeFighter,
		},
		"US-Torpedo": {
			Name:   "US-Torpedo",
			Nation: objUnit.NationUS,
			Type:   objUnit.PlaneTypeTorpedoBomber,
		},
		"JP-Dive": {
			Name:   "JP-Dive",
			Nation: objUnit.NationJP,
			Type:   objUnit.PlaneTypeDiveBomber,
		},
	}
	objUnit.AllPlaneNames = []string{"US-Fighter", "US-Torpedo", "JP-Dive"}

	ui := &CollectionUI{
		planeNation:     objUnit.NationUS,
		planeType:       planeTypeFighter,
		planeFirstIndex: 3,
	}

	ui.movePlaneType(1)

	require.Equal(t, planeTypeTorpedo, ui.planeType)
	require.Zero(t, ui.planeFirstIndex)
	require.True(t, ui.pendingRebuild)
}

// TestCollectionPlaneCardScaleUnified 验证飞机卡片共用真实比例尺且最大机型刚好填满图片框。
// 场景：B-17G 长 22.5m、海獭式 12.12m 用真实素材；cardScale=0.8 时图片框 233.6×92.8。
// 预期：所有机型绘制尺寸不超框，最大者贴住一边；B-17G 高度=boxH，海獭/B-17G≈12.12/22.5（±0.15）。
// 覆盖：临时限定两机型，算 ratio 后遍历原图与 GetDisplayScale 求最大显示尺寸，再断言比例。
func TestCollectionPlaneCardScaleUnified(t *testing.T) {
	// 统一比例尺必须让所有飞机按同一像素/米比例展示，且最大机型恰好占满卡片图片框。
	// B-17G 长 22.5m、海獭式长 12.12m，若都各自填满固定框，二者会显示得一样大。
	previousPlaneMap := objUnit.PlaneMap
	previousPlaneNames := objUnit.AllPlaneNames
	t.Cleanup(func() {
		objUnit.PlaneMap = previousPlaneMap
		objUnit.AllPlaneNames = previousPlaneNames
	})

	// 用真实素材名称构造最小集合，验证统一比例尺的等比规则（正式图鉴由初始化填充更多机型）。
	objUnit.AllPlaneNames = []string{"B-17G", "SeaOtter"}
	objUnit.PlaneMap = map[string]*objUnit.Plane{
		"B-17G":    {Name: "B-17G", Nation: objUnit.NationUS},
		"SeaOtter": {Name: "SeaOtter", Nation: objUnit.NationUS},
	}

	cardScale := 0.8
	ratio := collectionPlaneCardScale(cardScale)
	require.Greater(t, ratio, 0.0)

	boxW := (340 - 48) * cardScale
	boxH := 116 * cardScale
	var maxDisplayW, maxDisplayH float64
	var planes int
	for _, name := range objUnit.AllPlaneNames {
		plane := objUnit.PlaneMap[name]
		if plane == nil || plane.Nation == objUnit.NationSpecial {
			continue
		}
		img := planeImg.GetOriginal(name)
		scale := planeImg.GetDisplayScale(name)
		if img == nil || scale <= 0 {
			continue
		}
		planes++
		base := scale * ratio
		h := float64(img.Bounds().Dy()) * base
		w := float64(img.Bounds().Dx()) * base
		require.LessOrEqual(t, h, boxH, "%s drawn height %.2f exceeds box %.2f", name, h, boxH)
		require.LessOrEqual(t, w, boxW, "%s drawn width %.2f exceeds box %.2f", name, w, boxW)
		maxDisplayH = max(maxDisplayH, h)
		maxDisplayW = max(maxDisplayW, w)
	}
	require.Greater(t, planes, 0)

	// 倍率应取到约束边界：最大机型要么顶到框高、要么顶到框宽，否则说明缩放没等比贴合。
	require.True(
		t,
		math.Abs(maxDisplayH-boxH) < 1e-6 || math.Abs(maxDisplayW-boxW) < 1e-6,
		"expected largest plane to touch a box edge: maxH=%.3f (box %.3f), maxW=%.3f (box %.3f)",
		maxDisplayH, boxH, maxDisplayW, boxW,
	)

	// 机体比例：B-17G（最长机型）应占满框高，海獭式按真实长度缩小（长度比 12.12/22.5≈0.54）。
	b17 := planeImg.GetOriginal("B-17G")
	seaOtter := planeImg.GetOriginal("SeaOtter")
	if b17 != nil && seaOtter != nil {
		b17H := float64(b17.Bounds().Dy()) * planeImg.GetDisplayScale("B-17G") * ratio
		seaOtterH := float64(seaOtter.Bounds().Dy()) * planeImg.GetDisplayScale("SeaOtter") * ratio
		require.InDelta(t, b17H, boxH, 1e-6, "B-17G (longest plane) should fill the card image box height")
		require.InDelta(t, seaOtterH/b17H, 12.12/22.5, 0.15,
			"Sea Otter should look proportionally smaller than B-17G")
	}
}

// TestRadarScaleNoteBreaksFormationNoteIntoTwoCleanLines 验证编队说明按文案换行符断成两行。
// 场景：10 架编队的雷达说明若按字宽自动换行，会把 P95 短词拆到两行，需按内嵌换行符排版。
// 预期：radarScaleNoteLines 返回 2 行，第二行包含 P95，第一行不含；失败意味着短词仍被拆开。
// 覆盖：zh-Hans 下格式化 MsgCollectionFormationNote（Count=10），以宽度 300、14px 调用后断言。
func TestRadarScaleNoteBreaksFormationNoteIntoTwoCleanLines(t *testing.T) {
	// 飞机卡片的雷达说明按字宽换行会把 "P95" 拆到两行，这里要求按文案中的换行符断成两行。
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	note := i18n.Format(i18n.MsgCollectionFormationNote, map[string]any{"Count": 10})
	lines := radarScaleNoteLines(note, 300, 14)

	require.Len(t, lines, 2)
	require.Contains(t, lines[1], "P95")
	require.NotContains(t, lines[0], "P95")
}

// TestPlaneRadarProjectionIsLabelledRange 验证飞机「投送」维度改称「航程」，舰船仍称「投送」。
// 场景：飞机 projection 取的就是机体航程，若与舰船共用「投送」会让图鉴语义混淆。
// 预期：飞机 labelID 为「航程」、舰船为「投送」，飞机 tooltip 首行为「测试机 · 航程」。
// 覆盖：从 collectionAbilityDimensions 找 ID=projection，构造 plane/ship 两种 radarSubject 后断言。
func TestPlaneRadarProjectionIsLabelledRange(t *testing.T) {
	// 飞机的投送取值就是机体航程，图鉴里应显示为“航程”；舰船仍使用投送，避免两者混用。
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	var projection abilityDimension
	for _, dimension := range collectionAbilityDimensions {
		if dimension.ID == "projection" {
			projection = dimension
		}
	}
	require.Equal(t, "projection", projection.ID)

	plane := radarSubject{Name: "测试机", IsPlane: true}
	ship := radarSubject{Name: "测试舰"}
	require.Equal(t, "航程", i18n.Text(projection.labelID(plane)))
	require.Equal(t, "投送", i18n.Text(projection.labelID(ship)))
	require.Equal(t, "测试机 · 航程", radarTooltipLines(projection, plane, 100)[0])
}

// TestSubmarineShipClassFilter 验证潜艇是独立图鉴分类，不会落进驱逐舰筛选。
// 场景：潜艇与驱逐舰同属小型舰艇，旧分类若按吨位或大类合并会让两者互相串页。
// 预期：submarine 对 submarine/all 为 true，destroyer 与 submarine 双向筛选均为 false。
// 覆盖：四次 matchShipClass 调用，分别覆盖潜艇/驱逐舰类型与两个筛选项的组合。
func TestSubmarineShipClassFilter(t *testing.T) {
	// 潜艇是独立图鉴分类，不能再落进驱逐舰筛选。
	require.True(t, matchShipClass(objUnit.ShipTypeSubmarine, shipClassSubmarine))
	require.True(t, matchShipClass(objUnit.ShipTypeSubmarine, shipClassAll))
	require.False(t, matchShipClass(objUnit.ShipTypeDestroyer, shipClassSubmarine))
	require.False(t, matchShipClass(objUnit.ShipTypeSubmarine, shipClassDestroyer))
}

// TestCollectionScaleBarGeometryMatchesHundredMeterScale 验证 100 米比例尺的像素换算与左下角定位。
// 场景：493 米长的舰在蓝图中占 986 像素（2px/m），比例尺必须按同一比例绘制并贴蓝图左下角。
// 预期：collectionBlueprintScaleMeters=100；pixelsPerMeter(986,493)=2；比例尺距左/底边留白相等且条高小于 5。
// 覆盖：断言常量与像素换算，再用 1100×360 蓝图矩形调用 collectionBlueprintScaleBarOrigin 检查几何。
func TestCollectionScaleBarGeometryMatchesHundredMeterScale(t *testing.T) {
	require.Equal(t, 100.0, collectionBlueprintScaleMeters)
	require.InDelta(t, 2.0, collectionPixelsPerMeter(986, 493), 1e-9)

	rect := image.Rect(40, 20, 1140, 380)
	x, y, barH, margin := collectionBlueprintScaleBarOrigin(rect, 1)
	require.InDelta(t, margin, x-float64(rect.Min.X), 1e-9)
	require.InDelta(t, margin, float64(rect.Max.Y)-(y+barH), 1e-9)
	require.Less(t, barH, 5.0)
}
