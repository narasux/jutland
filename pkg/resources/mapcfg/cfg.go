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
	// waterRegion 是每格的水域连通域编号，0 表示非可航行格
	waterRegion [][]int32
	// waterRegionSizes 各水域连通域的可航行格数，下标即域编号（0 号占位）
	waterRegionSizes []int
	// clearance 是每格的离岸距离（格）：陆格为 0，可航行格为到最近陆格的切比雪夫距离
	clearance [][]int16
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
	cfg.PreparedGrid()
}

// PreparedGrid 返回这张地图只读的寻路网格。
// 网格带“离岸越近代价越高”的软代价，让航线尽量走在深水里，
// 减少舰体转弯时切到岸上被拦停。
func (cfg *MapCfg) PreparedGrid() *grid.Grid {
	if cfg.pathGrid == nil {
		cfg.pathGrid = grid.NewGrid(cfg.Cells)
		cfg.pathGrid.SetExtraCost(cfg.pathExtraCosts())
		cfg.pathGrid.Prepare()
	}
	return cfg.pathGrid
}

// pathExtraCosts 按离岸距离生成寻路用的额外代价表（行优先展平，float32 省内存）。
func (cfg *MapCfg) pathExtraCosts() []float32 {
	cfg.ensureClearance()
	if cfg.clearance == nil {
		return nil
	}
	costs := make([]float32, 0, cfg.Width*cfg.Height)
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			if !cfg.Map.IsSea(x, y) {
				costs = append(costs, 0)
				continue
			}
			costs = append(costs, pathCellExtraCost(cfg.clearance[y][x]))
		}
	}
	return costs
}

// pathCellExtraCost 贴岸水域的额外通行代价：离岸 1 格 +2、2 格 +1、其余 0。
// 代价很小，只让航线轻微偏向深水，不会把窄水道判成不可通行；
// 实际使用时还会按舰体尺寸缩放（大舰乘大、小艇乘小）。
func pathCellExtraCost(clearance int16) float32 {
	switch {
	case clearance <= 1:
		return 2
	case clearance == 2:
		return 1
	}
	return 0
}

// clearanceUnknown 离岸距离场的初始哨兵值，必须大于地图上任何真实距离。
const clearanceUnknown = int16(1) << 14

// ensureClearance 惰性计算离岸距离场（多源 BFS，一次性 O(宽×高)）。
func (cfg *MapCfg) ensureClearance() {
	if cfg.clearance != nil || len(cfg.Map) == 0 {
		return
	}
	cfg.clearance = make([][]int16, cfg.Height)
	queue := make([]grid.Point, 0, cfg.Width*cfg.Height/8)
	for y := 0; y < cfg.Height; y++ {
		cfg.clearance[y] = make([]int16, cfg.Width)
		for x := 0; x < cfg.Width; x++ {
			if cfg.Map.IsLand(x, y) {
				cfg.clearance[y][x] = 0
				queue = append(queue, grid.Point{X: x, Y: y})
				continue
			}
			cfg.clearance[y][x] = clearanceUnknown
		}
	}
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		next := cfg.clearance[cur.Y][cur.X] + 1
		for _, dir := range waterRegionDirs {
			nx, ny := cur.X+dir.X, cur.Y+dir.Y
			if nx < 0 || ny < 0 || ny >= cfg.Height || nx >= cfg.Width {
				continue
			}
			if cfg.clearance[ny][nx] <= next {
				continue
			}
			cfg.clearance[ny][nx] = next
			queue = append(queue, grid.Point{X: nx, Y: ny})
		}
	}
}

// ClearanceAt 返回指定格的离岸距离（格）：陆格为 0，可航行格为到最近陆格的切比雪夫距离。
// 没有地图数据时返回 0。
func (cfg *MapCfg) ClearanceAt(x, y int) int16 {
	cfg.ensureClearance()
	if cfg.clearance == nil || y < 0 || y >= cfg.Height || x < 0 || x >= cfg.Width {
		return 0
	}
	return cfg.clearance[y][x]
}

// GenPath 生成路径。
// 起终点落在海岸/陆地格上时（如点击码头、或舰船中心已搁浅），
// 先吸附到同一片连通水域的可航行格再寻路，保证搁浅舰也能退回水中继续机动。
func (cfg *MapCfg) GenPath(start, end grid.Point) []grid.Point {
	start, end = cfg.SnapPathEndpoints(start, end)
	return cfg.PreparedGrid().Search(start, end)
}

