package unit

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 声呐/水听器半径（地图格）。探测独立于视距与迷雾，只对水下潜艇生效。
const (
	// ShipSonarRangeEscort 驱逐舰 / 护卫舰的声呐半径
	ShipSonarRangeEscort = 7
	// ShipSonarRangeCapital 巡洋舰 / 战列舰的声呐半径
	ShipSonarRangeCapital = 4
	// ShipSonarRangeCarrier 航空母舰的声呐半径
	ShipSonarRangeCarrier = 3
	// HydrophoneRange 潜艇水听器半径（可探测敌方下潜潜艇）
	HydrophoneRange = 7
	// PlaneSonarRange 飞机目视探测下潜潜艇的半径
	PlaneSonarRange = 2
	// SubmarineDetectionMemorySeconds 失去探测后仍被点亮的时间
	SubmarineDetectionMemorySeconds = 8
)

// ShipDetection 某个玩家最近一次探测到潜艇的记录。
type ShipDetection struct {
	// UntilTick 记忆到期拍；到期前全队仍可看见、可锁定
	UntilTick int64
	// LastPos 最后一次探测到的位置，供 AI 反潜搜索
	LastPos objPos.MapPos
}

// SonarRange 本舰对水下潜艇的探测半径；0 表示没有声呐。
func (s *BattleShip) SonarRange() float64 {
	if s.CurHP <= 0 {
		return 0
	}
	switch s.Type {
	case ShipTypeDestroyer, ShipTypeFrigate:
		return ShipSonarRangeEscort
	case ShipTypeCruiser, ShipTypeBattleShip:
		return ShipSonarRangeCapital
	case ShipTypeAircraftCarrier:
		return ShipSonarRangeCarrier
	case ShipTypeSubmarine:
		return HydrophoneRange
	default:
		return 0
	}
}

// Detection 返回指定玩家对本体（潜艇）的探测记录。
func (s *BattleShip) Detection(player faction.Player) (ShipDetection, bool) {
	if len(s.DetectedBy) == 0 {
		return ShipDetection{}, false
	}
	d, ok := s.DetectedBy[player]
	return d, ok
}

// MarkDetected 记录一次探测；同一玩家重复探测只刷新到期时间与位置。
func (s *BattleShip) MarkDetected(player faction.Player, untilTick int64, pos objPos.MapPos) {
	if s.DetectedBy == nil {
		s.DetectedBy = map[faction.Player]ShipDetection{}
	}
	s.DetectedBy[player] = ShipDetection{UntilTick: untilTick, LastPos: pos}
}

// ClearDetection 清空所有探测记忆（上浮 / 过渡态时调用）。
func (s *BattleShip) ClearDetection() {
	if len(s.DetectedBy) == 0 {
		return
	}
	clear(s.DetectedBy)
}

// DetectedByAny 是否有任一玩家在当前拍点亮本艇。
func (s *BattleShip) DetectedByAny(now int64) bool {
	for _, d := range s.DetectedBy {
		if d.UntilTick > now {
			return true
		}
	}
	return false
}
