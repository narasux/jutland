package mapcfg

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/narasux/jutland/pkg/envs"
)

// 地图名称
type MapName string

// 地图数据
type MapData []string

func (m *MapData) Get(x, y int) rune {
	if y < 0 || y >= len(*m) {
		return ' '
	}
	return rune((*m)[y][x])
}

// 地图配置
type MapCfg struct {
	// 地图名称
	Name MapName
	// 地图数据
	Map MapData
	// 地图宽度
	Width int
	// 地图高度
	Height int
}

var maps = map[MapName]*MapCfg{}

const (
	// 默认地图（128 * 128）
	MapDefault MapName = "default"
	// 海洋地图（128 * 128）
	MapAllSea128 MapName = "allSea128"
)

func init() {
	log.Println("loading map config...")

	maps = make(map[MapName]*MapCfg)

	maps[MapDefault] = loadMapCfg(MapDefault)
	maps[MapAllSea128] = loadMapCfg(MapAllSea128)

	log.Println("map config loaded")
}

func loadMapCfg(name MapName) *MapCfg {
	mapPath := fmt.Sprintf("%s/%s.map", envs.MapResBaseDir, name)

	file, err := os.Open(mapPath)
	if err != nil {
		log.Fatalf("missing %s: %s", mapPath, err)
	}
	defer file.Close()

	cfg := MapCfg{Name: name, Map: MapData{}}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		cfg.Map = append(cfg.Map, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		log.Fatalf("error when load map %s: %s", name, err)
	}

	cfg.Width = len(cfg.Map[0])
	cfg.Height = len(cfg.Map)
	return &cfg
}

// GetByName 获取地图配置
func GetByName(name MapName) *MapCfg {
	return maps[name]
}
