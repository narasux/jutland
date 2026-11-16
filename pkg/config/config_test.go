package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// TestNormalizeSpeedMultiplierKeepsThreePresets 验证速度倍率被吸附到慢/标准/快三档。
// 场景：json5 里可能写 0.1 或 4.0 这类手改值；分档阈值是相邻档中点 0.75 与 1.5。
// 预期：0.1→0.50、0.5→0.50、1.0→1.00、2.0→2.00、4.0→2.00；失败表示任意倍率会直接进入游戏循环。
// 覆盖：表驱动遍历 5 个输入调用 normalizeSpeedMultiplier，逐个与 SpeedSlow/Standard/FastMultiplier 比较。
func TestNormalizeSpeedMultiplierKeepsThreePresets(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  float64
	}{
		{value: 0.1, want: SpeedSlowMultiplier},
		{value: SpeedSlowMultiplier, want: SpeedSlowMultiplier},
		{value: SpeedStandardMultiplier, want: SpeedStandardMultiplier},
		{value: SpeedFastMultiplier, want: SpeedFastMultiplier},
		{value: 4.0, want: SpeedFastMultiplier},
	} {
		if got := normalizeSpeedMultiplier(tc.value); got != tc.want {
			t.Fatalf("normalizeSpeedMultiplier(%v) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestMissingAIStrategiesStayEnabled 验证配置缺 aiStrategies 或只写部分键时，其余策略仍默认开启。
// 场景：老 game_settings.json5 没有该节；新配置可能只关 attack/scout，其余 bool 不能因此清零。
// 预期：只写 language 时五种策略全为 true；写 attack=false,scout=false 时仅这两项 false，
// Defend/Counter/Raid 仍为 true；失败表示玩家会意外失去防守/反击/袭扰 AI。
// 覆盖：以 NewDefaultGameSettings 副本为底，json5.Unmarshal 整节缺失与部分缺失两种输入后比较 AI。
func TestMissingAIStrategiesStayEnabled(t *testing.T) {
	settings := *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"language":"en"}`), &settings); err != nil {
		t.Fatalf("decode defaults: %v", err)
	}
	if settings.AI != DefaultAIStrategies() {
		t.Fatalf("missing strategies = %+v", settings.AI)
	}

	settings = *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"aiStrategies":{"attack":false,"scout":false}}`), &settings); err != nil {
		t.Fatalf("decode partial: %v", err)
	}
	if settings.AI.Attack || settings.AI.Scout || !settings.AI.Defend || !settings.AI.Counter || !settings.AI.Raid {
		t.Fatalf("partial strategies = %+v", settings.AI)
	}
}

// TestMinimapDefaultsToOpen 验证小地图默认开启，且老配置缺 enableMinimap 时不会被清零。
// 场景：NewDefaultGameSettings 应给出 EnableMinimap=true；以默认设置为底 json5.Unmarshal `{"language":"en"}` 模拟老配置文件，再显式写 `{"enableMinimap":false}` 验证关闭。
// 预期：前两次 EnableMinimap 均为 true，显式 false 时为 false；失败意味着升级后老玩家的小地图会被关掉。
// 覆盖：分别对默认值、缺键配置、显式 false 配置做三次断言。
func TestMinimapDefaultsToOpen(t *testing.T) {
	if !NewDefaultGameSettings().EnableMinimap {
		t.Fatal("minimap should default to open")
	}

	settings := *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"language":"en"}`), &settings); err != nil {
		t.Fatalf("decode defaults: %v", err)
	}
	if !settings.EnableMinimap {
		t.Fatal("missing enableMinimap should stay open")
	}

	settings = *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"enableMinimap":false}`), &settings); err != nil {
		t.Fatalf("decode disable: %v", err)
	}
	if settings.EnableMinimap {
		t.Fatal("explicit false should keep the minimap closed")
	}
}

