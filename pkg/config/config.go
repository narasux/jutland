package config

import (
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

const (
	// SpeedSlowMultiplier 慢档全局速度倍率。
	SpeedSlowMultiplier = 0.50
	// SpeedStandardMultiplier 标准档全局速度倍率。
	SpeedStandardMultiplier = 1.00
	// SpeedFastMultiplier 快档全局速度倍率。
	SpeedFastMultiplier = 2.00
)

// 战舰水面尾流的可调范围。1.0 表示保持原始观感，0 或缺失时按 1.0 处理。
const (
	WakeTrailLengthMin    = 0.40
	WakeTrailLengthMax    = 2.50
	WakeTrailWidthMin     = 0.40
	WakeTrailWidthMax     = 2.00
	WakeTrailOpacityMin   = 0.20
	WakeTrailOpacityMax   = 1.50
	WakeTrailDefaultScale = 1.00
)

// DefaultAudioVolume 默认音频音量，1.0 表示不衰减。
const DefaultAudioVolume = 1.00

var BaseDir = detectBaseDir()

// detectBaseDir 兼容从仓库子目录启动的 Go 测试，同时保留从可执行文件旁加载资源的能力。
func detectBaseDir() string {
	pwd, _ := os.Getwd()
	exePath, _ := os.Executable()
	for _, candidate := range []string{filepath.Dir(exePath), pwd} {
		for dir := candidate; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if isResourceBaseDir(dir) {
				return dir
			}
		}
	}
	return pwd
}

func isResourceBaseDir(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "configs")); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "resources"))
	return err == nil
}

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

// AIStrategies 是电脑舰队可以同时启用的策略。未出现在配置里的项保持默认开启。
type AIStrategies struct {
	Attack  bool `json:"attack"`
	Defend  bool `json:"defend"`
	Counter bool `json:"counter"`
	Raid    bool `json:"raid"`
	Scout   bool `json:"scout"`
}

// DefaultAIStrategies 五种策略全部开启，和未写配置时的行为一致。
func DefaultAIStrategies() AIStrategies {
	return AIStrategies{Attack: true, Defend: true, Counter: true, Raid: true, Scout: true}
}

// WakeTrailSettings 战舰水面尾流的观感参数，1.0 为原始值。
type WakeTrailSettings struct {
	// Length 尾流航迹长度倍率；越大尾流拖得越远。
	Length float64 `json:"length"`
	// Width 尾流宽度倍率；越大尾流扩散得越宽。
	Width float64 `json:"width"`
	// Opacity 尾流浓度倍率；越大尾流越白、越显眼。
	Opacity float64 `json:"opacity"`
}

// DefaultWakeTrailSettings 返回原始尾流观感。
func DefaultWakeTrailSettings() WakeTrailSettings {
	return WakeTrailSettings{
		Length:  WakeTrailDefaultScale,
		Width:   WakeTrailDefaultScale,
		Opacity: WakeTrailDefaultScale,
	}
}

func normalizeWakeScale(value, minValue, maxValue float64) float64 {
	if math.IsNaN(value) || value <= 0 {
		return WakeTrailDefaultScale
	}
	return min(max(value, minValue), maxValue)
}

// Normalize 把三个尾流参数夹到可用范围；0/NaN 视为未配置并回退到 1.0。
func (w WakeTrailSettings) Normalize() WakeTrailSettings {
	return WakeTrailSettings{
		Length:  normalizeWakeScale(w.Length, WakeTrailLengthMin, WakeTrailLengthMax),
		Width:   normalizeWakeScale(w.Width, WakeTrailWidthMin, WakeTrailWidthMax),
		Opacity: normalizeWakeScale(w.Opacity, WakeTrailOpacityMin, WakeTrailOpacityMax),
	}
}

