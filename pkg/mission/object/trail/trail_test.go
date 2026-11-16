package trail

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

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
