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

// TestHasAirburst 验证对空近炸只在有爆炸半径且目标为飞机时启用。
// 场景：BlastRadius=0.32 的弹药依次把目标设为 plane、ship，再在 plane 下把半径清零。
// 预期：仅 (0.32, plane) 返回 true，ship 与零半径均 false；失败意味着对舰炮弹误触发空爆。
// 覆盖：三次调用 HasAirburst，锁死半径大于 0 与目标类型两个条件的与逻辑。
func TestHasAirburst(t *testing.T) {
	bt := &Bullet{BlastRadius: 0.32, TargetObjType: object.TypePlane}
	assert.True(t, bt.HasAirburst())

	bt.TargetObjType = object.TypeShip
	assert.False(t, bt.HasAirburst())

	bt.TargetObjType = object.TypePlane
	bt.BlastRadius = 0
	assert.False(t, bt.HasAirburst())
}

// TestNewCopiesTemplateByValue 验证 New 按值拷贝弹药模板，实例修改不回写全局 Map。
// 场景：注册 Damage=42、Diameter=127 模板，从 (3,4) 向 (5,6) 发射后把实例 Damage 改为 99。
// 预期：模板仍为 42；实例保留 Name/Type/Diameter，CurPos、Plunge、Speed、Life、阵营取发射参数。
// 覆盖：Cleanup 恢复 Map；断言模板与实例 10 处字段，防共享指针串改同型弹药。
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

// TestPlungeForRangeIsMonotonic 验证落角随射程百分比单调不减，近距离为平射。
// 场景：炮弹按射程从近到远计算落角；若出现回落，弹道会在中段突然变平，命中/危险界计算随之跳变。
// 预期：射程 0 与 PlungeRangeStart 落角为 0；射程 1 为 1；0..1 采样全程不减且在 [0,1]。
// 覆盖：断言三个端点，再以 0.02 步长遍历并比较相邻值。
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

// TestDangerSpaceShrinksWithPlunge 验证危险界随落角增大而收窄：平射无限，垂直吊射最小。
// 场景：平射炮弹沿瞄准线有很长危险界；大落角炮弹只在一个短区间内可命中，超过即落水。
// 预期：Plunge=0 返回 +Inf；Plunge=1 返回 dangerHeight/dangerTanRef；中等落角大于垂直落角。
// 覆盖：构造三颗不同 Plunge 的 Bullet，调用 DangerSpace 断言端点与单调性。
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

// TestPassedAimOnlyForPlunging 验证只有吊射弹药会在越过瞄准点一个危险界后判定落水。
// 场景：平射炮弹沿地面飞行，不应“越过”瞄准点；大落角炮弹在瞄准点后方一个危险界即落水。
// 预期：Plunge=0 永远 false；Plunge=1 在瞄准点后方 9 格为 true；距瞄准点 0.01 格时为 false。
// 覆盖：构造三颗 Bullet，设置 TargetPos/CurPos，调用 PassedAim 断言。
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

// TestShellTrailWaitsForHalfCell 验证炮弹尾迹按累计 0.5 格采样且前 10 拍不生成。
// 场景：TypeShell、Speed=0.2、ForwardAge=11；连续三次调用 GenTrails，不推进年龄。
// 预期：前两次累计 0.2/0.4 格返回 nil，第三次 0.6 格生成 1 条矩形尾迹，余量 0.1。
// 覆盖：断言前两次长度为 0、第三次长度为 1 且 Shape=TrailShapeRect。
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

// TestRocketTrailAlternatesEveryThirdTick 验证火箭尾迹每 3 拍一个圆点并交替尾焰/灰烟。
// 场景：TypeRocket、Speed=0.4、CurPos=(1,1)，把 ForwardAge 依次设为 2/3/4/6 调用 GenTrails。
// 预期：2、4 拍无尾迹；3 拍为 1 个 Orange 尾焰；6 拍为 1 个 DarkSilver 烟点。
// 覆盖：断言数量与颜色，覆盖模 3 取点和奇偶颜色分支。
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

