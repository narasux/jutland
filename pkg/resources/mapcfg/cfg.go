package mapcfg

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/narasux/jutland/pkg/config"
)

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
	Name string
	// 地图数据
	Map MapData
	// 地图宽度
	Width int
	// 地图高度
	Height int
}

var maps map[string]*MapCfg

func loadMapCfg(name string) *MapCfg {
	mapPath := fmt.Sprintf("%s/%s.map", config.MapResBaseDir, name)

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

func init() {
	log.Println("loading map config...")

	maps = make(map[string]*MapCfg)

	// TODO 这里也改成从配置文件加载
	maps["default"] = loadMapCfg("default")
	maps["allSea128"] = loadMapCfg("allSea128")

	log.Println("map config loaded")
}

// GetByName 获取地图配置
func GetByName(name string) *MapCfg {
	return maps[name]
}
