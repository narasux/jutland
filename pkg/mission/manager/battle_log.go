package manager

import (
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// combatantInfo 优先使用弹药发射瞬间记录的攻击者信息；旧弹药再按 UID 回查。
func (m *MissionManager) combatantInfo(bt *objBullet.Bullet) (string, string, bool) {
	isPlane := bt.ShooterObjType == object.TypePlane
	if bt.ShooterName != "" {
		return bt.ShooterName, bt.ShooterType, isPlane
	}
	if bt.ShooterObjType == object.TypePlane {
		if plane := m.state.Arena.Planes[bt.Shooter]; plane != nil {
			return plane.Name, string(plane.Type), true
		}
	}
	if ship := m.state.Arena.Ships[bt.Shooter]; ship != nil {
		return ship.Name, string(ship.Type), false
	}
	if plane := m.state.Arena.Planes[bt.Shooter]; plane != nil {
		return plane.Name, string(plane.Type), true
	}
	return "", "", isPlane
}

// markBulletShooter 在弹药发射瞬间写入攻击者名称与类型。
func markBulletShooter(bullets []*objBullet.Bullet, name, unitType string) {
	for _, bt := range bullets {
		bt.ShooterName = name
		bt.ShooterType = unitType
	}
}

// bulletHitInfo 把弹药信息转成战报需要的武器与伤害结构。
func bulletHitInfo(bt *objBullet.Bullet) objUnit.HitInfo {
	info := objUnit.HitInfo{
		WeaponType: string(bt.Type),
		Diameter:   bt.Diameter,
		Caliber:    bulletCaliber(bt.Name),
		Critical:   bt.CriticalType,
	}
	if bt.Type == objBullet.TypeBomb {
		info.Weight = bulletBombWeight(bt.Name)
	}
	return info
}

// bulletCaliber 从 JP/GB/7.7/1922 这类弹药名里取精确口径文本。
func bulletCaliber(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return ""
	}
	switch parts[1] {
	case "GB", "TB", "AIR", "AST":
		return parts[2]
	}
	return ""
}

// bulletBombWeight 从 SU/BB/250/1938 这类名字里取炸弹公斤数。
func bulletBombWeight(name string) int {
	parts := strings.Split(name, "/")
	for idx := 0; idx+1 < len(parts); idx++ {
		if parts[idx] != "BB" {
			continue
		}
		if weight, err := strconv.Atoi(parts[idx+1]); err == nil {
			return weight
		}
	}
	return 0
}

// hurtShip 记录攻击者后结算伤害；未沉没时追加一条命中记录，击沉由 RecordShipLost 记录。
func (m *MissionManager) hurtShip(ship *objUnit.BattleShip, bt *objBullet.Bullet) {
	if ship == nil || ship.CurHP <= 0 {
		return
	}
	name, actorType, isPlane := m.combatantInfo(bt)
	ship.LastAttackerName = name
	ship.LastAttackerType = actorType
	ship.LastAttackerPlayer = bt.BelongPlayer
	ship.LastAttackerIsPlane = isPlane

	beforeDamage := bt.RealDamage
	ship.HurtBy(bt)
	info := bulletHitInfo(bt)
	info.Damage = bt.RealDamage - beforeDamage
	ship.LastHit = info
	ship.DamageTaken += info.Damage
	if ship.CurHP <= 0 {
		ship.SunkByHit = true
		return
	}
	m.state.Stats.RecordBattleLog(state.BattleLogEntry{
		Tick:         objUnit.SimTick(),
		ActorPlayer:  bt.BelongPlayer,
		ActorName:    name,
		ActorType:    actorType,
		ActorIsPlane: isPlane,
		TargetPlayer: ship.BelongPlayer,
		TargetName:   ship.Name,
		TargetType:   string(ship.Type),
		Action:       state.BattleLogShipHit,
		Count:        1,
		Hit:          info,
	})
}

