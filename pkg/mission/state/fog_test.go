package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
)

// TestUsesFogStaysOffUntilEnabled 验证 UsesFog 三态：未开迷雾全关、开启后双方都跑、IgnoreFog 只放过当前玩家。
// 场景：热路径每拍先问 UsesFog 决定是否扫描视野；Core.FogOfWar 与 IgnoreFog 组合决定结果，默认均 false。
// 预期：初始双方 false；FogOfWar=true 后均 true；IgnoreFog=true 后 HumanAlpha false、电脑仍 true。
// 覆盖：空 MissionState（CurPlayer=HumanAlpha）上三次改开关，分别断言双方 UsesFog 返回值。
func TestUsesFogStaysOffUntilEnabled(t *testing.T) {
	ms := &MissionState{Player: MissionPlayerState{CurPlayer: faction.HumanAlpha}}
	if ms.UsesFog(faction.HumanAlpha) || ms.UsesFog(faction.ComputerAlpha) {
		t.Fatal("fog stays off until the mission enables it")
	}

	ms.Core.FogOfWar = true
	if !ms.UsesFog(faction.HumanAlpha) || !ms.UsesFog(faction.ComputerAlpha) {
		t.Fatal("both sides use fog when the mission enables it")
	}

	ms.Player.IgnoreFog = true
	if ms.UsesFog(faction.HumanAlpha) {
		t.Fatal("current player ignores fog after the reveal cheat")
	}
	if !ms.UsesFog(faction.ComputerAlpha) {
		t.Fatal("computer keeps fog after the current player is revealed")
	}
}
