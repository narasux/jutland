package explosion

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// TestHitEffectLifecycle 验证命中特效按 tick 计数，Age 达到 Life 即停止绘制。
// 场景：NewHitEffect 创建 KindFlash，寿命 3 帧，连续调用 Update 三次。
// 预期：创建时 IsAlive=true；三次后 IsAlive=false 且 Frame()=3。
// 覆盖：断言存活、Age==Life 的死亡边界与帧号，锁死特效不会滞留。
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

// TestNewHitEffectClampsLife 验证 Life=0 的命中特效被钳到 1 帧，避免生成即过期。
// 场景：配置/调用方传入 Life=0；若直接使用，特效永远不会被绘制，命中反馈丢失。
// 预期：Life=1，且 IsAlive 在第一帧为 true。
// 覆盖：NewHitEffect(..., life=0, ...) 后断言 Life 与 IsAlive。
func TestNewHitEffectClampsLife(t *testing.T) {
	e := NewHitEffect(KindFlash, objPos.NewR(0, 0), 0, 0.22, 0, 17, faction.HumanAlpha)
	if e.Life != 1 {
		t.Fatalf("life = %d, want 1", e.Life)
	}
	if !e.IsAlive() {
		t.Fatal("clamped explosion should be alive for one frame")
	}
}

// TestOverShipsByKind 验证只有命中闪光绘制在舰船之上，水花/尘土/鱼雷水柱/火箭空爆在舰体之下。
// 场景：爆炸层按 Kind.OverShips 决定绘制顺序；水花画在舰体上会像船在冒水，尘土会盖住甲板。
// 预期：KindFlash.OverShips=true；Splash/Dust/TorpedoPlume/RocketAirburst 都为 false。
// 覆盖：逐 Kind 调用 OverShips 断言布尔值。
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

// TestFrameFollowsAge 验证运行时粒子特效的帧号等于已推进 tick 数。
// 场景：非序列帧特效（水花）用 Age 驱动粒子，Frame 必须跟随 Age；若帧号不前进，粒子动画会停住。
// 预期：Life=60 的特效推进 5 帧后 Frame()==5。
// 覆盖：NewHitEffect 后循环 Update 5 次，断言 Frame。
func TestFrameFollowsAge(t *testing.T) {
	e := NewHitEffect(KindSplash, objPos.NewR(0, 0), 0, 0.5, 60, 11, faction.HumanAlpha)
	for i := 0; i < 5; i++ {
		e.Update()
	}
	if e.Frame() != 5 {
		t.Fatalf("frame = %d after 5 ticks, want 5", e.Frame())
	}
}

// TestNewRocketUsesAirburstKindAndScale 验证火箭空爆固定使用空爆种类、尺寸与飞机爆炸帧寿命。
// 场景：以 (3,4)、45°、ComputerAlpha 调 NewRocket，内部转调 NewHitEffect。
// 预期：Kind=KindRocketAirburst、Size=0.8、Life=39；失败意味着复用了闪光/水花资源。
// 覆盖：断言种类、RocketAirburstSize 与 MaxPlaneExplodeState 三个字段。
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
