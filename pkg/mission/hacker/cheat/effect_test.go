package cheat

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestBlackSheepWallRevealsOnlyCurrentPlayer(t *testing.T) {
	ms := &state.MissionState{
		Core:   state.MissionCoreState{FogOfWar: true},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
	}

	got := (&BlackSheepWall{}).Exec(ms)
	if got == "Not Implemented" {
		t.Fatal("black sheep wall still reports unimplemented")
	}
	if !ms.Player.IgnoreFog {
		t.Fatal("cheat did not mark the current player")
	}
	if ms.UsesFog(faction.HumanAlpha) {
		t.Fatal("current player still uses fog")
	}
	if !ms.UsesFog(faction.ComputerAlpha) {
		t.Fatal("computer fog was cleared with the current player")
	}
}
