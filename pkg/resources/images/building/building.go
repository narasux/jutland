package building

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// ReinforcePoints 增援点
	ReinforcePoint *ebiten.Image

	// EnemyReinforcePoint 敌方增援点
	EnemyReinforcePoint *ebiten.Image

	// OilPlatform 油井
	OilPlatform *ebiten.Image
)

func init() {
	var err error

	log.Println("loading building image resources...")

	imgPath := "/buildings/reinforce_point.png"
	if ReinforcePoint, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/buildings/enemy_reinforce_point.png"
	if EnemyReinforcePoint, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/buildings/oil_platform.png"
	if OilPlatform, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("texture building resources loaded")
}
