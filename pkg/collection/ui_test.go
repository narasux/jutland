package collection

import (
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/stretchr/testify/require"
)

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
