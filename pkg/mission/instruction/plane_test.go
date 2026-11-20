package instruction

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func newLandingTestState(
	t *testing.T, planeName string, planeCount int,
) (*state.MissionState, *objUnit.BattleShip, []*objUnit.Plane) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name:         planeName,
		TotalHP:      100,
		CurHP:        100,
		MaxSpeed:     0.12,
		Acceleration: 0.01,
		RotateSpeed:  12,
		RemainRange:  100,
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})

	ship := &objUnit.BattleShip{
		Uid:          "carrier-" + planeName,
		Length:       128,
		CurPos:       objPos.NewR(50, 50),
		CurRotation:  0,
		BelongPlayer: faction.HumanAlpha,
		Aircraft: objUnit.ShipAircraft{
			Groups: []objUnit.PlaneGroup{{
				Name:     planeName,
				MaxCount: int64(planeCount),
				CurCount: 0,
			}},
		},
	}
	planes := make([]*objUnit.Plane, 0, planeCount)
	planeMap := make(map[string]*objUnit.Plane, planeCount)
	for idx := range planeCount {
		plane := objUnit.NewPlane(
			planeName,
			objPos.NewR(50, 53+float64(idx)*0.2),
			0,
			ship.Uid,
			faction.HumanAlpha,
		)
		planes = append(planes, plane)
		planeMap[plane.Uid] = plane
	}

	return &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: planeMap,
		},
	}, ship, planes
}

func execPlaneReturn(t *testing.T, instruction *PlaneReturn, missionState *state.MissionState) {
	t.Helper()
	if err := instruction.Exec(missionState); err != nil {
		t.Fatal(err)
	}
}

// TestPlaneReturnRecoversOnlyAfterLandingDeck 验证航母只在 LandingDeck 完成后才入库。
// 场景：航母长 128 在 (50,50) 朝北，1 架飞机在舰尾外侧返航，逐拍执行 PlaneReturn，
// 待场段用 Flight.EndPos 把飞机直接放到进近入口以缩短等待。
// 预期：Staging/Approach 期间 CurCount 恒为 0，进入 Deck 后仍为 0，Executed 后
// CurCount=1 且飞机从 Arena.Planes 移除。提前入库会让空中飞机凭空消失。
// 覆盖：300 拍推待场、140 拍推进近、220 拍推着舰滑跑，每段都断言库存未变。
func TestPlaneReturnRecoversOnlyAfterLandingDeck(t *testing.T) {
	ms, ship, planes := newLandingTestState(t, "test-landing-plane", 1)
	plane := planes[0]
	returnInstr := NewPlaneReturn(plane.Uid)

	execPlaneReturn(t, returnInstr, ms)
	if plane.Flight.Phase != objUnit.PlaneFlightPhaseLandingStaging {
		t.Fatalf("phase = %s, want %s", plane.Flight.Phase, objUnit.PlaneFlightPhaseLandingStaging)
	}
	if ship.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("plane recovered while waiting")
	}

	plane.CurPos = plane.Flight.EndPos.Copy()
	plane.CurRotation = 0
	for frame := 0; frame < 300 && plane.Flight.Phase == objUnit.PlaneFlightPhaseLandingStaging; frame++ {
		execPlaneReturn(t, returnInstr, ms)
		if ship.Aircraft.Groups[0].CurCount != 0 {
			t.Fatalf("plane recovered while aligning with landing approach")
		}
	}
	if plane.Flight.Phase != objUnit.PlaneFlightPhaseLandingApproach {
		t.Fatalf(
			"phase = %s, want %s (pos=%s rotation=%.2f slot=%d)",
			plane.Flight.Phase,
			objUnit.PlaneFlightPhaseLandingApproach,
			plane.CurPos.String(),
			plane.CurRotation,
			plane.Landing.Slot,
		)
	}

	for frame := 0; frame < 140 && plane.Flight.Phase == objUnit.PlaneFlightPhaseLandingApproach; frame++ {
		execPlaneReturn(t, returnInstr, ms)
		if ship.Aircraft.Groups[0].CurCount != 0 {
			t.Fatalf("plane recovered during final approach")
		}
	}
	if plane.Flight.Phase != objUnit.PlaneFlightPhaseLandingDeck {
		t.Fatalf("phase = %s, want %s", plane.Flight.Phase, objUnit.PlaneFlightPhaseLandingDeck)
	}
	if ship.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("plane recovered before deck landing")
	}

	for frame := 0; frame < 220 && !returnInstr.Executed(); frame++ {
		execPlaneReturn(t, returnInstr, ms)
	}
	if !returnInstr.Executed() {
		t.Fatalf("return instruction did not finish")
	}
	if ship.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("recovered plane count = %d, want 1", ship.Aircraft.Groups[0].CurCount)
	}
	if _, ok := ms.Arena.Planes[plane.Uid]; ok {
		t.Fatalf("plane still exists after recovery")
	}
}

