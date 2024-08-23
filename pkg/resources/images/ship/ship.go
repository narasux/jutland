package ship

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/loader"
)

func init() {
	log.Println("loading ship image resources...")

	// TODO 航母，鱼雷艇待支持
	shipTypes := []string{"battleship", "cruiser", "default", "destroyer"}

	for _, shipType := range shipTypes {
		entries, err := os.ReadDir(filepath.Join(config.ImgResBaseDir, "ships", shipType))
		if err != nil {
			log.Fatal(err)
		}
		for _, entry := range entries {
			// 限制原始图片必须是 PNG 格式
			if !strings.HasSuffix(entry.Name(), ".png") {
				continue
			}
			imgPath := fmt.Sprintf("/ships/%s/%s", shipType, entry.Name())
			shipImg, loadImgErr := loader.LoadImage(imgPath)
			if loadImgErr != nil {
				log.Fatalf("missing %s: %s", imgPath, loadImgErr)
			}
			ShipImgMap[strings.TrimSuffix(entry.Name(), ".png")] = shipImg
		}
	}

	log.Println("ship image resources loaded")
}

var ShipImgMap = map[string]*ebiten.Image{}

// Get 获取战舰图片
func Get(name string) *ebiten.Image {
	return ShipImgMap[name]
}
