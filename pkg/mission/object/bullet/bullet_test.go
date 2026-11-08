package bullet

import (
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/object/trail"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

func TestHasAirburst(t *testing.T) {
	bt := &Bullet{BlastRadius: 0.32, TargetObjType: object.TypePlane}
	assert.True(t, bt.HasAirburst())

	bt.TargetObjType = object.TypeShip
	assert.False(t, bt.HasAirburst())

	bt.TargetObjType = object.TypePlane
	bt.BlastRadius = 0
	assert.False(t, bt.HasAirburst())
}

func TestNewCopiesTemplateByValue(t *testing.T) {
	const name = "test/shallow-copy"
	original, existed := Map[name]
	t.Cleanup(func() {
		if existed {
			Map[name] = original
			return
		}
		delete(Map, name)
	})

	Map[name] = &Bullet{
		Name:     name,
		Type:     TypeShell,
		Diameter: 127,
		Damage:   42,
		CurPos:   objPos.NewR(99, 99),
	}

	b := New(
		name,
		objPos.NewR(3, 4),
		objPos.NewR(5, 6),
		"shooter",
		object.TypeShip,
		faction.HumanAlpha,
		object.TypeShip,
		1.5,
		20,
		0.4,
	)
	b.Damage = 99

	assert.Equal(t, 42.0, Map[name].Damage)
	assert.Equal(t, name, b.Name)
	assert.Equal(t, TypeShell, b.Type)
	assert.Equal(t, 127, b.Diameter)
	assert.Equal(t, 3.0, b.CurPos.RX)
	assert.Equal(t, 4.0, b.CurPos.RY)
	assert.Equal(t, 0.4, b.Plunge)
	assert.Equal(t, 1.5, b.Speed)
	assert.Equal(t, 20, b.Life)
	assert.Equal(t, faction.HumanAlpha, b.BelongPlayer)
}

// 落角必须随射程单调不减，且近距离完全是平射。
func TestPlungeForRangeIsMonotonic(t *testing.T) {
	if got := PlungeForRange(0); got != 0 {
		t.Fatalf("plunge at 0 range = %v, want 0", got)
	}
	if got := PlungeForRange(PlungeRangeStart); got != 0 {
		t.Fatalf("plunge at start range = %v, want 0", got)
	}
	if got := PlungeForRange(1); got != 1 {
		t.Fatalf("plunge at max range = %v, want 1", got)
	}
	prev := -1.0
	for p := 0.0; p <= 1.0; p += 0.02 {
		got := PlungeForRange(p)
		if got < prev {
			t.Fatalf("plunge decreases at rangePercent=%v: %v < %v", p, got, prev)
		}
		if got < 0 || got > 1 {
			t.Fatalf("plunge out of range at %v: %v", p, got)
		}
		prev = got
	}
}

// 危险界随落角收窄：平射不限制，吊射收敛到几十米。
func TestDangerSpaceShrinksWithPlunge(t *testing.T) {
	flat := &Bullet{Plunge: 0}
	if !math.IsInf(flat.DangerSpace(), 1) {
		t.Fatalf("flat danger space = %v, want +Inf", flat.DangerSpace())
	}
	steep := &Bullet{Plunge: 1}
	if got, want := steep.DangerSpace(), dangerHeight/dangerTanRef; math.Abs(got-want) > 1e-9 {
		t.Fatalf("steep danger space = %v, want %v", got, want)
	}
	mid := &Bullet{Plunge: PlungeForRange(0.55)}
	if mid.DangerSpace() <= steep.DangerSpace() {
		t.Fatalf("mid danger space = %v, want larger than steep %v", mid.DangerSpace(), steep.DangerSpace())
	}
}

// 平射弹药永远不算越过瞄准点，吊射弹药越过一个危险界即落水。
func TestPassedAimOnlyForPlunging(t *testing.T) {
	flat := &Bullet{Plunge: 0, Rotation: 0, TargetPos: objPos.NewR(10, 10), CurPos: objPos.NewR(10, 1)}
	if flat.PassedAim() {
		t.Fatal("flat bullet should never pass its aim point")
	}
	steep := &Bullet{Plunge: 1, Rotation: 0, TargetPos: objPos.NewR(10, 10), CurPos: objPos.NewR(10, 1)}
	if !steep.PassedAim() {
		t.Fatal("steep bullet should pass its aim point after a danger space")
	}
	near := &Bullet{Plunge: 1, Rotation: 0, TargetPos: objPos.NewR(10, 10), CurPos: objPos.NewR(10, 9.99)}
	if near.PassedAim() {
		t.Fatal("steep bullet at the aim point should not have passed it yet")
	}
}

func TestShellTrailWaitsForHalfCell(t *testing.T) {
	bt := &Bullet{
		Type:       TypeShell,
		ForwardAge: 11,
		Speed:      0.2,
		HitObjType: object.TypeNone,
	}
	if trails := bt.GenTrails(); trails != nil {
		t.Fatalf("first step trails = %d, want 0", len(trails))
	}
	if trails := bt.GenTrails(); trails != nil {
		t.Fatalf("second step trails = %d, want 0", len(trails))
	}
	trails := bt.GenTrails()
	if len(trails) != 1 {
		t.Fatalf("third step trails = %d, want 1", len(trails))
	}
	if trails[0].Shape != textureImg.TrailShapeRect {
		t.Fatalf("shape = %v, want rect", trails[0].Shape)
	}
}

func TestRocketTrailAlternatesEveryThirdTick(t *testing.T) {
	bt := &Bullet{
		Type:       TypeRocket,
		Speed:      0.4,
		HitObjType: object.TypeNone,
		CurPos:     objPos.NewR(1, 1),
	}
	bt.ForwardAge = 2
	if trails := bt.GenTrails(); trails != nil {
		t.Fatalf("age 2 trails = %d, want 0", len(trails))
	}
	bt.ForwardAge = 3
	flame := bt.GenTrails()
	if len(flame) != 1 || flame[0].Color != colorx.Orange {
		t.Fatalf("age 3 trails = %d color %v, want one orange flame", len(flame), colorOf(flame))
	}
	bt.ForwardAge = 4
	if trails := bt.GenTrails(); trails != nil {
		t.Fatalf("age 4 trails = %d, want 0", len(trails))
	}
	bt.ForwardAge = 6
	smoke := bt.GenTrails()
	if len(smoke) != 1 || smoke[0].Color != colorx.DarkSilver {
		t.Fatalf("age 6 trails = %d color %v, want one smoke puff", len(smoke), colorOf(smoke))
	}
}

func colorOf(trails []*trail.Trail) color.Color {
	if len(trails) == 0 {
		return nil
	}
	return trails[0].Color
}

// 转向必须走最短弧：350° 到 10° 是向右 +20°，而不是向左 -340°。
func TestTurnTowardsTakesShortestArc(t *testing.T) {
	testCases := []struct {
		name     string
		current  float64
		target   float64
		maxDelta float64
		want     float64
	}{
		{name: "clockwise wrap", current: 350, target: 10, maxDelta: 5, want: 355},
		{name: "counter-clockwise wrap", current: 10, target: 350, maxDelta: 5, want: 5},
		{name: "clamp to max delta", current: 0, target: 90, maxDelta: 2, want: 2},
		{name: "reaches target when allowed", current: 0, target: 90, maxDelta: 200, want: 90},
		{name: "zero delta aims directly", current: 0, target: 90, maxDelta: 0, want: 90},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := turnTowards(testCase.current, testCase.target, testCase.maxDelta)
			if math.Abs(got-testCase.want) > 1e-9 {
				t.Fatalf("turnTowards(%v, %v, %v) = %v, want %v",
					testCase.current, testCase.target, testCase.maxDelta, got, testCase.want)
			}
		})
	}
}

// 普通弹药不能被导引：没有 Guidance 时 SteerTowards 不改变朝向。
func TestSteerTowardsRequiresGuidance(t *testing.T) {
	bt := &Bullet{CurPos: objPos.NewR(0, 0), Rotation: 0}
	bt.SteerTowards(objPos.NewR(10, 0), 10)
	if bt.Rotation != 0 {
		t.Fatalf("unguided rotation = %v, want 0", bt.Rotation)
	}

	bt.Guidance = &Guidance{Mode: GuidanceModePursuit, TurnRate: 180}
	bt.SteerTowards(objPos.NewR(10, 0), 10)
	if math.Abs(bt.Rotation-10) > 1e-9 {
		t.Fatalf("guided rotation = %v, want 10", bt.Rotation)
	}
}
