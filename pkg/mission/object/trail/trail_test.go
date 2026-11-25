package trail

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// TestBeginStopFadeAcceleratesExistingFade 验证停船尾迹可以用更短的帧数再次加速淡出。
// 场景：尾迹先按 30 帧停船淡出，舰船随后原地转向，管理器改用 12 帧再次调用。
// 预期：LifeReductionRate 提升到 CurLife/12 且不小于原速率；若第二次调用被 stopFade 提前返回，
// 旧航迹会在舰艉转开后继续悬在舷侧。
// 覆盖：构造 CurLife=100、LifeReductionRate=1 的尾迹，先 BeginStopFade(30) 再 BeginStopFade(12)，
// 断言速率从约 3.33 提升到约 8.33，并验证更长的帧数不会把速率降回去。
func TestBeginStopFadeAcceleratesExistingFade(t *testing.T) {
	trail := New(
		objPos.NewR(1, 2), textureImg.TrailShapeCircle,
		4, 0.1,
		100, 1,
		0, 0, nil,
	)
	trail.BeginStopFade(30)
	slowRate := trail.LifeReductionRate
	if slowRate <= 1 {
		t.Fatalf("first stop fade rate = %v, want faster than normal", slowRate)
	}

	trail.BeginStopFade(12)
	if trail.LifeReductionRate <= slowRate {
		t.Fatalf("accelerated fade rate = %v, want faster than %v", trail.LifeReductionRate, slowRate)
	}
	if trail.LifeReductionRate < 100.0/12-0.01 {
		t.Fatalf("accelerated fade rate = %v, want about %v", trail.LifeReductionRate, 100.0/12)
	}

	fastRate := trail.LifeReductionRate
	trail.BeginStopFade(30)
	if trail.LifeReductionRate != fastRate {
		t.Fatalf("slower fade request reduced rate to %v, want %v", trail.LifeReductionRate, fastRate)
	}
}

// TestReleaseClearsOwnerAndStopFade 验证尾迹对象池复用时清零归属与停船淡出状态。
// 场景：circle 尾迹设 OwnerUid="ship" 并 BeginStopFade(10)，Release 后 New 一条 rect 尾迹。
// 预期：New 复用同一指针，stopFade=false、OwnerUid=""，且 Pos=(3,4)、Shape=rect 来自新参数。
// 覆盖：先断言淡出已置位，再断言池命中、旧状态清空与新字段生效。
func TestReleaseClearsOwnerAndStopFade(t *testing.T) {
	first := New(
		objPos.NewR(1, 2), textureImg.TrailShapeCircle,
		4, 0.1,
		20, 1,
		0, 0, nil,
	)
	first.OwnerUid = "ship"
	first.BeginStopFade(10)
	if !first.stopFade {
		t.Fatal("stop fade was not armed")
	}

	Release(first)
	second := New(
		objPos.NewR(3, 4), textureImg.TrailShapeRect,
		2, 0.2,
		8, 1,
		0, 15, nil,
	)
	if second != first {
		t.Fatal("released trail was not reused")
	}
	if second.stopFade || second.OwnerUid != "" {
		t.Fatalf("reused trail kept stopFade=%v owner=%q", second.stopFade, second.OwnerUid)
	}
	if second.Pos.RX != 3 || second.Pos.RY != 4 || second.Shape != textureImg.TrailShapeRect {
		t.Fatal("reused trail did not take the new fields")
	}
}
