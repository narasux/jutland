package mapblock

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pkg/errors"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
)

const (
	// 海洋地图块数量
	seaBlockCount = 7
	// 深海地图块数量
	deepSeaBlockCount = 3
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

var sceneBlockMap map[int]map[int]*ebiten.Image

// LoadMapSceneBlocks 加载地图贴图数据
// 注：不要使用 ebiten.Image.SubImage() 来裁剪图片，有性能问题
func LoadMapSceneBlocks(mission string) error {
	// 丢弃上一个关卡的地图贴图数据
	sceneBlockMap = map[int]map[int]*ebiten.Image{}

	imgPath := fmt.Sprintf("/map/scenes/%s.png", mission)
	imgData, err := os.ReadFile(config.ImgResBaseDir + imgPath)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return err
	}
	missionImg, ok := img.(*image.NRGBA)
	if !ok {
		return errors.New("mission image isn't image.NRGBA type")
	}

	blockSize := constants.MapBlockSize
	w, h := (missionImg.Bounds().Dx()+2)/blockSize, (missionImg.Bounds().Dy()+2)/blockSize
	for x := 0; x < w; x++ {
		mp := map[int]*ebiten.Image{}
		for y := 0; y < h; y++ {
			topLeftX, topLeftY := x*blockSize, y*blockSize
			cropRect := image.Rect(topLeftX, topLeftY, topLeftX+blockSize, topLeftY+blockSize)
			mp[y] = ebiten.NewImageFromImage(missionImg.SubImage(cropRect))
		}
		sceneBlockMap[x] = mp
	}
	log.Println("mission %s map scene blocks loaded", mission)
	return nil
}

// GetByCharAndPos 根据指定字符 & 坐标，获取地图块资源
func GetByCharAndPos(c rune, x, y int) []*ebiten.Image {
	hash := md5.Sum([]byte(fmt.Sprintf("%d:%d", x, y)))

	posBlocks := []*ebiten.Image{}
	// 字符映射关系：. 浅海 o 深海 # 陆地
	switch c {
	case '.':
		index := int(hash[0]) % seaBlockCount
		img := blocks[fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case 'o':
		index := int(hash[0]) % deepSeaBlockCount
		img := blocks[fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case 'L':
		posBlocks = append(posBlocks, sceneBlockMap[x][y])
	case 'S':
		// 浅滩/沙滩需要现有海洋贴图，再贴陆地/沙滩贴图
		index := int(hash[0]) % seaBlockCount
		img := blocks[fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img, sceneBlockMap[x][y])
	}

	return posBlocks
}
