package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// foggedState 构造一个开启迷雾、双方视野图都为空（初始不可见）的任务状态。
func foggedState() *MissionState {
	ms := &MissionState{
		Core:   MissionCoreState{SimTick: 100, FogOfWar: true},
		Player: MissionPlayerState{CurPlayer: faction.HumanAlpha},
	}
	ms.Player.Visions = map[faction.Player]*FactionVision{
		faction.HumanAlpha:    NewFactionVision(16, 16),
		faction.ComputerAlpha: NewFactionVision(16, 16),
	}
	return ms
}

// TestCanSeeShipSurfaceFollowsFog 验证水面舰可见性完全跟随迷雾格，己方舰船恒可见且不受迷雾影响。
// 场景：foggedState 的 16×16 视野图全暗，敌舰在 (5.5,5.5)、己方舰在 (9.5,9.5)，只手动点亮敌方格。
// 预期：点亮前 false，置 Visible[5*16+5]=1 后 true；己方舰在暗格上仍 true，失败即迷雾失效或己方被隐藏。
// 覆盖：一艘 ComputerAlpha 与一艘 HumanAlpha 的 BattleShip，三次断言 CanSeeShip。
func TestCanSeeShipSurfaceFollowsFog(t *testing.T) {
	ms := foggedState()
	enemy := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100, CurPos: objPos.NewR(5.5, 5.5),
	}
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, enemy), "迷雾图上没盖到就看不见")

	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, enemy))

	// 己方恒可见，且不受迷雾影响
	own := &objUnit.BattleShip{BelongPlayer: faction.HumanAlpha, CurHP: 100, CurPos: objPos.NewR(9.5, 9.5)}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, own))
}

// TestCanSeeShipSubmergedRequiresDetection 验证下潜潜艇的可见性独立于迷雾格，只认探测记忆的有效期。
// 场景：16×16 视野图全暗但 (5,5) 已点亮；SimTick=100 时对下潜潜艇 MarkDetected 到 150，随后推进到 150。
// 预期：仅点亮格时 false；MarkDetected 后 true；SimTick=150 时 false（UntilTick 需严格大于当前拍）。
// 覆盖：foggedState 加 ShipTypeSubmarine/DiveStateSubmerged，三次断言 CanSeeShip。
func TestCanSeeShipSubmergedRequiresDetection(t *testing.T) {
	ms := foggedState()
	ms.Core.SimTick = 100
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(5.5, 5.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	// 视野格全亮也不该看见水下潜艇
	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))

	sub.MarkDetected(faction.HumanAlpha, 150, sub.CurPos)
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	// 到期即失效
	ms.Core.SimTick = 150
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))
}

// TestCanSeeShipIgnoreFogRevealsAll 验证 IgnoreFog 只揭开当前玩家，电脑侧对第三方潜艇仍要探测。
// 场景：观战秘籍开启，HumanBeta 的下潜潜艇在 (1.5,1.5)，双方视野图全空，CurPlayer=HumanAlpha。
// 预期：CanSeeShip(HumanAlpha, sub)=true、CanSeeShip(ComputerAlpha, sub)=false，失败即秘籍顺带帮电脑开图。
// 覆盖：foggedState 上置 Player.IgnoreFog，构造不属于任何一方的 HumanBeta 潜艇，断言两个方向的结果。
func TestCanSeeShipIgnoreFogRevealsAll(t *testing.T) {
	ms := foggedState()
	ms.Player.IgnoreFog = true
	// 第三方潜艇：既不是当前玩家的，也不是电脑的
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.HumanBeta, CurHP: 100,
		CurPos: objPos.NewR(1.5, 1.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	// 秘籍只揭开当前玩家，电脑仍要靠探测
	require.False(t, ms.CanSeeShip(faction.ComputerAlpha, sub))
}

// TestRevealsSubmarineIsVisualOnly 验证 RevealsSubmarine 只是绘制层标记，不改变 CanSeeShip 的战斗可见性。
// 场景：FogOfWar 关闭、敌潜艇在 (4.5,4.5) 下潜且未被探测，切换 RevealSubmarines；另测己方与上浮潜艇。
// 预期：关时两函数均 false；开后 RevealsSubmarine=true 但 CanSeeShip=false；己方与上浮舰标记均 false。
// 覆盖：三个 BattleShip（敌方下潜、己方下潜、敌方水面）与两次开关切换，共断言 7 个结果。
func TestRevealsSubmarineIsVisualOnly(t *testing.T) {
	ms := foggedState()
	ms.Core.FogOfWar = false
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(4.5, 4.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}

	// 秘籍没开：不可见、也不暴露
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))

	ms.Player.RevealSubmarines = true
	require.True(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))
	// 只是绘制层暴露，战斗可见性必须保持原样
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))

	// 己方潜艇、水面/未下潜潜艇不需要红框
	own := &objUnit.BattleShip{
		BelongPlayer: faction.HumanAlpha, CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, own))

	surfaced := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(6.5, 6.5), Type: objUnit.ShipTypeSubmarine,
	}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, surfaced))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, surfaced), "已经可见的潜艇不用再加红框")
}
