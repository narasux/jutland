package ship

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/resources/images/utils"
)

// 战机图片（各类缩放尺寸）
var planeZoomImgMaps map[int]map[string]*ebiten.Image
var planeOriginalImgMap map[string]*ebiten.Image
var planeDisplayScaleMap map[string]float64

type planeImageSize struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
}

// Get 获取战机（顶部）图片
func Get(name string, zoom int) *ebiten.Image {
	if cache, ok := planeZoomImgMaps[zoom]; ok {
		return cache[name]
	}
	return planeZoomImgMaps[10][name]
}

// GetOriginal 获取未归一化尺寸的原始战机图片，用于图鉴等高清展示场景。
func GetOriginal(name string) *ebiten.Image {
	return planeOriginalImgMap[name]
}

// GetDisplayScale 返回原始战机图片缩放到配置尺寸（10px/米）的比例。
func GetDisplayScale(name string) float64 {
	if scale, ok := planeDisplayScaleMap[name]; ok && scale > 0 {
		return scale
	}
	return 1
}

func init() {
	log.Println("loading plane image resources...")

	planeTypes := []string{
		"fighter",
		"dive_bomber",
		"torpedo_bomber",
	}

	planeOriginalImgMap = map[string]*ebiten.Image{}
	planeDisplayScaleMap = map[string]float64{}
	basePlaneImgMap := map[string]*ebiten.Image{}
	loadPlaneImages(basePlaneImgMap, planeTypes)
	planeZoomImgMaps = map[int]map[string]*ebiten.Image{
		10: basePlaneImgMap,
		8:  utils.GenZoomImages(basePlaneImgMap, 1.25),
		4:  utils.GenZoomImages(basePlaneImgMap, 2.5),
		2:  utils.GenZoomImages(basePlaneImgMap, 5),
		1:  utils.GenZoomImages(basePlaneImgMap, 10),
	}

	log.Println("plane image resources loaded")
}

func loadPlaneImages(cache map[string]*ebiten.Image, planeTypes []string) {
	sizeByName := loadConfiguredPlaneImageSizes()
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
			name := strings.TrimSuffix(entry.Name(), ".png")
			planeOriginalImgMap[name] = shipImg
			planeDisplayScaleMap[name] = configuredPlaneDisplayScale(name, shipImg, sizeByName)
			cache[name] = normalizePlaneImageSize(shipImg, planeDisplayScaleMap[name])
		}
	}
}

func loadConfiguredPlaneImageSizes() map[string]planeImageSize {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "planes.json5"))
	if err != nil {
		log.Fatal("failed to open planes.json5 for plane image sizing: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)
	var planes []struct {
		Name   string  `json:"name"`
		Length float64 `json:"length"`
		Width  float64 `json:"width"`
	}
	if err = json5.Unmarshal(bytes, &planes); err != nil {
		log.Fatal("failed to unmarshal planes.json5 for plane image sizing: ", err)
	}

	sizeByName := make(map[string]planeImageSize, len(planes))
	for _, plane := range planes {
		sizeByName[plane.Name] = planeImageSize{Length: plane.Length, Width: plane.Width}
	}
	return sizeByName
}

func configuredPlaneDisplayScale(name string, img *ebiten.Image, sizeByName map[string]planeImageSize) float64 {
	size, ok := sizeByName[name]
	if !ok || size.Length <= 0 || size.Width <= 0 {
		return 1
	}

	targetW := int(math.Round(size.Width * 10))
	targetH := int(math.Round(size.Length * 10))
	if targetW <= 0 || targetH <= 0 {
		return 1
	}

	return min(
		float64(targetW)/float64(img.Bounds().Dx()),
		float64(targetH)/float64(img.Bounds().Dy()),
	)
}

func normalizePlaneImageSize(img *ebiten.Image, displayScale float64) *ebiten.Image {
	if displayScale <= 0 || displayScale >= 1 {
		return img
	}

	targetW := int(math.Round(float64(img.Bounds().Dx()) * displayScale))
	targetH := int(math.Round(float64(img.Bounds().Dy()) * displayScale))
	if targetW <= 0 || targetH <= 0 {
		return img
	}

	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(displayScale, displayScale)
	normalized := ebiten.NewImage(targetW, targetH)
	normalized.DrawImage(img, opts)
	return normalized
}
