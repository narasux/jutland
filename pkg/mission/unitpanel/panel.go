package unitpanel

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// systemTabHeight 是武器/航空页签的局部屏幕像素高度。
const systemTabHeight = 30.0

// Tab 表示单位面板当前展示的系统页。
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
	hitWeaponTab hitKind = iota
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
	Rect       Rect
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

// Panel 管理右侧合并面板「地图 + 战舰信息」页签内的单位信息内容。
// 内容在滚动视口中纵向堆叠（视觉 → 基础信息 → 系统控制），超出视口高度时由外层滚动条处理。
type Panel struct {
	layout    panelLayout
	tab       Tab
	lastFocus string
	hits      []hitRegion
	// targetRect 是单舰有攻击目标时，"→"居中到目标按钮的位置。
	targetRect Rect
	hasTarget  bool
	// renderBuf 是用于裁剪滚动内容的离屏缓冲，尺寸随视口变化。
	renderBuf *ebiten.Image
	// viewport 是本帧内容所在的滚动视口（屏幕坐标），用于把光标换算到局部坐标做悬停。
	viewport Rect
}

// cursorAt 判断屏幕光标（换算到视口局部坐标）是否落在 area 内。
func (p *Panel) cursorAt(area Rect) bool {
	x, y := ebiten.CursorPosition()
	return area.contains(x-int(p.viewport.X), y-int(p.viewport.Y))
}

// New 创建单位信息内容面板。
func New() *Panel { return &Panel{tab: TabWeapons} }

// MeasureContent 返回当前状态下内容相对于滚动视口的自然总高度，供外层计算滚动条范围。
// 视口高度 region.H 不会被用于压缩内容；溢出交由滚动条处理。
func (p *Panel) MeasureContent(ms *state.MissionState, region Rect) float64 {
	return p.calcLayout(ms, region, 0).contentHeight
}

// Update 读取鼠标输入并返回本帧产生的面板操作。
func (p *Panel) Update(ms *state.MissionState, region Rect, scrollY float64) []Action {
	sx, sy := ebiten.CursorPosition()
	return p.updateWithPointer(ms, region, scrollY, pointerInput{
		X:           sx,
		Y:           sy,
		JustPressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
	})
}

// updateWithPointer 使用显式鼠标快照刷新点击区域并处理本帧操作。
// region 为滚动视口的屏幕坐标，pointer 为屏幕坐标；命中检测会换算到视口局部坐标。
func (p *Panel) updateWithPointer(ms *state.MissionState, region Rect, scrollY float64, pointer pointerInput) []Action {
	if ms.Core.MissionStatus != state.MissionRunning {
		p.hits = p.hits[:0]
		return nil
	}
	p.layout = p.calcLayout(ms, region, scrollY)
	p.viewport = region
	p.syncDefaultTab(ms)
	p.rebuildHits(ms)

	if !pointer.JustPressed {
		return nil
	}
	local := pointerInput{X: pointer.X - int(region.X), Y: pointer.Y - int(region.Y)}
	for _, hit := range p.hits {
		if !hit.Rect.contains(local.X, local.Y) {
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

// rebuildHits 根据本帧布局与数据重建全部可点击区域（局部坐标）。
func (p *Panel) rebuildHits(ms *state.MissionState) {
	p.hits = p.hits[:0]
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
			p.hasTarget = true
			p.targetRect = p.computeTargetRect(ms, ships[0])
			p.hits = append(p.hits, hitRegion{
				Rect:    p.targetButtonRect(),
				Kind:    hitCenterTarget,
				ShipUid: target.Uid,
			})
		} else {
			p.hasTarget = false
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

func (p *Panel) tabRects() (Rect, Rect) {
	width := math.Min(96.0, (p.layout.Systems.W-8)/2)
	return Rect{X: p.layout.Systems.X, Y: p.layout.Systems.Y, W: width, H: systemTabHeight},
		Rect{X: p.layout.Systems.X + width + 8, Y: p.layout.Systems.Y, W: width, H: systemTabHeight}
}

func (p *Panel) allWeaponsToggleRect() Rect {
	return Rect{X: p.layout.Systems.X + p.layout.Systems.W - 36, Y: p.layout.Systems.Y + 34, W: 32, H: 32}
}

// weaponRowStep 返回固定的武器行步进；内容溢出交由滚动条处理，不再压缩行高。
func (p *Panel) weaponRowStep(_ int) float64 {
	return weaponRowH
}

// weaponToggleRect 返回第 index 个武器行左侧状态开关的点击区域（围绕状态圆点的小方块）。
func (p *Panel) weaponToggleRect(index, rowCount int) Rect {
	step := p.weaponRowStep(rowCount)
	y := p.layout.Systems.Y + 66 + float64(index)*step
	return Rect{X: p.layout.Systems.X + 8, Y: y + 5, W: 32, H: 32}
}

func (p *Panel) aircraftToggleRect() Rect {
	return Rect{X: p.layout.Systems.X + p.layout.Systems.W - 36, Y: p.layout.Systems.Y + 34, W: 32, H: 32}
}

func (p *Panel) previousFocusRect() Rect {
	return Rect{X: p.layout.Visual.X + p.layout.Visual.W - 58, Y: p.layout.Visual.Y + 4, W: 24, H: 20}
}

func (p *Panel) nextFocusRect() Rect {
	return Rect{X: p.layout.Visual.X + p.layout.Visual.W - 30, Y: p.layout.Visual.Y + 4, W: 24, H: 20}
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
	Rect Rect
}

// visibleFocusShips 返回以焦点舰为中心的一组行；选中过多时只展示 maxFocusRows 行，
// 其余通过上一艘/下一艘按钮循环查看。
func (p *Panel) visibleFocusShips(ships []*objUnit.BattleShip, focusedUid string) []focusEntry {
	rows := min(len(ships), maxFocusRows)
	start := 0
	for index, ship := range ships {
		if ship.Uid == focusedUid {
			start = max(0, index-rows/2)
			break
		}
	}
	if start+rows > len(ships) {
		start = max(0, len(ships)-rows)
	}
	end := min(len(ships), start+rows)
	entries := make([]focusEntry, 0, end-start)
	for index, ship := range ships[start:end] {
		entries = append(entries, focusEntry{
			Ship: ship,
			Rect: Rect{
				X: p.layout.Visual.X + 4,
				Y: p.layout.Visual.Y + 28 + float64(index)*focusRowH,
				W: p.layout.Visual.W - 8,
				H: focusRowH - 4,
			},
		})
	}
	return entries
}
