package human

import (
	"os"
	"strings"
	"testing"
)

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
