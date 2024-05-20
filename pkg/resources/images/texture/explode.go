package texture

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/loader"
)

// MaxExplodeState 最大爆炸图片状态
const MaxExplodeState = 60

var explodeImgMap = map[int]*ebiten.Image{}

func init() {
	var err error
	var img *ebiten.Image

	log.Println("loading explode image resources...")

	// 加载爆炸图片
	for hp := 0; hp < MaxExplodeState; hp++ {
		imgPath := fmt.Sprintf("/textures/explode/explo_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		explodeImgMap[hp] = img
	}

	log.Println("explode image resources loaded")
}

// GetExplodeImg 根据残留的生命值，获取爆炸图片
func GetExplodeImg(curHp float64) *ebiten.Image {
	return explodeImgMap[int(math.Floor(curHp))]
}
