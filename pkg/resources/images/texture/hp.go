package texture

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	hpImgMap      = map[int]*ebiten.Image{}
	enemyHpImgMap = map[int]*ebiten.Image{}
)

func init() {
	var err error
	var img *ebiten.Image

	log.Println("loading hp image resources...")

	// 加载生命值图片
	for hp := 0; hp <= 100; hp += 10 {
		imgPath := fmt.Sprintf("/textures/hp/hp_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		hpImgMap[hp/10] = img

		imgPath = fmt.Sprintf("/textures/hp/enemy_hp_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		enemyHpImgMap[hp/10] = img
	}

	log.Println("hp image resources loaded")
}

// GetHpImg 获取生命值图片
func GetHpImg(curHp, maxHp float64) *ebiten.Image {
	return hpImgMap[int(math.Floor(curHp/maxHp*10))]
}

// GetEnemyHpImg 获取敌人生命值图片
func GetEnemyHpImg(curHp, maxHp float64) *ebiten.Image {
	return enemyHpImgMap[int(math.Floor(curHp/maxHp*10))]
}
