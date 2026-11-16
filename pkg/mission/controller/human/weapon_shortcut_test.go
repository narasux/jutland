package human

import (
	"os"
	"strings"
	"testing"
)

// TestHumanCombatHandlerHasNoWeaponToggleShortcuts 验证玩家战斗处理器不再有武器切换快捷键。
// 场景：武器切换改为 UI 面板后，human.go 里遗留的 handleWeapon 与 Q/W/E/R/T
// 按键会与新的点击操作并存，玩家按键会意外切换武器。
// 预期：源码文本不含 "handleWeapon" 及 ebiten.KeyQ/W/E/R/T；命中任一项即失败，
// 提示旧键盘路径仍在生效。
// 覆盖：os.ReadFile 读取同目录 human.go 后做子串扫描，不启动 Ebiten。
func TestHumanCombatHandlerHasNoWeaponToggleShortcuts(t *testing.T) {
	source, err := os.ReadFile("human.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Contains(text, "handleWeapon") {
		t.Fatal("legacy keyboard weapon toggle handler still exists")
	}
	for _, key := range []string{"ebiten.KeyQ", "ebiten.KeyW", "ebiten.KeyE", "ebiten.KeyR", "ebiten.KeyT"} {
		if strings.Contains(text, key) {
			t.Fatalf("legacy weapon shortcut %s still exists in human combat handler", key)
		}
	}
}
