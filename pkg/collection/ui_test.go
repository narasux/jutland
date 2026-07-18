package collection

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/i18n"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/stretchr/testify/require"
)

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

func TestTruncatedTextAreaAt(t *testing.T) {
	areas := []truncatedTextHitArea{
		{Rect: image.Rect(10, 20, 110, 45), Lines: []string{"full armament"}},
	}

	require.Same(t, &areas[0], truncatedTextAreaAt(areas, image.Pt(50, 30)))
	require.Nil(t, truncatedTextAreaAt(areas, image.Pt(110, 30)))
}

func TestDropdownListWidthFitsLongestLabelAndScreen(t *testing.T) {
	fontSize := 20.0
	scale := 1.0
	labels := []string{"All", "Aircraft Carrier"}
	contentWidth := int(math.Ceil(estimateCollectionTextWidth(labels[1], fontSize) + 36*scale))

	require.Equal(t, contentWidth, dropdownListWidth(100, 500, scale, fontSize, labels))
	require.Equal(t, 120, dropdownListWidth(100, 120, scale, fontSize, labels))
}

func TestDropdownChevronGeometryFormsCenteredDownwardV(t *testing.T) {
	rect := image.Rect(100, 20, 260, 60)
	chevron := dropdownChevronGeometry(rect, 1)

	require.Equal(t, float32(247), chevron.CenterX)
	require.Equal(t, float32(42), chevron.CenterY)
	require.Equal(t, chevron.CenterX-chevron.LeftX, chevron.RightX-chevron.CenterX)
	require.Less(t, chevron.LeftY, chevron.CenterY)
	require.Equal(t, chevron.LeftY, chevron.RightY)
}

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
