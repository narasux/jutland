package texture

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
)

var (
	// ArrowWhite 箭头图片（白）
	ArrowWhiteImg *ebiten.Image
	// SelectBox 选择框图片
	SelectBoxWhiteImg *ebiten.Image

	// MainGunEnabledImg 主炮启用图片
	MainGunEnabledImg *ebiten.Image
	// MainGunDisabledImg 主炮禁用图片
	MainGunDisabledImg *ebiten.Image

	// SecondaryGunEnabledImg 副炮启用图片
	SecondaryGunEnabledImg *ebiten.Image
	// SecondaryGunDisabledImg 副炮禁用图片
	SecondaryGunDisabledImg *ebiten.Image

	// TorpedoEnabledImg 鱼雷启用图片
	TorpedoEnabledImg *ebiten.Image
	// TorpedoDisabledImg 鱼雷禁用图片
	TorpedoDisabledImg *ebiten.Image

	// ShipSelectedImg 选中战舰标志图片
	ShipSelectedImg *ebiten.Image

	// TargetPosImg 目标位置标志图片
	TargetPosImg *ebiten.Image
)

func init() {
	var err error

	log.Println("loading texture image resources...")

	imgPath := "/textures/flag/arrow_white.png"
	if ArrowWhiteImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/select_box_white.png"
	if SelectBoxWhiteImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载主炮启停用图标
	imgPath = "/textures/flag/main_gun_enabled.png"
	if MainGunEnabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/main_gun_disabled.png"
	if MainGunDisabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载副炮启停用图标
	imgPath = "/textures/flag/secondary_gun_enabled.png"
	if SecondaryGunEnabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/secondary_gun_disabled.png"
	if SecondaryGunDisabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载鱼雷启停用图标
	imgPath = "/textures/flag/torpedo_enabled.png"
	if TorpedoEnabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/torpedo_disabled.png"
	if TorpedoDisabledImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载选中战舰标志
	imgPath = "/textures/flag/ship_selected.png"
	if ShipSelectedImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载目标位置标志
	imgPath = "/textures/flag/target_pos.png"
	if TargetPosImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("texture image resources loaded")
}

var textImages = map[string]*ebiten.Image{}

// 获取行书文本图像（支持缓存）
func GetHangTextImg(textStr string, fontSize float64, clr color.Color) *ebiten.Image {
	r, g, b, a := clr.RGBA()
	key := fmt.Sprintf("%s:size:%d:clr:%d:%d:%d:%d", textStr, fontSize, r, g, b, a)
	if img, ok := textImages[key]; ok {
		return img
	}

	img := ebiten.NewImage(int(layout.CalcTextWidth(textStr, fontSize)*1.5), int(fontSize))
	opts := &text.DrawOptions{}
	opts.ColorScale.ScaleWithColor(clr)
	textFace := text.GoTextFace{
		Source: font.Hang,
		Size:   fontSize,
	}
	text.Draw(img, textStr, &textFace, opts)

	textImages[key] = img
	return img
}
