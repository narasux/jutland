package drawer

import (
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/stretchr/testify/require"
)

func TestTruncateReinforceTextKeepsEllipsisWithinPanel(t *testing.T) {
	textFont := font.JetbrainsMono
	fontSize := 20.0
	maxWidth := layout.CalcTextWidth("Anti-Submarine Weapons: 2x5", fontSize, textFont)

	display, truncated := truncateReinforceText(
		"Anti-Submarine Weapons: 2x5 250mm Type 65 ASW rockets",
		maxWidth,
		fontSize,
		textFont,
	)

	require.True(t, truncated)
	require.True(t, strings.HasSuffix(display, "…"))
	require.LessOrEqual(t, layout.CalcTextWidth(display, fontSize, textFont), maxWidth)
}

func TestReinforceArchiveColumnsKeepRussianLabelsAndValuesSeparated(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageRussian))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	textFont := font.LocalizedUI(font.Kai)
	items := []reinforceArchiveItem{
		{label: i18n.Text(i18n.MsgReinforceType), value: "ТВ / Торпедный катер"},
		{label: i18n.Text(i18n.MsgReinforceHP), value: "48"},
		{label: i18n.Text(i18n.MsgReinforceSpeed), value: "41.0 уз."},
		{label: i18n.Text(i18n.MsgReinforceCost), value: "$25 / 11с"},
	}
	card := reinforceUIPanel{X: 48, W: 420}

	labelX, valueX, valueWidth := reinforceArchiveColumns(card, items, 20, textFont)

	for _, item := range items {
		labelWidth := layout.CalcTextWidth(item.label, 20, font.ForText(item.label, textFont))
		require.LessOrEqual(t, labelX+labelWidth, valueX)
	}
	require.Positive(t, valueWidth)
	require.LessOrEqual(t, valueX+valueWidth, card.X+card.W-24)
}

func TestLayoutReinforceTipsWrapsWithinAvailableSpace(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageEnglish))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	tips := []string{
		"↑ ↓ Reinforcement points",
		"← → Ships",
		"Enter Summon",
		"Backspace Cancel reinforcement",
		"Click map Set rally point",
	}
	maxWidth := 230.0
	maxHeight := 160.0

	result := layoutReinforceTips(tips, maxWidth, maxHeight, font.JetbrainsMono)

	lineCount := 0
	for _, lines := range result.Lines {
		lineCount += len(lines)
		for _, line := range lines {
			require.LessOrEqual(
				t,
				layout.CalcTextWidth(line, result.FontSize, font.JetbrainsMono),
				maxWidth,
			)
		}
	}
	requiredHeight := float64(lineCount)*result.LineHeight +
		float64(len(result.Lines)-1)*result.GroupGap
	require.LessOrEqual(t, requiredHeight, maxHeight)
	require.GreaterOrEqual(t, result.FontSize, 14.0)
}
