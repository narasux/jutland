package unitpanel

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

const (
	// systemTabHeight 是武器/航空页签的屏幕像素高度。
	systemTabHeight = 30.0
	// focusRowHeight 是多选舰船列表单行占用的屏幕像素高度。
	focusRowHeight = 30.0
)

// Tab 表示单位面板右侧当前展示的系统页。
type Tab int

const (
	// TabWeapons 展示已装备舰载武器及其控制。
	TabWeapons Tab = iota
	// TabAircraft 展示舰载航空联队及其起飞控制。
	TabAircraft
)

// ActionKind 表示单位面板产生、等待 manager 执行的操作类型。
type ActionKind int

const (
	// ActionFocusShip 切换多选集合中的焦点舰。
	ActionFocusShip ActionKind = iota
	// ActionCenterTarget 将相机一次性居中到焦点舰的攻击目标。
	ActionCenterTarget
	// ActionToggleWeapon 批量启用或禁用一种武器。
	ActionToggleWeapon
	// ActionToggleAircraft 批量允许或禁止舰载机起飞。
	ActionToggleAircraft
)

// Action 是单位面板输出的类型化操作，不直接修改战斗对象。
type Action struct {
	Kind       ActionKind
	ShipUids   []string
	FocusUid   string
	TargetUid  string
	WeaponType objUnit.WeaponType
	Enable     bool
}

type hitKind int

const (
	hitHandle hitKind = iota
	hitWeaponTab
	hitAircraftTab
	hitFocusShip
	hitPreviousFocus
	hitNextFocus
	hitCenterTarget
	hitAllWeapons
	hitWeapon
	hitAircraft
)

type hitRegion struct {
	Rect       rect
	Kind       hitKind
	ShipUid    string
	WeaponType objUnit.WeaponType
}

// pointerInput 是面板本帧需要的最小鼠标快照。
// 将坐标与点击状态显式传入，可保证生产交互和确定性测试共用同一命中逻辑。
type pointerInput struct {
	X, Y        int
	JustPressed bool
}

// Panel 管理底部单位面板的布局、绘制状态与鼠标交互。
type Panel struct {
	layout     panelLayout
	rightInset float64
	tab        Tab
	lastFocus  string
	hits       []hitRegion
}

// New 创建默认折叠的底部单位面板。
func New() *Panel { return &Panel{tab: TabWeapons} }

// OccupiedHeight 返回展开面板遮挡战场的屏幕像素高度。
func (p *Panel) OccupiedHeight(ms *state.MissionState) float64 {
	if !ms.UI.UnitPanelExpanded {
		return 0
	}
	return expandedPanelHeight(ms.View.Layout)
}

// ConsumesCursor 判断当前鼠标位置是否位于底部面板或其把手内。
func (p *Panel) ConsumesCursor(ms *state.MissionState, rightInset float64) bool {
	sx, sy := ebiten.CursorPosition()
	return p.consumesCursorAt(ms, rightInset, sx, sy)
}

// consumesCursorAt 判断给定屏幕坐标是否位于底部面板或其把手内。
func (p *Panel) consumesCursorAt(ms *state.MissionState, rightInset float64, sx, sy int) bool {
	if ms.Core.MissionStatus != state.MissionRunning {
		return false
	}
	ui := calcLayout(ms.View.Layout, ms.UI.UnitPanelExpanded, rightInset)
	if ui.Handle.contains(sx, sy) {
		return true
	}
	return ms.UI.UnitPanelExpanded && ui.Panel.contains(sx, sy)
}

// Update 刷新点击区域并返回本帧产生的面板操作。
func (p *Panel) Update(ms *state.MissionState, rightInset float64) []Action {
	sx, sy := ebiten.CursorPosition()
	return p.updateWithPointer(ms, rightInset, pointerInput{
		X:           sx,
		Y:           sy,
		JustPressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
	})
}

// updateWithPointer 使用显式鼠标快照刷新点击区域并处理本帧操作。
func (p *Panel) updateWithPointer(ms *state.MissionState, rightInset float64, pointer pointerInput) []Action {
	if ms.Core.MissionStatus != state.MissionRunning {
		p.hits = p.hits[:0]
		return nil
	}
	p.rightInset = rightInset
	p.layout = calcLayout(ms.View.Layout, ms.UI.UnitPanelExpanded, rightInset)
	p.syncDefaultTab(ms)
	p.rebuildHits(ms)

	if !pointer.JustPressed {
		return nil
	}
	for _, hit := range p.hits {
		if !hit.Rect.contains(pointer.X, pointer.Y) {
			continue
		}
		return p.activate(ms, hit)
	}
	return nil
}

