package game

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/resources/font"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	"github.com/narasux/jutland/pkg/utils/colorx"
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

func (d *Drawer) drawMissionSelect(screen *ebiten.Image, curMission string, states *objStates) {
	screenW, screenH := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	abbrMap := d.getAbbrMap(curMission)

	// 暖色系（参照图鉴卡片）
	titleClr := color.RGBA{R: 240, G: 232, B: 218, A: 255}
	subtitleClr := color.RGBA{R: 175, G: 165, B: 150, A: 255}
	labelClr := color.RGBA{R: 214, G: 201, B: 178, A: 255}
	bodyClr := color.RGBA{R: 224, G: 210, B: 188, A: 255}
	cardFillClr := color.RGBA{R: 18, G: 18, B: 18, A: 172}
	cardBorderClr := color.RGBA{R: 214, G: 201, B: 178, A: 190}

	window := bgImg.MissionWindow
	windowWidth, windowHeight := window.Bounds().Dx(), window.Bounds().Dy()
	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(screenW/float64(windowWidth), screenH/float64(windowHeight))
	screen.DrawImage(window, opts)

	// 缩略地图（左侧）
	abbrX, abbrY := 50.0, 45.0
	abbrDisplaySize := min(screenW*0.48, screenH-130)
	abbrSrcW, abbrSrcH := float64(abbrMap.Bounds().Dx()), float64(abbrMap.Bounds().Dy())
	abbrScale := abbrDisplaySize / abbrSrcW
	abbrH := abbrSrcH * abbrScale

	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(abbrScale, abbrScale)
	opts.GeoM.Translate(abbrX, abbrY)
	screen.DrawImage(abbrMap, opts)

	// 地图银色边框
	strokeWidth := float32(4)
	vector.StrokeRect(
		screen,
		float32(abbrX), float32(abbrY)+strokeWidth,
		float32(abbrDisplaySize), float32(abbrH)-2*strokeWidth,
		strokeWidth, colorx.Silver, false,
	)

	// 右侧详情卡片（与地图等高）
	misMD := metadata.Get(curMission)
	cardGap := 60.0
	cardX := abbrX + abbrDisplaySize + cardGap
	cardY := abbrY
	cardW := screenW - cardX - 50
	cardH := abbrH

	// 卡片背景 + 边框（与图鉴卡片一致）
	vector.FillRect(
		screen,
		float32(cardX), float32(cardY), float32(cardW), float32(cardH),
		cardFillClr, false,
	)
	vector.StrokeRect(
		screen,
		float32(cardX), float32(cardY), float32(cardW), float32(cardH),
		2, cardBorderClr, false,
	)

	// 卡片内边距
	pad := 28.0
	panelX := cardX + pad
	curY := cardY + 32.0

	d.drawText(screen, misMD.DisplayName, panelX, curY, 36, font.LocalizedUI(font.Hang), titleClr)

	curY += 50
	d.drawText(screen, misMD.MapCfg.DisplayName, panelX, curY, 20, font.LocalizedUI(font.Kai), subtitleClr)

	// 标题分隔线
	curY += 32
	vector.StrokeLine(
		screen,
		float32(panelX), float32(curY),
		float32(cardX+cardW-pad), float32(curY),
		1, color.RGBA{R: 214, G: 201, B: 178, A: 120}, false,
	)

	curY += 22
	// 数据行
	dataLine := i18n.Format(i18n.MsgMissionStats, map[string]any{
		"Funds": misMD.InitFunds, "ShipLimit": misMD.MaxShipCount, "OilPlatforms": misMD.OilPlatformCount,
	})
	statsFontSize := 19.0
	statsLineHeight := 28.0
	statsMaxWidth := cardW - 2*pad
	dataLines := wrapText(dataLine, statsMaxWidth, statsFontSize)
	for idx, line := range dataLines {
		d.drawText(
			screen, line, panelX, curY+float64(idx)*statsLineHeight,
			statsFontSize, font.LocalizedUI(font.Kai), labelClr,
		)
	}

	curY += float64(len(dataLines))*statsLineHeight + 4
	// 战力对比
	battleLine := i18n.Format(i18n.MsgMissionBattleStats, map[string]any{
		"AllyShips": misMD.AllyShipCount, "EnemyShips": misMD.EnemyShipCount,
		"AllyPoints": misMD.AllyReinforceCount, "EnemyPoints": misMD.EnemyReinforceCount,
	})
	battleLines := wrapText(battleLine, statsMaxWidth, statsFontSize)
	for idx, line := range battleLines {
		d.drawText(
			screen, line, panelX, curY+float64(idx)*statsLineHeight,
			statsFontSize, font.LocalizedUI(font.Kai), bodyClr,
		)
	}

	curY += float64(len(battleLines))*statsLineHeight + 14
	// 描述区
	descFontSize := 18.0
	descLineHeight := 28.0
	descMaxWidth := cardW - 2*pad
	for idx, line := range wrapText(misMD.Description, descMaxWidth, descFontSize) {
		d.drawText(
			screen,
			line,
			panelX,
			curY+float64(idx)*descLineHeight,
			descFontSize,
			font.LocalizedUI(font.Kai),
			bodyClr,
		)
	}

	// 底部控件（卡片下方）
	arrowSize := 36.0
	startText := i18n.Text(i18n.MsgMissionStart)
	startFont := font.LocalizedUI(font.Hang)
	startTW := layout.CalcTextWidth(startText, 22, startFont)
	backText := i18n.Text(i18n.MsgBack)
	backFont := font.LocalizedUI(font.Hang)
	backTW := layout.CalcTextWidth(backText, 22, backFont)
	buttonW := max(150.0, max(startTW, backTW)+48)
	buttonH := 40.0
	controlY := screenH - 95

	// 左箭头
	leftX := screenW*0.38 - arrowSize/2
	leftArrow := clickableArea{X: leftX, Y: controlY - arrowSize/2, W: arrowSize, H: arrowSize}
	leftColor := colorx.Gray
	if isHoverArea(leftArrow) {
		leftColor = colorx.SkyBlue
	}
	d.drawText(screen, "<", leftX+4, controlY-12, 36, font.LocalizedUI(font.Hang), leftColor)

	// 右箭头
	rightX := screenW*0.62 - arrowSize/2
	rightArrow := clickableArea{X: rightX, Y: controlY - arrowSize/2, W: arrowSize, H: arrowSize}
	rightColor := colorx.Gray
	if isHoverArea(rightArrow) {
		rightColor = colorx.SkyBlue
	}
	d.drawText(screen, ">", rightX+4, controlY-12, 36, font.LocalizedUI(font.Hang), rightColor)

	// 关卡索引
	missions := metadata.AvailableMissions()
	for i, m := range missions {
		if m == curMission {
			idxText := fmt.Sprintf("%d / %d", i+1, len(missions))
			idxFont := font.LocalizedUI(font.Kai)
			idxW := layout.CalcTextWidth(idxText, 22, idxFont)
			d.drawText(screen, idxText, screenW/2-idxW/2, controlY-4, 22, idxFont, subtitleClr)
		}
	}

	// 开始任务按钮
	startX := screenW/2 - buttonW - 28
	startY := controlY + 40
	startBtn := clickableArea{X: startX, Y: startY, W: buttonW, H: buttonH}
	startColor := bodyClr
	borderColor := subtitleClr
	if isHoverArea(startBtn) {
		startColor = colorx.White
		borderColor = colorx.SkyBlue
	}
	vector.StrokeRect(
		screen,
		float32(startX),
		float32(startY),
		float32(buttonW),
		float32(buttonH),
		1,
		borderColor,
		false,
	)
	d.drawText(screen, startText, startX+(buttonW-startTW)/2, startY+10, 22, startFont, startColor)

	// 返回按钮
	backX := screenW/2 + 28
	backY := startY
	backBtn := clickableArea{X: backX, Y: backY, W: buttonW, H: buttonH}
	backColor := bodyClr
	backBorderColor := subtitleClr
	if isHoverArea(backBtn) {
		backColor = colorx.White
		backBorderColor = colorx.SkyBlue
	}
	vector.StrokeRect(
		screen,
		float32(backX),
		float32(backY),
		float32(buttonW),
		float32(buttonH),
		1,
		backBorderColor,
		false,
	)
	d.drawText(screen, backText, backX+(buttonW-backTW)/2, backY+10, 22, backFont, backColor)

	// 同步 UI 点击热区（基于实际屏幕尺寸，避免与 ebiten.WindowSize() 不一致）
	if states != nil {
		states.MissionSelectUI = &missionSelectUI{
			LeftArrow:   leftArrow,
			RightArrow:  rightArrow,
			StartButton: startBtn,
			BackButton:  backBtn,
		}
	}
}

