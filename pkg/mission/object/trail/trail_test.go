package trail

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

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
