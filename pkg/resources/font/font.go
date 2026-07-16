package font

import (
	"log"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/i18n"
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

	// GolosText 俄语界面字体
	GolosText *text.GoTextFaceSource

	// ZenKakuGothicNew 日语界面字体
	ZenKakuGothicNew *text.GoTextFaceSource
)

var fontStrMap map[*text.GoTextFaceSource]string

// FontToStr 字体转名称字符串
func FontToStr(f *text.GoTextFaceSource) string {
	return fontStrMap[f]
}

// LocalizedUI 返回当前语言应使用的正文界面字体。
func LocalizedUI(chinese *text.GoTextFaceSource) *text.GoTextFaceSource {
	return ForLanguage(i18n.CurrentLanguage(), chinese)
}

// ForLanguage 返回指定语言应使用的界面字体。
func ForLanguage(lang i18n.Language, chinese *text.GoTextFaceSource) *text.GoTextFaceSource {
	switch lang {
	case i18n.LanguageEnglish:
		return JetbrainsMono
	case i18n.LanguageRussian:
		return GolosText
	case i18n.LanguageJapanese:
		return ZenKakuGothicNew
	default:
		return chinese
	}
}

// LocalizedTitle 返回当前语言应使用的标题字体。
func LocalizedTitle(chinese *text.GoTextFaceSource) *text.GoTextFaceSource {
	switch i18n.CurrentLanguage() {
	case i18n.LanguageEnglish:
		return OpenSans
	case i18n.LanguageRussian:
		return GolosText
	case i18n.LanguageJapanese:
		return ZenKakuGothicNew
	default:
		return chinese
	}
}

// LanguageSelector 返回能覆盖四种语言自称的设置页字体。
func LanguageSelector() *text.GoTextFaceSource {
	return ZenKakuGothicNew
}

// ForText 按实际文本内容选择能覆盖字形的字体。
func ForText(value string, preferred *text.GoTextFaceSource) *text.GoTextFaceSource {
	hasHan := false
	for _, r := range value {
		if unicode.In(r, unicode.Cyrillic) {
			return GolosText
		}
		if unicode.In(r, unicode.Hiragana, unicode.Katakana) {
			return ZenKakuGothicNew
		}
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Bopomofo) {
			hasHan = true
		}
	}
	if hasHan {
		if i18n.CurrentLanguage() == i18n.LanguageJapanese {
			return ZenKakuGothicNew
		}
		if preferred == Hang {
			return Hang
		}
		return Kai
	}
	return preferred
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

	fontPath = "/golos_text.ttf"
	if GolosText, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontPath = "/zen_kaku_gothic_new_medium.ttf"
	if ZenKakuGothicNew, err = loader.LoadFont(fontPath); err != nil {
		log.Fatalf("missing %s: %s", fontPath, err)
	}

	fontStrMap = map[*text.GoTextFaceSource]string{
		Hang:                "hang",
		Kai:                 "kai",
		OpenSans:            "open_sans",
		OpenSansItalic:      "open_sans_italic",
		JetbrainsMono:       "jetbrains_mono",
		JetbrainsMonoItalic: "jetbrains_mono_italic",
		GolosText:           "golos_text",
		ZenKakuGothicNew:    "zen_kaku_gothic_new",
	}

	log.Println("font resources loaded")
}
