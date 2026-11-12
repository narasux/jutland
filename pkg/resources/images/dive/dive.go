package dive

import (
	"fmt"
	"log"
	"path"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
)

// State 潜艇深度状态图标。
type State string

const (
	// Surface 水面
	Surface State = "surface"
	// Submerged 水下
	Submerged State = "submerged"
	// Diving 下潜中
	Diving State = "diving"
	// Surfacing 上浮中
	Surfacing State = "surfacing"
)

var (
	// imagesByZoom 按资源倍率缓存启动期生成的 16/32/64px 绿色图标。
	imagesByZoom = map[int]map[State]*ebiten.Image{
		1: {},
		2: {},
		4: {},
	}
	// underwaterImagesByZoom 同状态的青色图标，用于下潜填充和水下稳定态。
	underwaterImagesByZoom = map[int]map[State]*ebiten.Image{
		1: {},
		2: {},
		4: {},
	}
	// transitionImagesByZoom 同状态的金色图标，用作上浮/下潜的底图。
	transitionImagesByZoom = map[int]map[State]*ebiten.Image{
		1: {},
		2: {},
		4: {},
	}
)

func init() {
	log.Println("loading dive state icon resources...")

	genZoomImages(loadSources("dive"), imagesByZoom)
	genZoomImages(loadSources("dive/underwater", Diving, Submerged), underwaterImagesByZoom)
	genZoomImages(loadSources("dive/transition", Diving, Surfacing), transitionImagesByZoom)

	log.Println("dive state icon resources loaded")
}

// loadSources 读取目录下指定状态的 128px 源图；不传状态时读取全部四种。
func loadSources(dir string, states ...State) map[State]*ebiten.Image {
	if len(states) == 0 {
		states = []State{Surface, Submerged, Diving, Surfacing}
	}
	source := map[State]*ebiten.Image{}
	for _, state := range states {
		relPath := path.Join("/", dir, fmt.Sprintf("%s.png", state))
		img, err := loader.LoadImage(relPath)
		if err != nil {
			log.Fatalf("missing %s: %s", filepath.Join(config.ImgResBaseDir, relPath), err)
		}
		source[state] = img
	}
	return source
}

// genZoomImages 逐级减半生成三档缓存，避免细圆环直接从 128px 缩到 16px 时丢线。
func genZoomImages(source map[State]*ebiten.Image, target map[int]map[State]*ebiten.Image) {
	genSizedImages(source, target[4], 64)
	genSizedImages(target[4], target[2], 32)
	genSizedImages(target[2], target[1], 16)
}

// genSizedImages 把源图等比缩放并居中到固定尺寸缓存。
// 缩放只在资源初始化时执行，绘制热路径直接复用对应 zoom 的缓存图片。
func genSizedImages(source, target map[State]*ebiten.Image, size int) {
	for state, img := range source {
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		width := img.Bounds().Dx()
		height := img.Bounds().Dy()
		scale := min(float64(size)/float64(width), float64(size)/float64(height))
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(
			(float64(size)-float64(width)*scale)/2,
			(float64(size)-float64(height)*scale)/2,
		)

		zoomImg := ebiten.NewImage(size, size)
		zoomImg.DrawImage(img, opts)
		target[state] = zoomImg
	}
}

// Get 获取深度状态图标；zoom 档位与武器图标一致（1→16px、2→32px、4→64px）。
func Get(state State, zoom int) *ebiten.Image {
	return get(imagesByZoom, state, zoom)
}

// GetUnderwater 获取同状态的青色图标，用于下潜填充和水下稳定态。
func GetUnderwater(state State, zoom int) *ebiten.Image {
	return get(underwaterImagesByZoom, state, zoom)
}

// GetTransition 获取同状态的金色图标，用作上浮/下潜的底图。
func GetTransition(state State, zoom int) *ebiten.Image {
	return get(transitionImagesByZoom, state, zoom)
}

func get(imagesByZoom map[int]map[State]*ebiten.Image, state State, zoom int) *ebiten.Image {
	imgs, ok := imagesByZoom[zoom]
	if !ok {
		imgs = imagesByZoom[4]
	}
	if img, ok := imgs[state]; ok {
		return img
	}
	return imagesByZoom[4][state]
}
