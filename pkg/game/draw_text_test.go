package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
)

// TestCenteredButtonTextYCentersEveryLanguage 验证任务选择按钮文字按实际字体 metrics 垂直居中。
// 场景：按钮高度 40px、字号 22px，五种语言各自使用楷体/粉圆/JetBrains/Golos/Zen Kaku；
// 旧版固定写 button.Y+10，在 Noto Serif TC 等高 ascent 字体下文字会整体偏下。
// 预期：line box 中心等于 buttonY+buttonH/2（误差 1e-9），失败说明又退回固定像素偏移。
// 覆盖：遍历 SupportedLanguages，SetLanguage 后用 ForText 实际字体算 ascent/descent 并断言中心。
func TestCenteredButtonTextYCentersEveryLanguage(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	const (
		buttonY  = 100.0
		buttonH  = 40.0
		fontSize = 22.0
	)
	for _, lang := range i18n.SupportedLanguages() {
		i18n.SetLanguage(string(lang))
		label := i18n.Text(i18n.MsgMissionStart)
		source := font.LocalizedUI(font.Hang)
		face := &text.GoTextFace{Source: font.ForText(label, source), Size: fontSize}
		metrics := face.Metrics()
		got := centeredButtonTextY(label, buttonY, buttonH, fontSize, source)
		require.InDelta(t, buttonY+buttonH/2, got+(metrics.HAscent+metrics.HDescent)/2, 1e-9, string(lang))
	}
}
