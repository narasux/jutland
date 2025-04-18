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
	ArrowWhite *ebiten.Image

	// ArrowKey 方向键
	ArrowKey *ebiten.Image
	// EnterKey 回车键
	EnterKey *ebiten.Image
	// BackspaceKey 退格键
	BackspaceKey *ebiten.Image

	// MainGunEnabled 主炮启用图片
	MainGunEnabled *ebiten.Image
	// MainGunDisabled 主炮禁用图片
	MainGunDisabled *ebiten.Image

	// SecondaryGunEnabled 副炮启用图片
	SecondaryGunEnabled *ebiten.Image
	// SecondaryGunDisabled 副炮禁用图片
	SecondaryGunDisabled *ebiten.Image

	// TorpedoEnabled 鱼雷启用图片
	TorpedoEnabled *ebiten.Image
	// TorpedoDisabled 鱼雷禁用图片
	TorpedoDisabled *ebiten.Image

	// ShipSelected 选中战舰标志图片
	ShipSelected *ebiten.Image

	// TargetPos 目标位置标志图片
	TargetPos *ebiten.Image
	// LockOnTarget 锁定目标标志图片
	LockOnTarget *ebiten.Image
	// AttackTarget 攻击目标标志图片
	AttackTarget *ebiten.Image
)

func init() {
	var err error

	log.Println("loading flag image resources...")

	imgPath := "/textures/flag/arrow_white.png"
	if ArrowWhite, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	imgPath = "/textures/flag/arrow_key.png"
	if ArrowKey, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	imgPath = "/textures/flag/enter_key.png"
	if EnterKey, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	imgPath = "/textures/flag/backspace_key.png"
	if BackspaceKey, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载主炮启停用图标
	imgPath = "/textures/flag/main_gun_enabled.png"
	if MainGunEnabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/main_gun_disabled.png"
	if MainGunDisabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载副炮启停用图标
	imgPath = "/textures/flag/secondary_gun_enabled.png"
	if SecondaryGunEnabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/secondary_gun_disabled.png"
	if SecondaryGunDisabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载鱼雷启停用图标
	imgPath = "/textures/flag/torpedo_enabled.png"
	if TorpedoEnabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/torpedo_disabled.png"
	if TorpedoDisabled, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载选中战舰标志
	imgPath = "/textures/flag/ship_selected.png"
	if ShipSelected, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载目标位置标志
	imgPath = "/textures/flag/target_pos.png"
	if TargetPos, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	// 加载锁定目标标志图片
	imgPath = "/textures/flag/lock_on_target.png"
	if LockOnTarget, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	// 攻击目标标志图片
	imgPath = "/textures/flag/attack_target.png"
	if AttackTarget, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("flag image resources loaded")
}

var textImages = map[string]*ebiten.Image{}

// GetText 获取文本图像（支持缓存）
func GetText(
	textStr string,
	textFont *text.GoTextFaceSource,
	fontSize float64,
	clr color.Color,
) *ebiten.Image {
	r, g, b, a := clr.RGBA()
	key := fmt.Sprintf(
		"%s:size:%.0f:clr:%d:%d:%d:%d:%s",
		textStr, fontSize, r, g, b, a, font.FontToStr(textFont),
	)
	if img, ok := textImages[key]; ok {
		return img
	}

	img := ebiten.NewImage(int(layout.CalcTextWidth(textStr, fontSize)*3), int(fontSize)+3)
	opts := &text.DrawOptions{}
	opts.ColorScale.ScaleWithColor(clr)
	textFace := text.GoTextFace{
		Source: textFont,
		Size:   fontSize,
	}
	text.Draw(img, textStr, &textFace, opts)

	textImages[key] = img
	return img
}
