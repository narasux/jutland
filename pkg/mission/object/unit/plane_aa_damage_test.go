package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// 舰对空破片毁伤：非暴击单发伤害有上限（约四成血量），一发 127mm 舰炮弹
// （伤害 120）不应直接击落满血 B-17G（115 HP）。
func TestPlaneHurtByShipShellSurvivesSingleHit(t *testing.T) {
	b17 := &Plane{TotalHP: 115, CurHP: 115, DamageReduction: 0.5}
	shell := &objBullet.Bullet{
		Damage: 120, CriticalRate: 0,
		ShooterObjType: object.TypeShip, TargetObjType: object.TypePlane,
	}

	b17.HurtBy(shell)

	if b17.CurHP <= 0 {
		t.Fatalf("single 127mm shell should not destroy a full-HP B-17G, HP = %.2f", b17.CurHP)
	}
	lost := 115 - b17.CurHP
	if lost > 115*0.4+1e-9 {
		t.Fatalf("ship-shell fragment damage %.2f exceeds 40%% max HP cap", lost)
	}
}

// 暴击代表直击要害（油箱/弹药舱），不受破片上限约束，可以直接击落。
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

// 空对空（战斗机/自卫机枪对空）伤害仍按 3 倍结算，不受舰对空破片上限影响。
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