// TestPlaneReturnAllPlanesReachDeckBeforeRecovery 验证 32 架飞机分批返航都先着舰再回收。
// 场景：航母以 0.055 速度每拍转 0.8° 边转向边前进，32 架飞机在同一拍下达返航，
// 预算 2000 拍；甲板槽位与刹车段让回收排队进行。
// 预期：所有飞机都在 LandingDeck 拍被记录，没有任何一架在其他阶段从 Arena 消失；
// 结束时 Arena.Planes 为空、CurCount=32、reachedDeck 覆盖 32 架。
// 覆盖：newLandingTestState 造 32 架 test-landing-batch，advanceLandingTestCarrier 模拟母舰机动。
func TestPlaneReturnAllPlanesReachDeckBeforeRecovery(t *testing.T) {
	// 着舰入口基线后移至 -5.5 舰长、进近与刹车段按实际路程推进后，
	// 32 架两波次回收的整体时间线拉长，预算从 1200 帧放宽到 2000 帧。
	const maxRecoveryFrames = 2000

	ms, ship, planes := newLandingTestState(t, "test-landing-batch", 32)
	ship.CurSpeed = 0.055
	returns := make([]*PlaneReturn, 0, len(planes))
	reachedDeck := make(map[string]bool, len(planes))
	for _, plane := range planes {
		returnInstr := NewPlaneReturn(plane.Uid)
		returns = append(returns, returnInstr)
		execPlaneReturn(t, returnInstr, ms)
	}

	for frame := 0; frame < maxRecoveryFrames && len(ms.Arena.Planes) > 0; frame++ {
		advanceLandingTestCarrier(ship)
		for idx, plane := range planes {
			if returns[idx].Executed() {
				continue
			}
			phaseBefore := plane.Flight.Phase
			execPlaneReturn(t, returns[idx], ms)
			if plane.Flight.Phase == objUnit.PlaneFlightPhaseLandingDeck {
				reachedDeck[plane.Uid] = true
			}
			if _, exists := ms.Arena.Planes[plane.Uid]; !exists &&
				phaseBefore != objUnit.PlaneFlightPhaseLandingDeck {
				t.Fatalf("plane %d disappeared from phase %s instead of the deck", idx, phaseBefore)
			}
		}
	}

	if active := len(ms.Arena.Planes); active != 0 {
		t.Fatalf("planes remained after %d recovery frames: %d", maxRecoveryFrames, active)
	}
	if recovered := ship.Aircraft.Groups[0].CurCount; recovered != int64(len(planes)) {
		t.Fatalf("recovered planes = %d, want %d", recovered, len(planes))
	}
	if len(reachedDeck) != len(planes) {
		t.Fatalf("planes reaching deck = %d, want %d", len(reachedDeck), len(planes))
	}
}

