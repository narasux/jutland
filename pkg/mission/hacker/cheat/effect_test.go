package cheat

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestBlackSheepWallRevealsOnlyCurrentPlayer 验证 black sheep wall 只为当前玩家关闭迷雾。
// 场景：任务开启 FogOfWar，当前玩家为 HumanAlpha；作弊执行后不应顺带揭开电脑方视野。
// 预期：Exec 返回 "Fog of war removed."（不是 "Not Implemented"），Player.IgnoreFog=true，
// UsesFog(HumanAlpha)=false 而 UsesFog(ComputerAlpha)=true；失败表示作弊影响到了 AI 阵营。
// 覆盖：构造 FogOfWar=true、CurPlayer=HumanAlpha 的 MissionState，执行一次后检查返回串与两侧 UsesFog。
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

// TestRevealSubmarinesCheatOnlyTogglesRenderingFlag 验证 reveal submarines 只开关绘制标记，可反复切换。
// 场景：调试时用红框暴露潜艇位置；它不能清迷雾（IgnoreFog 保持 false），否则会掩盖真实视野。
// 预期：Match("reveal submarines")=true 而 Match("reveal submarine")=false；首次 Exec 后
// RevealSubmarines=true 且 IgnoreFog 仍为 false；再次 Exec 翻回 false。
// 覆盖：构造 CurPlayer=HumanAlpha 的 MissionState，连续两次 Exec，断言精确匹配与标记翻转。
func TestRevealSubmarinesCheatOnlyTogglesRenderingFlag(t *testing.T) {
	ms := &state.MissionState{
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
	}
	cheat := &RevealSubmarines{}
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
