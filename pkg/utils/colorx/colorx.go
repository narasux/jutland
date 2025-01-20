package colorx

import (
	"image/color"
	"strings"
)

var (
	// Red 红色
	Red = color.RGBA{255, 0, 0, 255}
	// DarkRed 暗红
	DarkRed = color.RGBA{139, 0, 0, 255}
	// Green 绿色
	Green = color.RGBA{0, 255, 0, 255}
	// Blue 蓝色
	Blue = color.RGBA{0, 0, 255, 255}
	// SkyBlue 天蓝
	SkyBlue = color.RGBA{135, 206, 235, 255}
	// DarkBlue 暗蓝
	DarkBlue = color.RGBA{0, 0, 139, 255}
	// Yellow 黄色
	Yellow = color.RGBA{255, 255, 0, 255}
	// Black 黑色
	Black = color.RGBA{0, 0, 0, 255}
	// White 白色
	White = color.RGBA{255, 255, 255, 255}
	// Gray 灰色
	Gray = color.RGBA{128, 128, 128, 255}
	// Cyan 青色
	Cyan = color.RGBA{0, 255, 255, 255}
	// Magenta 洋红
	Magenta = color.RGBA{255, 0, 255, 255}
	// Orange 橙色
	Orange = color.RGBA{255, 165, 0, 255}
	// Brown 棕色
	Brown = color.RGBA{165, 42, 42, 255}
	// Pink 粉色
	Pink = color.RGBA{255, 192, 203, 255}
	// Purple 紫色
	Purple = color.RGBA{128, 0, 128, 255}
	// Violet 紫罗兰
	Violet = color.RGBA{238, 130, 238, 255}
	// Gold 金色
	Gold = color.RGBA{255, 215, 0, 255}
	// Silver 银色
	Silver = color.RGBA{192, 192, 192, 255}
	// DarkSilver 暗银
	DarkSilver = color.RGBA{105, 105, 105, 255}
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
}

// GetColorByName 根据名称获取颜色
func GetColorByName(clr string) color.Color {
	// 替换空格
	clr = strings.ReplaceAll(clr, " ", "")
	// 转换为小写
	clr = strings.ToLower(clr)

	return clrMap[clr]
}
