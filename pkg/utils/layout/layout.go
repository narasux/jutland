package layout

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	gamei18n "github.com/narasux/jutland/pkg/i18n"
)

// CalcTextWidth 计算文本宽度
func CalcTextWidth(textStr string, fontSize float64, source *text.GoTextFaceSource) float64 {
	width, _ := text.Measure(textStr, &text.GoTextFace{Source: source, Size: fontSize}, 0)
	return width
}

// WrapText 按当前语言和实际字体宽度换行。
func WrapText(value string, maxWidth, fontSize float64, source *text.GoTextFaceSource) []string {
	if CalcTextWidth(value, fontSize, source) <= maxWidth {
		return []string{value}
	}
	if gamei18n.CurrentLanguage() == gamei18n.LanguageEnglish {
		return wrapWords(value, maxWidth, fontSize, source)
	}
	return wrapRunes(value, maxWidth, fontSize, source)
}

func wrapWords(value string, maxWidth, fontSize float64, source *text.GoTextFaceSource) []string {
	words := strings.Fields(value)
	lines, line := []string{}, ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if CalcTextWidth(candidate, fontSize, source) <= maxWidth {
			line = candidate
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		if CalcTextWidth(word, fontSize, source) <= maxWidth {
			line = word
			continue
		}
		parts := wrapRunes(word, maxWidth, fontSize, source)
		lines = append(lines, parts[:len(parts)-1]...)
		line = parts[len(parts)-1]
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func wrapRunes(value string, maxWidth, fontSize float64, source *text.GoTextFaceSource) []string {
	lines, line := []string{}, ""
	for _, r := range value {
		candidate := line + string(r)
		if line != "" && CalcTextWidth(candidate, fontSize, source) > maxWidth {
			lines = append(lines, line)
			line = string(r)
			continue
		}
		line = candidate
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// ScreenPos 屏幕坐标
type ScreenPos struct {
	SX int
	SY int
}

// Screen 关卡屏幕
type ScreenLayout struct {
	// 屏幕宽高（一般是屏幕分辨率）
	Width  int
	Height int
}

// NewScreenLayout ...
func NewScreenLayout() ScreenLayout {
	width, height := ebiten.Monitor().Size()
	return ScreenLayout{Width: width, Height: height}
}
