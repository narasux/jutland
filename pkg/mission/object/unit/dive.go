package unit

import (
	"fmt"
	"math"

	"github.com/narasux/jutland/pkg/i18n"
)

// DiveState 潜艇深度状态。水面与水下是稳定态，下潜中/上浮中是过渡态。
type DiveState int

const (
	// DiveStateSurface 水面
	DiveStateSurface DiveState = iota
	// DiveStateDiving 下潜中
	DiveStateDiving
	// DiveStateSubmerged 水下
	DiveStateSubmerged
	// DiveStateSurfacing 上浮中
	DiveStateSurfacing
)

const (
	// diveTransitionSpeedRatio 过渡态航速倍率（相对水面最大速）。
	diveTransitionSpeedRatio = 0.5
	// defaultSubmergedSpeedRatio 未配置时的水下航速倍率。
	defaultSubmergedSpeedRatio = 0.35
	// defaultDiveTimeSeconds / defaultSurfaceTimeSeconds 未配置时的切换耗时。
	defaultDiveTimeSeconds    = 10
	defaultSurfaceTimeSeconds = 8
)

// Submerged 是否处于完全水下：只能用鱼雷，甲板炮/防空炮锁定。
func (s *BattleShip) Submerged() bool {
	return s.DiveState == DiveStateSubmerged
}

// DiveTransitioning 是否正在下潜/上浮；过渡期按水面规则处理。
func (s *BattleShip) DiveTransitioning() bool {
	return s.DiveState == DiveStateDiving || s.DiveState == DiveStateSurfacing
}

// NextDiveState 返回按下「下潜/上浮」后的目标状态。
// 过渡中也能随时反向切换，以最后一次指令为准：下潜中返回上浮、上浮中返回下潜。
// 非潜艇返回 false。
func (s *BattleShip) NextDiveState() (DiveState, bool) {
	if s.Type != ShipTypeSubmarine {
		return s.DiveState, false
	}
	switch s.DiveState {
	case DiveStateSurface, DiveStateSurfacing:
		return DiveStateSubmerged, true
	case DiveStateSubmerged, DiveStateDiving:
		return DiveStateSurface, true
	}
	return s.DiveState, false
}

// submergedSpeedRatio 水下航速倍率（相对水面最大速），未配置时用默认值。
func (s *BattleShip) submergedSpeedRatio() float64 {
	if s.SubmergedSpeedRatio > 0 {
		return s.SubmergedSpeedRatio
	}
	return defaultSubmergedSpeedRatio
}

// DiveSpeedRatio 当前深度状态下的航速倍率（相对水面最大速）。
func (s *BattleShip) DiveSpeedRatio() float64 {
	switch s.DiveState {
	case DiveStateDiving, DiveStateSurfacing:
		return diveTransitionSpeedRatio
	case DiveStateSubmerged:
		return s.submergedSpeedRatio()
	default:
		return 1
	}
}

// SubmergedMaxSpeedKnots 水下最大航速（节），供图鉴等静态档案展示。
// 非潜艇没有水下状态，直接返回水面最大航速。
func (s *BattleShip) SubmergedMaxSpeedKnots() float64 {
	if s.Type != ShipTypeSubmarine {
		return s.MaxSpeedKnots()
	}
	return s.MaxSpeed * s.submergedSpeedRatio() * ShipSpeedScale
}

// StartDive 开始下潜；潜艇可以在任意时刻改主意：上浮中会直接掉头下潜。
// 反向切换沿用当前潜航进度，按与完整下潜相同的速度回到水下；
// 已经在水下或下潜中时返回 false，不重启计时。
func (s *BattleShip) StartDive() bool {
	if s.Type != ShipTypeSubmarine {
		return false
	}
	switch s.DiveState {
	case DiveStateSurface, DiveStateSurfacing:
	default:
		return false
	}
	progress := s.DiveProgress()
	s.DiveState = DiveStateDiving
	s.diveTotalTicks = ReloadTicks(s.diveTimeSeconds())
	s.diveTicksLeft = transitionTicksLeft(s.diveTotalTicks, 1-progress)
	return true
}

// StartSurface 开始上浮；潜艇可以在任意时刻改主意：下潜中会直接掉头上浮。
// 反向切换沿用当前潜航进度，按与完整上浮相同的速度回到水面；
// 已经在水面或上浮中时返回 false，不重启计时。
func (s *BattleShip) StartSurface() bool {
	if s.Type != ShipTypeSubmarine {
		return false
	}
	switch s.DiveState {
	case DiveStateSubmerged, DiveStateDiving:
	default:
		return false
	}
	progress := s.DiveProgress()
	s.DiveState = DiveStateSurfacing
	s.diveTotalTicks = ReloadTicks(s.surfaceTimeSeconds())
	s.diveTicksLeft = transitionTicksLeft(s.diveTotalTicks, progress)
	return true
}

