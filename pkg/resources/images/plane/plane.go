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
	"github.com/narasux/jutland/pkg/resources/images/utils"
)

// 战机图片（各类缩放尺寸）
var planeZoom10ImgMap, planeZoom8ImgMap, planeZoom4ImgMap, planeZoom2ImgMap, planeZoom1ImgMap map[string]*ebiten.Image

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

func init() {
	log.Println("loading plane image resources...")

	planeTypes := []string{
		"fighter",
		"dive_bomber",
		"torpedo_bomber",
	}

	planeZoom10ImgMap = map[string]*ebiten.Image{}
	loadPlaneImages(planeZoom10ImgMap, planeTypes)
	planeZoom8ImgMap = utils.GenZoomImages(planeZoom10ImgMap, 1.25)
	planeZoom4ImgMap = utils.GenZoomImages(planeZoom10ImgMap, 2.5)
	planeZoom2ImgMap = utils.GenZoomImages(planeZoom10ImgMap, 5)
	planeZoom1ImgMap = utils.GenZoomImages(planeZoom10ImgMap, 10)

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
