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
	"github.com/narasux/jutland/pkg/resources/mapcfg"
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
	// S 符号块（调试用）
	imgPath = "/map/blocks/common/char_s.png"
	if blocks["char_s"], err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}
	// C 符号块（调试用）
	imgPath = "/map/blocks/common/char_c.png"
	if blocks["char_c"], err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)
	}

	log.Println("map block image resources loaded")
}

type sceneBlockCache struct {
	mapName string
	data    map[string]*ebiten.Image
}

// SceneBlockCache 场景地图块缓存
var SceneBlockCache = sceneBlockCache{}

// Init 加载地图贴图数据
// 注：不要使用 ebiten.Image.SubImage() 来裁剪图片，有性能问题
func (c *sceneBlockCache) Init(cfg *mapcfg.MapCfg) error {
	// 避免重复加载
	if c.mapName == cfg.Name {
		log.Println("reuse scene blocks cache of map", cfg.Name)
		return nil
	}
	c.mapName = cfg.Name
	// 丢弃上一个关卡的地图贴图数据
	c.data = map[string]*ebiten.Image{}

	imgPath := fmt.Sprintf("/map/abbrs/%s.png", cfg.Name)
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
		return errors.New("mission map image isn't image.NRGBA type")
	}

	blockSize := missionImg.Bounds().Dx() / cfg.Width
	for x := 0; x < cfg.Width; x++ {
		for y := 0; y < cfg.Height; y++ {
			// 纯海洋（不含浅滩）不需要贴图，可以跳过
			chr := cfg.Map.Get(x, y)
			if chr == mapcfg.ChrSea || chr == mapcfg.ChrDeepSea {
				continue
			}
			topLeftX, topLeftY := x*blockSize, y*blockSize
			cropRect := image.Rect(topLeftX, topLeftY, topLeftX+blockSize, topLeftY+blockSize)
			subImg := missionImg.SubImage(cropRect)
			// 计算缩放比例
			scaleX := float64(constants.MapBlockSize) / float64(blockSize)
			scaleY := float64(constants.MapBlockSize) / float64(blockSize)
			// 生成地图块
			blockImg := ebiten.NewImage(constants.MapBlockSize, constants.MapBlockSize)
			opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			opts.GeoM.Scale(scaleX, scaleY)
			blockImg.DrawImage(ebiten.NewImageFromImage(subImg), opts)

			c.data[c.genKey(x, y)] = blockImg
		}
	}
	log.Printf("mission %s map scene blocks loaded, total size: %d\n", cfg.Name, len(c.data))
	return nil
}

// Get 根据坐标获取地图块
func (c *sceneBlockCache) Get(x, y int) *ebiten.Image {
	return c.data[c.genKey(x, y)]
}

// 生成缓存键
func (c *sceneBlockCache) genKey(x, y int) string {
	return fmt.Sprintf("%d:%d", x, y)
}

// GetByCharAndPos 根据指定字符 & 坐标，获取地图块资源
func GetByCharAndPos(c rune, x, y int) []*ebiten.Image {
	hash := md5.Sum([]byte(fmt.Sprintf("%d:%d", x, y)))

	posBlocks := []*ebiten.Image{}
	// 字符映射关系：. 浅海 o 深海 # 陆地
	switch c {
	case mapcfg.ChrSea:
		index := int(hash[0]) % seaBlockCount
		img := blocks[fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case mapcfg.ChrDeepSea:
		index := int(hash[0]) % deepSeaBlockCount
		img := blocks[fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case mapcfg.ChrLand:
		posBlocks = append(posBlocks, SceneBlockCache.Get(x, y))
	case mapcfg.ChrShallow:
		fallthrough
	case mapcfg.ChrCoast:
		// 浅滩/海岸需要现有海洋贴图，再贴陆地/沙滩贴图
		index := int(hash[0]) % seaBlockCount
		img := blocks[fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img, SceneBlockCache.Get(x, y))
		// 调试地图浅海/海岸用（人工标记法 orz）
		// posBlocks = append(posBlocks, blocks[fmt.Sprintf("char_%s", strings.ToLower(string(c)))])
	}

	return posBlocks
}
