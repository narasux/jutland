package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

var (
	pwd, _     = os.Getwd()
	exePath, _ = os.Executable()
	exeDir     = filepath.Dir(exePath)
	BaseDir    = lo.Ternary(strings.Contains(exeDir, pwd), exeDir, pwd)
)

var (
	// ImgResBaseDir 图片资源根目录
	ImgResBaseDir = filepath.Join(BaseDir, "resources/images")

	// FontResBaseDir 字体资源根目录
	FontResBaseDir = filepath.Join(BaseDir, "resources/fonts")

	// AudioResBaseDir 音频资源根目录
	AudioResBaseDir = filepath.Join(BaseDir, "resources/audios")

	// MapResBaseDir 地图资源根目录
	MapResBaseDir = filepath.Join(BaseDir, "resources/maps")

	// ConfigBaseDir 配置文件根目录
	ConfigBaseDir = filepath.Join(BaseDir, "configs")
)