// TestDisplayOptionsDefaultToOpen 验证命中特效、伤害数字、战舰状态三个显示项默认开启，且老配置缺键保持开启。
// 场景：默认设置三项均为 true；json5.Unmarshal `{"language":"en"}` 模拟旧配置；再显式把 displayHitEffects/displayDamageNumber/forceDisplayState 都设为 false。
// 预期：默认与缺键时三项全 true，显式 false 时三项全 false；失败说明升级会静默关掉命中反馈或常显状态。
// 覆盖：三次构造 GameSettings 副本，分别断言三个 bool 的组合。
func TestDisplayOptionsDefaultToOpen(t *testing.T) {
	defaults := NewDefaultGameSettings()
	if !defaults.DisplayHitEffects || !defaults.DisplayDamageNumber || !defaults.ForceDisplayState {
		t.Fatalf("display options should default to open: %+v", defaults)
	}

	settings := *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"language":"en"}`), &settings); err != nil {
		t.Fatalf("decode defaults: %v", err)
	}
	if !settings.DisplayHitEffects || !settings.DisplayDamageNumber || !settings.ForceDisplayState {
		t.Fatalf("missing display keys should stay open: %+v", settings)
	}

	settings = *NewDefaultGameSettings()
	disabled := `{"displayHitEffects":false,"displayDamageNumber":false,"forceDisplayState":false}`
	if err := json5.Unmarshal([]byte(disabled), &settings); err != nil {
		t.Fatalf("decode disable: %v", err)
	}
	if settings.DisplayHitEffects || settings.DisplayDamageNumber || settings.ForceDisplayState {
		t.Fatalf("explicit false should stay closed: %+v", settings)
	}
}

// TestGameSettingsValidateNormalizesSpeedAndLanguage 验证设置校验把任意速度归一到三档，并把非法语言回退中文。
// 场景：旧存档/手写 game_settings.json5 可能写入 0.4、1.7、NaN 或 "fr"；这些值直接参与移动速度与本地化选表。
// 预期：0.75 及以下→0.50，0.75~1.5→1.00，1.5 以上→2.00，NaN→1.00；非法语言→zh-Hans。
// 覆盖：表驱动构造速度/语言组合，调用 validate 后逐项断言两个字段。
func TestGameSettingsValidateNormalizesSpeedAndLanguage(t *testing.T) {
	cases := []struct {
		name     string
		speed    float64
		language string
		want     float64
		wantLang string
	}{
		{name: "slow lower bound", speed: 0.4, language: "en", want: SpeedSlowMultiplier, wantLang: "en"},
		{name: "slow upper bound", speed: 0.75, language: "ru", want: SpeedSlowMultiplier, wantLang: "ru"},
		{name: "standard interior", speed: 1.2, language: "ja", want: SpeedStandardMultiplier, wantLang: "ja"},
		{
			name:     "standard upper bound",
			speed:    1.5,
			language: "zh-Hans",
			want:     SpeedStandardMultiplier,
			wantLang: "zh-Hans",
		},
		{name: "fast above bound", speed: 1.51, language: "en", want: SpeedFastMultiplier, wantLang: "en"},
		{name: "fast upper bound", speed: 5, language: "", want: SpeedFastMultiplier, wantLang: "zh-Hans"},
		{
			name:     "NaN falls back to standard",
			speed:    math.NaN(),
			language: "unknown",
			want:     SpeedStandardMultiplier,
			wantLang: "zh-Hans",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settings := &GameSettings{SpeedMultiplier: tc.speed, Language: tc.language}
			settings.validate()
			require.Equal(t, tc.want, settings.SpeedMultiplier)
			require.Equal(t, tc.wantLang, settings.Language)
		})
	}
}

// withTempConfigDir 把配置目录指向临时目录，并在测试结束后恢复全局目录与 G。
func withTempConfigDir(t *testing.T) (string, func()) {
	t.Helper()
	previousDir, previousSettings := ConfigBaseDir, G
	ConfigBaseDir = t.TempDir()
	return ConfigBaseDir, func() {
		ConfigBaseDir, G = previousDir, previousSettings
	}
}

// TestLoadGameSettingsCreatesDefaultWhenMissing 验证首次启动没有配置文件时自动写出一份默认配置。
// 场景：新安装/删除 game_settings.json5 后启动；游戏必须在没有用户配置的情况下仍能进入菜单。
// 预期：LoadGameSettings 后 G 非 nil、文件存在，且速度/语言/开关与 NewDefaultGameSettings 一致。
// 覆盖：把 ConfigBaseDir 指到空临时目录，调用 Load，断言 G 字段与 os.Stat 结果。
func TestLoadGameSettingsCreatesDefaultWhenMissing(t *testing.T) {
	dir, cleanup := withTempConfigDir(t)
	defer cleanup()

	G = nil
	LoadGameSettings()

	require.NotNil(t, G)
	require.Equal(t, SpeedStandardMultiplier, G.SpeedMultiplier)
	require.Equal(t, "zh-Hans", G.Language)
	require.True(t, G.EnableMinimap)
	require.True(t, G.DisplayHitEffects)

	_, err := os.Stat(filepath.Join(dir, "game_settings.json5"))
	require.NoError(t, err)
}

// TestLoadGameSettingsFallsBackOnInvalidJSON 验证配置文件损坏时回退默认值而不是崩溃。
// 场景：玩家手动编辑 game_settings.json5 写坏语法，或磁盘写入被截断；启动不能因此 panic。
// 预期：LoadGameSettings 不 panic，G 为默认配置；坏文件保留供用户修复。
// 覆盖：临时目录写入非法 JSON5，调用 Load 后断言 G 默认值与文件仍存在。
func TestLoadGameSettingsFallsBackOnInvalidJSON(t *testing.T) {
	dir, cleanup := withTempConfigDir(t)
	defer cleanup()

	settingsPath := filepath.Join(dir, "game_settings.json5")
	require.NoError(t, os.WriteFile(settingsPath, []byte("not a json5 file"), 0o644))
	G = nil
	LoadGameSettings()

	require.NotNil(t, G)
	require.Equal(t, SpeedStandardMultiplier, G.SpeedMultiplier)
	require.Equal(t, "zh-Hans", G.Language)
	_, err := os.Stat(settingsPath)
	require.NoError(t, err)
}

// TestSaveGameSettingsRoundTripsAndRemovesTemp 验证保存后能原样读回，且临时文件被原子重命名清理。
// 场景：玩家在设置页改速度/语言/开关后保存，下次启动应恢复这些值；写入中断时不能留下 .tmp。
// 预期：保存后无 game_settings.json5.tmp；清空 G 重新 Load 得到保存的速度/语言/开关。
// 覆盖：临时目录构造自定义 G，调用 Save，再 Load 并逐字段断言；os.Stat 断言临时文件不存在。
func TestSaveGameSettingsRoundTripsAndRemovesTemp(t *testing.T) {
	dir, cleanup := withTempConfigDir(t)
	defer cleanup()

	G = &GameSettings{
		SpeedMultiplier:     SpeedFastMultiplier,
		Language:            "ja",
		AI:                  AIStrategies{Attack: true, Scout: true},
		EnableLandAirfield:  false,
		EnableFogOfWar:      true,
		EnableMinimap:       false,
		DisplayHitEffects:   false,
		DisplayDamageNumber: false,
		ForceDisplayState:   false,
	}
	require.NoError(t, SaveGameSettings())

	_, err := os.Stat(filepath.Join(dir, "game_settings.json5.tmp"))
	require.True(t, os.IsNotExist(err), "tmp file should be renamed away")

	G = nil
	LoadGameSettings()
	require.NotNil(t, G)
	require.Equal(t, SpeedFastMultiplier, G.SpeedMultiplier)
	require.Equal(t, "ja", G.Language)
	require.True(t, G.AI.Attack)
	require.True(t, G.AI.Scout)
	require.False(t, G.AI.Defend)
	require.False(t, G.EnableLandAirfield)
	require.True(t, G.EnableFogOfWar)
	require.False(t, G.EnableMinimap)
	require.False(t, G.DisplayHitEffects)
	require.False(t, G.DisplayDamageNumber)
	require.False(t, G.ForceDisplayState)
}
