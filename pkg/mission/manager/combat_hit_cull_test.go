package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestDirectShellHitsCornerBeyondHalfLength 验证炮弹擦过舷侧角点时不能按半舰长提前剔除。
// 场景：256×256 像素靶舰中心在 (10,10)，炮弹从 (10.99,10.99) 以 135°、速度 1 飞出，到舰心约 2.4 格。
// 预期：距离超过半长 1 格加弹速 1 格，但小于对角线半径 1.414 加弹速；应命中并扣 10 HP。
// 覆盖：先断言样本在半长外、对角线内，再跑 updateShotBullets，断言 CurHP=90 且命中类型为舰。
func TestDirectShellHitsCornerBeyondHalfLength(t *testing.T) {
	const (
		speed    = 1.0
		rotation = 135.0
		length   = 256.0
		width    = 256.0
	)
	// updateShotBullets 会先 Forward 再判定。起点放在舷侧角点内侧，
	// 沿对角线飞出后，到舰心的距离大于半长加弹速，但仍小于对角线半径加弹速。
	start := objPos.NewR(10.99, 10.99)
	sinR := math.Sin(rotation * math.Pi / 180)
	cosR := math.Cos(rotation * math.Pi / 180)
	endX := start.RX + sinR*speed
	endY := start.RY - cosR*speed

	dx, dy := endX-10, endY-10
	dist2 := dx*dx + dy*dy
	halfLengthReach := 0.5*length/constants.MapBlockSize + speed
	diagReach := targetHitRadius(length, width) + speed
	if dist2 <= halfLengthReach*halfLengthReach {
		t.Fatal("fixture no longer exceeds half-length reach")
	}
	if dist2 > diagReach*diagReach {
		t.Fatal("fixture is outside the diagonal reach")
	}

	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        start,
		Rotation:      rotation,
		Speed:         speed,
		Life:          10,
	}

	newShellImpactManager(target, bullet).updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 for a corner hit beyond half length", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}

// TestDirectShellIgnoresShipOutsideItsBuckets 验证空间桶粗筛不会误伤查询范围外的舰船。
// 场景：炮弹在 (10.99,10.99)，近舰 (10,10)、远舰 (80,10)，同为 256×256；桶查询半径只有 1+3 格。
// 预期：近舰沿 135° 弹道被命中剩 90 HP，远舰即使舰体朝向相同也保持 100 HP。
// 覆盖：MissionState 同时放入 target/far 两舰与一发炮弹，断言双方 CurHP，证明远舰未进入桶查询。
func TestDirectShellIgnoresShipOutsideItsBuckets(t *testing.T) {
	const (
		speed    = 1.0
		rotation = 135.0
		length   = 256.0
		width    = 256.0
	)
	start := objPos.NewR(10.99, 10.99)
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	far := &objUnit.BattleShip{
		Uid:          "far",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(80, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        start,
		Rotation:      rotation,
		Speed:         speed,
		Life:          10,
	}
	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				target.Uid: target,
				far.Uid:    far,
			},
			ForwardingBullets: []*objBullet.Bullet{bullet},
		},
	}}
	manager.updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("near HP = %.1f, want 90", target.CurHP)
	}
	if far.CurHP != 100 {
		t.Fatalf("far HP = %.1f, want 100", far.CurHP)
	}
}

// TestDirectShellSkipsDistantShip 验证相距过远的炮弹不会命中舰船且继续留作飞行中弹药。
// 场景：靶舰在 (10,10)，炮弹在 (50,50)、速度 1、寿命 10，两者相距约 56.6 格。
// 预期：target 保持 100 HP，bullet.HitObjType 仍为 TypeNone，ForwardingBullets 仍保留这 1 发。
// 覆盖：newShellImpactManager 只放入靶舰与炮弹，跑一拍 updateShotBullets 后断言 HP、命中类型与队列长度。
func TestDirectShellSkipsDistantShip(t *testing.T) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       256,
		Width:        256,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        objPos.NewR(50, 50),
		Speed:         1,
		Life:          10,
	}

	manager := newShellImpactManager(target, bullet)
	manager.updateShotBullets()

	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100 for a distant miss", target.CurHP)
	}
	if bullet.HitObjType != object.TypeNone {
		t.Fatalf("hit type = %v, want none", bullet.HitObjType)
	}
	if len(manager.state.Arena.ForwardingBullets) != 1 {
		t.Fatalf("forwarding bullets = %d, want 1", len(manager.state.Arena.ForwardingBullets))
	}
}
