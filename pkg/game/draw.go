package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// Drawer 图像绘制工具
type Drawer struct {
	abbrMaps map[string]*ebiten.Image
}

type collectionLayout struct {
	GraphX, GraphY, GraphW, GraphH float64
	InfoY, InfoH                   float64
	CardGap                        float64
	ArchiveCard                    collectionCard
	ArmamentCard                   collectionCard
	SourceCard                     collectionCard
}

type collectionCard struct {
	X, Y, W, H float64
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
	x, y := float64(misLayout.Width)-300, float64(misLayout.Height)-300
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
	abbrMap.DrawImage(abbrMapImg.Get(misMD.MapCfg.Source), opts)

	d.abbrMaps[curMission] = abbrMap
	return abbrMap
}

// 绘制战舰图鉴
func (d *Drawer) drawCollection(screen *ebiten.Image, curShipName string, refLinks []*refLink) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	collLayout := calcCollectionLayout(screenWidth, screenHeight)

	d.drawCollectionDesignGraph(screen, curShipName, collLayout)
	d.drawCollectionInfoCards(screen, curShipName, refLinks, collLayout)
}

func (d *Drawer) drawCollectionDesignGraph(screen *ebiten.Image, curShipName string, collLayout collectionLayout) {
	bgWidth, bgHeight := collLayout.GraphW, collLayout.GraphH
	xOffset, yOffset := collLayout.GraphX, collLayout.GraphY

	opts := d.genDefaultDrawImageOptions()
	designGraphImg := bgImg.MissionWindowParchment
	opts.GeoM.Scale(
		bgWidth/float64(designGraphImg.Bounds().Dx()),
		bgHeight/float64(designGraphImg.Bounds().Dy()),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(designGraphImg, opts)

	// 设计图添加银色边框
	vector.StrokeRect(
		screen, float32(xOffset), float32(yOffset),
		float32(bgWidth), float32(bgHeight),
		5, colorx.Silver, false,
	)

	// 绘制战舰 侧视图 & 俯视图
	sideImg := shipImg.GetSide(curShipName, 4)
	sideImgDx, sideImgDy := sideImg.Bounds().Dx(), sideImg.Bounds().Dy()

	topImg := shipImg.GetTop(curShipName, 4)
	// x, y 互换，因为需要顺时针旋转 90 度
	topImgDx, topImgDy := topImg.Bounds().Dy(), topImg.Bounds().Dx()

	paddingX := (bgWidth - float64(sideImgDx)) / 7 * 3
	paddingY := (bgHeight - float64(topImgDy+sideImgDy)) / 3

	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(xOffset+paddingX, yOffset+paddingY)
	screen.DrawImage(sideImg, opts)

	opts = d.genDefaultDrawImageOptions()
	ebutil.SetOptsCenterRotation(opts, topImg, 90)
	opts.GeoM.Translate(xOffset+paddingX, yOffset+2*paddingY+float64(sideImgDy))
	opts.GeoM.Translate(float64(topImgDx-topImgDy)/2, float64(topImgDy-topImgDx)/2)
	screen.DrawImage(topImg, opts)
}

func (d *Drawer) drawCollectionInfoCards(
	screen *ebiten.Image, curShipName string, refLinks []*refLink, layout collectionLayout,
) {
	ref := objRef.GetReference(curShipName)
	allShipNames := objUnit.GetAllShipNames()

	d.drawCollectionCard(screen, layout.ArchiveCard, "舰船档案")
	shipIndexText := fmt.Sprintf("%d/%d", lo.IndexOf(allShipNames, curShipName)+1, len(allShipNames))
	d.drawText(
		screen, shipIndexText,
		layout.ArchiveCard.X+layout.ArchiveCard.W-24-estimateCollectionTextWidth(shipIndexText, 16),
		layout.ArchiveCard.Y+20,
		16, font.Kai, color.RGBA{230, 218, 194, 220},
	)
	shipDisplayName := objUnit.GetShipDisplayName(curShipName)
	d.drawText(screen, shipDisplayName, layout.ArchiveCard.X+24, layout.ArchiveCard.Y+54, 36, font.Hang, colorx.White)
	d.drawCollectionInfoItems(
		screen, ref.Specs, layout.ArchiveCard.X+24, layout.ArchiveCard.Y+104,
		layout.ArchiveCard.W-140, 28,
	)

	d.drawCollectionCard(screen, layout.ArmamentCard, "武装配置")
	d.drawCollectionInfoItems(
		screen, ref.Armaments, layout.ArmamentCard.X+24, layout.ArmamentCard.Y+54,
		layout.ArmamentCard.W-140, 30,
	)

	d.drawCollectionCard(screen, layout.SourceCard, "历史与来源")
	descriptionLines := wrapCollectionText(ref.Description, layout.SourceCard.W-48, 20)
	authorY := collectionSourceMetaY(layout, len(descriptionLines))
	maxDescriptionLines := max(1, int((authorY-layout.SourceCard.Y-78)/26))
	d.drawCollectionLines(
		screen, descriptionLines, layout.SourceCard.X+24, layout.SourceCard.Y+54,
		layout.SourceCard.W-48, 20, 26, font.Kai, maxDescriptionLines, colorx.White,
	)
	d.drawText(screen, fmt.Sprintf("素材原作者：%s", ref.Author), layout.SourceCard.X+24, authorY, 20, font.Kai, colorx.White)
	d.drawText(screen, "参考资料：", layout.SourceCard.X+24, authorY+34, 20, font.Kai, colorx.White)
	for _, link := range refLinks {
		d.drawText(screen, link.Text, link.PosX, link.PosY, link.FontSize, link.Font, link.Color)
	}
}

func (d *Drawer) drawCollectionCard(screen *ebiten.Image, card collectionCard, title string) {
	vector.FillRect(
		screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		color.RGBA{18, 18, 18, 172}, false,
	)
	vector.StrokeRect(
		screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		2, color.RGBA{214, 201, 178, 190}, false,
	)
	d.drawText(screen, title, card.X+20, card.Y+18, 20, font.Kai, color.RGBA{230, 218, 194, 255})
	vector.StrokeLine(
		screen, float32(card.X+20), float32(card.Y+44), float32(card.X+card.W-20), float32(card.Y+44),
		1, color.RGBA{214, 201, 178, 120}, false,
	)
}

func (d *Drawer) drawCollectionInfoItems(
	screen *ebiten.Image, items []objRef.InfoItem, x, y, valueMaxWidth, lineHeight float64,
) {
	drawn := 0
	for _, item := range items {
		lineY := y + float64(drawn)*lineHeight
		d.drawText(screen, item.Label, x, lineY, 20, font.Kai, color.RGBA{214, 201, 178, 255})

		valueFont := font.Kai
		wrappedValues := wrapCollectionText(item.Value, valueMaxWidth, 20)
		for idx, value := range wrappedValues {
			d.drawText(screen, value, x+76, lineY+float64(idx)*lineHeight, 20, valueFont, colorx.White)
		}
		drawn += max(1, len(wrappedValues))
	}
}

func (d *Drawer) drawCollectionLines(
	screen *ebiten.Image,
	lines []string,
	x, y, maxWidth, fontSize, lineHeight float64,
	textFont *text.GoTextFaceSource,
	maxLines int,
	textColor color.Color,
) {
	drawn := 0
	for _, line := range lines {
		for _, wrapped := range wrapCollectionText(line, maxWidth, fontSize) {
			if drawn >= maxLines {
				return
			}
			d.drawText(screen, wrapped, x, y+float64(drawn)*lineHeight, fontSize, textFont, textColor)
			drawn++
		}
	}
}

func calcCollectionLayout(screenWidth, screenHeight int) collectionLayout {
	screenW, screenH := float64(screenWidth), float64(screenHeight)
	graphW, graphH := screenW*0.88, screenH*0.48
	graphX, graphY := (screenW-graphW)/2, 50.0
	infoY := graphY + graphH + 52
	infoH := screenH - infoY - 54
	gap := 24.0
	infoX, infoW := graphX, graphW
	archiveW, armamentW := infoW*0.23, infoW*0.24
	sourceW := infoW - archiveW - armamentW - 2*gap
	return collectionLayout{
		GraphX: graphX, GraphY: graphY, GraphW: graphW, GraphH: graphH,
		InfoY: infoY, InfoH: infoH, CardGap: gap,
		ArchiveCard:  collectionCard{X: infoX, Y: infoY, W: archiveW, H: infoH},
		ArmamentCard: collectionCard{X: infoX + archiveW + gap, Y: infoY, W: armamentW, H: infoH},
		SourceCard:   collectionCard{X: infoX + archiveW + armamentW + 2*gap, Y: infoY, W: sourceW, H: infoH},
	}
}

func collectionRefLinkOriginByDescription(screenWidth, screenHeight int, description string) (float64, float64) {
	collLayout := calcCollectionLayout(screenWidth, screenHeight)
	descriptionLines := wrapCollectionText(description, collLayout.SourceCard.W-48, 20)
	return collLayout.SourceCard.X + 24, collectionSourceMetaY(collLayout, len(descriptionLines)) + 48
}

func collectionSourceMetaY(layout collectionLayout, descriptionLineCount int) float64 {
	descriptionBottomY := layout.SourceCard.Y + 54 + float64(descriptionLineCount)*30
	return min(descriptionBottomY+42, layout.SourceCard.Y+layout.SourceCard.H-100)
}

func wrapCollectionText(text string, maxWidth, fontSize float64) []string {
	if estimateCollectionTextWidth(text, fontSize) <= maxWidth {
		return []string{text}
	}
	lines := []string{}
	line := ""
	for _, r := range text {
		next := line + string(r)
		if line != "" && estimateCollectionTextWidth(next, fontSize) > maxWidth {
			lines = append(lines, line)
			line = string(r)
			continue
		}
		line = next
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// estimateCollectionTextWidth 粗略估算图鉴文本宽度，用于换行和同排文本定位。
func estimateCollectionTextWidth(text string, fontSize float64) float64 {
	width := 0.0
	for _, r := range text {
		if r <= 127 {
			width += fontSize * 0.55
		} else {
			width += fontSize
		}
	}
	return width
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
