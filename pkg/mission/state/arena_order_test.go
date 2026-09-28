package state

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

func TestOrderedShipsStaySortedAfterPutAndRemove(t *testing.T) {
	arena := MissionArenaState{}
	arena.PutShip(&objUnit.BattleShip{Uid: "b"})
	arena.PutShip(&objUnit.BattleShip{Uid: "a"})
	arena.PutShip(&objUnit.BattleShip{Uid: "c"})

	got := arena.OrderedShips()
	if len(got) != 3 || got[0].Uid != "a" || got[1].Uid != "b" || got[2].Uid != "c" {
		t.Fatalf("ordered = %v", uids(got))
	}
	if arena.Ships["b"] == nil || arena.Ships["b"].Uid != "b" {
		t.Fatal("map lookup missed ship b")
	}

	arena.RemoveShip("b")
	got = arena.OrderedShips()
	if len(got) != 2 || got[0].Uid != "a" || got[1].Uid != "c" {
		t.Fatalf("after remove = %v", uids(got))
	}
	if _, ok := arena.Ships["b"]; ok {
		t.Fatal("removed ship still in map")
	}
}

func uids(ships []*objUnit.BattleShip) []string {
	out := make([]string, len(ships))
	for i, ship := range ships {
		out[i] = ship.Uid
	}
	return out
}
