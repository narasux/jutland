package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

// 小地图默认打开：配置没加载或没关掉时，开局都直接展开战术面板。
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
