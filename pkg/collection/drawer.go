package collection

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
)

type collectionCard struct {
	// collectionCard 是图鉴下半区固定卡片的几何描述，只保存渲染时需要的矩形参数。
	X, Y, W, H float64
}

type clickableArea struct {
	// clickableArea 表示纯几何命中区，不依赖 EbitenUI 控件树。
	X, Y, W, H float64
}

// Drawer 封装图鉴专用绘制逻辑。
// 把文本、图片和卡片样式集中在这里，能减少 UI 代码里重复的绘制参数。
type Drawer struct{}

func NewDrawer() *Drawer { return &Drawer{} }

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
	text.Draw(screen, textStr, &text.GoTextFace{Source: textFont, Size: fontSize}, opts)
}

func (d *Drawer) drawCollectionImageFit(
	screen *ebiten.Image,
	img *ebiten.Image,
	x, y, width, height, rotation float64,
	allowUpscale bool,
) {
	// 图鉴中的舰船和飞机素材尺寸不统一，这里按目标框等比缩放，必要时允许放大。
	if img == nil || width <= 0 || height <= 0 {
		return
	}
	scale := collectionImageFitScale(img, width, height, rotation, allowUpscale)
	d.drawCollectionImageScaled(screen, img, x, y, width, height, rotation, scale)
}

func collectionImageFitScale(img *ebiten.Image, width, height, rotation float64, allowUpscale bool) float64 {
	if img == nil || width <= 0 || height <= 0 {
		return 0
	}
	imgW, imgH := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	rotatedW, rotatedH := imgW, imgH
	if int(math.Mod(math.Abs(rotation), 180)) == 90 {
		rotatedW, rotatedH = imgH, imgW
	}
	scale := min(width/rotatedW, height/rotatedH)
	if !allowUpscale {
		scale = min(1, scale)
	} else {
		scale = min(8, scale)
	}
	return scale
}

func (d *Drawer) drawCollectionImageScaled(
	screen *ebiten.Image,
	img *ebiten.Image,
	x, y, width, height, rotation, scale float64,
) {
	if img == nil || width <= 0 || height <= 0 || scale <= 0 {
		return
	}
	imgW, imgH := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Translate(-imgW/2, -imgH/2)
	opts.GeoM.Rotate(rotation * math.Pi / 180)
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(x+width/2, y+height/2)
	screen.DrawImage(img, opts)
}

func planeArmamentItems(plane *objUnit.Plane) []objRef.InfoItem {
	// 飞机武装按武器类型分组，重复同名武器压缩成“数量 x 名称”的形式。
	// 这样飞机页能在较窄的纵列里同时展示基础数据和武器配置。
	items := []objRef.InfoItem{}
	appendGroup := func(label string, names []string) {
		counts, order := map[string]int{}, []string{}
		for _, name := range names {
			if counts[name] == 0 {
				order = append(order, name)
			}
			counts[name]++
		}
		for _, name := range order {
			items = append(items, objRef.InfoItem{
				Label: label,
				Value: i18n.Format(i18n.MsgItemCount, map[string]any{
					"Name": name, "Count": counts[name],
				}),
			})
		}
	}
	gunNames := make([]string, 0, len(plane.Weapon.Guns))
	for _, gun := range plane.Weapon.Guns {
		gunNames = append(gunNames, gun.Name)
	}
	bombNames := make([]string, 0, len(plane.Weapon.Bombs))
	for _, bomb := range plane.Weapon.Bombs {
		bombNames = append(bombNames, bomb.Name)
	}
	torpedoNames := make([]string, 0, len(plane.Weapon.Torpedoes))
	for _, torpedo := range plane.Weapon.Torpedoes {
		torpedoNames = append(torpedoNames, torpedo.Name)
	}
	rocketNames := make([]string, 0, len(plane.Weapon.Rockets))
	for _, rocket := range plane.Weapon.Rockets {
		rocketNames = append(rocketNames, rocket.Name)
	}
	appendGroup(i18n.Text(i18n.MsgWeaponGun), gunNames)
	appendGroup(i18n.Text(i18n.MsgWeaponBomb), bombNames)
	appendGroup(i18n.Text(i18n.MsgWeaponTorpedo), torpedoNames)
	appendGroup(i18n.Text(i18n.MsgWeaponRocket), rocketNames)
	if len(items) == 0 {
		items = append(items, objRef.InfoItem{
			Label: i18n.Text(i18n.MsgCollectionArmaments),
			Value: i18n.Text(i18n.MsgCollectionNone),
		})
	}
	return items
}

func (d *Drawer) drawCollectionCard(
	screen *ebiten.Image, card collectionCard, title string, metrics collectionMetrics,
) {
	// 卡片统一的底色、边框和标题栏，避免舰船页各卡片样式漂移。
	scale := metrics.Scale
	vector.FillRect(
		screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		color.RGBA{R: 18, G: 18, B: 18, A: 172}, false,
	)
	vector.StrokeRect(
		screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		2, color.RGBA{R: 214, G: 201, B: 178, A: 190}, false,
	)
	titleFontSize := metrics.CardTitle
	if i18n.CurrentLanguage() == i18n.LanguageEnglish {
		titleFontSize *= 0.82
	}
	maxTitleWidth := card.W - 40*scale
	if titleWidth := estimateCollectionTextWidth(title, titleFontSize); titleWidth > maxTitleWidth {
		titleFontSize *= maxTitleWidth / titleWidth
	}
	d.drawText(
		screen, title, card.X+20*scale, card.Y+18*scale,
		titleFontSize, font.LocalizedUI(font.Kai), color.RGBA{R: 230, G: 218, B: 194, A: 255},
	)
	vector.StrokeLine(
		screen, float32(card.X+20*scale), float32(card.Y+44*scale),
		float32(card.X+card.W-20*scale), float32(card.Y+44*scale),
		1, color.RGBA{R: 214, G: 201, B: 178, A: 120}, false,
	)
}

func (d *Drawer) drawCollectionLines(
	screen *ebiten.Image,
	lines []string,
	x, y, maxWidth, fontSize, lineHeight float64,
	textFont *text.GoTextFaceSource,
	maxLines int,
	textColor color.Color,
) {
	// 文本卡片需要按宽度折行，但又不想引入复杂排版器，所以这里使用保守的手写折行逻辑。
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

func wrapCollectionText(value string, maxWidth, fontSize float64) []string {
	textFont := font.ForText(value, font.LocalizedUI(font.Kai))
	return layout.WrapText(value, maxWidth, fontSize, textFont)
}

func estimateCollectionTextWidth(value string, fontSize float64) float64 {
	textFont := font.ForText(value, font.LocalizedUI(font.Kai))
	return layout.CalcTextWidth(value, fontSize, textFont)
}

func isHoverArea(area clickableArea) bool {
	// 由于外部链接区域是手工维护的矩形，所以只需要简单的鼠标命中判断。
	rect := image.Rect(int(area.X), int(area.Y), int(area.X+area.W), int(area.Y+area.H))
	return image.Pt(ebiten.CursorPosition()).In(rect)
}

func isMouseButtonLeftJustPressed() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}
