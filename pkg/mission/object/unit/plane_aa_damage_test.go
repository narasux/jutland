package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// TestPlaneHurtByShipShellSurvivesSingleHit 验证舰对空破片毁伤对非暴击单发有“六成血量”上限。
// 场景：满血 B-17G（TotalHP=115、DamageReduction=0.5）挨一发大威力的 127mm 舰炮弹
// （Damage=300、CriticalRate=0 即非暴击、ShooterObjType=TypeShip）；
// 若无上限，300×(1-0.5)=150 会直接击落 115 HP 的重型轰炸机。
// 预期：实扣 min(150, 115×0.6)=69，CurHP=46 > 0，单发不致命。
// 覆盖：构造 B-17G 调用 HurtBy，断言未归零且 115-CurHP 精确等于 115×0.6。
func TestPlaneHurtByShipShellSurvivesSingleHit(t *testing.T) {
	b17 := &Plane{TotalHP: 115, CurHP: 115, DamageReduction: 0.5}
	shell := &objBullet.Bullet{
		Damage: 300, CriticalRate: 0,
		ShooterObjType: object.TypeShip, TargetObjType: object.TypePlane,
	}

	b17.HurtBy(shell)

	if b17.CurHP <= 0 {
		t.Fatalf("single 127mm shell should not destroy a full-HP B-17G, HP = %.2f", b17.CurHP)
	}
	lost := 115 - b17.CurHP
	if math.Abs(lost-115*0.6) > 1e-9 {
		t.Fatalf("ship-shell fragment damage %.2f should be capped at 60%% of max HP (%.2f)", lost, 115*0.6)
	}
}

// TestPlaneHurtByShipShellCritBypassesCap 验证暴击不受舰对空破片上限约束，可以直接击落。
// 场景：满血 B-17G（115 HP、减伤 0.5）挨 120 伤害舰炮弹，CriticalRate=1；
// rand.Float64() 必然小于 0.1，触发 10 倍暴击，代表直击要害（油箱/弹药舱）。
// 预期：暴击伤害 120×(1-0.5)×10=600，远超 115 HP，CurHP 归零。
// 覆盖：构造必暴击弹调用 HurtBy，断言 CurHP==0。
func TestPlaneHurtByShipShellCritBypassesCap(t *testing.T) {
	b17 := &Plane{TotalHP: 115, CurHP: 115, DamageReduction: 0.5}
	// CriticalRate 1 -> rand.Float64() 必然小于 0.1，必定触发 10 倍暴击
	shell := &objBullet.Bullet{
		Damage: 120, CriticalRate: 1,
		ShooterObjType: object.TypeShip, TargetObjType: object.TypePlane,
	}

	b17.HurtBy(shell)

	if b17.CurHP != 0 {
		t.Fatalf("critical direct hit should destroy the plane, HP = %.2f", b17.CurHP)
	}
}

// TestPlaneHurtByAircraftBulletKeepsTripleDamage 验证空对空（战斗机/自卫机枪对空）伤害仍按 3 倍结算，
// 不受舰对空破片上限影响。
// 场景：满血 B-17G（115 HP、减伤 0.5）吃空对空机枪弹：Damage=0.25、CriticalRate=0、
// ShooterObjType=TypePlane 且 TargetObjType=TypePlane；若误用舰对空分支，伤害会被 0.6 上限截断。
// 预期：实扣 0.25×(1-0.5)×3=0.375，CurHP=114.625。
// 覆盖：构造飞机对飞机弹药调用 HurtBy，断言 115-CurHP 精确等于 0.375。
func TestPlaneHurtByAircraftBulletKeepsTripleDamage(t *testing.T) {
	b17 := &Plane{TotalHP: 115, CurHP: 115, DamageReduction: 0.5}
	bullet := &objBullet.Bullet{
		Damage: 0.25, CriticalRate: 0,
		ShooterObjType: object.TypePlane, TargetObjType: object.TypePlane,
	}

	b17.HurtBy(bullet)

	lost := 115 - b17.CurHP
	// 0.25 * (1 - 0.5) * 3 = 0.375
	if math.Abs(lost-0.375) > 1e-9 {
		t.Fatalf("air-to-air damage = %.4f, want 0.375", lost)
	}
}
