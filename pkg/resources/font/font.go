package font

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	// Hang 行书字体
	Hang *text.GoTextFaceSource

	// Kai 楷体字体
	Kai *text.GoTextFaceSource

	// OpenSans OpenSans 字体
	OpenSans *text.GoTextFaceSource

	// OpenSansItalic OpenSans 斜体
	OpenSansItalic *text.GoTextFaceSource
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

	fontPath = "/open_sans.ttf"
	if OpenSans, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontPath = "/open_sans_italic.ttf"
	if OpenSansItalic, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	log.Println("font resources loaded")
}
