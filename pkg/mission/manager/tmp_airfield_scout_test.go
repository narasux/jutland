package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
)

// 临时诊断：电脑方拥有陆地机场时，自动侦察是否会从机场起飞。
func TestTmpComputerAirfieldAutoScout(t *testing.T) {
	useFogSettings(t)
	for _, side := range []faction.Side{faction.SideP1, faction.SideP2} {
		m := New("MidwayFourCornersTest", side)
		t.Logf("side=%s player=%s fog=%v airfields=%d", side, m.state.Player.CurPlayer, m.state.Core.FogOfWar, len(m.state.Arena.Airfields))
		launched := 0
		for tick := 0; tick < 60; tick++ {
			stepMission(m)
			launched = 0
			for _, p := range m.state.Arena.Planes {
				if _, ok := m.state.Arena.Airfields[p.BelongShip]; ok {
					launched++
				}
			}
			if launched > 0 {
				break
			}
		}
		t.Logf("  airfield-launched planes after stepping: %d", launched)
		for uid, af := range m.state.Arena.Airfields {
			t.Logf("  airfield %s owner=%s scoutStock=%d canSearch=%v canManual=%v groups=%d",
				uid, af.BelongPlayer, af.Aircraft.ScoutStock(), af.Aircraft.CanSearch(),
				af.Aircraft.CanManualScout(), len(af.Aircraft.Groups))
		}
	}
}