// SnapPathEndpoints 把寻路起终点吸附到互相连通的可航行水域。
// 两端都在水面时原样返回；一端在陆格时，吸附到与另一端同一片水域的最近
// 可航行格。返回的起终点可以直接交给 Grid.Search（异步寻路线程同样适用），
// 避免“吸附到被陆地隔开的封闭水域 → A* 直接失败 → 指令静默结束”。
func (cfg *MapCfg) SnapPathEndpoints(start, end grid.Point) (grid.Point, grid.Point) {
	cfg.ensureWaterRegions()
	startSea, endSea := cfg.Map.IsSea(start.X, start.Y), cfg.Map.IsSea(end.X, end.Y)
	switch {
	case startSea && endSea:
		return start, end
	case startSea:
		return start, cfg.snapToSea(end, cfg.WaterRegionAt(start.X, start.Y))
	case endSea:
		return cfg.snapToSea(start, cfg.WaterRegionAt(end.X, end.Y)), end
	default:
		// 两端都点在陆格上：先按最近水面确定一片水域，再把另一端也吸附到同一片。
		snappedStart := cfg.snapToSea(start, 0)
		return snappedStart, cfg.snapToSea(end, cfg.WaterRegionAt(snappedStart.X, snappedStart.Y))
	}
}

// WaterRegionAt 返回指定格的水域连通域编号，非可航行格返回 0。
func (cfg *MapCfg) WaterRegionAt(x, y int) int32 {
	cfg.ensureWaterRegions()
	if y < 0 || y >= cfg.Height || x < 0 || x >= cfg.Width {
		return 0
	}
	return cfg.waterRegion[y][x]
}

// waterRegionDirs 是水域连通域的相邻方向，与 grid.getNeighbors 保持一致：
// 对角方向要求两个正交邻格都可航行，避免从两块陆地的夹角“斜穿”出假连通。
var waterRegionDirs = [8]grid.Point{
	{X: 1, Y: 0},
	{X: -1, Y: 0},
	{X: 0, Y: 1},
	{X: 0, Y: -1},
	{X: 1, Y: 1},
	{X: 1, Y: -1},
	{X: -1, Y: 1},
	{X: -1, Y: -1},
}

// ensureWaterRegions 惰性标注水域连通域（一次性 O(宽×高)，之后只查表）。
func (cfg *MapCfg) ensureWaterRegions() {
	if cfg.waterRegion != nil {
		return
	}
	cfg.waterRegion = make([][]int32, cfg.Height)
	for y := range cfg.waterRegion {
		cfg.waterRegion[y] = make([]int32, cfg.Width)
	}
	cfg.waterRegionSizes = []int{0}
	stack := make([]grid.Point, 0, 64)
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			if cfg.waterRegion[y][x] != 0 || !cfg.Map.IsSea(x, y) {
				continue
			}
			regionID := int32(len(cfg.waterRegionSizes))
			stack = append(stack[:0], grid.Point{X: x, Y: y})
			cfg.waterRegion[y][x] = regionID
			size := 0
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++
				for _, dir := range waterRegionDirs {
					nx, ny := cur.X+dir.X, cur.Y+dir.Y
					if !cfg.Map.IsSea(nx, ny) || cfg.waterRegion[ny][nx] != 0 {
						continue
					}
					if dir.X != 0 && dir.Y != 0 &&
						(!cfg.Map.IsSea(cur.X, ny) || !cfg.Map.IsSea(nx, cur.Y)) {
						continue
					}
					cfg.waterRegion[ny][nx] = regionID
					stack = append(stack, grid.Point{X: nx, Y: ny})
				}
			}
			cfg.waterRegionSizes = append(cfg.waterRegionSizes, size)
		}
	}
}

// snapToSeaSnapRadius 兜底吸附搜索的最大半径（格）。
// 舰体中心最多压进岸边一两格，这个范围足够覆盖搁浅恢复场景。
const snapToSeaSnapRadius = 5

// snapToSeaPreferredRadius 同域吸附搜索的最大半径（格）。
// 玩家点在离岸较远的陆地上时，同域水面可能超出兜底半径，
// 放宽搜索范围可以给出一条“绕到最近可停靠水面”的航线。
const snapToSeaPreferredRadius = 16

// snapToSea 把点吸附到切比雪夫半径内最近的可航行格。
// preferRegion > 0 时优先接受该水域连通域内的格；找不到同域格时
// 退回半径内的最近可航行格，保证“同域优先”不会把原本可用的吸附变成失败。
func (cfg *MapCfg) snapToSea(p grid.Point, preferRegion int32) grid.Point {
	if cfg.Map.IsSea(p.X, p.Y) {
		return p
	}
	var fallback grid.Point
	hasFallback := false
	for radius := 1; radius <= snapToSeaPreferredRadius; radius++ {
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
				if !cfg.Map.IsSea(x, y) {
					continue
				}
				// 不限定水域，或正好落在目标水域：最近的可航行格即为答案。
				if preferRegion <= 0 || cfg.waterRegion[y][x] == preferRegion {
					return grid.Point{X: x, Y: y}
				}
				// 兜底只在原半径内放宽，避免把落点吸附到过远的另一片水域。
				if radius <= snapToSeaSnapRadius && !hasFallback {
					fallback, hasFallback = grid.Point{X: x, Y: y}, true
				}
			}
		}
	}
	if hasFallback {
		return fallback
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
