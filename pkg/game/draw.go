package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/resources/font"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// Drawer 图像绘制工具
type Drawer struct {
	abbrMaps map[string]*ebiten.Image
}

// NewDrawer ...
func NewDrawer() *Drawer {
	return &Drawer{
		abbrMaps: map[string]*ebiten.Image{},
	}
}

// 绘制背景
func (d *Drawer) drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()

	scaleX := float64(screen.Bounds().Dx()) / float64(w)
	scaleY := float64(screen.Bounds().Dy()) / float64(h)

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(bg, opts)
}

func (d *Drawer) drawMissionSelect(screen *ebiten.Image, curMission string) {
	misLayout := layout.NewScreenLayout()
	abbrMap := d.getAbbrMap(curMission)

	window := bgImg.MissionWindow
	windowWidth, windowHeight := window.Bounds().Dx(), window.Bounds().Dy()
	abbrMapWidth, abbrMapHeight := abbrMap.Bounds().Dx(), abbrMap.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(float64(misLayout.Width)/float64(windowWidth), float64(misLayout.Height)/float64(windowHeight))
	screen.DrawImage(window, opts)

	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(50, 50)
	screen.DrawImage(abbrMap, opts)
	// 缩略地图添加银色边框
	strokeWidth := float32(4)
	vector.StrokeRect(
		screen,
		float32(50), float32(50)+strokeWidth, float32(abbrMapWidth),
		float32(abbrMapHeight)-2*strokeWidth, strokeWidth,
		colorx.Silver, false,
	)

	// 关卡名称，描述，配置等
	misMD := metadata.Get(curMission)
	xOffset, yOffset := float64(abbrMapWidth+120), float64(120)
	d.drawText(screen, misMD.DisplayName, xOffset, yOffset, 40, font.Hang, colorx.White)

	yOffset += 80
	for idx, line := range misMD.Description {
		d.drawText(screen, line, xOffset, yOffset+float64(idx)*50, 24, font.Hang, colorx.White)
	}

	// 方向键 + 提示
	x, y := float64(abbrMapWidth+120), float64(misLayout.Height)-250
	drawArrowKey := func(xOffset, yOffset, rotation float64) {
		opts = d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, textureImg.ArrowKey, rotation)
		opts.GeoM.Translate(x+xOffset, y+yOffset)
		screen.DrawImage(textureImg.ArrowKey, opts)
	}
	drawArrowKey(45, 90, 90)
	drawArrowKey(-45, 90, 270)

	d.drawText(screen, "选择", x+15, y+200, 24, font.Hang, colorx.White)

	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(x+145, y+80)
	screen.DrawImage(textureImg.EnterKey, opts)

	d.drawText(screen, "确定", x+145, y+200, 24, font.Hang, colorx.White)
}

func (d *Drawer) getAbbrMap(curMission string) *ebiten.Image {
	if img, ok := d.abbrMaps[curMission]; ok {
		return img
	}
	// 初始化
	misMD := metadata.Get(curMission)
	misLayout := layout.NewScreenLayout()

	abbrMapSize := float64(misLayout.Height) - 2*50
	abbrMap := ebiten.NewImage(int(abbrMapSize), int(abbrMapSize))

	bg := abbrMapImg.Background
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(abbrMapSize/float64(w), abbrMapSize/float64(h))

	abbrMap.DrawImage(abbrMapImg.Background, opts)
	abbrMap.DrawImage(abbrMapImg.Get(misMD.MapCfg.Name), opts)

	d.abbrMaps[curMission] = abbrMap
	return abbrMap
}

// 绘制游戏标题
func (d *Drawer) drawGameTitle(screen *ebiten.Image) {
	textStr := "怒 海 激 战"
	fontSize := float64(128)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize)) / 2
	posY := float64(screen.Bounds().Dy()) / 5 * 4
	d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
}

// 绘制游戏菜单
func (d *Drawer) drawGameMenu(screen *ebiten.Image, states *menuButtonStates) {
	for _, b := range []*menuButton{
		states.MissionSelect,
		states.Collection,
		states.GameSetting,
		states.ExitGame,
	} {
		d.drawText(screen, b.Text, b.PosX, b.PosY, b.FontSize, b.Font, b.Color)
	}
}

// 绘制游戏通用提示
func (d *Drawer) drawGameTip(screen *ebiten.Image, textStr string) {
	fontSize := float64(64)
	posX := float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize) - 50
	posY := float64(screen.Bounds().Dy()) / 10 * 9
	d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
}

// 绘制任务结果
func (d *Drawer) drawMissionResult(screen *ebiten.Image, textStr string, textColor color.Color) {
	fontSize := float64(96)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize)) / 7
	posY := float64(screen.Bounds().Dy() / 8 * 7)
	d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, textColor)
}

// 绘制鸣谢
func (d *Drawer) drawCredits(screen *ebiten.Image) {
	// 注：英文感叹号字体是一样的，但是末尾留白少一些，对齐比较好看 :D
	textStr := "祝君武运昌隆!"
	fontSize := float64(128)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize)) / 2
	posY := float64(screen.Bounds().Dy()) / 6 * 5
	d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
}

// 绘制文本
func (d *Drawer) drawText(
	screen *ebiten.Image,
	textStr string,
	posX, posY, fontSize float64,
	textFont *text.GoTextFaceSource,
	textColor color.Color,
) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(posX, posY)
	opts.ColorScale.ScaleWithColor(textColor)
	textFace := text.GoTextFace{
		Source: textFont,
		Size:   fontSize,
	}
	text.Draw(screen, textStr, &textFace, opts)
}

// 默认绘制配置
func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
