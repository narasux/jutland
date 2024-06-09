package mapblock

import (
	"crypto/sha256"
	"fmt"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/loader"
)

const (
	// 海洋地图块数量
	seaBlockCount = 7
	// 深海地图块数量
	deepSeaBlockCount = 3
	// 陆地地图块数量
	landBlockCount = 5
)

var blocks map[string]*ebiten.Image

func init() {
	var err error

	log.Println("loading map block image resources...")

	blocks = make(map[string]*ebiten.Image)

	// 海洋地图块（浅海）
	for i := 0; i < seaBlockCount; i++ {
		imgName := fmt.Sprintf("%d_%d", constants.MapBlockSize, i)
		imgPath := fmt.Sprintf("/map/blocks/sea/%s.png", imgName)
		if blocks["sea_"+imgName], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}

	// 深海地图块（浅海）
	for i := 0; i < deepSeaBlockCount; i++ {
		imgName := fmt.Sprintf("%d_%d", constants.MapBlockSize, i)
		imgPath := fmt.Sprintf("/map/blocks/deep_sea/%s.png", imgName)
		if blocks["deep_sea_"+imgName], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}

	// 空白地图块
	imgPath := "/map/blocks/common/white.png"
	if blocks["blank"], err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("map block image resources loaded")
}

var sceneBlockMap map[string]*ebiten.Image

// LoadMapSceneRes 加载地图场景资源
func LoadMapSceneRes(mission string) {
	sceneBlockMap = map[string]*ebiten.Image{}

	imgPath := fmt.Sprintf("/map/scenes/%s.png", mission)
	missionImg, err := loader.LoadImage(imgPath)
	if err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	blockSize := constants.MapBlockSize
	w, h := missionImg.Bounds().Dx()/blockSize, missionImg.Bounds().Dy()/blockSize
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			topLeftX, topLeftY := x*blockSize, y*blockSize
			cropRect := image.Rect(topLeftX, topLeftY, topLeftX+blockSize, topLeftY+blockSize)

			key := fmt.Sprintf("%d:%d", x, y)
			sceneBlockMap[key] = missionImg.SubImage(cropRect).(*ebiten.Image)
		}
	}
}

// GetByCharAndPos 根据指定字符 & 坐标，获取地图块资源
func GetByCharAndPos(c rune, x, y int) *ebiten.Image {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", x, y)))
	// 字符映射关系：. 浅海 o 深海 # 陆地
	if c == '.' {
		index := int(hash[0]) % seaBlockCount
		return blocks[fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
	}
	if c == 'o' {
		index := int(hash[0]) % deepSeaBlockCount
		return blocks[fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, index)]
	}
	if c == '#' {
		key := fmt.Sprintf("%d:%d", x, y)
		if img, ok := sceneBlockMap[key]; ok {
			return img
		}
	}
	return blocks["blank"]
}
