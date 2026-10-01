package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
)

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
