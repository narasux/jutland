package manager

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func newEventReportManager(events int) *MissionManager {
	manager := &MissionManager{state: &state.MissionState{
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha, CurEnemy: faction.ComputerAlpha,
		},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{BattleReportPage: state.BattleReportPageEvents},
	}}
	manager.state.Stats = state.NewBattleStats()
	manager.state.Stats.BattleLog = make([]state.BattleLogEntry, events)
	return manager
}

func TestScrollBattleReportEventsClamps(t *testing.T) {
	manager := newEventReportManager(30)

	manager.ScrollBattleReportEvents(100)
	require.Equal(t, 30-state.BattleReportEventRows, manager.state.UI.BattleReportEventScroll)

	manager.ScrollBattleReportEvents(-100)
	require.Zero(t, manager.state.UI.BattleReportEventScroll)

	layout := manager.BattleReportLayout(true)
	manager.ScrollBattleReportByWheel(
		-1,
		int(layout.Content.X+layout.Content.W/2),
		int(layout.Content.Y+layout.Content.H/2),
	)
	require.Greater(t, manager.state.UI.BattleReportEventScroll, 0)
}

func TestClearBattleStatsResetsReportData(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		UI: state.MissionUIState{
			BattleReportPage:        state.BattleReportPageEvents,
			BattleReportEventScroll: 7,
			BattleReportShipScroll:  [2]int{3, 4},
		},
	}}
	manager.state.Stats = state.NewBattleStats()
	manager.state.Stats.BattleLog = append(manager.state.Stats.BattleLog, state.BattleLogEntry{})
	manager.state.Stats.FundsStart = 100

	manager.ClearBattleStats()

	require.Empty(t, manager.state.Stats.BattleLog)
	require.Zero(t, manager.state.Stats.FundsStart)
	require.Equal(t, state.BattleReportPageOverview, manager.state.UI.BattleReportPage)
	require.Zero(t, manager.state.UI.BattleReportEventScroll)
	require.Equal(t, state.BattleReportEventFilterAll, manager.state.UI.BattleReportEventFilter)
	require.Equal(t, [2]int{}, manager.state.UI.BattleReportShipScroll)
}

func TestScrollBattleReportShipsClamps(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha, CurEnemy: faction.ComputerAlpha,
		},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{BattleReportPage: state.BattleReportPageShips},
	}}
	manager.state.Stats = state.NewBattleStats()
	own := &state.BattleSideStats{ShipsInitialByClass: map[string]int{}}
	for idx := 0; idx < 30; idx++ {
		own.ShipsInitialByClass[fmt.Sprintf("Ship%02d", idx)] = 1
	}
	manager.state.Stats.Sides = map[faction.Player]*state.BattleSideStats{
		faction.HumanAlpha: own,
	}

	reportLayout := manager.BattleReportLayout(true)
	left := reportLayout.DetailSides[0]
	leftX, leftY := int(left.X+left.W/2), int(left.Y+left.H/2)
	manager.ScrollBattleReportByWheel(-1, leftX, leftY)
	require.Greater(t, manager.state.UI.BattleReportShipScroll[0], 0)

	manager.ScrollBattleReportByWheel(100, leftX, leftY)
	require.Zero(t, manager.state.UI.BattleReportShipScroll[0])

	// 敌军没有舰级，右侧滚动保持为 0。
	right := reportLayout.DetailSides[1]
	manager.ScrollBattleReportByWheel(-1, int(right.X+right.W/2), int(right.Y+right.H/2))
	require.Zero(t, manager.state.UI.BattleReportShipScroll[1])
}

func TestHurtShipRecordsHitAndSink(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{}}
	manager.state.Stats = state.NewBattleStats()
	attacker := &objUnit.BattleShip{
		Uid: "a", Name: "Attacker", Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.HumanAlpha,
	}
	target := &objUnit.BattleShip{
		Uid: "t", Name: "Target", Type: objUnit.ShipTypeBattleShip,
		BelongPlayer: faction.ComputerAlpha, CurHP: 100, TotalHP: 100,
	}
	manager.state.Arena.Ships = map[string]*objUnit.BattleShip{
		attacker.Uid: attacker,
		target.Uid:   target,
	}

	manager.hurtShip(target, &objBullet.Bullet{
		Name: "TEST/GB/127/1940", Type: objBullet.TypeShell, Diameter: 127,
		Shooter: "a", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, Damage: 10,
	})
	require.Len(t, manager.state.Stats.BattleLog, 1)
	hit := manager.state.Stats.BattleLog[0]
	require.Equal(t, state.BattleLogShipHit, hit.Action)
	require.Equal(t, "Attacker", hit.ActorName)
	require.Equal(t, string(objUnit.ShipTypeDestroyer), hit.ActorType)
	require.Equal(t, "Target", hit.TargetName)
	require.Equal(t, string(objUnit.ShipTypeBattleShip), hit.TargetType)
	require.Equal(t, string(objBullet.TypeShell), hit.Hit.WeaponType)
	require.Equal(t, 127, hit.Hit.Diameter)
	require.Equal(t, "127", hit.Hit.Caliber)
	require.Equal(t, 10.0, hit.Hit.Damage)

	manager.hurtShip(target, &objBullet.Bullet{
		Name: "TEST/GB/127/1940", Type: objBullet.TypeShell, Diameter: 127,
		Shooter: "a", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, Damage: 1000,
	})
	require.True(t, target.SunkByHit)
	manager.state.Stats.RecordShipLost(target)

	require.Len(t, manager.state.Stats.BattleLog, 2)
	sink := manager.state.Stats.BattleLog[1]
	require.Equal(t, state.BattleLogShipSunk, sink.Action)
	require.Equal(t, "Attacker", sink.ActorName)
	require.Equal(t, faction.HumanAlpha, sink.ActorPlayer)
	require.Equal(t, "Target", sink.TargetName)
	require.Equal(t, 1000.0, sink.Hit.Damage)
	require.Equal(t, 1010.0, sink.TotalDamage)
}

