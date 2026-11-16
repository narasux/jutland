package unit

import "testing"

// TestShipAnimationSequenceAndIdle 验证舰船动画按 FrameTicks 取帧，停船时回到 idle 帧。
// 场景：剑鱼等有尾迹动画的舰船在航行时按 8 帧序列播放；速度降为 0 后应显示停泊帧而不是继续摆动。
// 预期：初始 01；AnimationAge=24（6 tick/帧）取 05；CurSpeed=0 取 IdleTopFrame=03。
// 覆盖：直接修改 AnimationAge/CurSpeed，调用 CurrentTopImageName 逐点断言。
func TestShipAnimationSequenceAndIdle(t *testing.T) {
	ship := &BattleShip{
		Name: "swordfish", CurSpeed: 1,
		Animation: ShipAnimation{
			TopFrames:    []string{"01", "02", "03", "04", "05", "04", "03", "02"},
			IdleTopFrame: "03", FrameTicks: 6,
		},
	}
	if got := ship.CurrentTopImageName(); got != "01" {
		t.Fatalf("initial frame=%s", got)
	}
	ship.AnimationAge = 24
	if got := ship.CurrentTopImageName(); got != "05" {
		t.Fatalf("middle frame=%s", got)
	}
	ship.CurSpeed = 0
	if got := ship.CurrentTopImageName(); got != "03" {
		t.Fatalf("idle frame=%s", got)
	}
}
