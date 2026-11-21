package settings

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// TestSpeedOptionsOnlyExposeThreePresets 验证设置页只暴露慢/标准/快三档速度且顺序固定。
// 场景：速度下拉按索引映射配置倍率，增删或换序会让已保存的 SpeedMultiplier 对应错档。
// 预期：len(speedOptions)=3，Value 依次等于 Slow/Standard/Fast 三个配置常量。
// 覆盖：以 config 常量构造 want 切片，先比长度再逐项比较 Value。
func TestSpeedOptionsOnlyExposeThreePresets(t *testing.T) {
	want := []float64{
		config.SpeedSlowMultiplier,
		config.SpeedStandardMultiplier,
		config.SpeedFastMultiplier,
	}
	if len(speedOptions) != len(want) {
		t.Fatalf("speed option count = %d, want %d", len(speedOptions), len(want))
	}
	for idx, value := range want {
		if speedOptions[idx].Value != value {
			t.Fatalf("speed option %d = %v, want %v", idx, speedOptions[idx].Value, value)
		}
	}
}

// TestClosedControlsShareTheWidestLabel 验证关闭态下拉宽度取所有候选标签的最宽者再加内边距。
// 场景：22px 下候选含 5/5、三档速度文案（当前语言字体）和四种语言 NativeName（各自语言字体）；
// 切换选中语言后宽度会变化，不同字体不能混用测量。
// 预期：closedControlWidth = max(速度类文案宽, 各语言 NativeName 宽) + controlPadX，而不是硬编码“标准”。
// 覆盖：用 comboFace 量 5/5 与 speedOptions，用 ForLanguage 的字体逐个量 NativeName，取最大值后与函数返回值比较。
func TestClosedControlsShareTheWidestLabel(t *testing.T) {
	comboFace := *newFace(22, font.LocalizedUI(font.Kai))

	want := int(text.Advance("5/5", comboFace) + 0.5)
	for _, option := range speedOptions {
		want = max(want, int(text.Advance(i18n.Text(option.Label), comboFace)+0.5))
	}
	for _, lang := range i18n.SupportedLanguages() {
		face := *newFace(22, font.ForLanguage(lang, font.Kai))
		want = max(want, int(text.Advance(lang.NativeName(), face)+0.5))
	}
	want += controlPadX

	if got := closedControlWidth(comboFace, 22); got != want {
		t.Fatalf("closed width = %d, want %d (widest label + padding)", got, want)
	}
}

// TestLanguageNameFaceUsesNativeFont 验证语言自称按语言选字体，中文使用楷体。
// 场景：语言下拉每个选项和关闭态都用该语言自己的字体；中文若继续用日文字体会与周围楷体不一致。
// 预期：zh-Hans→Kai、en→JetbrainsMono、ru→GolosText、ja→ZenKakuGothicNew，失败说明又退回单一复合字体。
// 覆盖：构造 buttonSize=22 的 UI，逐个语言调用 languageNameFace，断言 *text.GoTextFace 的 Source 指针。
func TestLanguageNameFaceUsesNativeFont(t *testing.T) {
	ui := &UI{metrics: settingsMetrics{buttonSize: 22}}
	for _, tc := range []struct {
		lang i18n.Language
		want *text.GoTextFaceSource
	}{
		{i18n.LanguageZhHans, font.Kai},
		{i18n.LanguageEnglish, font.JetbrainsMono},
		{i18n.LanguageRussian, font.GolosText},
		{i18n.LanguageJapanese, font.ZenKakuGothicNew},
	} {
		face := ui.languageNameFace(tc.lang)
		goFace, ok := (*face).(*text.GoTextFace)
		require.True(t, ok)
		require.Same(t, tc.want, goFace.Source)
	}
}

// TestStrategySummaryListsEnabledStrategies 验证 AI 策略摘要按已启用项计数并显示 n/5。
// 场景：设置页按钮需在收起状态提示当前启用了多少策略，用户可能全关或只开两项。
// 预期：Attack+Raid 为 true 时显示「2/5」，全 false 时显示「0/5」；失败意味着漏计或多计。
// 覆盖：构造 localAI 只开 Attack/Raid，再换成空结构体，两次调用 strategySummary 断言字符串。
func TestStrategySummaryListsEnabledStrategies(t *testing.T) {
	ui := &UI{localAI: config.AIStrategies{Attack: true, Raid: true}}
	if got := ui.strategySummary(); got != "2/5" {
		t.Fatalf("summary = %q", got)
	}
	ui.localAI = config.AIStrategies{}
	if got := ui.strategySummary(); got != "0/5" {
		t.Fatalf("empty summary = %q", got)
	}
}

