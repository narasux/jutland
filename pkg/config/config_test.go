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
			name:     "traditional chinese",
			speed:    1.2,
			language: "zh-Hant",
			want:     SpeedStandardMultiplier,
			wantLang: "zh-Hant",
		},
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
		EnableShipCollision: true,
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
	require.True(t, G.EnableShipCollision)
	require.False(t, G.EnableMinimap)
	require.False(t, G.DisplayHitEffects)
	require.False(t, G.DisplayDamageNumber)
	require.False(t, G.ForceDisplayState)
}

// TestWakeTrailSettingsDefaultAndClamp 验证尾流参数缺省为 1.0，非法值按范围夹紧。
// 场景：旧 game_settings.json5 没有 wakeTrail 节，或玩家手写 0、NaN、超大倍率。
// 预期：零值/NaN 三项都回退 1.0；0.1 与 9.0 分别被夹到 Length/Width/Opacity 的上下限。
// 覆盖：表驱动 Normalize 五种输入，逐项断言 Length/Width/Opacity，并检查默认设置本身。
func TestWakeTrailSettingsDefaultAndClamp(t *testing.T) {
	defaults := NewDefaultGameSettings()
	require.Equal(t, DefaultWakeTrailSettings(), defaults.WakeTrail)

	cases := []struct {
		name string
		in   WakeTrailSettings
		want WakeTrailSettings
	}{
		{
			name: "missing fields fall back to one",
			in:   WakeTrailSettings{},
			want: DefaultWakeTrailSettings(),
		},
		{
			name: "NaN falls back to one",
			in:   WakeTrailSettings{Length: math.NaN(), Width: math.NaN(), Opacity: math.NaN()},
			want: DefaultWakeTrailSettings(),
		},
		{
			name: "below range clamps",
			in:   WakeTrailSettings{Length: 0.1, Width: 0.1, Opacity: 0.1},
			want: WakeTrailSettings{
				Length: WakeTrailLengthMin, Width: WakeTrailWidthMin, Opacity: WakeTrailOpacityMin,
			},
		},
		{
			name: "above range clamps",
			in:   WakeTrailSettings{Length: 9, Width: 9, Opacity: 9},
			want: WakeTrailSettings{
				Length: WakeTrailLengthMax, Width: WakeTrailWidthMax, Opacity: WakeTrailOpacityMax,
			},
		},
		{
			name: "valid values pass through",
			in:   WakeTrailSettings{Length: 1.6, Width: 1.4, Opacity: 0.8},
			want: WakeTrailSettings{Length: 1.6, Width: 1.4, Opacity: 0.8},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.in.Normalize())
		})
	}
}

// TestValidateNormalizesWakeTrail 验证 Load/Save 共用的 validate 会写回尾流默认值与范围。
// 场景：GameSettings 由测试或旧配置直接构造，WakeTrail 零值不能原样进入游戏循环。
// 预期：validate 后 WakeTrail 等于 DefaultWakeTrailSettings；越界长度被夹到 2.50。
// 覆盖：先构造零值设置，再构造只有 Length=9 的设置，各调用一次 validate 后断言字段。
func TestValidateNormalizesWakeTrail(t *testing.T) {
	settings := &GameSettings{SpeedMultiplier: SpeedStandardMultiplier, Language: "zh-Hans"}
	settings.validate()
	require.Equal(t, DefaultWakeTrailSettings(), settings.WakeTrail)

	settings = &GameSettings{
		SpeedMultiplier: SpeedStandardMultiplier,
		Language:        "zh-Hans",
		WakeTrail:       WakeTrailSettings{Length: 9},
	}
	settings.validate()
	require.Equal(t, WakeTrailLengthMax, settings.WakeTrail.Length)
	require.Equal(t, WakeTrailDefaultScale, settings.WakeTrail.Width)
	require.Equal(t, WakeTrailDefaultScale, settings.WakeTrail.Opacity)
}

// TestSaveGameSettingsRoundTripsWakeTrail 验证尾流参数能写入并原样读回配置文件。
// 场景：玩家在设置页把长度/宽度/浓度调成非默认值后保存，下次启动应继续生效。
// 预期：Load 后三项与保存前完全一致；失败说明 wakeTrail 没有序列化或被 validate 改写。
// 覆盖：临时配置目录构造 Length=1.8/Width=0.7/Opacity=1.2，Save 后清空 G 再 Load 断言三项。
func TestSaveGameSettingsRoundTripsWakeTrail(t *testing.T) {
	_, cleanup := withTempConfigDir(t)
	defer cleanup()

	G = &GameSettings{
		SpeedMultiplier: SpeedStandardMultiplier,
		Language:        "zh-Hans",
		AI:              DefaultAIStrategies(),
		EnableMinimap:   true,
		WakeTrail:       WakeTrailSettings{Length: 1.8, Width: 0.7, Opacity: 1.2},
	}
	require.NoError(t, SaveGameSettings())

	G = nil
	LoadGameSettings()
	require.NotNil(t, G)
	require.Equal(t, WakeTrailSettings{Length: 1.8, Width: 0.7, Opacity: 1.2}, G.WakeTrail)
}

