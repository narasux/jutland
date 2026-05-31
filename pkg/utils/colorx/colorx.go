package colorx

import (
	"image/color"
	"strings"
)

var (
	// Red 红色
	Red = color.RGBA{R: 255, A: 255}
	// DarkRed 暗红
	DarkRed = color.RGBA{R: 139, A: 255}
	// Green 绿色
	Green = color.RGBA{G: 255, A: 255}
	// Blue 蓝色
	Blue = color.RGBA{B: 255, A: 255}
	// SkyBlue 天蓝
	SkyBlue = color.RGBA{R: 135, G: 206, B: 235, A: 255}
	// DarkBlue 暗蓝
	DarkBlue = color.RGBA{B: 139, A: 255}
	// Yellow 黄色
	Yellow = color.RGBA{R: 255, G: 255, A: 255}
	// Black 黑色
	Black = color.RGBA{A: 255}
	// White 白色
	White = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	// Gray 灰色
	Gray = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	// Cyan 青色
	Cyan = color.RGBA{G: 255, B: 255, A: 255}
	// Magenta 洋红
	Magenta = color.RGBA{R: 255, B: 255, A: 255}
	// Orange 橙色
	Orange = color.RGBA{R: 255, G: 165, A: 255}
	// Brown 棕色
	Brown = color.RGBA{R: 165, G: 42, B: 42, A: 255}
	// Pink 粉色
	Pink = color.RGBA{R: 255, G: 192, B: 203, A: 255}
	// Purple 紫色
	Purple = color.RGBA{R: 128, B: 128, A: 255}
	// Violet 紫罗兰
	Violet = color.RGBA{R: 238, G: 130, B: 238, A: 255}
	// Gold 金色
	Gold = color.RGBA{R: 255, G: 215, A: 255}
	// Silver 银色
	Silver = color.RGBA{R: 192, G: 192, B: 192, A: 255}
	// DarkSilver 暗银
	DarkSilver = color.RGBA{R: 105, G: 105, B: 105, A: 255}
	// LightGreen 浅绿色（半透明，用于医疗船治疗范围圈）
	LightGreen = color.RGBA{R: 144, G: 238, B: 144, A: 100}
)

var clrMap = map[string]color.Color{
	"red":        Red,
	"darkred":    DarkRed,
	"green":      Green,
	"blue":       Blue,
	"skyblue":    SkyBlue,
	"darkblue":   DarkBlue,
	"yellow":     Yellow,
	"black":      Black,
	"white":      White,
	"gray":       Gray,
	"cyan":       Cyan,
	"magenta":    Magenta,
	"orange":     Orange,
	"brown":      Brown,
	"pink":       Pink,
	"purple":     Purple,
	"violet":     Violet,
	"gold":       Gold,
	"silver":     Silver,
	"darksilver": DarkSilver,
	"lightgreen": LightGreen,
}

// GetColorByName 根据名称获取颜色
func GetColorByName(clr string) color.Color {
	// 替换空格
	clr = strings.ReplaceAll(clr, " ", "")
	// 转换为小写
	clr = strings.ToLower(clr)

	return clrMap[clr]
}
