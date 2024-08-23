package texture

import (
	"log"

	"github.com/samber/lo"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/loader"
)

var (
	// AbbrShipLight 轻型战舰缩略图（< 1w ton）
	AbbrShipLight *ebiten.Image
	// AbbrShipMedium 中型战舰缩略图（1w-3w ton)
	AbbrShipMedium *ebiten.Image
	// AbbrShipHeavy 重型战舰缩略图（> 3w ton）
	AbbrShipHeavy *ebiten.Image

	// AbbrEnemyLight 轻型战舰缩略图
	AbbrEnemyLight *ebiten.Image
	// AbbrEnemyMedium 中型战舰缩略图
	AbbrEnemyMedium *ebiten.Image
	// AbbrEnemyHeavy 重型战舰缩略图
	AbbrEnemyHeavy *ebiten.Image
)

func init() {
	var err error

	log.Println("loading abbreviation object image resources...")

	imgPath := "/textures/abbr_obj/light.png"
	if AbbrShipLight, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/medium.png"
	if AbbrShipMedium, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/heavy.png"
	if AbbrShipHeavy, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_light.png"
	if AbbrEnemyLight, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_medium.png"
	if AbbrEnemyMedium, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_heavy.png"
	if AbbrEnemyHeavy, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("abbreviation object image resources loaded")
}

// GetAbbrShip 获取缩略图
func GetAbbrShip(ton float64, isEnemy bool) *ebiten.Image {
	if ton < 10000 {
		return lo.Ternary(isEnemy, AbbrEnemyLight, AbbrShipLight)
	} else if ton < 30000 {
		return lo.Ternary(isEnemy, AbbrEnemyMedium, AbbrShipMedium)
	} else {
		return lo.Ternary(isEnemy, AbbrEnemyHeavy, AbbrShipHeavy)
	}
}
