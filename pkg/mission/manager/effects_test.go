package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestDestroyedLandingPlaneReleasesSlot 验证战机被击落后 updateMissionPlanes 会回收其着舰槽位。
// 场景：航母 "carrier" 上两架飞机已用 RequestLanding 占住槽位 0/1，其中 "first" 的 CurHP 被扣到 0，
// 下一帧 updateMissionPlanes 既要把残骸移出 Arena.Planes，也要取消它的回收预约。
// 预期：Arena.Planes 中不再有 "first"；新申请降落的 "replacement" 拿到槽位 0 而不是 2。
// 覆盖：构造 Ships/Planes 映射与 100x100 地图配置，断言残骸被移除且复用槽位返回值等于 0。
func TestDestroyedLandingPlaneReleasesSlot(t *testing.T) {
	ship := &objUnit.BattleShip{Uid: "carrier"}
	first := &objUnit.Plane{
		Uid:        "first",
		BelongShip: ship.Uid,
		CurHP:      0,
		CurPos:     objPos.NewR(10, 10),
	}
	second := &objUnit.Plane{
		Uid:        "second",
		BelongShip: ship.Uid,
		CurHP:      10,
	}
	ship.Aircraft.RequestLanding(first.Uid)
	ship.Aircraft.RequestLanding(second.Uid)

	manager := &MissionManager{state: &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: map[string]*objUnit.Plane{
				first.Uid:  first,
				second.Uid: second,
			},
		},
	}}
	manager.updateMissionPlanes()

	if _, exists := manager.state.Arena.Planes[first.Uid]; exists {
		t.Fatalf("destroyed plane was not removed from arena")
	}
	if slot := ship.Aircraft.RequestLanding("replacement"); slot != 0 {
		t.Fatalf("destroyed plane slot was not released: replacement slot = %d", slot)
	}
}

func newWakeTrail(ownerUID string) *objTrail.Trail {
	trail := objTrail.New(
		objPos.NewR(10, 10), textureImg.TrailShapeCircle,
		10, 1,
		100, 1,
		0, 0, nil,
	)
	trail.OwnerUid = ownerUID
	return trail
}

// TestStoppedShipWakeFadesOutInsteadOfLeavingIsolatedDots 验证停船后的尾流会加速淡出并被回收。
// 场景：战舰 CurSpeed=0 且原地不动，但战场上还留着一组 CurLife=100、LifeReductionRate=1 的尾流，
// updateObjectTrails 发现船速为 0 后调用 BeginStopFade(stoppedShipWakeFadeFrames=30)。
// 预期：首帧后 LifeReductionRate 大于 1（约 100/30），再推进 30 帧尾流全部消亡、Arena.Trails 清空；
// 失败意味着停船尾流按原速率滞留，海面上会留下孤立的小圆点。
// 覆盖：newWakeTrail 生成 owner 为 "ship" 的尾流，配置默认 GameSettings，循环 stoppedShipWakeFadeFrames 帧。
func TestStoppedShipWakeFadesOutInsteadOfLeavingIsolatedDots(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	trail := newWakeTrail("ship")
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					"ship": {
						Uid:          "ship",
						CurHP:        100,
						TotalHP:      100,
						CurSpeed:     0,
						CurPos:       objPos.NewR(10, 10),
						BelongPlayer: faction.HumanAlpha,
					},
				},
				Trails: []*objTrail.Trail{trail},
			},
		},
	}

	manager.updateObjectTrails()
	if trail.LifeReductionRate <= 1 {
		t.Fatalf("stopped wake fade rate = %v, want faster than normal", trail.LifeReductionRate)
	}
	for range stoppedShipWakeFadeFrames {
		manager.updateObjectTrails()
	}
	if len(manager.state.Arena.Trails) != 0 {
		t.Fatalf("stopped wake trails = %d, want 0", len(manager.state.Arena.Trails))
	}
}

// TestStoppedTurningShipWakeFadesFasterThanIdleStop 验证停船后原地转向会把旧尾迹更快淡掉。
// 场景：战舰 CurSpeed=0，海面留着 CurLife=100、LifeReductionRate=1 的尾迹；首帧按停船 30 帧淡出，
// 随后舰体原地转到 90°，旧尾迹若继续按 30 帧淡出会悬在转开的舷侧，看起来像脱节的白条。
// 预期：转向后 LifeReductionRate 从约 100/30 提升到约 100/12，再推进 12 帧尾迹全部回收；
// 失败意味着舰体转完时旧航迹还留在原地。
// 覆盖：同一组 Ships/Trails 连续调用 updateObjectTrails，先断言停船速率，再改 CurRotation 断言提速与回收。
func TestStoppedTurningShipWakeFadesFasterThanIdleStop(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	trail := newWakeTrail("ship")
	ship := &objUnit.BattleShip{
		Uid:          "ship",
		CurHP:        100,
		TotalHP:      100,
		CurSpeed:     0,
		CurPos:       objPos.NewR(10, 10),
		CurRotation:  0,
		BelongPlayer: faction.HumanAlpha,
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships:  map[string]*objUnit.BattleShip{"ship": ship},
				Trails: []*objTrail.Trail{trail},
			},
		},
	}

	manager.updateObjectTrails()
	idleRate := trail.LifeReductionRate
	if idleRate <= 1 || idleRate > 100.0/stoppedShipTurningWakeFadeFrames {
		t.Fatalf("idle stop fade rate = %v, want about 100/%d", idleRate, stoppedShipWakeFadeFrames)
	}

	ship.CurRotation = 90
	manager.updateObjectTrails()
	if trail.LifeReductionRate <= idleRate {
		t.Fatalf(
			"turning stop fade rate = %v, want faster than idle %v", trail.LifeReductionRate, idleRate,
		)
	}
	if trail.LifeReductionRate < 100.0/stoppedShipTurningWakeFadeFrames-0.5 {
		t.Fatalf(
			"turning stop fade rate = %v, want at least about 100/%d",
			trail.LifeReductionRate, stoppedShipTurningWakeFadeFrames,
		)
	}
	for range stoppedShipTurningWakeFadeFrames {
		manager.updateObjectTrails()
	}
	if len(manager.state.Arena.Trails) != 0 {
		t.Fatalf("turning stop wake trails = %d, want 0", len(manager.state.Arena.Trails))
	}
}

// TestMovingShipWakeKeepsNormalFade 验证航行中的战舰尾流不会套用停船加速淡出。
// 场景：同一组尾流挂在 CurSpeed=0.1 的友方战舰 "ship" 上，updateObjectTrails 执行一帧。
// 预期：trail.LifeReductionRate 保持 1；若大于 1 说明停船判定把航行中的船也算进去，
// 尾流会在几帧内消失，看不到正常的航迹长度。
// 覆盖：复用 newWakeTrail 与默认配置，只调用一次 updateObjectTrails 后断言速率等于 1。
func TestMovingShipWakeKeepsNormalFade(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	trail := newWakeTrail("ship")
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					"ship": {
						Uid:          "ship",
						CurHP:        100,
						TotalHP:      100,
						CurSpeed:     0.1,
						CurPos:       objPos.NewR(10, 10),
						BelongPlayer: faction.HumanAlpha,
					},
				},
				Trails: []*objTrail.Trail{trail},
			},
		},
	}

	manager.updateObjectTrails()
	if trail.LifeReductionRate != 1 {
		t.Fatalf("moving wake fade rate = %v, want 1", trail.LifeReductionRate)
	}
}
