package envs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/utils/envx"
)

var (
	pwd, _     = os.Getwd()
	exePath, _ = os.Executable()
	exeDir     = filepath.Dir(exePath)
	baseDir    = lo.Ternary(strings.Contains(exeDir, pwd), exeDir, pwd)
)

// 以下变量值可通过环境变量指定
var (
	// ImgResBaseDir 图片资源根目录
	ImgResBaseDir = envx.Get("IMG_RES_BASE_DIR", filepath.Join(baseDir, "resources/images"))

	// FontResBaseDir 字体资源根目录
	FontResBaseDir = envx.Get("FONT_RES_BASE_DIR", filepath.Join(baseDir, "resources/fonts"))

	// AudioResBaseDir 音频资源根目录
	AudioResBaseDir = envx.Get("AUDIO_RES_BASE_DIR", filepath.Join(baseDir, "resources/audios"))

	// MapResBaseDir 地图资源根目录
	MapResBaseDir = envx.Get("MAP_RES_BASE_DIR", filepath.Join(baseDir, "resources/maps"))

	// ConfigBaseDir 配置文件根目录
	ConfigBaseDir = envx.Get("CONFIG_BASE_DIR", filepath.Join(baseDir, "configs"))
)
