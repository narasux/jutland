package settings

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
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
// 场景：22px 字体下候选含 5/5、三档速度文案和当前语言 NativeName；切换语言后最宽标签会变化。
// 预期：closedControlWidth = max(Advance(label)) + controlPadX，而不是硬编码“标准”。
// 覆盖：从 speedOptions、5/5、NativeName 重建候选集，逐个算宽度取最大值后与函数返回值比较。
func TestClosedControlsShareTheWidestLabel(t *testing.T) {
	face := font.LanguageSelectorFace(22)
	labels := []string{"5/5"}
	for _, option := range speedOptions {
		labels = append(labels, i18n.Text(option.Label))
	}
	labels = append(labels, i18n.CurrentLanguage().NativeName())

	want := 0
	for _, label := range labels {
		want = max(want, int(text.Advance(label, face)+0.5))
	}
	want += controlPadX

	if got := closedControlWidth(face); got != want {
		t.Fatalf("closed width = %d, want %d (widest label + padding)", got, want)
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
