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

var (
	blocks     map[string]*ebiten.Image
	zoomBlocks map[int]map[string]*ebiten.Image
)

// supportedZooms 是地图块缓存支持的主战场缩放档位。
var supportedZooms = []int{1, 2, 4, 8, 16}

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

	zoomBlocks = genZoomBlockMap(blocks)

	log.Println("map block image resources loaded")
}

type sceneBlockCache struct {
	mapName       string
	data          map[string]*ebiten.Image
	zoomData      map[int]map[string]*ebiten.Image
	prewarmJobs   []sceneBlockPrewarmJob
	prewarmQueued map[string]bool
}

type sceneBlockPrewarmJob struct {
	key  string
	zoom int
}

// DrawBlock 是地图块绘制层，Scale 用于缺少预热缓存时临时缩放原始场景块。
type DrawBlock struct {
	Image *ebiten.Image
	Scale float64
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
	c.zoomData = map[int]map[string]*ebiten.Image{}
	c.prewarmJobs = nil
	c.prewarmQueued = map[string]bool{}

	imgPath := fmt.Sprintf("/map/abbrs/%s.png", cfg.Source)
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

// GetZoom 根据坐标和缩放档位获取已缓存的场景地图块
func (c *sceneBlockCache) GetZoom(x, y int, zoom int) *ebiten.Image {
	zoom = normalizeZoom(zoom)

	zoomMap := c.zoomData[zoom]
	if zoomMap == nil {
		return nil
	}
	return zoomMap[c.genKey(x, y)]
}

// GetZoomDrawBlock 获取场景地图块绘制层；缓存未就绪时返回原始图和临时缩放比例。
func (c *sceneBlockCache) GetZoomDrawBlock(x, y int, zoom int) DrawBlock {
	zoom = normalizeZoom(zoom)
	if img := c.GetZoom(x, y, zoom); img != nil {
		return DrawBlock{Image: img, Scale: 1}
	}
	baseImg := c.Get(x, y)
	if baseImg == nil {
		return DrawBlock{}
	}
	return DrawBlock{
		Image: baseImg,
		Scale: float64(zoomedBlockSize(zoom)+1) / float64(constants.MapBlockSize),
	}
}

// SchedulePrewarmAround 把相机附近的场景地图块加入预热队列；已缓存或已排队的块会被跳过。
func (c *sceneBlockCache) SchedulePrewarmAround(minX, minY, width, height int, zooms []int, margin int) {
	for _, zoom := range zooms {
		zoom = normalizeZoom(zoom)
		zoomMap := c.zoomData[zoom]
		if zoomMap == nil {
			zoomMap = map[string]*ebiten.Image{}
			c.zoomData[zoom] = zoomMap
		}
		for x := minX - margin; x <= minX+width+margin; x++ {
			for y := minY - margin; y <= minY+height+margin; y++ {
				key := c.genKey(x, y)
				if c.data[key] == nil || zoomMap[key] != nil {
					continue
				}
				queueKey := c.prewarmQueueKey(key, zoom)
				if c.prewarmQueued[queueKey] {
					continue
				}
				c.prewarmQueued[queueKey] = true
				c.prewarmJobs = append(c.prewarmJobs, sceneBlockPrewarmJob{key: key, zoom: zoom})
			}
		}
	}
}

// HasMissingAround 判断相机附近是否仍有指定 zoom 的场景地图块缓存缺口。
func (c *sceneBlockCache) HasMissingAround(minX, minY, width, height int, zoom int, margin int) bool {
	zoom = normalizeZoom(zoom)
	zoomMap := c.zoomData[zoom]
	if zoomMap == nil {
		return true
	}
	for x := minX - margin; x <= minX+width+margin; x++ {
		for y := minY - margin; y <= minY+height+margin; y++ {
			key := c.genKey(x, y)
			if c.data[key] != nil && zoomMap[key] == nil {
				return true
			}
		}
	}
	return false
}

// ResetPrewarmQueue 清空待预热队列，用于缩放变更后丢弃旧优先级任务。
func (c *sceneBlockCache) ResetPrewarmQueue() {
	c.prewarmJobs = nil
	c.prewarmQueued = map[string]bool{}
}

// StepPrewarm 按预算生成场景地图块缩放缓存，避免在单帧集中创建过多图片。
func (c *sceneBlockCache) StepPrewarm(budget int) int {
	processed := 0
	for budget > 0 && len(c.prewarmJobs) > 0 {
		job := c.prewarmJobs[0]
		c.prewarmJobs = c.prewarmJobs[1:]
		delete(c.prewarmQueued, c.prewarmQueueKey(job.key, job.zoom))

		baseImg := c.data[job.key]
		if baseImg == nil {
			continue
		}
		zoomMap := c.zoomData[job.zoom]
		if zoomMap == nil {
			zoomMap = map[string]*ebiten.Image{}
			c.zoomData[job.zoom] = zoomMap
		}
		if zoomMap[job.key] != nil {
			continue
		}
		zoomMap[job.key] = genZoomBlock(baseImg, job.zoom)
		budget--
		processed++
	}
	return processed
}

func (c *sceneBlockCache) prewarmQueueKey(key string, zoom int) string {
	return fmt.Sprintf("%d:%s", zoom, key)
}

// 生成缓存键
func (c *sceneBlockCache) genKey(x, y int) string {
	return fmt.Sprintf("%d:%d", x, y)
}

// GetByCharAndPos 根据指定字符 & 坐标，获取地图块资源
func GetByCharAndPos(c rune, x, y int) []*ebiten.Image {
	return GetByCharAndPosZoom(c, x, y, 4)
}

// GetByCharAndPosZoom 根据指定字符、坐标和缩放档位获取地图块资源
func GetByCharAndPosZoom(c rune, x, y int, zoom int) []*ebiten.Image {
	hash := md5.Sum([]byte(fmt.Sprintf("%d:%d", x, y)))
	zoom = normalizeZoom(zoom)

	posBlocks := []*ebiten.Image{}
	// 字符映射关系：. 浅海 o 深海 # 陆地
	switch c {
	case mapcfg.ChrSea:
		index := int(hash[0]) % seaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case mapcfg.ChrDeepSea:
		index := int(hash[0]) % deepSeaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img)
	case mapcfg.ChrLand:
		posBlocks = append(posBlocks, SceneBlockCache.GetZoom(x, y, zoom))
	case mapcfg.ChrShallow:
		fallthrough
	case mapcfg.ChrCoast:
		// 浅滩/海岸需要现有海洋贴图，再贴陆地/沙滩贴图
		index := int(hash[0]) % seaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, img, SceneBlockCache.GetZoom(x, y, zoom))
		// 调试地图浅海/海岸用（人工标记法 orz）
		// posBlocks = append(posBlocks, blocks[fmt.Sprintf("char_%s", strings.ToLower(string(c)))])
	}

	return posBlocks
}

