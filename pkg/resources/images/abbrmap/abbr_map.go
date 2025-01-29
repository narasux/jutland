package abbrmap

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// Background 地图缩略图背景
var Background *ebiten.Image

// abbrMapImgMap 地图缩略图
var abbrMapImgMap map[string]*ebiten.Image

func init() {
	var err error

	log.Println("loading background image resources...")

	imgPath := "/map/abbrs/background.png"
	if Background, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	abbrMapImgMap = make(map[string]*ebiten.Image)

	for _, mapSource := range mapcfg.GetAllMapSources() {
		imgPath = fmt.Sprintf("/map/abbrs/%s.png", mapSource)
		if abbrMapImgMap[mapSource], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}
	log.Println("background image resources loaded")
}

// Get 获取地图缩略图
func Get(mapName string) *ebiten.Image {
	return abbrMapImgMap[mapName]
}
