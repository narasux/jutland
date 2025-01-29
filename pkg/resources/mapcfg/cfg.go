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
	if x < 0 || x >= len(*m) || y < 0 || y >= len(*m) {
		return ' '
	}
	return rune((*m)[y][x])
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
	// 原始素材名
	Source string `json:"source"`
	// 地图数据
	Map MapData
	// 地图网格数据
	Cells grid.Cells
	// 地图宽度
	Width int
	// 地图高度
	Height int
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

	cfg.Cells = cfg.Map.ToGridCells()
	cfg.Width = len(cfg.Map[0])
	cfg.Height = len(cfg.Map)
}

// GenPath 生成路径
func (cfg *MapCfg) GenPath(start, end grid.Point) []grid.Point {
	return grid.NewGrid(cfg.Cells).Search(start, end)
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
