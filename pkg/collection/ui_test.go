package collection

import (
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
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
