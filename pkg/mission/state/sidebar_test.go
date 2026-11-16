package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

// TestMinimapFromSettingsDefaultsToOpen 验证小地图默认展开，显式关闭后才折叠。
// 场景：老版本 game_settings.json5 没有 enableMinimap 字段，或启动早期 config.G 尚未加载；
// 此时开局仍应直接展开战术面板，否则玩家会以为小地图功能丢失。
// 预期：config.G 为 nil/默认配置时返回 true；EnableMinimap=false 时返回 false。
// 覆盖：构造上述三种 config.G 状态，直接调用 minimapFromSettings 并断言布尔值。
func TestMinimapFromSettingsDefaultsToOpen(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })

	config.G = nil
	if !minimapFromSettings() {
		t.Fatal("without settings the minimap should stay open")
	}

	config.G = config.NewDefaultGameSettings()
	if !minimapFromSettings() {
		t.Fatal("default settings should open the minimap")
	}

	config.G.EnableMinimap = false
	if minimapFromSettings() {
		t.Fatal("disabling the minimap should keep the panel collapsed")
	}
}
