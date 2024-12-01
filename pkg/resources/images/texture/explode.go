package texture

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/loader"
)

// MaxShipExplodeState 最大爆炸图片状态
const MaxShipExplodeState = 60

var shipExplodeImgMap = map[int]*ebiten.Image{}

func init() {
	var err error
	var img *ebiten.Image

	log.Println("loading explode image resources...")

	// 加载爆炸图片
	for hp := 0; hp < MaxShipExplodeState; hp++ {
		// TODO 尝试加了另外三种爆炸效果，但是不协调（太亮太短）先不使用，后面考虑作为飞机的爆炸效果
		imgPath := fmt.Sprintf("/textures/explode/explo_0_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		shipExplodeImgMap[hp] = img
	}

	log.Println("explode image resources loaded")
}

// GetShipExplode 根据残留的生命值，获取爆炸图片
func GetShipExplode(curHp float64) *ebiten.Image {
	return shipExplodeImgMap[int(math.Floor(curHp))]
}