// GetDrawBlocksByCharAndPosZoom 根据指定字符、坐标和缩放档位获取可绘制地图块层。
func GetDrawBlocksByCharAndPosZoom(c rune, x, y int, zoom int) []DrawBlock {
	hash := md5.Sum([]byte(fmt.Sprintf("%d:%d", x, y)))
	zoom = normalizeZoom(zoom)

	posBlocks := []DrawBlock{}
	switch c {
	case mapcfg.ChrSea:
		index := int(hash[0]) % seaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, DrawBlock{Image: img, Scale: 1})
	case mapcfg.ChrDeepSea:
		index := int(hash[0]) % deepSeaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("deep_sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(posBlocks, DrawBlock{Image: img, Scale: 1})
	case mapcfg.ChrLand:
		posBlocks = append(posBlocks, SceneBlockCache.GetZoomDrawBlock(x, y, zoom))
	case mapcfg.ChrShallow:
		fallthrough
	case mapcfg.ChrCoast:
		index := int(hash[0]) % seaBlockCount
		img := zoomBlocks[zoom][fmt.Sprintf("sea_%d_%d", constants.MapBlockSize, index)]
		posBlocks = append(
			posBlocks,
			DrawBlock{Image: img, Scale: 1},
			SceneBlockCache.GetZoomDrawBlock(x, y, zoom),
		)
	}

	return posBlocks
}

// genZoomBlockMap 为一组地图块生成所有支持缩放档位的缓存。
// 它只用于通用海面等少量资源；关卡陆地块走按需缓存以缩短加载时间
func genZoomBlockMap(source map[string]*ebiten.Image) map[int]map[string]*ebiten.Image {
	target := make(map[int]map[string]*ebiten.Image, len(supportedZooms))
	for _, zoom := range supportedZooms {
		zoomMap := make(map[string]*ebiten.Image, len(source))
		for key, img := range source {
			zoomMap[key] = genZoomBlock(img, zoom)
		}
		target[zoom] = zoomMap
	}
	return target
}

// genZoomBlock 生成单张地图块在指定 zoom 下的缓存图。
// 缓存图使用最近邻缩放，并额外保留 1px 覆盖边缘以减少 tile 缝隙
func genZoomBlock(img *ebiten.Image, zoom int) *ebiten.Image {
	// 缩放后的 tile 多保留 1px 覆盖边缘，避免子像素相机位置下露出背景缝。
	size := zoomedBlockSize(zoom) + 1
	zoomImg := ebiten.NewImage(size, size)
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	scale := float64(size) / float64(constants.MapBlockSize)
	opts.GeoM.Scale(scale, scale)
	zoomImg.DrawImage(img, opts)
	return zoomImg
}

// normalizeZoom 将地图块缩放值修正为受支持档位。
// 非法值回到默认 4，保持与任务主战场的默认 1x 语义一致
func normalizeZoom(zoom int) int {
	for _, supportedZoom := range supportedZooms {
		if zoom == supportedZoom {
			return zoom
		}
	}
	return 4
}

// zoomedBlockSize 返回指定 zoom 下地图块的目标边长。
// zoom 使用“倍率 * 4”的整数语义，因此 4 对应原始 MapBlockSize
func zoomedBlockSize(zoom int) int {
	return max(1, constants.MapBlockSize*normalizeZoom(zoom)/4)
}
