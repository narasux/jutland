package envs

import (
	"path/filepath"

	"github.com/narasux/jutland/pkg/utils/envx"
	"github.com/narasux/jutland/pkg/utils/pathx"
)

// 以下变量值可通过环境变量指定
var (
	// Debug 调试模式
	Debug = envx.Get("DEBUG", "false") == "true"

	// ImgResBaseDir 图片资源根目录
	ImgResBaseDir = envx.Get("IMG_RES_BASE_DIR", filepath.Join(pathx.GetCurPKGPath(), "../../resources/images"))

	// FontResBaseDir 字体资源根目录
	FontResBaseDir = envx.Get("FONT_RES_BASE_DIR", filepath.Join(pathx.GetCurPKGPath(), "../../resources/fonts"))

	// AudioResBaseDir 音频资源根目录
	AudioResBaseDir = envx.Get("AUDIO_RES_BASE_DIR", filepath.Join(pathx.GetCurPKGPath(), "../../resources/audios"))

	// MapResBaseDir 地图资源根目录
	MapResBaseDir = envx.Get("MAP_RES_BASE_DIR", filepath.Join(pathx.GetCurPKGPath(), "../../resources/maps"))
)
