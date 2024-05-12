package texture

import (
	"fmt"
	"log"
	"math"

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
)

var hpImgMap = map[int]*ebiten.Image{}

func init() {
	var err error

	log.Println("loading texture image resources...")

	imgPath := "/textures/arrow_white.png"
	if ArrowWhiteImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/select_box_white.png"
	if SelectBoxWhiteImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载生命值图片
	for hp := 0; hp <= 100; hp += 10 {
		imgPath = fmt.Sprintf("/textures/hp/hp_%d.png", hp)
		var img *ebiten.Image
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		hpImgMap[hp/10] = img
	}

	// 加载火炮启停用图标
	imgPath = "/textures/gun_enable.png"
	if GunEnableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/gun_disable.png"
	if GunDisableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// 加载鱼雷启停用图标
	imgPath = "/textures/torpedo_enable.png"
	if TorpedoEnableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/torpedo_disable.png"
	if TorpedoDisableImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("texture image resources loaded")
}

// GetHpImg 获取生命值图片
func GetHpImg(curHp, maxHp int) *ebiten.Image {
	return hpImgMap[int(math.Floor(float64(curHp)/float64(maxHp)*10))]
}