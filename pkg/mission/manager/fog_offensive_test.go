package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// pearlHarborContactTick 是电脑找到并打到我方之前允许跑的任务帧数。
// 实测首次接触约 2750 拍、首次造成伤害约 3490 拍；伤害取决于攻击是否命中，
// 未命中的分支会拖到 5200 拍左右，这里取 7200 留足余量。
const pearlHarborContactTick = 7200

// useFogSettings 打开战争迷雾跑一个测试，结束后还原全局设置。
func useFogSettings(t *testing.T) {
	t.Helper()
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableFogOfWar = true
	t.Cleanup(func() { config.G = previous })
}

// stepMission 推进一拍任务模拟，等价于 Update 里的战斗阶段。
func stepMission(m *MissionManager) {
	m.simTick++
	m.state.Core.SimTick = m.simTick
	objUnit.SetSimTick(m.simTick)
	m.updateCommandPhase()
	m.updateSupportPhase()
	m.updateCombatPhase()
}

func countExplored(vision *state.FactionVision) int {
	if vision == nil {
		return 0
	}
	count := 0
	for _, cell := range vision.Explored {
		if cell != 0 {
			count++
		}
	}
	return count
}

// 侦察机起飞后不能被 updatePlaneAttackOrReturn 当成「没目标」直接叫返航，
// 否则迷雾下的自动侦察和玩家手动侦察都是废的。
func TestScoutKeepsFlyingWithoutAttackTarget(t *testing.T) {
	useFogSettings(t)
	m := New("MidwayFourCornersTest", faction.SideP1)

	var base objUnit.AircraftBase
	for _, ship := range m.state.Arena.Ships {
		if ship.Aircraft.ScoutStock() >= 2 {
			base = ship
			break
		}
	}
	if base == nil {
		for _, airfield := range m.state.Arena.Airfields {
			if airfield.Aircraft.ScoutStock() >= 2 {
				base = airfield
				break
			}
		}
	}
	if base == nil {
		t.Fatal("测试关卡里应该有带侦察机的基地")
	}

	plane := base.BaseAircraft().TakeOffScout(base)
	if plane == nil {
		t.Fatal("TakeOffScout 失败")
	}
	plane.FlightPhase = objUnit.PlaneFlightPhaseCruising
	m.state.Arena.PutPlane(plane)
	scoutUid := instr.GenInstrUid(instr.NamePlaneScout, plane.Uid)
	m.instructionSet.Add(instr.NewPlaneScout(plane.Uid, plane.CurPos, false))

	m.updatePlaneAttackOrReturn()

	if m.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneReturn, plane.Uid)) {
		t.Fatal("侦察机刚起飞就被下达返航指令")
	}
	if !m.instructionSet.Exists(scoutUid) {
		t.Fatal("侦察指令不该被撤掉")
	}
}

// 珍珠港 + 迷雾：电脑必须自己派飞机搜索、点亮我方舰队，然后发动进攻。
// 回归「迷雾下敌人站着不动、不会进攻」。
func TestComputerScoutsAndAttacksUnderFog(t *testing.T) {
	useFogSettings(t)
	m := New("PearlHarbor1941", faction.SideP1)

	startExplored := countExplored(m.state.Player.Visions[faction.ComputerAlpha])
	launched, damage := false, 0.0
	for tick := 0; tick < pearlHarborContactTick && damage == 0; tick++ {
		stepMission(m)
		if !launched {
			for _, plane := range m.state.Arena.Planes {
				if plane.BelongPlayer == faction.ComputerAlpha {
					launched = true
					break
				}
			}
		}
		for _, ship := range m.state.Arena.Ships {
			if ship.BelongPlayer == faction.HumanAlpha {
				damage += ship.TotalHP - ship.CurHP
			}
		}
	}

	if !launched {
		t.Fatal("电脑在迷雾下一架飞机都没派，永远点不亮我方")
	}
	if damage <= 0 {
		t.Fatalf("%d 拍内电脑始终没有打到我方", pearlHarborContactTick)
	}
	if end := countExplored(m.state.Player.Visions[faction.ComputerAlpha]); end < startExplored*2 {
		t.Fatalf("电脑已探索 %d 格（起始 %d），搜索机没有真的在开图", end, startExplored)
	}
}