// TestMinimapSettingDefaultsToOpen 验证设置页小地图开关默认显示「开」，并跟随已保存配置。
// 场景：老版本 game_settings.json5 没有 enableMinimap 字段，或启动早期 config.G 尚未加载；
// 设置页仍应显示开启，只有玩家显式保存 false 后才显示关闭。
// 预期：config.G 为 nil/默认配置时 minimapSetting 返回 true；EnableMinimap=false 时返回 false。
// 覆盖：构造三种 config.G 状态，直接调用 minimapSetting 并断言开关值。
func TestMinimapSettingDefaultsToOpen(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })

	config.G = nil
	if !minimapSetting() {
		t.Fatal("without settings the minimap switch should show open")
	}

	config.G = config.NewDefaultGameSettings()
	if !minimapSetting() {
		t.Fatal("default settings should show the minimap switch open")
	}

	config.G.EnableMinimap = false
	if minimapSetting() {
		t.Fatal("saved false should show the minimap switch closed")
	}
}

// TestDisplaySwitchesFollowConfig 验证命中特效/伤害数字/战舰状态三个开关默认开启并跟随配置。
// 场景：老配置文件缺少这些键，或玩家在设置页关闭某一项后重开；UI 必须反映已保存值。
// 预期：config.G 为 nil/默认配置时三个开关都为 true；分别置 false 后对应开关为 false。
// 覆盖：构造三种 config.G 状态，逐个调用 hitEffectsSetting/damageNumberSetting/displayStateSetting。
func TestDisplaySwitchesFollowConfig(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })

	config.G = nil
	if !hitEffectsSetting() || !damageNumberSetting() || !displayStateSetting() {
		t.Fatal("without settings the display switches should show open")
	}

	config.G = config.NewDefaultGameSettings()
	if !hitEffectsSetting() || !damageNumberSetting() || !displayStateSetting() {
		t.Fatal("default settings should show the display switches open")
	}

	config.G.DisplayHitEffects = false
	config.G.DisplayDamageNumber = false
	config.G.ForceDisplayState = false
	if hitEffectsSetting() || damageNumberSetting() || displayStateSetting() {
		t.Fatal("saved false should show the display switches closed")
	}
}

// TestSettingsMetricsKeepFooterOnScreen 验证设置页排版在 800×600 等矮屏下仍完整落在屏幕内。
// 场景：游戏页 8 行、视听页 7 行，卡片按 8 行预留高度，页脚位于卡片下方；标题/标签/页脚都可能把内容顶出屏幕。
// 预期：卡片左右不越界，页脚底部不超过屏幕高度，且单行控件宽度能放进卡片内容区。
// 覆盖：800×600、1024×768、1920×1080 三种 ScreenLayout × 7/8 行调用 newSettingsMetrics，断言几何边界。
func TestSettingsMetricsKeepFooterOnScreen(t *testing.T) {
	for _, screen := range []layout.ScreenLayout{
		{Width: 800, Height: 600},
		{Width: 1024, Height: 768},
		{Width: 1920, Height: 1080},
	} {
		for _, rows := range []int{7, 8} {
			m := newSettingsMetrics(screen, rows)
			if m.panelX < 0 || m.panelX+m.panelW > screen.Width {
				t.Fatalf(
					"%dx%d rows=%d panel X=%d W=%d out of screen",
					screen.Width, screen.Height, rows, m.panelX, m.panelW,
				)
			}
			if m.panelY < 0 || m.footerY+m.footerH > screen.Height {
				t.Fatalf(
					"%dx%d rows=%d panel Y=%d footer bottom=%d out of screen",
					screen.Width, screen.Height, rows, m.panelY, m.footerY+m.footerH,
				)
			}
			if rowWidth := m.panelW - 2*m.panelPad; m.controlW+2*m.rowPadX > rowWidth {
				t.Fatalf(
					"%dx%d rows=%d control W=%d does not fit row W=%d",
					screen.Width, screen.Height, rows, m.controlW, rowWidth,
				)
			}
		}
	}
}

