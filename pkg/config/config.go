package config

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/yosuke-furukawa/json5/encoding/json5"
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

// GameSettings 游戏设置配置
type GameSettings struct {
	// SpeedMultiplier 全局速度倍率，影响战舰、炮弹、鱼雷、飞机等移动速度
	// 默认值为1.0，最小值为0.1，最大值为10.0
	SpeedMultiplier float64 `json:"speedMultiplier"`
}

// G 游戏设置全局变量
var G *GameSettings

// NewDefaultGameSettings 创建默认游戏设置
func NewDefaultGameSettings() *GameSettings {
	return &GameSettings{
		SpeedMultiplier: 1.0,
	}
}

// validate 校验并修正游戏设置
func (s *GameSettings) validate() {
	// 处理 NaN 特殊情况（NaN 无法通过 min/max 处理）
	if math.IsNaN(s.SpeedMultiplier) {
		s.SpeedMultiplier = 1.0
	}
	// 限制范围 0.25 ~ 4.0
	s.SpeedMultiplier = math.Max(0.25, math.Min(4.0, s.SpeedMultiplier))
}

// LoadGameSettings 加载游戏设置配置文件
// 如果配置文件不存在则创建默认配置
// 如果加载失败则使用默认配置
func LoadGameSettings() {
	settingsPath := filepath.Join(ConfigBaseDir, "game_settings.json5")

	// 检查配置文件是否存在
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		log.Println("[INFO] game_settings.json5 not found, creating default configuration")
		G = NewDefaultGameSettings()
		if err = SaveGameSettings(); err != nil {
			log.Printf("[ERROR] Failed to save default game settings: %v", err)
		}
		return
	}

	// 读取配置文件
	file, err := os.Open(settingsPath)
	if err != nil {
		log.Printf("[ERROR] Failed to open game_settings.json5: %v, using default settings", err)
		G = NewDefaultGameSettings()
		return
	}
	defer file.Close()

	// 解析 JSON5
	var settings GameSettings
	decoder := json5.NewDecoder(file)
	if err = decoder.Decode(&settings); err != nil {
		log.Printf("[ERROR] Failed to parse game_settings.json5: %v, using default settings", err)
		G = NewDefaultGameSettings()
		return
	}

	// 校验配置
	settings.validate()
	G = &settings

	log.Printf("[INFO] Game settings loaded: SpeedMultiplier=%.2f", G.SpeedMultiplier)
}

// SaveGameSettings 保存游戏设置到配置文件
// 使用临时文件+重命名保证原子性写入
// 保存前会执行校验
func SaveGameSettings() error {
	if G == nil {
		G = NewDefaultGameSettings()
	}

	// 保存前校验
	G.validate()

	settingsPath := filepath.Join(ConfigBaseDir, "game_settings.json5")
	tempPath := settingsPath + ".tmp"

	// 创建临时文件
	file, err := os.Create(tempPath)
	if err != nil {
		log.Printf("[ERROR] Failed to create temp file for game settings: %v", err)
		return err
	}

	// 写入JSON5格式的注释头部
	_, _ = file.WriteString("// 游戏设置配置文件\n")
	_, _ = file.WriteString("// SpeedMultiplier: 全局速度倍率，影响战舰、炮弹、鱼雷、飞机等移动/转向速度\n")
	_, _ = file.WriteString("// 范围: 0.25 ~ 4.0，默认值: 1.0\n\n")

	// 编码并写入配置
	data, err := json5.MarshalIndent(G, "", "  ")
	if err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		log.Printf("[ERROR] Failed to encode game settings: %v", err)
		return err
	}
	_, _ = file.Write(data)
	_ = file.Close()

	// 原子性重命名
	if err = os.Rename(tempPath, settingsPath); err != nil {
		_ = os.Remove(tempPath)
		log.Printf("[ERROR] Failed to rename temp file: %v", err)
		return err
	}

	log.Printf("[INFO] Game settings saved: SpeedMultiplier=%.2f", G.SpeedMultiplier)
	return nil
}
