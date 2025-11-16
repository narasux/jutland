package texture

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/resources/images/utils"
)

const (
	// MaxShipExplodeState 最大爆炸图片状态
	MaxShipExplodeState = 60

	// MaxPlaneExplodeState 最大飞机爆炸图片状态
	MaxPlaneExplodeState = 39
)

var shipExplodeImgMap, planeExplodeImgMap map[int]*ebiten.Image

func init() {
	var err error
	var img *ebiten.Image

	log.Println("loading explode image resources...")

	// 加载战舰爆炸图片
	shipExplodeImgMap = map[int]*ebiten.Image{}
	for hp := 0; hp < MaxShipExplodeState; hp++ {
		// TODO 尝试加了另外三种爆炸效果，但是不协调（太亮太短）先不使用，后面考虑作为飞机的爆炸效果
		imgPath := fmt.Sprintf("/textures/explode/explo_0_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		shipExplodeImgMap[hp] = img
	}

	// 加载飞机爆炸图片
	planeExplodeImgMap = map[int]*ebiten.Image{}
	for hp := 0; hp < MaxPlaneExplodeState; hp++ {
		imgPath := fmt.Sprintf("/textures/explode/explo_3_%d.png", hp)
		if img, err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
		planeExplodeImgMap[hp] = img
	}

	// FIXME 飞机爆炸图片需要压缩成 1/4
	planeExplodeImgMap = utils.GenZoomImages(planeExplodeImgMap, 4)

	log.Println("explode image resources loaded")
}

// GetShipExplode 根据残留的生命值，获取爆炸图片
func GetShipExplode(curHp float64) *ebiten.Image {
	return shipExplodeImgMap[int(math.Floor(curHp))]
}

// 根据残留的生命值，获取飞机爆炸图片
func GetPlaneExplode(curHp float64) *ebiten.Image {
	return planeExplodeImgMap[int(math.Floor(curHp))]
}
