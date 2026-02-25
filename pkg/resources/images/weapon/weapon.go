package weapon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
)

// WeaponType 武器类型
type WeaponType string

const (
	WeaponTypeMainGun         WeaponType = "main_gun"
	WeaponTypeSecondaryGun    WeaponType = "secondary_gun"
	WeaponTypeAntiAircraftGun WeaponType = "antiaircraft_gun"
	WeaponTypeTorpedo         WeaponType = "torpedo"
)

// WeaponStatus 武器状态
type WeaponStatus string

const (
	WeaponStatusLoaded    WeaponStatus = "loaded"
	WeaponStatusReloading WeaponStatus = "reloading"
	WeaponStatusDisabled  WeaponStatus = "disabled"
)

func init() {
	log.Println("loading weapon icon resources...")

	// 加载原始图标（放大4倍，用于高分辨率）
	loadWeaponIcons(weaponZoom4ImgMap)

	// 生成不同缩放级别的图标
	genZoomWeaponImages(weaponZoom4ImgMap, weaponZoom2ImgMap, 2)
	genZoomWeaponImages(weaponZoom4ImgMap, weaponZoom1ImgMap, 4)

	log.Println("weapon icon resources loaded")
}

// loadWeaponIcons 加载武器图标资源
func loadWeaponIcons(cache map[string]*ebiten.Image) {
	weapons := []WeaponType{
		WeaponTypeMainGun,
		WeaponTypeSecondaryGun,
		WeaponTypeAntiAircraftGun,
		WeaponTypeTorpedo,
	}

	statuses := []WeaponStatus{
		WeaponStatusLoaded,
		WeaponStatusReloading,
		WeaponStatusDisabled,
	}

	for _, weapon := range weapons {
		// 防空炮只有两种状态
		weaponStatuses := statuses
		if weapon == WeaponTypeAntiAircraftGun {
			weaponStatuses = []WeaponStatus{WeaponStatusLoaded, WeaponStatusDisabled}
		}

		for _, status := range weaponStatuses {
			filename := fmt.Sprintf("%s_%s.png", weapon, status)
			imgPath := filepath.Join(config.ImgResBaseDir, "weapons", filename)

			// 检查文件是否存在
			if _, err := os.Stat(imgPath); os.IsNotExist(err) {
				log.Printf("warning: weapon icon not found: %s", imgPath)
				continue
			}

			img, err := loader.LoadImage(filepath.Join("/weapons", filename))
			if err != nil {
				log.Printf("warning: failed to load weapon icon %s: %s", filename, err)
				continue
			}

			cache[genKey(weapon, status)] = img
		}
	}
}

// genZoomWeaponImages 生成缩放后的武器图标
func genZoomWeaponImages(source, target map[string]*ebiten.Image, arcZoom int) {
	for key, img := range source {
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/float64(arcZoom), 1/float64(arcZoom))

		zoomImg := ebiten.NewImage(img.Bounds().Dx()/arcZoom, img.Bounds().Dy()/arcZoom)
		zoomImg.DrawImage(img, opts)
		target[key] = zoomImg
	}
}

// genKey 生成缓存键
func genKey(weapon WeaponType, status WeaponStatus) string {
	return fmt.Sprintf("%s_%s", weapon, status)
}

var (
	weaponZoom4ImgMap = map[string]*ebiten.Image{}
	weaponZoom2ImgMap = map[string]*ebiten.Image{}
	weaponZoom1ImgMap = map[string]*ebiten.Image{}
)

// Get 获取武器图标
func Get(weapon WeaponType, status WeaponStatus, zoom int) *ebiten.Image {
	key := genKey(weapon, status)

	var imgMap map[string]*ebiten.Image
	switch zoom {
	case 1:
		imgMap = weaponZoom1ImgMap
	case 2:
		imgMap = weaponZoom2ImgMap
	default:
		imgMap = weaponZoom4ImgMap
	}

	img, exists := imgMap[key]
	if !exists {
		// 降级处理：尝试从其他缩放级别获取
		if img = weaponZoom4ImgMap[key]; img != nil {
			return img
		}
		if img = weaponZoom2ImgMap[key]; img != nil {
			return img
		}
		if img = weaponZoom1ImgMap[key]; img != nil {
			return img
		}
		// 如果都没有，返回 nil（调用者需要进行降级处理）
		return nil
	}

	return img
}
