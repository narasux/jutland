package texture

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// ArrowWhite 箭头图片（白）
	ArrowWhiteImg *ebiten.Image
	// SelectBox 选择框图片
	SelectBoxWhiteImg *ebiten.Image

	// GunEnableImg 火炮启用图片
	GunEnableImg *ebiten.Image
	// GunDisableImg 火炮禁用图片
	GunDisableImg *ebiten.Image

	// TorpedoEnableImg 鱼雷启用图片
	TorpedoEnableImg *ebiten.Image
	// TorpedoDisableImg 鱼雷禁用图片
	TorpedoDisableImg *ebiten.Image

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

	// 加载火炮启停用图标
	imgPath = "/textures/flag/gun_enable.png"
	if GunEnableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/gun_disable.png"
	if GunDisableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载鱼雷启停用图标
	imgPath = "/textures/flag/torpedo_enable.png"
	if TorpedoEnableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/flag/torpedo_disable.png"
	if TorpedoDisableImg, err = loader.LoadImage(imgPath); err != nil {
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
