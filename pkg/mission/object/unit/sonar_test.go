package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestSonarRangeByShipType 验证各舰种声呐/水听器半径，以及沉没舰不再提供声呐。
// 场景：驱逐/护卫用护航声呐，巡洋/战列用主力声呐，航母用航母声呐，潜艇用水听器，货船/鱼雷艇没有。
// 预期：各类型返回对应常量；CurHP=0 的驱逐舰返回 0。
// 覆盖：表驱动遍历 8 种舰型，再补一个沉没舰用例。
func TestSonarRangeByShipType(t *testing.T) {
	cases := []struct {
		shipType ShipType
		want     float64
	}{
		{ShipTypeDestroyer, ShipSonarRangeEscort},
		{ShipTypeFrigate, ShipSonarRangeEscort},
		{ShipTypeCruiser, ShipSonarRangeCapital},
		{ShipTypeBattleShip, ShipSonarRangeCapital},
		{ShipTypeAircraftCarrier, ShipSonarRangeCarrier},
		{ShipTypeSubmarine, HydrophoneRange},
		{ShipTypeCargo, 0},
		{ShipTypeTorpedoBoat, 0},
	}
	for _, tc := range cases {
		ship := &BattleShip{Type: tc.shipType, CurHP: 100}
		require.InDelta(t, tc.want, ship.SonarRange(), 1e-9, "type %s", tc.shipType)
	}
	// 沉没的舰不再提供声呐
	dead := &BattleShip{Type: ShipTypeDestroyer, CurHP: 0}
	require.Zero(t, dead.SonarRange())
}

// TestSubmarineDetectionMemory 验证潜艇被声呐点亮后的记忆：到期失效、重复探测刷新位置/时间、可清除。
// 场景：反潜舰探测到潜艇后，即使下一拍失去接触，也应记住最后位置一段时间；到期后不能再看见。
// 预期：MarkDetected 后 UntilTick 前可见；到期不可见；再次探测刷新 LastPos/UntilTick；
// ClearDetection 全部清除。
// 覆盖：对同一艘潜艇依次调用 MarkDetected/DetectedByAny/Detection/ClearDetection 并断言。
func TestSubmarineDetectionMemory(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100}
	pos := objPos.NewR(3.5, 4.5)

	require.False(t, sub.DetectedByAny(100))
	sub.MarkDetected("CA", 150, pos)
	require.True(t, sub.DetectedByAny(100))
	require.False(t, sub.DetectedByAny(150), "到期后不再算被点亮")

	detection, ok := sub.Detection("CA")
	require.True(t, ok)
	require.Equal(t, pos, detection.LastPos)

	// 同一玩家再次探测只刷新位置与到期时间
	sub.MarkDetected("CA", 200, objPos.NewR(5.5, 6.5))
	detection, _ = sub.Detection("CA")
	require.Equal(t, objPos.NewR(5.5, 6.5), detection.LastPos)
	require.EqualValues(t, 200, detection.UntilTick)

	sub.ClearDetection()
	require.False(t, sub.DetectedByAny(100))
}

// TestSubmergedShipEmitsNoWake 验证水下潜艇不生成尾流。
// 场景：潜艇下潜后仍在移动；若继续画尾流会暴露位置，破坏潜艇隐蔽性。
// 预期：GenTrails 返回 nil。
// 覆盖：构造水下、有速度的潜艇，直接调用 GenTrails。
func TestSubmergedShipEmitsNoWake(t *testing.T) {
	sub := &BattleShip{
		Type: ShipTypeSubmarine, CurHP: 100, CurSpeed: 0.03, MaxSpeed: 0.03,
		DiveState: DiveStateSubmerged,
	}
	require.Nil(t, sub.GenTrails(), "水下不画尾流，避免泄露位置")
}
