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

// NewComposite 按地图实际宽高比生成背景与地图素材的合成缩略图。
func NewComposite(mapSource string, mapWidth, mapHeight, targetHeight int) *ebiten.Image {
	targetWidth := targetHeight * mapWidth / mapHeight
	composite := ebiten.NewImage(targetWidth, targetHeight)

	drawScaled := func(img *ebiten.Image) {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(float64(targetWidth)/float64(w), float64(targetHeight)/float64(h))
		composite.DrawImage(img, opts)
	}
	drawScaled(Background)
	drawScaled(Get(mapSource))
	return composite
}