// TestLoadFromSettingsRestoresDefaults 验证「恢复默认」读取默认设置后清掉全部本地改动。
// 场景：玩家已把速度、语言、背景音乐/战场音效、迷雾、AI、显示开关和尾流参数都改成非默认值，随后点击恢复默认。
// 预期：localValue=标准速度、语言 zh-Hans、BGM/战场音效回到默认 1.0、AI 全开、迷雾关闭、
// 小地图/显示/机场全开，尾流三项回到 1.0；失败意味着恢复默认只重置了部分页签。
// 覆盖：先手工填充 UI 的全部 local 字段，再 loadFromSettings(NewDefaultGameSettings) 后逐项断言。
func TestLoadFromSettingsRestoresDefaults(t *testing.T) {
	ui := &UI{
		localValue:         config.SpeedFastMultiplier,
		localLanguage:      i18n.LanguageEnglish,
		localBGMVolume:     0.2,
		localEffectsVolume: 0.6,
		localAI:            config.AIStrategies{Attack: true},
		localFog:           true,
		localMinimap:       false,
		localRam:           true,
		localLand:          false,
		localHitEffects:    false,
		localDamageNumbers: false,
		localShipState:     false,
		localWake:          config.WakeTrailSettings{Length: 2, Width: 0.5, Opacity: 0.4},
	}
	ui.loadFromSettings(config.NewDefaultGameSettings())

	if ui.localValue != config.SpeedStandardMultiplier || ui.localLanguage != i18n.LanguageZhHans {
		t.Fatalf("speed/language = %v/%v, want standard/zh-Hans", ui.localValue, ui.localLanguage)
	}
	if ui.localBGMVolume != config.DefaultAudioVolume || ui.localEffectsVolume != config.DefaultAudioVolume {
		t.Fatalf(
			"volumes = %v/%v, want default %v",
			ui.localBGMVolume, ui.localEffectsVolume, config.DefaultAudioVolume,
		)
	}
	if ui.localAI != config.DefaultAIStrategies() {
		t.Fatalf("AI = %+v, want all enabled", ui.localAI)
	}
	if ui.localFog || !ui.localMinimap || ui.localRam || !ui.localLand {
		t.Fatalf(
			"fog/minimap/ram/land = %v/%v/%v/%v, want false/true/false/true",
			ui.localFog, ui.localMinimap, ui.localRam, ui.localLand,
		)
	}
	if !ui.localHitEffects || !ui.localDamageNumbers || !ui.localShipState {
		t.Fatalf(
			"display switches = %v/%v/%v, want all true",
			ui.localHitEffects, ui.localDamageNumbers, ui.localShipState,
		)
	}
	if ui.localWake != config.DefaultWakeTrailSettings() {
		t.Fatalf("wake = %+v, want defaults", ui.localWake)
	}
}

