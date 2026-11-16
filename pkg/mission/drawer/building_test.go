package drawer

import (
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/stretchr/testify/require"
)

// TestTruncateReinforceTextKeepsEllipsisWithinPanel 验证超宽武装文本裁剪后仍带省略号且宽度不超限。
// 场景：增援卡片里 "Anti-Submarine Weapons: 2x5 250mm Type 65 ASW rockets" 比同标题短串宽。
// 预期：truncated=true，结果以「…」结尾且宽度≤maxWidth；失败意味着文字溢出或被丢弃。
// 覆盖：以 20px JetbrainsMono 量 "Anti-Submarine Weapons: 2x5" 为上限，断言返回值与三个条件。
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

// TestCalcReinforcePreviewScaleUsesOneScaleForBothViews 验证侧视/俯视共用倍率，且由侧视高度决定。
// 场景：增援面板同时画中途岛级侧视图 1180×228 与俯视图 1180×226，内容区仅 1178×569。
// 预期：scale≈569*0.35/228≈0.873，两图宽高都不超配额；失败意味着预览溢出卡片或两视图比例不一致。
// 覆盖：调用 calcReinforcePreviewScale(1180,228,1180,226,1178,569)，断言 scale 及四条 ≤ 约束。
func TestCalcReinforcePreviewScaleUsesOneScaleForBothViews(t *testing.T) {
	// 中途岛的两张原图长度相同，但侧视图更高，统一倍率应受侧视图高度限制。
	scale := calcReinforcePreviewScale(1180, 228, 1180, 226, 1178, 569)

	require.InDelta(t, 569*0.35/228, scale, 1e-9)
	require.LessOrEqual(t, 1180*scale, 1178.0)
	require.LessOrEqual(t, 228*scale, 569*0.35)
	require.LessOrEqual(t, 1180*scale, 1178.0)
	require.LessOrEqual(t, 226*scale, 569*0.48)
}

// TestCalcReinforceArmamentTextLayoutFitsAllMidwayItems 验证 7 条武装在 202 高卡片内缩字号并留底边。
// 场景：中途岛级武装条目多，默认 20px 字号 + 32px 行距共需 212px，超过卡片可用高度 124px。
// 预期：fontSize/lineHeight>0，最后一行底部 58+6*lineHeight+fontSize≤182；失败意味着末条被裁掉。
// 覆盖：calcReinforceArmamentTextLayout(202,7) 后复算 lastLineBottom，断言 ≤182 且字号/行距为正。
func TestCalcReinforceArmamentTextLayoutFitsAllMidwayItems(t *testing.T) {
	const (
		cardH     = 202.0
		itemCount = 7
	)

	fontSize, lineHeight := calcReinforceArmamentTextLayout(cardH, itemCount)
	lastLineBottom := 58 + float64(itemCount-1)*lineHeight + fontSize

	require.LessOrEqual(t, lastLineBottom, cardH-20)
	require.Positive(t, fontSize)
	require.Positive(t, lineHeight)
}

// TestReinforceArchiveColumnsKeepRussianLabelsAndValuesSeparated 验证俄语标签变宽时档案卡两列仍不重叠。
// 场景：语言切到 ru，标签与「ТВ / Торпедный катер」等长值同卡展示；按英文宽度估列会把值挤出。
// 预期：各标签右缘 ≤ valueX，valueWidth>0 且值列不越卡片右内边距；失败意味着俄语下串列或溢出。
// 覆盖：临时 SetLanguage(russian)、Cleanup 还原，4 个 reinforceArchiveItem + 420 宽卡片逐项量宽。
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

// TestLayoutReinforceTipsWrapsWithinAvailableSpace 验证提示列在 230×160 内自动换行并从 18px 降到 14px。
// 场景：增援界面右下角空间窄小，"Backspace Cancel reinforcement" 等英文提示在 18px 时总高超过 160。
// 预期：每行宽 ≤230、总高 ≤160，最终字号 ≥14；失败意味着提示被裁切或字号跌破可读下限。
// 覆盖：5 条英文提示调用 layoutReinforceTips(230,160,JetbrainsMono)，遍历 Lines 量宽并按行数复算高度。
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
