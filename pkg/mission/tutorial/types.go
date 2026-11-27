package tutorial

import (
	"github.com/narasux/jutland/pkg/i18n"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// Mode 教学步骤的推进方式。
type Mode string

const (
	// ModeInfo 信息卡：冻结战场，只等待下一步/跳过/退出。
	ModeInfo Mode = "info"
	// ModeGuided 操作步骤：战场继续运行，教学层只观察状态。
	ModeGuided Mode = "guided"
	// ModeScripted 演示步骤：保留给自动演示，第一版未使用。
	ModeScripted Mode = "scripted"
	// ModeFree 自由练习：不自动判定，点“完成练习”结束。
	ModeFree Mode = "free"
)

func (m Mode) Valid() bool {
	switch m {
	case ModeInfo, ModeGuided, ModeScripted, ModeFree:
		return true
	default:
		return false
	}
}

// CompletionKind 单个步骤的完成条件。
type CompletionKind string

const (
	// CompletionManual 只能通过下一步/完成按钮推进。
	CompletionManual CompletionKind = "manual"
	// CompletionSelectedShips 选中舰船数量达到 Min。
	CompletionSelectedShips CompletionKind = "selectedShips"
	// CompletionShipMoved 选中舰船相对进入步骤时移动达到 Distance 格。
	CompletionShipMoved CompletionKind = "shipMoved"
	// CompletionGroupAssigned 选中舰船被写入 Group 编组。
	CompletionGroupAssigned CompletionKind = "groupAssigned"
	// CompletionGroupSelected 当前选中的编组等于 Group。
	CompletionGroupSelected CompletionKind = "groupSelected"
	// CompletionDiveState 选中潜艇的潜航状态相对进入步骤时发生变化。
	CompletionDiveState CompletionKind = "diveState"
	// CompletionSonarVisible 按住 V 的声呐范围可见。
	CompletionSonarVisible CompletionKind = "sonarVisible"
	// CompletionMapOpened 打开 M 全图。
	CompletionMapOpened CompletionKind = "mapOpened"
	// CompletionMapReturned 从全图返回战场。
	CompletionMapReturned CompletionKind = "mapReturned"
	// CompletionPaused 进入暂停。
	CompletionPaused CompletionKind = "paused"
	// CompletionResumed 从暂停恢复。
	CompletionResumed CompletionKind = "resumed"
	// CompletionZoomChanged 缩放档位相对进入步骤时发生变化。
	CompletionZoomChanged CompletionKind = "zoomChanged"
	// CompletionCameraMoved 相机相对进入步骤时发生位移。
	CompletionCameraMoved CompletionKind = "cameraMoved"
	// CompletionScoutLaunched 手动侦察机数量增加。
	CompletionScoutLaunched CompletionKind = "scoutLaunched"
	// CompletionReinforceOpened 打开增援页。
	CompletionReinforceOpened CompletionKind = "reinforceOpened"
	// CompletionReinforceSummoned 增援队列长度增加。
	CompletionReinforceSummoned CompletionKind = "reinforceSummoned"
	// CompletionReinforceCanceled 增援队列长度减少。
	CompletionReinforceCanceled CompletionKind = "reinforceCanceled"
	// CompletionReinforceClosed 从增援页返回战场。
	CompletionReinforceClosed CompletionKind = "reinforceClosed"
	// CompletionSidebarExpanded 右侧战术面板展开。
	CompletionSidebarExpanded CompletionKind = "sidebarExpanded"
	// CompletionTerminalOpened 进入 Hacker 终端。
	CompletionTerminalOpened CompletionKind = "terminalOpened"
	// CompletionAirfieldProduction 陆地机场切换了生产机型。
	CompletionAirfieldProduction CompletionKind = "airfieldProduction"
	// CompletionEnemyHit 我方命中敌方舰船（可按武器类型 / 攻击者类型筛选）。
	CompletionEnemyHit CompletionKind = "enemyHit"
	// CompletionPlaneShotDown 我方击落一架敌机（防空炮或战斗机皆可）。
	CompletionPlaneShotDown CompletionKind = "planeShotDown"
	// CompletionAirStrike 我方舰载机已受命攻击敌方舰船。
	CompletionAirStrike CompletionKind = "airStrike"
)

func (k CompletionKind) Valid() bool {
	switch k {
	case CompletionManual, CompletionSelectedShips, CompletionShipMoved,
		CompletionGroupAssigned, CompletionGroupSelected, CompletionDiveState,
		CompletionSonarVisible, CompletionMapOpened, CompletionMapReturned,
		CompletionPaused, CompletionResumed, CompletionZoomChanged,
		CompletionCameraMoved, CompletionScoutLaunched, CompletionReinforceOpened,
		CompletionReinforceSummoned, CompletionReinforceCanceled, CompletionReinforceClosed,
		CompletionSidebarExpanded, CompletionTerminalOpened, CompletionAirfieldProduction,
		CompletionEnemyHit, CompletionPlaneShotDown, CompletionAirStrike:
		return true
	default:
		return false
	}
}

// Localized 一段按语言保存的文案。
type Localized struct {
	values map[i18n.Language]string
}

// Value 按当前语言与既有回退顺序返回文案。
func (l Localized) Value() string {
	return i18n.LocalizedValue(l.values)
}

func newLocalized(zhHans, zhHant, en, ru, ja string) Localized {
	return Localized{values: map[i18n.Language]string{
		i18n.LanguageZhHans:   zhHans,
		i18n.LanguageZhHant:   zhHant,
		i18n.LanguageEnglish:  en,
		i18n.LanguageRussian:  ru,
		i18n.LanguageJapanese: ja,
	}}
}

// Script 一个教学章节的完整脚本。
type Script struct {
	ID      string
	Mission string
	Title   Localized
	Steps   []Step
}

// Step 单个教学步骤。
type Step struct {
	ID         string
	Mode       Mode
	Title      Localized
	Body       Localized
	Hint       Hint
	Setup      []SetupAction
	Completion Completion
	AllowSkip  bool
	// Target 可选的地图目标点；有值时绘制脉冲圈，提示玩家点哪里。
	Target *objPos.MapPos
}

// Hint 键位/鼠标提示。
type Hint struct {
	Actions []ActionID
	Hold    bool
}

// SetupAction 进入步骤时执行的安全状态调整。
type SetupAction string

const (
	// SetupClearSelection 清空舰船选择、焦点舰和编组选择。
	SetupClearSelection SetupAction = "clear-selection"
	// SetupSelectPlayerFleet 选中当前玩家的全部存活舰船。
	SetupSelectPlayerFleet SetupAction = "select-player-fleet"
	// SetupOpenSidebar 展开右侧战术面板。
	SetupOpenSidebar SetupAction = "open-sidebar"
	// SetupCloseSidebar 收起右侧战术面板。
	SetupCloseSidebar SetupAction = "close-sidebar"
	// SetupFocusAirfield 把镜头直接居中到己方陆地机场，用于讲解机场面板。
	SetupFocusAirfield SetupAction = "focus-airfield"
	// SetupDisarmEnemy 关闭敌方舰船的全部武器，让教学靶舰不还击。
	SetupDisarmEnemy SetupAction = "disarm-enemy"
	// SetupOrderAirStrike 让己方航母先对最近敌舰下达空袭指令，便于演示舰载机出击。
	SetupOrderAirStrike SetupAction = "order-air-strike"
)

func (s SetupAction) Valid() bool {
	switch s {
	case SetupClearSelection, SetupSelectPlayerFleet, SetupOpenSidebar, SetupCloseSidebar,
		SetupFocusAirfield, SetupDisarmEnemy, SetupOrderAirStrike:
		return true
	default:
		return false
	}
}

// Completion 完成条件参数。
type Completion struct {
	Kind     CompletionKind
	Min      int
	Distance float64
	Group    int
	// Weapon 命中武器类型（shell / torpedo / bomb / rocket），仅 enemyHit 使用；空表示不限。
	Weapon string
	// Actor 攻击者类型（ship / plane），仅 enemyHit 使用；空表示不限。
	Actor string
}