// TestTurnTowardsTakesShortestArc 验证转向走最短弧并受单帧最大转角限制。
// 场景：炮弹/鱼雷从 350° 追 10° 时，应向右 +20° 而不是向左 -340°；大角度差要按 maxDelta 分帧转。
// 预期：顺时针跨 0° 得 355；逆时针得 5；0→90 且 maxDelta=2 得 2；maxDelta 足够时到 90；maxDelta=0 直指目标。
// 覆盖：表驱动五组 (current,target,maxDelta)，调用 turnTowards 断言结果。
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

// TestSteerTowardsRequiresGuidance 验证没有 Guidance 的普通弹药不会被转向，有制导才按最大转角转向。
// 场景：普通炮弹/炸弹应保持发射朝向；制导鱼雷/导弹才允许朝目标修正。
// 预期：无 Guidance 时 Rotation 保持 0；加 Pursuit/TurnRate=180 后朝 (10,0) 转到 10°。
// 覆盖：同一颗 Bullet 先无制导调用一次，再加 Guidance 调用一次，分别断言 Rotation。
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

// TestCanHitSubmergedByAmmoType 验证能否命中下潜潜艇由弹药类型与反潜标记决定。
// 场景：表驱动 8 组：深弹、反潜鱼雷、潜艇鱼雷、普通鱼雷、炮弹、炸弹、普通火箭、反潜火箭。
// 预期：深弹、反潜鱼雷、潜艇鱼雷、反潜火箭返回 true，其余 false；失败意味着反潜火箭失效或普通火箭误伤水下目标。
// 覆盖：逐项 assert.Equal 覆盖 Type 分支与 AntiSubmarine/SubmarineLaunched 两个标记。
func TestCanHitSubmergedByAmmoType(t *testing.T) {
	cases := []struct {
		name string
		bt   Bullet
		want bool
	}{
		{"depth charge", Bullet{Type: TypeDepthCharge}, true},
		{"anti-submarine torpedo", Bullet{Type: TypeTorpedo, AntiSubmarine: true}, true},
		{"submarine torpedo", Bullet{Type: TypeTorpedo, SubmarineLaunched: true}, true},
		{"surface torpedo", Bullet{Type: TypeTorpedo}, false},
		{"shell", Bullet{Type: TypeShell}, false},
		{"bomb", Bullet{Type: TypeBomb}, false},
		{"regular rocket", Bullet{Type: TypeRocket}, false},
		{"anti-submarine rocket", Bullet{Type: TypeRocket, AntiSubmarine: true}, true},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.bt.CanHitSubmerged(), tc.name)
	}
}

// TestTorpedoDetonationFactor 验证鱼雷诱爆系数随战斗部变重而下降，并夹在上下限内。
// 场景：表驱动航空雷 3400 伤害、基准 4500、重雷 6250、回天 24233、超轻 1000、零伤害。
// 预期：3400 → 1.32、4500 → 1、6250 → 0.72、24233 与 1000 分别夹到 0.45 与 2.2，
// 零伤害回落 1；失败意味着航空雷不再比舰载重雷更好拦截，或极端数值把概率放大到必中/归零。
// 覆盖：TorpedoDetonationFactor 的比值分支、Damage<=0 分支与 min/max 夹紧。
func TestTorpedoDetonationFactor(t *testing.T) {
	cases := []struct {
		name   string
		damage float64
		want   float64
	}{
		{"air torpedo 3400", 3400, 1.3235294117647058},
		{"reference 4500", 4500, 1},
		{"heavy type 93", 6250, 0.72},
		{"kaiten clamps low", 24233, torpedoDetonationFactorMin},
		{"tiny clamps high", 1000, torpedoDetonationFactorMax},
		{"unknown damage", 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bt := &Bullet{Type: TypeTorpedo, Damage: tc.damage}
			assert.InDelta(t, tc.want, bt.TorpedoDetonationFactor(), 1e-9)
		})
	}
}