// hurtPlane 记录攻击者后结算伤害；未损失时追加一条命中记录，损失由 RecordPlaneLost 记录。
func (m *MissionManager) hurtPlane(plane *objUnit.Plane, bt *objBullet.Bullet) {
	if plane == nil || plane.CurHP <= 0 {
		return
	}
	name, actorType, isPlane := m.combatantInfo(bt)
	plane.LastAttackerName = name
	plane.LastAttackerType = actorType
	plane.LastAttackerPlayer = bt.BelongPlayer
	plane.LastAttackerIsPlane = isPlane

	beforeDamage := bt.RealDamage
	plane.HurtBy(bt)
	info := bulletHitInfo(bt)
	info.Damage = bt.RealDamage - beforeDamage
	plane.LastHit = info
	plane.DamageTaken += info.Damage
	if plane.CurHP <= 0 {
		plane.ShotDown = true
		return
	}
	m.state.Stats.RecordBattleLog(state.BattleLogEntry{
		Tick:          objUnit.SimTick(),
		ActorPlayer:   bt.BelongPlayer,
		ActorName:     name,
		ActorType:     actorType,
		ActorIsPlane:  isPlane,
		TargetPlayer:  plane.BelongPlayer,
		TargetName:    plane.Name,
		TargetType:    string(plane.Type),
		TargetIsPlane: true,
		Action:        state.BattleLogPlaneHit,
		Count:         1,
		Hit:           info,
	})
}

// ScrollBattleReportEvents 滚动战况记录；0 为最新，增大向历史翻。
func (m *MissionManager) ScrollBattleReportEvents(delta int) {
	if m.state == nil || delta == 0 {
		return
	}
	total := state.FilteredBattleLogCount(
		m.state.Stats.BattleLog, m.state.UI.BattleReportEventFilter,
	)
	maxScroll := max(0, total-state.BattleReportEventRows)
	next := m.state.UI.BattleReportEventScroll + delta
	m.state.UI.BattleReportEventScroll = min(max(0, next), maxScroll)
}

// HandleBattleReportEventFilterClick 处理详情页「全部 / 击杀」筛选点击。
func (m *MissionManager) HandleBattleReportEventFilterClick(x, y int) bool {
	if m.state == nil || m.state.UI.BattleReportPage != state.BattleReportPageEvents {
		return false
	}
	layout := m.BattleReportLayout(true)
	for idx, rect := range layout.EventFilters {
		if !rect.Contains(x, y) {
			continue
		}
		filter := state.BattleReportEventFilter(idx)
		if filter != m.state.UI.BattleReportEventFilter {
			m.state.UI.BattleReportEventFilter = filter
			m.state.UI.BattleReportEventScroll = 0
		}
		return true
	}
	return false
}

// UpdateBattleReportScrollDrag 处理战报滚动条的按下 / 拖动 / 释放。
// 返回 true 表示本次鼠标事件已被滚动条消费，调用方不要再处理页签或按钮点击。
func (m *MissionManager) UpdateBattleReportScrollDrag(x, y int) bool {
	if m.state == nil {
		return false
	}
	if m.state.UI.BattleReportScrollDrag != state.BattleReportScrollDragNone {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			m.state.UI.BattleReportScrollDrag = state.BattleReportScrollDragNone
			m.state.UI.BattleReportScrollDragOffset = 0
			return true
		}
		m.dragBattleReportScrollTo(y)
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	kind, scrollbar, ok := m.battleReportScrollbarAt(x, y)
	if !ok {
		return false
	}
	offset := scrollbar.ThumbH / 2
	cursorY := float64(y)
	if cursorY >= scrollbar.ThumbY && cursorY <= scrollbar.ThumbY+scrollbar.ThumbH {
		offset = cursorY - scrollbar.ThumbY
	}
	m.state.UI.BattleReportScrollDrag = kind
	m.state.UI.BattleReportScrollDragOffset = offset
	m.dragBattleReportScrollTo(y)
	return true
}

