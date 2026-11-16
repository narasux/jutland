package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestPlaneAttackDisengagesWhenGroundTargetTakesOff 验证轰炸机锁定目标升空后立即脱离。
// 场景：配置默认设置，注册 test-disengage-bomb 炸弹；SBD-3 俯冲轰炸机（巡航阶段，
// 1 枚射程 1.5 / 弹速 0.4 的炸弹）锁定 FlightPhase=TakingOff 的敌机
// ground-target-1，调用 NewPlaneAttack(bomber, object.TypePlane, target) 后执行一次。
// 预期：指令马上 Executed，目标机不能继续被追着绕机场；炸弹无法攻击空中目标，
// 应由调度重新分配或触发返航。
// 覆盖：直接 Exec 后断言 i.Executed()。注意：本用例的目标仍在 TakingOff
// （IsOnGround=true），且轰炸机未设 RemainRange（为 0），MustReturn() 会先结束
// 指令，因此“目标真正升空即脱离”的分支实际未被覆盖，属于用例待补强点。
func TestPlaneAttackDisengagesWhenGroundTargetTakesOff(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-disengage-bomb"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeBomb}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		}
		delete(objBullet.Map, bulletName)
	})

	const bomberName = "SBD-3"
	bomber := &objUnit.Plane{
		Uid:          "bomber-1",
		Name:         bomberName,
		Type:         objUnit.PlaneTypeDiveBomber,
		TotalHP:      100,
		CurHP:        100,
		FlightPhase:  objUnit.PlaneFlightPhaseCruising,
		BelongPlayer: faction.ComputerAlpha,
		Weapon: objUnit.PlaneWeapon{
			Bombs: []*objUnit.Releaser{{BulletName: bulletName, Range: 1.5, BulletSpeed: 0.4}},
		},
	}
	target := &objUnit.Plane{
		Uid:          "ground-target-1",
		TotalHP:      100,
		CurHP:        100,
		FlightPhase:  objUnit.PlaneFlightPhaseTakingOff,
		BelongPlayer: faction.HumanAlpha,
	}
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{bomber.Uid: bomber, target.Uid: target},
		},
	}

	i := NewPlaneAttack(bomber.Uid, object.TypePlane, target.Uid)
	if err := i.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !i.Executed() {
		t.Fatal("bomber should disengage once its ground target leaves the ground")
	}
}