// TestPlaneReturnRecoversBatchNearMapBorders 验证航母贴四边时 32 架飞机仍能全部回收。
// 场景：航母分别摆在左 (1,50)、右 (97,50)、上 (50,1)、下 (50,97)，朝向依次为
// 90、270、180、0，舰长放大到 256，飞机围绕母舰散开且初始朝向为 (idx*47)%360。
// 预期：四个子用例都在 1800 拍内清空 Arena.Planes、CurCount=32、每架都经过
// LandingDeck；若有飞机在待场或进近阶段消失，说明贴边布局打乱了回收时序。
// 覆盖：每个边界一个 t.Run，逐架下发 PlaneReturn 并核对 reachedDeck 与库存。
func TestPlaneReturnRecoversBatchNearMapBorders(t *testing.T) {
	const (
		planeCount        = 32
		maxRecoveryFrames = 1800
	)
	testCases := []struct {
		name            string
		carrierPos      objPos.MapPos
		carrierRotation float64
		planeStart      func(int) objPos.MapPos
	}{
		{
			name:            "left",
			carrierPos:      objPos.NewR(1, 50),
			carrierRotation: 90,
			planeStart: func(idx int) objPos.MapPos {
				return objPos.NewR(5+float64(idx%4)*0.25, 35+float64(idx)*0.9)
			},
		},
		{
			name:            "right",
			carrierPos:      objPos.NewR(97, 50),
			carrierRotation: 270,
			planeStart: func(idx int) objPos.MapPos {
				return objPos.NewR(93-float64(idx%4)*0.25, 35+float64(idx)*0.9)
			},
		},
		{
			name:            "top",
			carrierPos:      objPos.NewR(50, 1),
			carrierRotation: 180,
			planeStart: func(idx int) objPos.MapPos {
				return objPos.NewR(35+float64(idx)*0.9, 5+float64(idx%4)*0.25)
			},
		},
		{
			name:            "bottom",
			carrierPos:      objPos.NewR(50, 97),
			carrierRotation: 0,
			planeStart: func(idx int) objPos.MapPos {
				return objPos.NewR(35+float64(idx)*0.9, 93-float64(idx%4)*0.25)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ms, ship, planes := newLandingTestState(
				t,
				"test-landing-border-"+testCase.name,
				planeCount,
			)
			ship.Length = 256
			ship.CurPos = testCase.carrierPos
			ship.CurRotation = testCase.carrierRotation
			returns := make([]*PlaneReturn, 0, len(planes))
			reachedDeck := make(map[string]bool, len(planes))
			for idx, plane := range planes {
				plane.CurPos = testCase.planeStart(idx)
				plane.CurRotation = float64((idx * 47) % 360)
				returnInstr := NewPlaneReturn(plane.Uid)
				returns = append(returns, returnInstr)
				execPlaneReturn(t, returnInstr, ms)
			}

			for frame := 0; frame < maxRecoveryFrames && len(ms.Arena.Planes) > 0; frame++ {
				for idx, plane := range planes {
					if returns[idx].Executed() {
						continue
					}
					phaseBefore := plane.Flight.Phase
					execPlaneReturn(t, returns[idx], ms)
					if plane.Flight.Phase == objUnit.PlaneFlightPhaseLandingDeck {
						reachedDeck[plane.Uid] = true
					}
					if _, exists := ms.Arena.Planes[plane.Uid]; !exists &&
						phaseBefore != objUnit.PlaneFlightPhaseLandingDeck {
						t.Fatalf(
							"plane %d disappeared from phase %s instead of the deck",
							idx,
							phaseBefore,
						)
					}
				}
			}

			if active := len(ms.Arena.Planes); active != 0 {
				t.Fatalf(
					"planes remained after %d recovery frames: %d",
					maxRecoveryFrames,
					active,
				)
			}
			if recovered := ship.Aircraft.Groups[0].CurCount; recovered != planeCount {
				t.Fatalf("recovered planes = %d, want %d", recovered, planeCount)
			}
			if len(reachedDeck) != len(planes) {
				t.Fatalf("planes reaching deck = %d, want %d", len(reachedDeck), len(planes))
			}
		})
	}
}

func advanceLandingTestCarrier(ship *objUnit.BattleShip) {
	ship.CurRotation = math.Mod(ship.CurRotation+0.8, 360)
	radians := ship.CurRotation * math.Pi / 180
	ship.CurPos.AssignRxy(
		ship.CurPos.RX+math.Sin(radians)*ship.CurSpeed,
		ship.CurPos.RY-math.Cos(radians)*ship.CurSpeed,
	)
}