// transitionTicksLeft 把“剩余比例”换算成剩余拍数。
// 计时总量始终是一次完整下潜/上浮的拍数，所以中途反向不会改变切换速度。
func transitionTicksLeft(total int64, remaining float64) int64 {
	remaining = min(1, max(0, remaining))
	return max(1, int64(math.Round(float64(total)*remaining)))
}

// AdvanceDive 推进过渡计时，由任务主循环每拍调用。
func (s *BattleShip) AdvanceDive() {
	if !s.DiveTransitioning() {
		return
	}
	s.diveTicksLeft--
	if s.diveTicksLeft > 0 {
		return
	}
	switch s.DiveState {
	case DiveStateDiving:
		s.DiveState = DiveStateSubmerged
	case DiveStateSurfacing:
		s.DiveState = DiveStateSurface
	}
	s.diveTicksLeft = 0
	s.diveTotalTicks = 0
}

// DiveProgress 返回 0（水面）到 1（完全水下）的潜航进度，
// 供下潜/上浮过渡动画与状态牌进度条使用。稳定态直接返回 0 或 1。
func (s *BattleShip) DiveProgress() float64 {
	if s.Type != ShipTypeSubmarine {
		return 0
	}
	switch s.DiveState {
	case DiveStateDiving:
		if s.diveTotalTicks <= 0 {
			return 1
		}
		progress := 1 - float64(s.diveTicksLeft)/float64(s.diveTotalTicks)
		return min(1, max(0, progress))
	case DiveStateSubmerged:
		return 1
	case DiveStateSurfacing:
		if s.diveTotalTicks <= 0 {
			return 0
		}
		progress := float64(s.diveTicksLeft) / float64(s.diveTotalTicks)
		return min(1, max(0, progress))
	default:
		return 0
	}
}

// DiveStateText 深度状态的本地化名称，供单位面板使用。
func (s *BattleShip) DiveStateText() string {
	switch s.DiveState {
	case DiveStateDiving:
		return i18n.Text(i18n.MsgShipDiving)
	case DiveStateSubmerged:
		return i18n.Text(i18n.MsgShipSubmerged)
	case DiveStateSurfacing:
		return i18n.Text(i18n.MsgShipSurfacing)
	default:
		return i18n.Text(i18n.MsgShipSurface)
	}
}

// DiveActionText 供面板按钮显示：按钮永远代表“最后一次指令”的反向，
// 水面与上浮中提示「下潜」，水下与下潜中提示「上浮」。
func (s *BattleShip) DiveActionText() string {
	switch s.DiveState {
	case DiveStateSurface, DiveStateSurfacing:
		return i18n.Text(i18n.MsgShipDive)
	case DiveStateSubmerged, DiveStateDiving:
		return i18n.Text(i18n.MsgShipSurfaceAction)
	}
	return s.DiveStateText()
}

// ResolveDiveConfig 补齐潜艇下潜配置的默认值并解析入场深度状态。
// 未配置时默认水下；水面开局需要在配置里显式写 surface。
// 非潜艇不写入深度字段；配置非法时返回错误，由初始化阶段 Fatal。
func (s *BattleShip) ResolveDiveConfig() error {
	if s.Type != ShipTypeSubmarine {
		return nil
	}
	if s.DiveTime <= 0 {
		s.DiveTime = defaultDiveTimeSeconds
	}
	if s.SurfaceTime <= 0 {
		s.SurfaceTime = defaultSurfaceTimeSeconds
	}
	if s.SubmergedSpeedRatio <= 0 {
		s.SubmergedSpeedRatio = defaultSubmergedSpeedRatio
	}
	switch s.DefaultDiveState {
	case "", "submerged":
		s.DiveState = DiveStateSubmerged
	case "surface":
		s.DiveState = DiveStateSurface
	default:
		return fmt.Errorf("ship %q has unknown defaultDiveState %q", s.Name, s.DefaultDiveState)
	}
	return nil
}

func (s *BattleShip) diveTimeSeconds() float64 {
	if s.DiveTime > 0 {
		return s.DiveTime
	}
	return defaultDiveTimeSeconds
}

func (s *BattleShip) surfaceTimeSeconds() float64 {
	if s.SurfaceTime > 0 {
		return s.SurfaceTime
	}
	return defaultSurfaceTimeSeconds
}

// diveStateName 调试面板用的英文状态名；界面文案走 DiveStateText。
func diveStateName(state DiveState) string {
	switch state {
	case DiveStateDiving:
		return "Diving"
	case DiveStateSubmerged:
		return "Submerged"
	case DiveStateSurfacing:
		return "Surfacing"
	default:
		return "Surface"
	}
}
