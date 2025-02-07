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
	log.Println("loading plane image resources...")

	planeTypes := []string{
		"fighter",
		"dive_bomber",
		"torpedo_bomber",
	}

	loadPlaneImages(planeZoom10ImgMap, planeTypes)
	genZoomImages(planeZoom10ImgMap, planeZoom8ImgMap, 1.25)
	genZoomImages(planeZoom10ImgMap, planeZoom4ImgMap, 2.5)
	genZoomImages(planeZoom10ImgMap, planeZoom2ImgMap, 5)
	genZoomImages(planeZoom10ImgMap, planeZoom1ImgMap, 10)

	log.Println("plane image resources loaded")
}

func loadPlaneImages(cache map[string]*ebiten.Image, planeTypes []string) {
	for _, planeType := range planeTypes {
		entries, err := os.ReadDir(filepath.Join(config.ImgResBaseDir, "planes", planeType))
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			// 限制原始图片必须是 PNG 格式
			if !strings.HasSuffix(entry.Name(), ".png") {
				continue
			}
			imgPath := fmt.Sprintf("/planes/%s/%s", planeType, entry.Name())
			shipImg, loadImgErr := loader.LoadImage(imgPath)
			if loadImgErr != nil {
				log.Fatalf("missing %s: %s", imgPath, loadImgErr)
			}
			cache[strings.TrimSuffix(entry.Name(), ".png")] = shipImg
		}
	}
}

func genZoomImages(
	source map[string]*ebiten.Image, target map[string]*ebiten.Image, arcZoom float64,
) {
	for name, img := range source {
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/arcZoom, 1/arcZoom)

		width := int(float64(img.Bounds().Dx())/arcZoom) + 1
		height := int(float64(img.Bounds().Dy())/arcZoom) + 1
		zoomImg := ebiten.NewImage(width, height)
		zoomImg.DrawImage(img, opts)
		target[name] = zoomImg
	}
}

var (
	planeZoom10ImgMap = map[string]*ebiten.Image{}
	planeZoom8ImgMap  = map[string]*ebiten.Image{}
	planeZoom4ImgMap  = map[string]*ebiten.Image{}
	planeZoom2ImgMap  = map[string]*ebiten.Image{}
	planeZoom1ImgMap  = map[string]*ebiten.Image{}
)

// Get 获取战机（顶部）图片
func Get(name string, zoom int) *ebiten.Image {
	if zoom == 1 {
		return planeZoom1ImgMap[name]
	} else if zoom == 2 {
		return planeZoom2ImgMap[name]
	} else if zoom == 4 {
		return planeZoom4ImgMap[name]
	} else if zoom == 8 {
		return planeZoom8ImgMap[name]
	}
	return planeZoom10ImgMap[name]
}
