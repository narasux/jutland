package explosion

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

func TestHitEffectLifecycle(t *testing.T) {
	e := NewHitEffect(KindFlash, objPos.NewR(1, 2), 90, 0.35, 3, 7, faction.HumanAlpha)
	if !e.IsAlive() {
		t.Fatal("fresh explosion should be alive")
	}
	for i := 0; i < 3; i++ {
		e.Update()
	}
	if e.IsAlive() {
		t.Fatal("explosion should expire after Life frames")
	}
	if e.Frame() != 3 {
		t.Fatalf("frame = %d, want 3", e.Frame())
	}
}

// Life 至少为 1，避免零帧特效生成即过期。
func TestNewHitEffectClampsLife(t *testing.T) {
	e := NewHitEffect(KindFlash, objPos.NewR(0, 0), 0, 0.22, 0, 17, faction.HumanAlpha)
	if e.Life != 1 {
		t.Fatalf("life = %d, want 1", e.Life)
	}
	if !e.IsAlive() {
		t.Fatal("clamped explosion should be alive for one frame")
	}
}

// 只有命中闪光压在舰船之上：水花、尘土、鱼雷水柱都在海面 / 地面，画在舰体下才自然。
func TestOverShipsByKind(t *testing.T) {
	if !KindFlash.OverShips() {
		t.Fatal("hit flash should be drawn over ships")
	}
	for _, kind := range []Kind{KindSplash, KindDust, KindTorpedoPlume, KindRocketAirburst} {
		if kind.OverShips() {
			t.Fatalf("kind %v should stay under ships", kind)
		}
	}
}

// 运行时粒子方案：帧号就是 tick 数（序列帧只留给火箭空爆）。
func TestFrameFollowsAge(t *testing.T) {
	e := NewHitEffect(KindSplash, objPos.NewR(0, 0), 0, 0.5, 60, 11, faction.HumanAlpha)
	for i := 0; i < 5; i++ {
		e.Update()
	}
	if e.Frame() != 5 {
		t.Fatalf("frame = %d after 5 ticks, want 5", e.Frame())
	}
}

func TestNewRocketUsesAirburstKindAndScale(t *testing.T) {
	e := NewRocket(objPos.NewR(3, 4), 45, faction.ComputerAlpha)
	if e.Kind != KindRocketAirburst {
		t.Fatalf("kind = %v, want airburst", e.Kind)
	}
	if e.Size != RocketAirburstSize {
		t.Fatalf("size = %v, want %v", e.Size, RocketAirburstSize)
	}
	if e.Life != textureImg.MaxPlaneExplodeState {
		t.Fatalf("life = %d, want %d", e.Life, textureImg.MaxPlaneExplodeState)
	}
}
