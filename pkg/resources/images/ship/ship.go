package ship

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
)

func init() {
	log.Println("loading ship image resources...")

	// TODO 航母，鱼雷艇待支持
	shipTypes := []string{"battleship", "cruiser", "default", "destroyer"}

	loadShipImages(topShipZoom4ImgMap, shipTypes, "top")
	genZoomShipImages(topShipZoom4ImgMap, topShipZoom2ImgMap, 2)
	genZoomShipImages(topShipZoom4ImgMap, topShipZoom1ImgMap, 4)

	loadShipImages(sideShipZoom4ImgMap, shipTypes, "side")
	genZoomShipImages(sideShipZoom4ImgMap, sideShipZoom2ImgMap, 2)
	genZoomShipImages(sideShipZoom4ImgMap, sideShipZoom1ImgMap, 4)

	log.Println("ship image resources loaded")
}

func loadShipImages(cache map[string]*ebiten.Image, shipTypes []string, direction string) {
	for _, shipType := range shipTypes {
		entries, err := os.ReadDir(filepath.Join(config.ImgResBaseDir, "ships", direction, shipType))
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			// 限制原始图片必须是 PNG 格式
			if !strings.HasSuffix(entry.Name(), ".png") {
				continue
			}
			imgPath := fmt.Sprintf("/ships/%s/%s/%s", direction, shipType, entry.Name())
			shipImg, loadImgErr := loader.LoadImage(imgPath)
			if loadImgErr != nil {
				log.Fatalf("missing %s: %s", imgPath, loadImgErr)
			}
			cache[strings.TrimSuffix(entry.Name(), ".png")] = shipImg
		}
	}
}

func genZoomShipImages(
	source map[string]*ebiten.Image, target map[string]*ebiten.Image, arcZoom int,
) {
	for name, img := range source {
		// 小黄鸭 & 水滴比较特殊，不提供缩放，只有一个尺寸
		if name == "duck" || name == "waterdrop" {
			target[name] = img
			continue
		}
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/float64(arcZoom), 1/float64(arcZoom))

		zoomImg := ebiten.NewImage(img.Bounds().Dx()/arcZoom, img.Bounds().Dy()/arcZoom)
		zoomImg.DrawImage(img, opts)
		target[name] = zoomImg
	}
}

var (
	topShipZoom4ImgMap = map[string]*ebiten.Image{}
	topShipZoom2ImgMap = map[string]*ebiten.Image{}
	topShipZoom1ImgMap = map[string]*ebiten.Image{}
)

var (
	sideShipZoom4ImgMap = map[string]*ebiten.Image{}
	sideShipZoom2ImgMap = map[string]*ebiten.Image{}
	sideShipZoom1ImgMap = map[string]*ebiten.Image{}
)

// GetTop 获取战舰顶部图片
func GetTop(name string, zoom int) *ebiten.Image {
	if zoom == 1 {
		return topShipZoom1ImgMap[name]
	} else if zoom == 2 {
		return topShipZoom2ImgMap[name]
	}
	return topShipZoom4ImgMap[name]
}

// GetSide 获取战舰侧面图片
func GetSide(name string, zoom int) *ebiten.Image {
	if zoom == 1 {
		return sideShipZoom1ImgMap[name]
	} else if zoom == 2 {
		return sideShipZoom2ImgMap[name]
	}
	return sideShipZoom4ImgMap[name]
}