// battleReportScrollbarAt 返回鼠标命中的滚动条及其拖动目标。
func (m *MissionManager) battleReportScrollbarAt(
	x, y int,
) (int, state.BattleReportScrollbar, bool) {
	layout := m.BattleReportLayout(true)
	if m.state.UI.BattleReportPage == state.BattleReportPageEvents {
		total := state.FilteredBattleLogCount(
			m.state.Stats.BattleLog, m.state.UI.BattleReportEventFilter,
		)
		scrollbar, ok := state.BattleReportEventScrollbar(
			layout.Content, total,
			m.state.UI.BattleReportEventScroll, layout.Scale,
		)
		if ok && battleReportScrollbarHit(scrollbar, x, y, layout.Scale) {
			return state.BattleReportScrollDragEvents, scrollbar, true
		}
		return 0, state.BattleReportScrollbar{}, false
	}
	if m.state.UI.BattleReportPage != state.BattleReportPageShips {
		return 0, state.BattleReportScrollbar{}, false
	}
	for side := 0; side < 2; side++ {
		stats := m.battleReportSideStats(side)
		scrollbar, ok := state.BattleReportDetailScrollbar(
			layout.DetailSides[side], stats.ShipClassCount(),
			m.state.UI.BattleReportShipScroll[side], layout.Scale,
		)
		if !ok || !battleReportScrollbarHit(scrollbar, x, y, layout.Scale) {
			continue
		}
		kind := state.BattleReportScrollDragShipsLeft
		if side == 1 {
			kind = state.BattleReportScrollDragShipsRight
		}
		return kind, scrollbar, true
	}
	return 0, state.BattleReportScrollbar{}, false
}

// battleReportScrollbarHit 判定鼠标是否落在滚动条上，横向命中区略放宽。
func battleReportScrollbarHit(scrollbar state.BattleReportScrollbar, x, y int, scale float64) bool {
	fx, fy := float64(x), float64(y)
	padX := 5 * scale
	return fx >= scrollbar.TrackX-padX &&
		fx <= scrollbar.TrackX+scrollbar.TrackW+padX &&
		fy >= scrollbar.TrackY &&
		fy <= scrollbar.TrackY+scrollbar.TrackH
}

// dragBattleReportScrollTo 根据鼠标 Y 计算拖动后的滚动位置。
func (m *MissionManager) dragBattleReportScrollTo(y int) {
	scrollbar, ok := m.currentBattleReportScrollbar(m.state.UI.BattleReportScrollDrag)
	if !ok {
		m.state.UI.BattleReportScrollDrag = state.BattleReportScrollDragNone
		m.state.UI.BattleReportScrollDragOffset = 0
		return
	}
	available := scrollbar.TrackH - scrollbar.ThumbH
	if available <= 0 {
		m.setBattleReportScroll(m.state.UI.BattleReportScrollDrag, 0)
		return
	}
	ratio := (float64(y) - scrollbar.TrackY - m.state.UI.BattleReportScrollDragOffset) / available
	ratio = min(max(0, ratio), 1)
	target := int(math.Round(ratio * float64(scrollbar.MaxScroll)))
	m.setBattleReportScroll(m.state.UI.BattleReportScrollDrag, target)
}

// currentBattleReportScrollbar 按拖动目标重新计算当前滚动条。
func (m *MissionManager) currentBattleReportScrollbar(kind int) (state.BattleReportScrollbar, bool) {
	layout := m.BattleReportLayout(true)
	switch kind {
	case state.BattleReportScrollDragEvents:
		if m.state.UI.BattleReportPage != state.BattleReportPageEvents {
			return state.BattleReportScrollbar{}, false
		}
		total := state.FilteredBattleLogCount(
			m.state.Stats.BattleLog, m.state.UI.BattleReportEventFilter,
		)
		return state.BattleReportEventScrollbar(
			layout.Content, total,
			m.state.UI.BattleReportEventScroll, layout.Scale,
		)
	case state.BattleReportScrollDragShipsLeft, state.BattleReportScrollDragShipsRight:
		if m.state.UI.BattleReportPage != state.BattleReportPageShips {
			return state.BattleReportScrollbar{}, false
		}
		side := 0
		if kind == state.BattleReportScrollDragShipsRight {
			side = 1
		}
		return state.BattleReportDetailScrollbar(
			layout.DetailSides[side], m.battleReportSideStats(side).ShipClassCount(),
			m.state.UI.BattleReportShipScroll[side], layout.Scale,
		)
	}
	return state.BattleReportScrollbar{}, false
}

// battleReportSideStats 返回战报左侧（0）或右侧（1）阵营的战果。
func (m *MissionManager) battleReportSideStats(side int) state.BattleSideStats {
	player := m.state.Player.CurPlayer
	if side == 1 {
		player = m.state.Player.CurEnemy
	}
	return m.state.Stats.Side(player)
}