// syncDefaultTab 在焦点舰改变时按舰种恢复默认系统页。
func (p *Panel) syncDefaultTab(ms *state.MissionState) {
	uid := ms.Interaction.FocusedShipUid
	if uid == p.lastFocus {
		return
	}
	p.lastFocus = uid
	p.tab = TabWeapons
	if ship := focusedShip(ms); ship != nil && ship.Type == objUnit.ShipTypeAircraftCarrier {
		p.tab = TabAircraft
	}
}

// rebuildHits 根据本帧布局与数据重建全部可点击区域。
func (p *Panel) rebuildHits(ms *state.MissionState) {
	p.hits = append(p.hits[:0], hitRegion{Rect: p.layout.Handle, Kind: hitHandle})
	if !ms.UI.UnitPanelExpanded {
		return
	}

	weaponTab, aircraftTab := p.tabRects()
	p.hits = append(p.hits,
		hitRegion{Rect: weaponTab, Kind: hitWeaponTab},
		hitRegion{Rect: aircraftTab, Kind: hitAircraftTab},
	)
	ships := selectedShips(ms)
	if len(ships) == 0 {
		return
	}

	if len(ships) > 1 {
		p.hits = append(p.hits,
			hitRegion{Rect: p.previousFocusRect(), Kind: hitPreviousFocus},
			hitRegion{Rect: p.nextFocusRect(), Kind: hitNextFocus},
		)
		for _, entry := range p.visibleFocusShips(ships, ms.Interaction.FocusedShipUid) {
			p.hits = append(p.hits, hitRegion{
				Rect:    entry.Rect,
				Kind:    hitFocusShip,
				ShipUid: entry.Ship.Uid,
			})
		}
	}
	if len(ships) == 1 {
		if target := p.focusTarget(ms); target != nil {
			p.hits = append(p.hits, hitRegion{
				Rect:    p.targetButtonRect(),
				Kind:    hitCenterTarget,
				ShipUid: target.Uid,
			})
		}
	}

	if p.tab == TabAircraft {
		for _, ship := range ships {
			if ship.Aircraft.HasPlane {
				p.hits = append(p.hits, hitRegion{Rect: p.aircraftToggleRect(), Kind: hitAircraft})
				break
			}
		}
		return
	}

	rows := weaponRows(ms, nowMillis())
	if len(rows) == 0 {
		return
	}
	p.hits = append(p.hits, hitRegion{Rect: p.allWeaponsToggleRect(), Kind: hitAllWeapons})
	for index, row := range rows {
		p.hits = append(p.hits, hitRegion{
			Rect:       p.weaponToggleRect(index, len(rows)),
			Kind:       hitWeapon,
			WeaponType: row.Type,
		})
	}
}

// activate 将命中区域转换为本地页签变化或类型化游戏动作。
func (p *Panel) activate(ms *state.MissionState, hit hitRegion) []Action {
	switch hit.Kind {
	case hitHandle:
		ms.UI.UnitPanelExpanded = !ms.UI.UnitPanelExpanded
		p.layout = calcLayout(ms.View.Layout, ms.UI.UnitPanelExpanded, p.rightInset)
		return nil
	case hitWeaponTab:
		p.tab = TabWeapons
		return nil
	case hitAircraftTab:
		p.tab = TabAircraft
		return nil
	case hitFocusShip:
		return []Action{{Kind: ActionFocusShip, FocusUid: hit.ShipUid}}
	case hitPreviousFocus:
		return []Action{{Kind: ActionFocusShip, FocusUid: p.adjacentFocusUid(ms, -1)}}
	case hitNextFocus:
		return []Action{{Kind: ActionFocusShip, FocusUid: p.adjacentFocusUid(ms, 1)}}
	case hitCenterTarget:
		return []Action{{Kind: ActionCenterTarget, TargetUid: hit.ShipUid}}
	case hitAllWeapons:
		return []Action{p.weaponAction(ms, objUnit.WeaponTypeAll)}
	case hitWeapon:
		return []Action{p.weaponAction(ms, hit.WeaponType)}
	case hitAircraft:
		return []Action{p.aircraftAction(ms)}
	default:
		return nil
	}
}

// weaponAction 按“存在禁用则全部启用，否则全部禁用”的规则生成批量操作。
func (p *Panel) weaponAction(ms *state.MissionState, weaponType objUnit.WeaponType) Action {
	ships := selectedShips(ms)
	action := Action{Kind: ActionToggleWeapon, WeaponType: weaponType}
	anyDisabled := false
	for _, ship := range ships {
		if weaponType == objUnit.WeaponTypeAll {
			hasAny := false
			for _, currentType := range equippedWeaponTypes([]*objUnit.BattleShip{ship}) {
				hasAny = true
				anyDisabled = anyDisabled || weaponDisabled(ship, currentType)
			}
			if hasAny {
				action.ShipUids = append(action.ShipUids, ship.Uid)
			}
			continue
		}
		if !hasWeapon(ship, weaponType) {
			continue
		}
		action.ShipUids = append(action.ShipUids, ship.Uid)
		anyDisabled = anyDisabled || weaponDisabled(ship, weaponType)
	}
	action.Enable = anyDisabled
	return action
}

