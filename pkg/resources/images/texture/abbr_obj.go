package texture

import (
	"log"

	"github.com/samber/lo"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/loader"
)

var (
	// AbbrShipLightImg 轻型战舰缩略图（< 1w ton）
	AbbrShipLightImg *ebiten.Image
	// AbbrShipMediumImg 中型战舰缩略图（1w-3w ton)
	AbbrShipMediumImg *ebiten.Image
	// AbbrShipHeavyImg 重型战舰缩略图（> 3w ton）
	AbbrShipHeavyImg *ebiten.Image

	// AbbrEnemyLightImg 轻型战舰缩略图
	AbbrEnemyLightImg *ebiten.Image
	// AbbrEnemyMediumImg 中型战舰缩略图
	AbbrEnemyMediumImg *ebiten.Image
	// AbbrEnemyHeavyImg 重型战舰缩略图
	AbbrEnemyHeavyImg *ebiten.Image
)

func init() {
	var err error

	log.Println("loading abbreviation object image resources...")

	imgPath := "/textures/abbr_obj/light.png"
	if AbbrShipLightImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/medium.png"
	if AbbrShipMediumImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/heavy.png"
	if AbbrShipHeavyImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_light.png"
	if AbbrEnemyLightImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_medium.png"
	if AbbrEnemyMediumImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_heavy.png"
	if AbbrEnemyHeavyImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("abbreviation object image resources loaded")
}

// GetAbbrShipImg 获取缩略图
func GetAbbrShipImg(ton float64, isEnemy bool) *ebiten.Image {
	if ton < 10000 {
		return lo.Ternary(isEnemy, AbbrEnemyLightImg, AbbrShipLightImg)
	} else if ton < 30000 {
		return lo.Ternary(isEnemy, AbbrEnemyMediumImg, AbbrShipMediumImg)
	} else {
		return lo.Ternary(isEnemy, AbbrEnemyHeavyImg, AbbrShipHeavyImg)
	}
}