// TestAudioVolumesDefaultAndClamp 验证 BGM/战场音效音量缺省为 1.0、显式 0 保持静音、越界值夹紧。
// 场景：旧 game_settings.json5 没有 bgmVolume/effectsVolume（指针 nil），玩家也可能把任一滑杆拖到 0 或手写 2.0。
// 预期：nil 回退 1.00、NaN 视为静音 0、显式 0 保留、负数夹到 0、大于 1 夹到 1，两条通道互不影响。
// 覆盖：表驱动六种输入分别走 ResolvedBGMVolume 与 ResolvedEffectsVolume，并检查默认设置、validate 写回与交叉赋值。
func TestAudioVolumesDefaultAndClamp(t *testing.T) {
	floatPtr := func(value float64) *float64 { return &value }

	defaults := NewDefaultGameSettings()
	require.Equal(t, DefaultAudioVolume, defaults.ResolvedBGMVolume())
	require.Equal(t, DefaultAudioVolume, defaults.ResolvedEffectsVolume())
	require.NotNil(t, defaults.BGMVolume)
	require.NotNil(t, defaults.EffectsVolume)

	settings := &GameSettings{SpeedMultiplier: SpeedStandardMultiplier, Language: "zh-Hans"}
	settings.validate()
	require.NotNil(t, settings.BGMVolume, "validate should materialize the default BGM volume")
	require.NotNil(t, settings.EffectsVolume, "validate should materialize the default effects volume")
	require.Equal(t, DefaultAudioVolume, settings.ResolvedBGMVolume())
	require.Equal(t, DefaultAudioVolume, settings.ResolvedEffectsVolume())

	for _, tc := range []struct {
		name string
		in   *float64
		want float64
	}{
		{name: "missing keeps default", in: nil, want: DefaultAudioVolume},
		{name: "negative clamps to mute", in: floatPtr(-0.2), want: 0},
		{name: "explicit mute stays", in: floatPtr(0), want: 0},
		{name: "valid passes through", in: floatPtr(0.35), want: 0.35},
		{name: "above one clamps", in: floatPtr(1.5), want: 1},
		{name: "NaN mutes", in: floatPtr(math.NaN()), want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, (&GameSettings{BGMVolume: tc.in}).ResolvedBGMVolume())
			require.Equal(t, tc.want, (&GameSettings{EffectsVolume: tc.in}).ResolvedEffectsVolume())
		})
	}

	settings = &GameSettings{}
	settings.SetBGMVolume(0)
	settings.SetEffectsVolume(0.6)
	require.Equal(t, 0.0, settings.ResolvedBGMVolume())
	require.Equal(t, 0.6, settings.ResolvedEffectsVolume())
}

// TestSaveGameSettingsRoundTripsAudioVolumes 验证 BGM/战场音效音量（含显式静音 0）能写入并原样读回。
// 场景：玩家把 BGM 拖到 0、战场音效调到 0.6 保存；若把 0 当作“未配置”，重启后会突然恢复满音量。
// 预期：Load 后两个指针非 nil，BGM=0、Effects=0.6；失败说明音量未序列化或被 validate 改写。
// 覆盖：withTempConfigDir 下写入 BGM=0 / Effects=0.6，Save 后清空 G 再 Load 断言两个 Resolved 值。
func TestSaveGameSettingsRoundTripsAudioVolumes(t *testing.T) {
	_, cleanup := withTempConfigDir(t)
	defer cleanup()

	G = &GameSettings{
		SpeedMultiplier: SpeedStandardMultiplier,
		Language:        "zh-Hans",
		AI:              DefaultAIStrategies(),
		EnableMinimap:   true,
	}
	G.SetBGMVolume(0)
	G.SetEffectsVolume(0.6)
	require.NoError(t, SaveGameSettings())

	G = nil
	LoadGameSettings()
	require.NotNil(t, G)
	require.NotNil(t, G.BGMVolume)
	require.NotNil(t, G.EffectsVolume)
	require.Equal(t, 0.0, G.ResolvedBGMVolume())
	require.Equal(t, 0.6, G.ResolvedEffectsVolume())
}

// TestTutorialCompletedLookup 验证教学完成进度按任务名查询且不受其他任务影响。
// 场景：设置里只记录 TutorialBasicControls，查询同一任务、另一个任务和非教学任务。
// 预期：IsTutorialCompleted 只对已记录任务返回 true；失败意味着任务选择页会把所有教学都标成已完成。
// 覆盖：临时替换全局 G，直接设置 TutorialCompleted 后查询三个名称，Cleanup 恢复旧指针。
func TestTutorialCompletedLookup(t *testing.T) {
	previous := G
	t.Cleanup(func() { G = previous })

	G = NewDefaultGameSettings()
	require.False(t, IsTutorialCompleted("TutorialBasicControls"))

	G.TutorialCompleted = []string{"TutorialBasicControls"}
	require.True(t, IsTutorialCompleted("TutorialBasicControls"))
	require.False(t, IsTutorialCompleted("TutorialFormation"))
	require.False(t, IsTutorialCompleted("PearlHarbor1941"))
}
