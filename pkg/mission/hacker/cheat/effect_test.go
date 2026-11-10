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

func TestRevealSubmarinesCheatOnlyTogglesRenderingFlag(t *testing.T) {
	ms := &state.MissionState{
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
	}
	cheat := &ShowMeTheSubmarines{}
	if !cheat.Match("reveal submarines") {
		t.Fatal("cheat command did not match")
	}
	if cheat.Match("reveal submarine") {
		t.Fatal("cheat command should match the exact phrase")
	}

	cheat.Exec(ms)
	if !ms.Player.RevealSubmarines {
		t.Fatal("cheat did not enable the reveal flag")
	}
	// 只影响绘制：探测相关状态不能被改动
	if ms.Player.IgnoreFog {
		t.Fatal("reveal submarines must not clear fog")
	}

	cheat.Exec(ms)
	if ms.Player.RevealSubmarines {
		t.Fatal("cheat should toggle the reveal flag off")
	}
}
