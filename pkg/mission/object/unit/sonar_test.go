package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

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

func TestSubmergedShipEmitsNoWake(t *testing.T) {
	sub := &BattleShip{
		Type: ShipTypeSubmarine, CurHP: 100, CurSpeed: 0.03, MaxSpeed: 0.03,
		DiveState: DiveStateSubmerged,
	}
	require.Nil(t, sub.GenTrails(), "水下不画尾流，避免泄露位置")
}
