package background

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// GameStartImg 游戏开始
	GameStartImg *ebiten.Image
	// GameMenuImg 游戏菜单
	GameMenuImg *ebiten.Image
	// GameEndImg 游戏结束
	GameEndImg *ebiten.Image

	// MissionsMapImg 任务地图
	MissionsMapImg *ebiten.Image
	// MissionStartImg 任务开始
	MissionStartImg *ebiten.Image
	// MissionSuccessImg 任务成功
	MissionSuccessImg *ebiten.Image
	// MissionFailedImg 任务失败
	MissionFailedImg *ebiten.Image
)

func init() {
	var err error

	log.Println("loading background resources...")

	imgPath := "/backgrounds/game_start.png"
	if GameStartImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/game_menu.png"
	if GameMenuImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/game_end.png"
	if GameEndImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// TODO 替换该资源
	imgPath = "/backgrounds/missions_map.png"
	if MissionsMapImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_start.png"
	if MissionStartImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_success.png"
	if MissionSuccessImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_failed.png"
	if MissionFailedImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("background resources loaded")
}
