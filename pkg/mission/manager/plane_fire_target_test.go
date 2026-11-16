package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestPlaneFireTargetPrioritizesCurrentAttackTarget 验证飞机开火优先选择当前追击目标而非随机候选。
// 场景：A6M5 战斗机在 (10,10)、MaxToPlaneRange=5，CurAttackTarget 指向尾后的 "tail-target"(10,9.5)；
// 侧翼另有一架 "flank-target"(12,10) 同样在 5 格射程内，构成随机挑选的干扰项。
// 预期：planeFireTarget 返回 "tail-target"；失败意味着目标选择走随机分支，机头会转向射界外的敌机，
// 追击中的火力被打断，弹药也会浪费在侧翼目标上。
// 覆盖：Arena.Planes 同时放入三架飞机，比较返回目标的 ID() 与 tailTarget.Uid。
func TestPlaneFireTargetPrioritizesCurrentAttackTarget(t *testing.T) {
	fighter := &objUnit.Plane{
		Uid:             "fighter",
		Name:            "A6M5",
		Type:            objUnit.PlaneTypeFighter,
		BelongPlayer:    faction.HumanAlpha,
		CurPos:          objPos.NewR(10, 10),
		CurRotation:     0,
		CurAttackTarget: "tail-target",
		Weapon: objUnit.PlaneWeapon{
			MaxToPlaneRange: 5,
		},
	}
	tailTarget := &objUnit.Plane{
		Uid:          "tail-target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(10, 9.5),
	}
	flankTarget := &objUnit.Plane{
		Uid:          "flank-target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(12, 10),
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Planes: map[string]*objUnit.Plane{
					fighter.Uid:     fighter,
					tailTarget.Uid:  tailTarget,
					flankTarget.Uid: flankTarget,
				},
			},
		},
	}

	target := manager.planeFireTarget(fighter)
	if target == nil || target.ID() != tailTarget.Uid {
		t.Fatalf("fire target = %v, want current tail target %q", target, tailTarget.Uid)
	}
}
