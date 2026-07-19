package instruction

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/grid"
)

func TestShipMovePathDoesNotReturnToPositionAtCommandTime(t *testing.T) {
	cells := make(grid.Cells, 20)
	for y := range cells {
		cells[y] = make([]int, 40)
	}
	mapCfg := &mapcfg.MapCfg{Cells: cells, Width: 40, Height: 20}

	commandPos := objPos.New(10, 10)
	targetPos := objPos.New(30, 10)
	ship := &objUnit.BattleShip{Uid: "ship", CurPos: objPos.NewR(11, 10)}
	misState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: mapCfg},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}

	move := NewShipMovePath(ship.Uid, commandPos, targetPos, 0)
	move.genPath(misState)

	require.Equal(t, []objPos.MapPos{targetPos}, move.path)
}
