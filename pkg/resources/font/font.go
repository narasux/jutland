package font

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// FontHang 行书字体
	Hang *text.GoTextFaceSource

	// FontKai 楷体字体
	Kai *text.GoTextFaceSource
)

func init() {
	var err error

	log.Println("loading font resources...")

	fontPath := "/hang.ttf"
	if Hang, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontPath = "/kai.ttf"
	if Kai, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	log.Println("font resources loaded")
}
