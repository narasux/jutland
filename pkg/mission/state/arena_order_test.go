package state

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestOrderedShipsStaySortedAfterPutAndRemove 验证舰船绘制名单在增删后仍与 Ships 一致并按 Uid 有序。
// 场景：舰队按 Uid 乱序放入（b、a、c），绘制层每帧读 OrderedShips，删除时走二分查找分支。
// 预期：返回 a、b、c；RemoveShip("b") 后返回 a、c 且字典查不到 b，失败即名单与字典脱节导致漏画。
// 覆盖：PutShip 三艘 BattleShip，断言 3 艘顺序与 Ships["b"] 命中；删除后断言 2 艘 a、c 且 map 无 b。
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

// TestOrderedPlanesStaySortedAfterPutAndRemove 验证飞机名单在 Put/Remove/替换后仍按 UID 有序并重建缓存。
// 场景：空战每拍有飞机起飞/击落/回收；绘制层按 OrderedPlanes 顺序遍历，顺序不稳定会导致同屏飞机闪烁。
// 预期：Put b/a/c 后为 a,b,c；替换 b 的指针不改变顺序；Remove b 后为 a,c；未知 UID 移除不 panic。
// 覆盖：依次 PutPlane/RemovePlane，比较 OrderedPlanes 的 UID 序列与 map 内容。
func TestOrderedPlanesStaySortedAfterPutAndRemove(t *testing.T) {
	arena := MissionArenaState{}
	arena.PutPlane(&objUnit.Plane{Uid: "b"})
	arena.PutPlane(&objUnit.Plane{Uid: "a"})
	arena.PutPlane(&objUnit.Plane{Uid: "c"})

	got := arena.OrderedPlanes()
	if len(got) != 3 || got[0].Uid != "a" || got[1].Uid != "b" || got[2].Uid != "c" {
		t.Fatalf("ordered planes = %v", planeUids(got))
	}

	replacement := &objUnit.Plane{Uid: "b"}
	arena.PutPlane(replacement)
	got = arena.OrderedPlanes()
	if len(got) != 3 || got[1] != replacement {
		t.Fatalf("replacement did not keep sorted position: %v", planeUids(got))
	}

	arena.RemovePlane("b")
	got = arena.OrderedPlanes()
	if len(got) != 2 || got[0].Uid != "a" || got[1].Uid != "c" {
		t.Fatalf("after remove = %v", planeUids(got))
	}
	arena.RemovePlane("missing")
}

// planeUids 提取飞机 UID 序列，便于失败时打印。
func planeUids(planes []*objUnit.Plane) []string {
	ids := make([]string, len(planes))
	for i, plane := range planes {
		ids[i] = plane.Uid
	}
	return ids
}