func (d *Drawer) getAbbrMap(curMission string) *ebiten.Image {
	if img, ok := d.abbrMaps[curMission]; ok {
		return img
	}
	// 初始化
	misMD := metadata.Get(curMission)
	_, wHeight := ebiten.WindowSize()

	abbrMapSize := float64(wHeight) - 2*50
	abbrMap := ebiten.NewImage(int(abbrMapSize), int(abbrMapSize))

	bg := abbrMapImg.Background
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(abbrMapSize/float64(w), abbrMapSize/float64(h))

	abbrMap.DrawImage(abbrMapImg.Background, opts)
	abbrMap.DrawImage(abbrMapImg.Get(misMD.MapCfg.Source), opts)

	d.abbrMaps[curMission] = abbrMap
	return abbrMap
}

func wrapText(text string, maxWidth, fontSize float64) []string {
	textFont := font.ForText(text, font.LocalizedUI(font.Kai))
	return layout.WrapText(text, maxWidth, fontSize, textFont)
}

// 绘制游戏标题
func (d *Drawer) drawGameTitle(screen *ebiten.Image) {
	textStr := i18n.Text(i18n.MsgGameTitle)
	fontSize := float64(128)
	titleFont := font.LocalizedUI(font.Hang)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize, titleFont)) / 2
	posY := float64(screen.Bounds().Dy()) / 5 * 4
	d.drawText(screen, textStr, posX, posY, fontSize, titleFont, colorx.White)
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
	tipFont := font.LocalizedUI(font.Hang)
	posX := float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize, tipFont) - 50
	posY := float64(screen.Bounds().Dy()) / 10 * 9
	d.drawText(screen, textStr, posX, posY, fontSize, tipFont, colorx.White)
}