// TestSaveWritesAllLocalValues 验证设置页保存会把所有页签的本地值写回配置并持久化。
// 场景：玩家在游戏/视听两个页签分别改动速度、语言、机场、AI、迷雾、小地图、撞击开关、战舰状态、BGM/战场音效、命中/伤害开关和尾流参数。
// 预期：config.G 与重新 Load 的配置文件都得到这些值；失败说明某个 local 字段漏写或 wakeTrail 未序列化。
// 覆盖：临时 ConfigBaseDir + 手工填充 UI local 字段（含 localRam=true），调用 save 后逐项断言 config.G，再 Load 断言持久化结果。
func TestSaveWritesAllLocalValues(t *testing.T) {
	previousDir, previousSettings := config.ConfigBaseDir, config.G
	previousBGM, previousEffects := audio.BGMVolume(), audio.EffectsVolume()
	config.ConfigBaseDir = t.TempDir()
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() {
		config.ConfigBaseDir, config.G = previousDir, previousSettings
		audio.SetBGMVolume(previousBGM)
		audio.SetEffectsVolume(previousEffects)
	})

	ui := &UI{
		localValue:         config.SpeedFastMultiplier,
		localLanguage:      i18n.LanguageEnglish,
		localBGMVolume:     0.2,
		localEffectsVolume: 0.6,
		localAI:            config.AIStrategies{Attack: true, Scout: true},
		localFog:           true,
		localMinimap:       false,
		localRam:           true,
		localLand:          false,
		localHitEffects:    false,
		localDamageNumbers: false,
		localShipState:     false,
		localWake:          config.WakeTrailSettings{Length: 1.8, Width: 0.7, Opacity: 1.2},
	}
	ui.save()

	if !ui.backPressed {
		t.Fatal("save should close the settings page")
	}
	require.Equal(t, config.SpeedFastMultiplier, config.G.SpeedMultiplier)
	require.Equal(t, "en", config.G.Language)
	require.Equal(t, 0.2, config.G.ResolvedBGMVolume())
	require.Equal(t, 0.6, config.G.ResolvedEffectsVolume())
	require.Equal(t, 0.2, audio.BGMVolume())
	require.Equal(t, 0.6, audio.EffectsVolume())
	require.Equal(t, config.AIStrategies{Attack: true, Scout: true}, config.G.AI)
	require.True(t, config.G.EnableFogOfWar)
	require.False(t, config.G.EnableMinimap)
	require.True(t, config.G.EnableRamming)
	require.False(t, config.G.EnableLandAirfield)
	require.False(t, config.G.DisplayHitEffects)
	require.False(t, config.G.DisplayDamageNumber)
	require.False(t, config.G.ForceDisplayState)
	require.Equal(t, config.WakeTrailSettings{Length: 1.8, Width: 0.7, Opacity: 1.2}, config.G.WakeTrail)

	config.G = nil
	config.LoadGameSettings()
	require.Equal(t, config.SpeedFastMultiplier, config.G.SpeedMultiplier)
	require.Equal(t, 0.2, config.G.ResolvedBGMVolume())
	require.Equal(t, 0.6, config.G.ResolvedEffectsVolume())
	require.True(t, config.G.EnableRamming)
	require.Equal(t, config.WakeTrailSettings{Length: 1.8, Width: 0.7, Opacity: 1.2}, config.G.WakeTrail)
}

// TestRestoreDefaultsSavesAndCloses 验证确认恢复会立即写回默认值、持久化并退出设置页。
// 场景：玩家把速度、语言、BGM/战场音效、迷雾、显示开关和尾流都改成非默认值，在二次确认弹窗点「确认恢复」。
// 预期：config.G 整个结构等于默认设置、音频两通道回到 1.0、backPressed=true，重新 Load 后仍为默认。
// 覆盖：临时 ConfigBaseDir + 先污染 config.G/UI local 字段，调用 restoreDefaults 后整体比较，再 Load 断言持久化。
func TestRestoreDefaultsSavesAndCloses(t *testing.T) {
	previousDir, previousSettings := config.ConfigBaseDir, config.G
	previousBGM, previousEffects := audio.BGMVolume(), audio.EffectsVolume()
	config.ConfigBaseDir = t.TempDir()
	config.G = &config.GameSettings{SpeedMultiplier: config.SpeedFastMultiplier, Language: "en"}
	t.Cleanup(func() {
		config.ConfigBaseDir, config.G = previousDir, previousSettings
		audio.SetBGMVolume(previousBGM)
		audio.SetEffectsVolume(previousEffects)
	})

	ui := &UI{
		localValue:         config.SpeedFastMultiplier,
		localLanguage:      i18n.LanguageEnglish,
		localBGMVolume:     0.2,
		localEffectsVolume: 0.6,
		localAI:            config.AIStrategies{Attack: true},
		localFog:           true,
		localMinimap:       false,
		localRam:           true,
		localLand:          false,
		localHitEffects:    false,
		localDamageNumbers: false,
		localShipState:     false,
		localWake:          config.WakeTrailSettings{Length: 2, Width: 0.5, Opacity: 0.4},
	}
	ui.restoreDefaults()

	require.True(t, ui.backPressed, "restore should close the settings page")
	require.False(t, ui.localRam)
	require.Equal(t, config.NewDefaultGameSettings(), config.G)
	require.Equal(t, config.DefaultAudioVolume, audio.BGMVolume())
	require.Equal(t, config.DefaultAudioVolume, audio.EffectsVolume())

	config.G = nil
	config.LoadGameSettings()
	require.Equal(t, config.NewDefaultGameSettings(), config.G)
}
