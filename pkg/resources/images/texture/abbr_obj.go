package texture

import (
	"log"

	"github.com/samber/lo"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

var (
	// AbbrReinforcePoint 增援点
	AbbrReinforcePoint *ebiten.Image
	// AbbrSelectedReinforcePoint 选中的增援点
	AbbrSelectedReinforcePoint *ebiten.Image
	// AbbrOilPlatform 油井
	AbbrOilPlatform *ebiten.Image
	// AbbrEnemyReinforcePoint 敌方增援点
	AbbrEnemyReinforcePoint *ebiten.Image

	// AbbrPlane 己方战机
	AbbrPlane = ebutil.NewImageWithColor(1, 2, colorx.Green)
	// AbbrEnemyPlane 敌方战机
	AbbrEnemyPlane = ebutil.NewImageWithColor(1, 2, colorx.Red)
)

func init() {
	var err error

	log.Println("loading abbreviation object image resources...")

	imgPath := "/textures/abbr_obj/reinforce_point.png"
	if AbbrReinforcePoint, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/selected_reinforce_point.png"
	if AbbrSelectedReinforcePoint, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/oil_platform.png"
	if AbbrOilPlatform, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/textures/abbr_obj/enemy_reinforce_point.png"
	if AbbrEnemyReinforcePoint, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("abbreviation object image resources loaded")
}

// GetAbbrPlane 获取缩略图
func GetAbbrPlane(isEnemy bool) *ebiten.Image {
	return lo.Ternary(isEnemy, AbbrEnemyPlane, AbbrPlane)
}
