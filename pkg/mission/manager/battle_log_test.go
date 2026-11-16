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

// TestScrollBattleReportEventsClamps 验证战况列表滚动偏移始终夹在 [0, 总数-每屏行数]。
// 场景：战报有 30 条记录而战况页仅显示 BattleReportEventRows=13 行，先向历史滚 100 行再反向滚 -100。
// 预期：最大偏移停在 30-13=17，最小回到 0；鼠标位于列表面板中心时滚轮 -1 能让偏移大于 0。
// 覆盖：newEventReportManager(30) 构造日志，断言两处滚动入口的边界。
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

// TestClearBattleStatsResetsReportData 验证关闭结算后清空本局战果并复位战报 UI 状态。
// 场景：遗留 1 条 BattleLog、FundsStart=100，战报停在战况页且滚动为 7、明细滚动为 [3,4]。
// 预期：ClearBattleStats 后日志为空、资金基线归零，页面回总览、筛选回全部，两处滚动都清零。
// 覆盖：构造带脏 UI 与 Stats 的 MissionManager，断言 BattleLog/FundsStart/BattleReportPage/滚动/筛选。
func TestClearBattleStatsResetsReportData(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		UI: state.MissionUIState{
			BattleReportPage:         state.BattleReportPageEvents,
			BattleReportEventScroll:  7,
			BattleReportDetailScroll: [2]int{3, 4},
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
	require.Equal(t, [2]int{}, manager.state.UI.BattleReportDetailScroll)
}

// TestScrollBattleReportShipsClamps 验证舰艇明细页按己方舰级总数限制单侧滚动范围。
// 场景：HumanAlpha 侧有 30 个舰级，远超明细面板可见行数，鼠标先落在左侧列表滚轮 -1 再反向滚 100。
// 预期：左列偏移先大于 0，再被夹回 0；敌方没有舰级，右列偏移始终保持 0。
// 覆盖：构造 30 个 ShipsInitialByClass 条目，用 BattleReportLayout 的 DetailSides 定位左右列并断言偏移。
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
	require.Greater(t, manager.state.UI.BattleReportDetailScroll[0], 0)

	manager.ScrollBattleReportByWheel(100, leftX, leftY)
	require.Zero(t, manager.state.UI.BattleReportDetailScroll[0])

	// 敌军没有舰级，右侧滚动保持为 0。
	right := reportLayout.DetailSides[1]
	manager.ScrollBattleReportByWheel(-1, int(right.X+right.W/2), int(right.Y+right.H/2))
	require.Zero(t, manager.state.UI.BattleReportDetailScroll[1])
}

// TestScrollBattleReportPlanesClamps 验证飞机明细页按己方机型总数限制单侧滚动范围。
// 场景：HumanAlpha 侧有 30 个机型，飞机页复用 DetailSides 与明细滚动状态，先在左列滚轮 -1 再反向滚 100。
// 预期：左列偏移先大于 0、再夹回 0；敌方 PlanesInitialByType 为空，右列偏移保持 0。
// 覆盖：构造 30 个 Plane%02d 机型，断言 BattleReportDetailScroll[0]/[1] 的上下界。
func TestScrollBattleReportPlanesClamps(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha, CurEnemy: faction.ComputerAlpha,
		},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{BattleReportPage: state.BattleReportPagePlanes},
	}}
	manager.state.Stats = state.NewBattleStats()
	own := &state.BattleSideStats{PlanesInitialByType: map[string]int{}}
	for idx := 0; idx < 30; idx++ {
		own.PlanesInitialByType[fmt.Sprintf("Plane%02d", idx)] = 1
	}
	manager.state.Stats.Sides = map[faction.Player]*state.BattleSideStats{
		faction.HumanAlpha: own,
	}

	reportLayout := manager.BattleReportLayout(true)
	left := reportLayout.DetailSides[0]
	leftX, leftY := int(left.X+left.W/2), int(left.Y+left.H/2)
	manager.ScrollBattleReportByWheel(-1, leftX, leftY)
	require.Greater(t, manager.state.UI.BattleReportDetailScroll[0], 0)

	manager.ScrollBattleReportByWheel(100, leftX, leftY)
	require.Zero(t, manager.state.UI.BattleReportDetailScroll[0])

	// 敌军没有机型记录，右侧滚动保持为 0。
	right := reportLayout.DetailSides[1]
	manager.ScrollBattleReportByWheel(-1, int(right.X+right.W/2), int(right.Y+right.H/2))
	require.Zero(t, manager.state.UI.BattleReportDetailScroll[1])
}

// TestHurtShipRecordsHitAndSink 验证舰艇命中与击沉分别由 hurtShip 和 RecordShipLost 写入战报。
// 场景：驱逐舰 Attacker 用 127mm 炮弹先打 10 点，再用 1000 点伤害击沉 100 HP 的战列舰 Target。
// 预期：首击生成 BattleLogShipHit 并记录攻击者/目标/武器；致死后不重复记命中，击沉记录累计伤害 1010。
// 覆盖：Arena.Ships 注册双方 UID，断言日志条数、Action、ActorName/Type、Hit.Caliber=127 与 TotalDamage。
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

// TestHurtPlaneRecordsHitAndShotDown 验证飞机被命中与击落分别记录，并保留空战 ×3 伤害。
// 场景：A6M 用 20mm 炮弹先对 F4F 造成 10 点基础伤害，再用 1000 点基础伤害将其击落。
// 预期：首条为 BattleLogPlaneHit 且伤害 30；致死后无额外命中条目，击落记录 Damage=3000、TotalDamage=3030。
// 覆盖：Arena.Planes 注册双方，断言日志 Action、Actor/Target 名称与类型、伤害倍率及累计值。
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

