package background

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// GameStart 游戏开始
	GameStart *ebiten.Image
	// GameMenu 游戏菜单
	GameMenu *ebiten.Image
	// GameEnd 游戏结束
	GameEnd *ebiten.Image

	// MissionsMap 任务地图
	MissionsMap *ebiten.Image
	// MissionStart 任务开始
	MissionStart *ebiten.Image
	// MissionSuccess 任务成功
	MissionSuccess *ebiten.Image
	// MissionFailed 任务失败
	MissionFailed *ebiten.Image
	// MissionWindow 任务窗口（设置，配置等背景）
	MissionWindow *ebiten.Image
)

func init() {
	var err error

	log.Println("loading background image resources...")

	imgPath := "/backgrounds/game_start.png"
	if GameStart, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/game_menu.png"
	if GameMenu, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/game_end.png"
	if GameEnd, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	// TODO 替换该资源
	imgPath = "/backgrounds/missions_map.png"
	if MissionsMap, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_start.png"
	if MissionStart, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_success.png"
	if MissionSuccess, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_failed.png"
	if MissionFailed, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	imgPath = "/backgrounds/mission_window.png"
	if MissionWindow, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("background image resources loaded")
}
