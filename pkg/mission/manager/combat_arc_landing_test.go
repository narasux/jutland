package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func newShellImpactManager(ship *objUnit.BattleShip, bullet *objBullet.Bullet) *MissionManager {
	return &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships:             map[string]*objUnit.BattleShip{ship.Uid: ship},
				ForwardingBullets: []*objBullet.Bullet{bullet},
			},
		},
	}
}

// resolveShell 反复推进，直到炮弹结算完毕（命中或落水）。
func resolveShell(m *MissionManager, maxTicks int) {
	for i := 0; i < maxTicks; i++ {
		m.updateShotBullets()
		if len(m.state.Arena.ForwardingBullets) == 0 {
			return
		}
	}
}

// newTestShip 造一条 51.2 x 12.8 像素（0.4 x 0.1 地图格）的靶舰，
// 航向与炮弹方向一致，舰体沿 Y 轴展开。
func newTestShip(x, y float64) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       51.2,
		Width:        12.8,
		CurPos:       objPos.NewR(x, y),
		BelongPlayer: faction.ComputerAlpha,
	}
}

// newTestShell 造一发沿 -Y 方向飞行的炮弹（Rotation 0）。
func newTestShell(plunge, startY, targetY float64) *objBullet.Bullet {
	return &objBullet.Bullet{
		Type:           objBullet.TypeShell,
		Damage:         10,
		Plunge:         plunge,
		TargetObjType:  object.TypeShip,
		Shooter:        "shooter",
		ShooterObjType: object.TypeShip,
		BelongPlayer:   faction.HumanAlpha,
		CurPos:         objPos.NewR(10, startY),
		TargetPos:      objPos.NewR(10, targetY),
		Speed:          0.5,
		Life:           60,
	}
}

// TestFlatShellUsesSweptCollision 验证平射炮弹按本拍弹道线段扫掠命中，落点差一个舰身也能打中。
// 场景：0.4×0.1 格靶舰在 (10,10)，Plunge=0、速度 0.5 的炮弹从 y=12 瞄准 y=9（舰艏前方 1 格），推进 200 拍。
// 预期：target.CurHP 从 100 降到 90，bt.HitObjType 为 TypeShip。
// 覆盖：updateShotBullets 用 IsSegmentIntersectRotatedRectangle 做扫掠相交，平射 DangerSpace=+Inf 不限制。
func TestFlatShellUsesSweptCollision(t *testing.T) {
	target := newTestShip(10, 10)
	// 瞄准点在舰艏前方 1 格，平射危险界不限制，扫过舰体即命中
	bullet := newTestShell(0, 12, 9)

	resolveShell(newShellImpactManager(target, bullet), 200)

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 for a swept flat hit", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}

// TestPlungingShellUsesLandingPoint 验证吊射炮弹的落点本身在舰体内时按落点命中。
// 场景：靶舰在 (10,10)，Plunge=1 的炮弹从 y=12 瞄准 y=10（舰体中心），推进 200 拍。
// 预期：target.CurHP 降到 90，bt.HitObjType 为 TypeShip。
// 覆盖：吊射落点/扫掠相交分支与 DangerSpace=0.20/0.70≈0.286 格内的命中判定。
func TestPlungingShellUsesLandingPoint(t *testing.T) {
	target := newTestShip(10, 10)
	bullet := newTestShell(1, 12, 10)

	resolveShell(newShellImpactManager(target, bullet), 200)

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 after final-point hit", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}

// TestPlungingShellStraddleWithinDangerSpaceHits 验证吊射落点擦着舰尾落水但仍在危险界内时算跨射命中。
// 场景：舰尾在 y=9.8，Plunge=1 的炮弹从 y=12 瞄准 y=9.75，已落在舰体外 0.05 格，但小于危险界 0.286 格。
// 预期：炮弹越过瞄准点后扫到舰体，target.CurHP 降到 90。
// 覆盖：DangerSpace=0.20/(1×0.70)≈0.286 格内的跨射判定与 PassedAim 前的扫掠相交。
func TestPlungingShellStraddleWithinDangerSpaceHits(t *testing.T) {
	target := newTestShip(10, 10)
	// 舰尾在 y = 9.8，落点 9.75 已经落在舰体之外，但在 0.286 格的危险界之内
	bullet := newTestShell(1, 12, 9.75)

	resolveShell(newShellImpactManager(target, bullet), 200)

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 for a straddle inside the danger space", target.CurHP)
	}
}

// TestPlungingShellBeyondDangerSpaceMisses 验证吊射打远一个舰身、超出危险界时判定脱靶并落水。
// 场景：靶舰在 (10,10)，Plunge=1 的炮弹从 y=12 瞄准 y=9.2，离舰心 0.8 格，大于危险界 0.286 格。
// 预期：target.CurHP 保持 100，bt.HitObjType 为 TypeWater。
// 覆盖：沿弹道方向 |along| > DangerSpace 的剔除与 PassedAim 的落水分支。
func TestPlungingShellBeyondDangerSpaceMisses(t *testing.T) {
	target := newTestShip(10, 10)
	bullet := newTestShell(1, 12, 9.2)

	resolveShell(newShellImpactManager(target, bullet), 200)

	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100 for a miss beyond the danger space", target.CurHP)
	}
	if bullet.HitObjType != object.TypeWater {
		t.Fatalf("hit type = %v, want water", bullet.HitObjType)
	}
}

// TestFlatShellIgnoresLargeRangeError 验证平射的沿弹道容差不受危险界限制，舰体离瞄准点 2 格仍命中。
// 场景：靶舰在 (10,10)，Plunge=0 的炮弹从 y=14 瞄准 y=8（越过舰心 2 格），推进 200 拍。
// 预期：弹道线段扫过舰体，target.CurHP 降到 90。
// 覆盖：Plunge<=0 时 DangerSpace 返回 +Inf，平射不做打远/打近剔除。
func TestFlatShellIgnoresLargeRangeError(t *testing.T) {
	target := newTestShip(10, 10)
	bullet := newTestShell(0, 14, 8)

	resolveShell(newShellImpactManager(target, bullet), 200)

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90, flat fire should still sweep the hull", target.CurHP)
	}
}
