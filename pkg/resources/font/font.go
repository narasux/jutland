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

	// JetbrainsMono Mono 字体
	JetbrainsMono *text.GoTextFaceSource

	// JetbrainsMonoItalic Mono 斜体
	JetbrainsMonoItalic *text.GoTextFaceSource
)

var fontStrMap map[*text.GoTextFaceSource]string

// FontToStr 字体转名称字符串
func FontToStr(f *text.GoTextFaceSource) string {
	return fontStrMap[f]
}

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

	fontPath = "/jetbrains_mono.ttf"
	if JetbrainsMono, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontPath = "/jetbrains_mono_italic.ttf"
	if JetbrainsMonoItalic, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontStrMap = map[*text.GoTextFaceSource]string{
		Hang:                "hang",
		Kai:                 "kai",
		OpenSans:            "open_sans",
		OpenSansItalic:      "open_sans_italic",
		JetbrainsMono:       "jetbrains_mono",
		JetbrainsMonoItalic: "jetbrains_mono_italic",
	}

	log.Println("font resources loaded")
}
