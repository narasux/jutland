package unit

import (
	"fmt"

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
// 非潜艇与过渡态都返回 false，避免无效指令与计时器抖动。
func (s *BattleShip) NextDiveState() (DiveState, bool) {
	if s.Type != ShipTypeSubmarine {
		return s.DiveState, false
	}
	switch s.DiveState {
	case DiveStateSurface:
		return DiveStateSubmerged, true
	case DiveStateSubmerged:
		return DiveStateSurface, true
	}
	return s.DiveState, false
}

// DiveSpeedRatio 当前深度状态下的航速倍率（相对水面最大速）。
func (s *BattleShip) DiveSpeedRatio() float64 {
	switch s.DiveState {
	case DiveStateDiving, DiveStateSurfacing:
		return diveTransitionSpeedRatio
	case DiveStateSubmerged:
		if s.SubmergedSpeedRatio > 0 {
			return s.SubmergedSpeedRatio
		}
		return defaultSubmergedSpeedRatio
	default:
		return 1
	}
}

// StartDive 开始下潜；非潜艇或非水面状态时拒绝。
func (s *BattleShip) StartDive() bool {
	if s.Type != ShipTypeSubmarine || s.DiveState != DiveStateSurface {
		return false
	}
	s.DiveState = DiveStateDiving
	s.diveTicksLeft = ReloadTicks(s.diveTimeSeconds())
	s.diveTotalTicks = s.diveTicksLeft
	return true
}

// StartSurface 开始上浮；非潜艇或非水下状态时拒绝。
func (s *BattleShip) StartSurface() bool {
	if s.Type != ShipTypeSubmarine || s.DiveState != DiveStateSubmerged {
		return false
	}
	s.DiveState = DiveStateSurfacing
	s.diveTicksLeft = ReloadTicks(s.surfaceTimeSeconds())
	s.diveTotalTicks = s.diveTicksLeft
	return true
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

// DiveActionText 供面板按钮显示：水面时提示「下潜」，水下时提示「上浮」，
// 过渡态直接显示当前状态。
func (s *BattleShip) DiveActionText() string {
	switch s.DiveState {
	case DiveStateSurface:
		return i18n.Text(i18n.MsgShipDive)
	case DiveStateSubmerged:
		return i18n.Text(i18n.MsgShipSurface)
	default:
		return s.DiveStateText()
	}
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