// TestBulletWeaponMetadata 验证从弹药名解析炸弹重量与口径文本的规则。
// 场景：名称可能带 BB 公斤段（250/5000）、GB/TB 等口径段、7.7/12.7 小数口径，也可能是不含 BB 的非法名。
// 预期：bulletBombWeight 取 BB 后整数、坏名给 0；bulletCaliber 只认 GB/TB/AIR/AST，BB 名给空串。
// 覆盖：断言 250、5000、0 三种重量，以及 7.7、7.62、12.7、20、533 与空串的口径输出。
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

	info := bulletHitInfo(&objBullet.Bullet{
		Type: objBullet.TypeShell, Name: "US/GB/127/1943",
		CriticalType: objBullet.CriticalTypeThreeTimes,
	})
	require.Equal(t, "127", info.Caliber)
	require.Equal(t, objBullet.CriticalTypeThreeTimes, info.Critical)
}

// TestBattleReportScrollbarDragMapping 验证战况/舰艇/飞机页滚动条命中后的拖动目标与最大滚动值。
// 场景：战况 30 条、己方明细各 30 行；分别在三条滚动条轨道左上角做命中测试并拖到底部。
// 预期：命中分别给出 Events、DetailLeft、DetailLeft；拖到底后偏移等于各自 MaxScroll，而不是按像素越界。
// 覆盖：由 layout 与 state 滚动条几何算出轨道位置，断言命中类型与拖到底后的 MaxScroll。
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
	require.Equal(t, state.BattleReportScrollDragDetailLeft, kind)

	manager.state.UI.BattleReportScrollDrag = kind
	manager.dragBattleReportScrollTo(int(scrollbar.TrackY + scrollbar.TrackH - scrollbar.ThumbH))
	require.Equal(t, scrollbar.MaxScroll, manager.state.UI.BattleReportDetailScroll[0])

	// 飞机页复用同一组明细滚动条：拖动目标与滚动状态都应走 DetailLeft。
	manager.state.UI.BattleReportPage = state.BattleReportPagePlanes
	manager.state.UI.BattleReportDetailScroll = [2]int{}
	own.PlanesInitialByType = map[string]int{}
	for idx := 0; idx < 30; idx++ {
		own.PlanesInitialByType[fmt.Sprintf("Plane%02d", idx)] = 1
	}
	reportLayout = manager.BattleReportLayout(true)
	scrollbar, ok = state.BattleReportDetailScrollbar(
		reportLayout.DetailSides[0], own.PlaneTypeCount(), 0, reportLayout.Scale,
	)
	require.True(t, ok)
	kind, _, ok = manager.battleReportScrollbarAt(
		int(scrollbar.TrackX+1), int(scrollbar.TrackY+1),
	)
	require.True(t, ok)
	require.Equal(t, state.BattleReportScrollDragDetailLeft, kind)

	manager.state.UI.BattleReportScrollDrag = kind
	manager.dragBattleReportScrollTo(int(scrollbar.TrackY + scrollbar.TrackH - scrollbar.ThumbH))
	require.Equal(t, scrollbar.MaxScroll, manager.state.UI.BattleReportDetailScroll[0])
}

// TestCombatantInfoUsesBulletShooterSnapshot 验证战报优先使用弹药发射时快照的攻击者名称与类型。
// 场景：弹药 Shooter 指向已不在 Arena 的 gone-ship/gone-plane，但 ShooterName、ShooterType 已在发射时写入。
// 预期：仍返回 yukikaze/destroyer（isPlane=false）与 A6M5/fighter（isPlane=true），不因单位消失而丢失署名。
// 覆盖：构造空 MissionState 与两种 ShooterObjType 的 Bullet，断言 combatantInfo 的三个返回值。
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

// TestHandleBattleReportEventFilterClick 验证战况页「全部/击杀」按钮点击切换筛选并复位滚动。
// 场景：日志含 1 条命中与 1 条击沉，当前滚动为 4，先点索引 1 的击杀按钮，再点索引 0 的全部按钮。
// 预期：两次都返回命中；切到击杀后筛选为 Kills 且滚动清零，切回全部后筛选恢复 All。
// 覆盖：用 layout.EventFilters 取按钮中心，断言点击返回值、筛选值与滚动复位。
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

// TestScrollBattleReportWheelAccumulatesTrackpadDeltas 验证触控板小数滚轮先累积到 1 个标准刻度再翻行。
// 场景：30 条战况、指针在 1280x720 战报 Content 中心，连续 4 次调用 ScrollBattleReportByWheel(-0.25)。
// 预期：前 3 次累积不足 1 个刻度，BattleReportEventScroll 保持 0；第 4 次凑满 -1 刻度后滚动 reportWheelRowsPerUnit=3 行。
// 覆盖：consumeReportWheelRows 的 reportWheelAccum 累加/取整分支与 ScrollBattleReportEvents，断言归零后跳到 3。
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

// TestScrollBattleReportWheelIgnoresOutsidePanel 验证滚轮只作用于战况列表面板内的指针位置。
// 场景：30 条战况的战报页上，指针位于屏幕左上角 (0,0)，明显落在 layout.Content 之外。
// 预期：ScrollBattleReportByWheel 直接返回并把累积量清零，BattleReportEventScroll 保持 0。
// 覆盖：newEventReportManager(30) 后以 -1 滚动，断言事件滚动偏移不变。
func TestScrollBattleReportWheelIgnoresOutsidePanel(t *testing.T) {
	manager := newEventReportManager(30)
	manager.ScrollBattleReportByWheel(-1, 0, 0)
	require.Zero(t, manager.state.UI.BattleReportEventScroll)
}
