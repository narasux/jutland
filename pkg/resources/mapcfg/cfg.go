package mapcfg

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/utils/grid"
)

const (
	// 海洋
	ChrSea = '.'
	// 深海
	ChrDeepSea = 'O'
	// 浅海（可航行）
	ChrShallow = 'S'
	// 海岸（沙滩/崖壁/码头...）
	ChrCoast = 'C'
	// 陆地
	ChrLand = 'L'
)

// 地图数据
type MapData []string

func (m *MapData) Get(x, y int) rune {
	if y < 0 || y >= len(*m) {
		return ' '
	}
	row := (*m)[y]
	if x < 0 || x >= len(row) {
		return ' '
	}
	return rune(row[x])
}

// IsSea ...
func (m *MapData) IsSea(x, y int) bool {
	chr := m.Get(x, y)
	return chr == ChrSea || chr == ChrDeepSea || chr == ChrShallow
}

// IsLand ...
func (m *MapData) IsLand(x, y int) bool {
	chr := m.Get(x, y)
	return chr == ChrCoast || chr == ChrLand
}

// ToGridCells 转换成网格（路径计算用）
func (m *MapData) ToGridCells() grid.Cells {
	cells := grid.Cells{}
	for y := 0; y < len(*m); y++ {
		line := []int{}
		for x := 0; x < len((*m)[y]); x++ {
			switch (*m)[y][x] {
			case ChrShallow:
				line = append(line, grid.SD)
			case ChrSea, ChrDeepSea:
				line = append(line, grid.O)
			case ChrCoast, ChrLand:
				line = append(line, grid.W)
			}
		}
		cells = append(cells, line)
	}
	return cells
}

// 地图配置
type MapCfg struct {
	// 地图名称
	Name string `json:"name"`
	// 展示名称
	DisplayName string `json:"displayName"`
	// 英文展示名称
	DisplayNameEn string `json:"displayNameEn"`
	// 俄文展示名称
	DisplayNameRu string `json:"displayNameRu"`
	// 日文展示名称
	DisplayNameJa string `json:"displayNameJa"`
	// 原始素材名
	Source string `json:"source"`
	// 地图数据
	Map MapData
	// 地图网格数据
	Cells grid.Cells
	// pathGrid 是按地形预计算好的只读寻路网格
	pathGrid *grid.Grid
	// 地图宽度
	Width int
	// 地图高度
	Height int
}

// LocalizedDisplayName 返回当前语言下的地图展示名称。
func (cfg *MapCfg) LocalizedDisplayName() string {
	return i18n.LocalizedValue(map[i18n.Language]string{
		i18n.LanguageZhHans:   cfg.DisplayName,
		i18n.LanguageEnglish:  cfg.DisplayNameEn,
		i18n.LanguageRussian:  cfg.DisplayNameRu,
		i18n.LanguageJapanese: cfg.DisplayNameJa,
	})
}

// 初始化方块信息
func (cfg *MapCfg) initMapCells() {
	mapPath := fmt.Sprintf("%s/%s.map", config.MapResBaseDir, cfg.Name)

	file, err := os.Open(mapPath)
	if err != nil {
		log.Fatalf("missing %s: %s", mapPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cfg.Map = append(cfg.Map, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		log.Fatalf("error when load map %s: %s", cfg.Name, err)
	}
	if len(cfg.Map) == 0 {
		log.Fatalf("map %s is empty", cfg.Name)
	}
	width := len(cfg.Map[0])
	for y, row := range cfg.Map {
		if len(row) != width {
			log.Fatalf("map %s row %d has width %d, expected %d", cfg.Name, y, len(row), width)
		}
	}

	cfg.Cells = cfg.Map.ToGridCells()
	cfg.Width = width
	cfg.Height = len(cfg.Map)
	cfg.pathGrid = grid.NewGrid(cfg.Cells)
	cfg.pathGrid.Prepare()
}

// PreparedGrid 返回这张地图只读的寻路网格。
func (cfg *MapCfg) PreparedGrid() *grid.Grid {
	if cfg.pathGrid == nil {
		cfg.pathGrid = grid.NewGrid(cfg.Cells)
		cfg.pathGrid.Prepare()
	}
	return cfg.pathGrid
}

// GenPath 生成路径。
// 起终点落在海岸/陆地格上时（如点击码头、或舰船中心已搁浅），
// 先吸附到最近的可航行格再寻路，保证搁浅舰也能退回水中继续机动。
func (cfg *MapCfg) GenPath(start, end grid.Point) []grid.Point {
	return cfg.PreparedGrid().Search(cfg.snapToSea(start), cfg.snapToSea(end))
}

// snapToSeaSnapRadius 吸附搜索的最大半径（格）。
// 舰体中心最多压进岸边一两格，这个范围足够覆盖搁浅恢复场景。
const snapToSeaSnapRadius = 5

// snapToSea 把点吸附到切比雪夫半径内最近的可航行格；找不到时原样返回，
// 交给 Grid.Search 按原有语义判定失败。
func (cfg *MapCfg) snapToSea(p grid.Point) grid.Point {
	if cfg.Map.IsSea(p.X, p.Y) {
		return p
	}
	for radius := 1; radius <= snapToSeaSnapRadius; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				// 只扫外圈，避免同一格被多圈重复检查。
				if max(abs(dx), abs(dy)) != radius {
					continue
				}
				x, y := p.X+dx, p.Y+dy
				if x < 0 || y < 0 || y >= cfg.Height || x >= cfg.Width {
					continue
				}
				if cfg.Map.IsSea(x, y) {
					return grid.Point{X: x, Y: y}
				}
			}
		}
	}
	return p
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var maps map[string]*MapCfg

func init() {
	log.Println("loading map config...")

	maps = make(map[string]*MapCfg)

	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "maps.json5"))
	if err != nil {
		log.Fatal("failed to open maps.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var mapConfigs []MapCfg
	if err = json5.Unmarshal(bytes, &mapConfigs); err != nil {
		log.Fatal("failed to unmarshal maps.json5: ", err)
	}

	for _, cfg := range mapConfigs {
		cfg.initMapCells()
		maps[cfg.Name] = &cfg
	}

	log.Println("map config loaded")
}

// GetByName 获取地图配置
func GetByName(name string) *MapCfg {
	return maps[name]
}

// GetAllMapSources 获取所有地图原始资源名称
func GetAllMapSources() []string {
	sources := []string{}
	for _, m := range maps {
		sources = append(sources, m.Source)
	}
	return sources
}