// aircraftAction 对选区内实际搭载飞机的舰船生成批量起飞控制操作。
func (p *Panel) aircraftAction(ms *state.MissionState) Action {
	action := Action{Kind: ActionToggleAircraft}
	for _, ship := range selectedShips(ms) {
		if !ship.Aircraft.HasPlane {
			continue
		}
		action.ShipUids = append(action.ShipUids, ship.Uid)
		action.Enable = action.Enable || ship.Aircraft.Disable
	}
	return action
}

func (p *Panel) focusTarget(ms *state.MissionState) *objUnit.BattleShip {
	ship := focusedShip(ms)
	if ship == nil || ship.AttackTarget == "" {
		return nil
	}
	return ms.Arena.Ships[ship.AttackTarget]
}

func (p *Panel) tabRects() (rect, rect) {
	width := min(104.0, (p.layout.Systems.W-8)/2)
	return rect{X: p.layout.Systems.X, Y: p.layout.Systems.Y, W: width, H: systemTabHeight},
		rect{X: p.layout.Systems.X + width + 8, Y: p.layout.Systems.Y, W: width, H: systemTabHeight}
}

func (p *Panel) allWeaponsToggleRect() rect {
	return rect{X: p.layout.Systems.X + p.layout.Systems.W - 78, Y: p.layout.Systems.Y + 40, W: 78, H: 26}
}

// weaponRowStep 根据行数压缩武器行，确保五种武器在最低面板高度内可见。
func (p *Panel) weaponRowStep(rowCount int) float64 {
	if rowCount <= 0 {
		return 30
	}
	return min(34, max(21, (p.layout.Systems.H-70)/float64(rowCount)))
}

func (p *Panel) weaponToggleRect(index, rowCount int) rect {
	step := p.weaponRowStep(rowCount)
	y := p.layout.Systems.Y + 68 + float64(index)*step
	return rect{
		X: p.layout.Systems.X + p.layout.Systems.W - 78,
		Y: y,
		W: 78,
		H: min(26, step-2),
	}
}

func (p *Panel) aircraftToggleRect() rect {
	return rect{X: p.layout.Systems.X + p.layout.Systems.W - 90, Y: p.layout.Systems.Y + 40, W: 90, H: 26}
}

func (p *Panel) infoRowHeight() float64 {
	return min(23, max(15, (p.layout.Info.H-24)/10))
}

func (p *Panel) targetButtonRect() rect {
	y := p.layout.Info.Y + 26 + p.infoRowHeight()*5
	return rect{X: p.layout.Info.X + p.layout.Info.W - 30, Y: y - 2, W: 26, H: 20}
}

func (p *Panel) previousFocusRect() rect {
	return rect{X: p.layout.Visual.X + p.layout.Visual.W - 58, Y: p.layout.Visual.Y + 4, W: 24, H: 20}
}

func (p *Panel) nextFocusRect() rect {
	return rect{X: p.layout.Visual.X + p.layout.Visual.W - 30, Y: p.layout.Visual.Y + 4, W: 24, H: 20}
}

// adjacentFocusUid 按稳定展示顺序循环选择相邻焦点舰。
func (p *Panel) adjacentFocusUid(ms *state.MissionState, direction int) string {
	ships := selectedShips(ms)
	if len(ships) == 0 {
		return ""
	}
	index := 0
	for currentIndex, ship := range ships {
		if ship.Uid == ms.Interaction.FocusedShipUid {
			index = currentIndex
			break
		}
	}
	index = (index + direction + len(ships)) % len(ships)
	return ships[index].Uid
}

type focusEntry struct {
	Ship *objUnit.BattleShip
	Rect rect
}

// visibleFocusShips 截取以当前焦点为中心、能够放入视图区的舰船列表窗口。
func (p *Panel) visibleFocusShips(ships []*objUnit.BattleShip, focusedUid string) []focusEntry {
	maxRows := max(1, int((p.layout.Visual.H-30)/focusRowHeight))
	start := 0
	for index, ship := range ships {
		if ship.Uid == focusedUid {
			start = max(0, index-maxRows/2)
			break
		}
	}
	if start+maxRows > len(ships) {
		start = max(0, len(ships)-maxRows)
	}
	end := min(len(ships), start+maxRows)
	entries := make([]focusEntry, 0, end-start)
	for index, ship := range ships[start:end] {
		entries = append(entries, focusEntry{
			Ship: ship,
			Rect: rect{
				X: p.layout.Visual.X + 4,
				Y: p.layout.Visual.Y + 28 + float64(index)*focusRowHeight,
				W: p.layout.Visual.W - 8,
				H: focusRowHeight - 4,
			},
		})
	}
	return entries
}