// GameSettings 游戏设置配置
type GameSettings struct {
	// SpeedMultiplier 全局速度倍率，影响战舰、炮弹、鱼雷、飞机等移动速度
	// 可选值为 0.50 / 1.00 / 2.00
	SpeedMultiplier float64 `json:"speedMultiplier"`
	// Language 游戏界面语言，当前正式启用 zh-Hans / en / ru / ja。
	Language string `json:"language"`
	// BGMVolume 背景音乐音量，范围 0.00 ~ 1.00；0 为静音，字段缺失时按 1.00 处理。
	// 使用指针区分「旧配置没写」和「玩家显式静音」。
	BGMVolume *float64 `json:"bgmVolume"`
	// EffectsVolume 战场音效（火炮 / 鱼雷 / 爆炸等）音量，范围同上。
	EffectsVolume *float64 `json:"effectsVolume"`
	// AI 电脑舰队允许使用的策略。
	AI AIStrategies `json:"aiStrategies"`
	// EnableLandAirfield 陆地机场功能总开关，false 时所有任务不生成机场
	EnableLandAirfield bool `json:"enableLandAirfield"`
	// EnableFogOfWar 战争迷雾总开关。关闭时任务不建视野图，逻辑与全图可见一致。
	EnableFogOfWar bool `json:"enableFogOfWar"`
	// EnablePenetration 实验性穿甲伤害总开关。默认关闭；开启后按弹药穿深/舰体装甲
	// 计算伤害倍率，未穿透会显著减伤，过穿会略微增伤。
	EnablePenetration bool `json:"enablePenetration"`
	// EnableMinimap 小地图总开关。默认开启：开局自动展开右上战术面板展示小地图；
	// 关闭后依然可以点面板把手手动展开。
	EnableMinimap bool `json:"enableMinimap"`
	// DisplayHitEffects 命中特效总开关。关闭后炮弹命中舰体 / 落水 / 落陆都不再生成特效，
	// 伤害数字与战舰状态不受影响。
	DisplayHitEffects bool `json:"displayHitEffects"`
	// DisplayDamageNumber 是否在命中位置显示伤害数字。
	DisplayDamageNumber bool `json:"displayDamageNumber"`
	// ForceDisplayState 是否常显战舰状态（HP 与武器装填）；关闭后仅在选中时显示。
	ForceDisplayState bool `json:"forceDisplayState"`
	// WakeTrail 战舰水面尾流的长度 / 宽度 / 浓度自定义参数。
	WakeTrail WakeTrailSettings `json:"wakeTrail"`
}

// G 游戏设置全局变量
var G *GameSettings

// NewDefaultGameSettings 创建默认游戏设置
func NewDefaultGameSettings() *GameSettings {
	settings := &GameSettings{
		SpeedMultiplier:     SpeedStandardMultiplier,
		Language:            "zh-Hans",
		AI:                  DefaultAIStrategies(),
		EnableLandAirfield:  true,
		EnableMinimap:       true,
		DisplayHitEffects:   true,
		DisplayDamageNumber: true,
		ForceDisplayState:   true,
		WakeTrail:           DefaultWakeTrailSettings(),
	}
	settings.SetBGMVolume(DefaultAudioVolume)
	settings.SetEffectsVolume(DefaultAudioVolume)
	return settings
}

// EnabledAIStrategies 返回当前设置里的电脑策略。设置还没加载时五种都算开启。
func EnabledAIStrategies() AIStrategies {
	if G == nil {
		return DefaultAIStrategies()
	}
	return G.AI
}

// ResolvedWakeTrail 返回校验后的尾流参数；设置未加载或字段缺失时使用原始观感。
func (s *GameSettings) ResolvedWakeTrail() WakeTrailSettings {
	if s == nil {
		return DefaultWakeTrailSettings()
	}
	return s.WakeTrail.Normalize()
}

func normalizeAudioVolume(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	return min(max(value, 0), 1)
}

func resolvedAudioVolume(value *float64) float64 {
	if value == nil {
		return DefaultAudioVolume
	}
	return normalizeAudioVolume(*value)
}

// ResolvedBGMVolume 返回校验后的背景音乐音量；字段缺失时使用默认值。
func (s *GameSettings) ResolvedBGMVolume() float64 {
	if s == nil {
		return DefaultAudioVolume
	}
	return resolvedAudioVolume(s.BGMVolume)
}

// ResolvedEffectsVolume 返回校验后的战场音效音量；字段缺失时使用默认值。
func (s *GameSettings) ResolvedEffectsVolume() float64 {
	if s == nil {
		return DefaultAudioVolume
	}
	return resolvedAudioVolume(s.EffectsVolume)
}

// SetBGMVolume 写入并夹紧背景音乐音量；显式 0 表示静音。
func (s *GameSettings) SetBGMVolume(value float64) {
	value = normalizeAudioVolume(value)
	s.BGMVolume = &value
}