// setBattleReportScroll 写入拖动目标对应的滚动偏移。
func (m *MissionManager) setBattleReportScroll(kind, value int) {
	switch kind {
	case state.BattleReportScrollDragEvents:
		m.state.UI.BattleReportEventScroll = max(0, value)
	case state.BattleReportScrollDragShipsLeft:
		m.state.UI.BattleReportShipScroll[0] = max(0, value)
	case state.BattleReportScrollDragShipsRight:
		m.state.UI.BattleReportShipScroll[1] = max(0, value)
	}
}

// ScrollBattleReportByWheel 按当前战报页与鼠标位置分发滚轮。
// 只有指针落在对应列表面板内才滚动，避免在页签 / 页脚上误滚。
func (m *MissionManager) ScrollBattleReportByWheel(dy float64, cursorX, cursorY int) {
	if m.state == nil || dy == 0 {
		return
	}
	layout := m.BattleReportLayout(true)
	switch m.state.UI.BattleReportPage {
	case state.BattleReportPageEvents:
		if !layout.Content.Contains(cursorX, cursorY) {
			m.reportWheelAccum = 0
			return
		}
		m.ScrollBattleReportEvents(m.consumeReportWheelRows(dy))
	case state.BattleReportPageShips:
		for side := range layout.DetailSides {
			if !layout.DetailSides[side].Contains(cursorX, cursorY) {
				continue
			}
			m.scrollBattleReportShip(m.consumeReportWheelRows(dy), side)
			return
		}
		m.reportWheelAccum = 0
	}
}

// consumeReportWheelRows 把滚轮增量换算成列表行数。
// 触控板会按帧上报小数增量，直接取整再兜底 ±1 会让一次轻扫滚出去很远；
// 这里先累加到 1 个标准刻度，再按每刻度固定行数换算。
func (m *MissionManager) consumeReportWheelRows(dy float64) int {
	if dy == 0 {
		return 0
	}
	// 反向滚动先丢弃上一方向的残余，避免转方向后要先把旧累积抵消。
	if (dy > 0) != (m.reportWheelAccum >= 0) {
		m.reportWheelAccum = 0
	}
	m.reportWheelAccum += dy
	units := int(m.reportWheelAccum)
	if units == 0 {
		return 0
	}
	m.reportWheelAccum -= float64(units)
	rows := -units * reportWheelRowsPerUnit
	return max(-reportWheelMaxRowsPerEvent, min(rows, reportWheelMaxRowsPerEvent))
}

// scrollBattleReportShip 滚动一侧舰级列表，并按该侧舰级数夹紧。
func (m *MissionManager) scrollBattleReportShip(rows, side int) {
	if side < 0 || side > 1 || rows == 0 {
		return
	}
	player := m.state.Player.CurPlayer
	if side == 1 {
		player = m.state.Player.CurEnemy
	}
	layout := m.BattleReportLayout(true)
	visible := state.BattleReportDetailVisibleRows(layout.DetailSides[side].H, layout.Scale)
	total := m.state.Stats.Side(player).ShipClassCount()
	maxScroll := max(0, total-visible)
	if maxScroll == 0 {
		m.state.UI.BattleReportShipScroll[side] = 0
		return
	}
	next := m.state.UI.BattleReportShipScroll[side] + rows
	m.state.UI.BattleReportShipScroll[side] = min(max(0, next), maxScroll)
}

// ClearBattleStats 关闭结算后清空本局战果与详细战况，释放内存。
func (m *MissionManager) ClearBattleStats() {
	if m.state == nil {
		return
	}
	m.state.Stats = state.NewBattleStats()
	m.state.UI.BattleReportPage = state.BattleReportPageOverview
	m.state.UI.BattleReportEventScroll = 0
	m.state.UI.BattleReportEventFilter = state.BattleReportEventFilterAll
	m.state.UI.BattleReportShipScroll = [2]int{}
	m.state.UI.BattleReportScrollDrag = state.BattleReportScrollDragNone
	m.state.UI.BattleReportScrollDragOffset = 0
}