func TestHurtPlaneRecordsHitAndShotDown(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{}}
	manager.state.Stats = state.NewBattleStats()
	attacker := &objUnit.Plane{
		Uid: "a", Name: "A6M", Type: objUnit.PlaneTypeFighter,
		BelongPlayer: faction.ComputerAlpha,
	}
	target := &objUnit.Plane{
		Uid: "t", Name: "F4F", Type: objUnit.PlaneTypeDiveBomber,
		BelongPlayer: faction.HumanAlpha, CurHP: 100, TotalHP: 100,
	}
	manager.state.Arena.Planes = map[string]*objUnit.Plane{
		attacker.Uid: attacker,
		target.Uid:   target,
	}

	manager.hurtPlane(target, &objBullet.Bullet{
		Name: "TEST/GB/20/1940", Type: objBullet.TypeShell, Diameter: 20,
		Shooter: "a", ShooterObjType: object.TypePlane,
		BelongPlayer: faction.ComputerAlpha, Damage: 10,
	})
	require.Len(t, manager.state.Stats.BattleLog, 1)
	hit := manager.state.Stats.BattleLog[0]
	require.Equal(t, state.BattleLogPlaneHit, hit.Action)
	require.Equal(t, "A6M", hit.ActorName)
	require.Equal(t, string(objUnit.PlaneTypeFighter), hit.ActorType)
	require.Equal(t, "F4F", hit.TargetName)
	require.Equal(t, string(objUnit.PlaneTypeDiveBomber), hit.TargetType)
	// 空对空伤害在 Plane.HurtBy 中 ×3。
	require.Equal(t, 30.0, hit.Hit.Damage)

	manager.hurtPlane(target, &objBullet.Bullet{
		Name: "TEST/GB/20/1940", Type: objBullet.TypeShell, Diameter: 20,
		Shooter: "a", ShooterObjType: object.TypePlane,
		BelongPlayer: faction.ComputerAlpha, Damage: 1000,
	})
	require.True(t, target.ShotDown)
	manager.state.Stats.RecordPlaneLost(target)

	require.Len(t, manager.state.Stats.BattleLog, 2)
	shotDown := manager.state.Stats.BattleLog[1]
	require.Equal(t, state.BattleLogPlaneShotDown, shotDown.Action)
	require.Equal(t, 3000.0, shotDown.Hit.Damage)
	require.Equal(t, 3030.0, shotDown.TotalDamage)
}

func TestBulletWeaponMetadata(t *testing.T) {
	require.Equal(t, 250, bulletBombWeight("US/BB/250/1938"))
	require.Equal(t, 5000, bulletBombWeight("SU/BB/5000/1938"))
	require.Zero(t, bulletBombWeight("bad"))
	require.Equal(t, "7.7", bulletCaliber("JP/GB/7.7/1922"))
	require.Equal(t, "7.62", bulletCaliber("SU/GB/7.62/1932"))
	require.Equal(t, "12.7", bulletCaliber("US/GB/12.7/1925"))
	require.Equal(t, "20", bulletCaliber("JP/GB/20/1936"))
	require.Equal(t, "533", bulletCaliber("SU/TB/533/1932/53-27"))
	require.Empty(t, bulletCaliber("US/BB/250/1938"))
}