// 绘制任务结果
func (d *Drawer) drawMissionResult(screen *ebiten.Image, textStr string, textColor color.Color) {
	fontSize := float64(96)
	resultFont := font.LocalizedUI(font.Hang)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize, resultFont)) / 7
	posY := float64(screen.Bounds().Dy() / 8 * 7)
	d.drawText(screen, textStr, posX, posY, fontSize, resultFont, textColor)
}

// 绘制鸣谢
func (d *Drawer) drawCredits(screen *ebiten.Image) {
	// 注：英文感叹号字体是一样的，但是末尾留白少一些，对齐比较好看 :D
	textStr := i18n.Text(i18n.MsgGameCredits)
	fontSize := float64(128)
	creditsFont := font.LocalizedUI(font.Hang)
	posX := (float64(screen.Bounds().Dx()) - layout.CalcTextWidth(textStr, fontSize, creditsFont)) / 2
	posY := float64(screen.Bounds().Dy()) / 6 * 5
	d.drawText(screen, textStr, posX, posY, fontSize, creditsFont, colorx.White)
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
	textFont = font.ForText(textStr, textFont)
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

// isHoverArea 判定鼠标是否在可点击区域内
func isHoverArea(area clickableArea) bool {
	r := image.Rectangle{
		Min: image.Point{X: int(area.X), Y: int(area.Y)},
		Max: image.Point{X: int(area.X + area.W), Y: int(area.Y + area.H)},
	}
	return image.Pt(ebiten.CursorPosition()).In(r)
}
