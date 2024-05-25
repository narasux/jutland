package mapblock

import (
	"crypto/sha256"
	"fmt"
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
		imgName := fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, i)
		imgPath := fmt.Sprintf("/blocks/%s.png", imgName)
		if blocks[imgName], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}

	// 深海地图块（浅海）
	for i := 0; i < deepSeaBlockCount; i++ {
		imgName := fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, i)
		imgPath := fmt.Sprintf("/blocks/%s.png", imgName)
		if blocks[imgName], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}

	// 陆地地图块
	for i := 0; i < landBlockCount; i++ {
		imgName := fmt.Sprintf("land_%d_%d", constants.MapBlockSize, i)
		imgPath := fmt.Sprintf("/blocks/%s.png", imgName)
		if blocks[imgName], err = loader.LoadImage(imgPath); err != nil {
			log.Fatalf("missing %s: %s", imgPath, err)
		}
	}

	// 空白地图块
	imgPath := "/blocks/white.png"
	if blocks["blank"], err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("map block image resources loaded")
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
		index := int(hash[0]) % landBlockCount
		return blocks[fmt.Sprintf("land_%d_%d", constants.MapBlockSize, index)]
	}
	return blocks["blank"]
}