func TestBattleReportScrollbarDragMapping(t *testing.T) {
	manager := newEventReportManager(30)
	reportLayout := manager.BattleReportLayout(true)
	scrollbar, ok := state.BattleReportEventScrollbar(
		reportLayout.Content, len(manager.state.Stats.BattleLog), 0, reportLayout.Scale,
	)
	require.True(t, ok)

	kind, _, ok := manager.battleReportScrollbarAt(
		int(scrollbar.TrackX+1), int(scrollbar.TrackY+1),
	)
	require.True(t, ok)
	require.Equal(t, state.BattleReportScrollDragEvents, kind)

	manager.state.UI.BattleReportScrollDrag = kind
	manager.dragBattleReportScrollTo(int(scrollbar.TrackY + scrollbar.TrackH - scrollbar.ThumbH))
	require.Equal(t, scrollbar.MaxScroll, manager.state.UI.BattleReportEventScroll)

	manager.state.UI.BattleReportPage = state.BattleReportPageShips
	own := &state.BattleSideStats{ShipsInitialByClass: map[string]int{}}
	for idx := 0; idx < 30; idx++ {
		own.ShipsInitialByClass[fmt.Sprintf("Ship%02d", idx)] = 1
	}
	manager.state.Stats.Sides = map[faction.Player]*state.BattleSideStats{
		faction.HumanAlpha: own,
	}
	reportLayout = manager.BattleReportLayout(true)
	scrollbar, ok = state.BattleReportDetailScrollbar(
		reportLayout.DetailSides[0], own.ShipClassCount(), 0, reportLayout.Scale,
	)
	require.True(t, ok)
	kind, _, ok = manager.battleReportScrollbarAt(
		int(scrollbar.TrackX+1), int(scrollbar.TrackY+1),
	)
	require.True(t, ok)
	require.Equal(t, state.BattleReportScrollDragShipsLeft, kind)

	manager.state.UI.BattleReportScrollDrag = kind
	manager.dragBattleReportScrollTo(int(scrollbar.TrackY + scrollbar.TrackH - scrollbar.ThumbH))
	require.Equal(t, scrollbar.MaxScroll, manager.state.UI.BattleReportShipScroll[0])
}

func TestCombatantInfoUsesBulletShooterSnapshot(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{}}
	shipShot := &objBullet.Bullet{
		Shooter: "gone-ship", ShooterName: "yukikaze", ShooterType: "destroyer",
		ShooterObjType: object.TypeShip,
	}
	name, unitType, isPlane := manager.combatantInfo(shipShot)
	require.Equal(t, "yukikaze", name)
	require.Equal(t, "destroyer", unitType)
	require.False(t, isPlane)

	planeShot := &objBullet.Bullet{
		Shooter: "gone-plane", ShooterName: "A6M5", ShooterType: "fighter",
		ShooterObjType: object.TypePlane,
	}
	name, unitType, isPlane = manager.combatantInfo(planeShot)
	require.Equal(t, "A6M5", name)
	require.Equal(t, "fighter", unitType)
	require.True(t, isPlane)
}

func TestHandleBattleReportEventFilterClick(t *testing.T) {
	manager := newEventReportManager(0)
	manager.state.UI.BattleReportEventScroll = 4
	manager.state.Stats.BattleLog = []state.BattleLogEntry{
		{Action: state.BattleLogShipHit},
		{Action: state.BattleLogShipSunk},
	}

	layout := manager.BattleReportLayout(true)
	killsRect := layout.EventFilters[1]
	hit := manager.HandleBattleReportEventFilterClick(
		int(killsRect.X+killsRect.W/2), int(killsRect.Y+killsRect.H/2),
	)
	require.True(t, hit)
	require.Equal(t, state.BattleReportEventFilterKills, manager.state.UI.BattleReportEventFilter)
	require.Zero(t, manager.state.UI.BattleReportEventScroll)

	allRect := layout.EventFilters[0]
	hit = manager.HandleBattleReportEventFilterClick(
		int(allRect.X+allRect.W/2), int(allRect.Y+allRect.H/2),
	)
	require.True(t, hit)
	require.Equal(t, state.BattleReportEventFilterAll, manager.state.UI.BattleReportEventFilter)
}

// 触控板按帧上报小数增量，滚轮必须先累加到标准刻度再翻行；
// 否则每帧都会被兜底成 ±1 行，一次轻扫就滚出去很远。
func TestScrollBattleReportWheelAccumulatesTrackpadDeltas(t *testing.T) {
	manager := newEventReportManager(30)
	layout := manager.BattleReportLayout(true)
	x := int(layout.Content.X + layout.Content.W/2)
	y := int(layout.Content.Y + layout.Content.H/2)

	for idx := 0; idx < 3; idx++ {
		manager.ScrollBattleReportByWheel(-0.25, x, y)
		require.Zero(t, manager.state.UI.BattleReportEventScroll)
	}
	manager.ScrollBattleReportByWheel(-0.25, x, y)
	require.Equal(t, reportWheelRowsPerUnit, manager.state.UI.BattleReportEventScroll)
}

func TestScrollBattleReportWheelIgnoresOutsidePanel(t *testing.T) {
	manager := newEventReportManager(30)
	manager.ScrollBattleReportByWheel(-1, 0, 0)
	require.Zero(t, manager.state.UI.BattleReportEventScroll)
}

func TestBulletHitInfoRecordsCritical(t *testing.T) {
	info := bulletHitInfo(&objBullet.Bullet{
		Type:         objBullet.TypeShell,
		Name:         "US/GB/127/1943",
		CriticalType: objBullet.CriticalTypeThreeTimes,
	})
	require.Equal(t, objBullet.CriticalTypeThreeTimes, info.Critical)
}