// SetEffectsVolume 写入并夹紧战场音效音量；显式 0 表示静音。
func (s *GameSettings) SetEffectsVolume(value float64) {
	value = normalizeAudioVolume(value)
	s.EffectsVolume = &value
}

func normalizeSpeedMultiplier(value float64) float64 {
	if math.IsNaN(value) {
		return SpeedStandardMultiplier
	}
	switch {
	case value <= (SpeedSlowMultiplier+SpeedStandardMultiplier)/2:
		return SpeedSlowMultiplier
	case value <= (SpeedStandardMultiplier+SpeedFastMultiplier)/2:
		return SpeedStandardMultiplier
	default:
		return SpeedFastMultiplier
	}
}

// validate 校验并修正游戏设置
func (s *GameSettings) validate() {
	s.SpeedMultiplier = normalizeSpeedMultiplier(s.SpeedMultiplier)
	switch s.Language {
	case "zh-Hans", "en", "ru", "ja":
	default:
		s.Language = "zh-Hans"
	}
	s.WakeTrail = s.WakeTrail.Normalize()
	s.SetBGMVolume(s.ResolvedBGMVolume())
	s.SetEffectsVolume(s.ResolvedEffectsVolume())
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

	// 从默认值开始解析，这样旧配置里没写的策略开关保持开启。
	settings := *NewDefaultGameSettings()
	decoder := json5.NewDecoder(file)
	if err = decoder.Decode(&settings); err != nil {
		log.Printf("[ERROR] Failed to parse game_settings.json5: %v, using default settings", err)
		G = NewDefaultGameSettings()
		return
	}

	// 校验配置
	settings.validate()
	G = &settings

	log.Printf("[INFO] Game settings loaded: SpeedMultiplier=%.2f Language=%s", G.SpeedMultiplier, G.Language)
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
	_, _ = file.WriteString(
		"// SpeedMultiplier: 全局速度倍率（慢=0.50 / 标准=1.00 / 快=2.00），" +
			"影响战舰、炮弹、鱼雷、飞机等移动/转向速度\n",
	)
	_, _ = file.WriteString("// 范围: 0.50 ~ 2.00，默认值: 1.00\n\n")
	_, _ = file.WriteString("// Language: 游戏界面语言，当前正式启用 zh-Hans / en / ru / ja\n")
	_, _ = file.WriteString("// bgmVolume: 背景音乐音量 0.00 ~ 1.00，0 为静音，默认 1.00\n")
	_, _ = file.WriteString("// effectsVolume: 战场音效（火炮 / 鱼雷 / 爆炸等）音量 0.00 ~ 1.00，默认 1.00\n")
	_, _ = file.WriteString("// EnableLandAirfield: 陆地机场功能总开关，false 时所有任务不生成机场\n")
	_, _ = file.WriteString("// aiStrategies: 电脑可用策略，attack/defend/counter/raid/scout 为 false 时对局中不使用\n")
	_, _ = file.WriteString("// EnableFogOfWar: 战争迷雾总开关，默认关闭；关闭时不计算迷雾\n")
	_, _ = file.WriteString("// EnableMinimap: 小地图总开关，默认开启；开启时开局自动展开战术面板显示小地图\n")
	_, _ = file.WriteString("// DisplayHitEffects: 命中特效总开关，默认开启；关闭后命中舰体 / 落水 / 落陆不再生成特效\n")
	_, _ = file.WriteString("// DisplayDamageNumber: 是否显示命中伤害数字，默认开启\n")
	_, _ = file.WriteString("// ForceDisplayState: 是否常显战舰状态（HP / 武器装填），默认开启；关闭后仅选中时显示\n")
	_, _ = file.WriteString(
		"// wakeTrail: 战舰水面尾流参数，length 航迹长度 0.40~2.50、width 宽度 0.40~2.00、" +
			"opacity 浓度 0.20~1.50，默认均为 1.00\n",
	)
	_, _ = file.WriteString("// EnablePenetration: 实验性穿甲伤害开关，默认关闭；开启后按弹药穿深/舰体装甲缩放伤害\n\n")

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

	log.Printf("[INFO] Game settings saved: SpeedMultiplier=%.2f Language=%s", G.SpeedMultiplier, G.Language)
	return nil
}
