package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestNavigableDestSnapsLandTargetToWater 验证 AI 目标点落在陆地时按同域水域吸附。
// 场景：pearl_harbor 地图上 ComputerAlpha 战列舰 bb 位于 (60,130)，AI 的环形阵位
// (59,122) 是海岸格（前提断言 IsLand(59,122)）；另以水面 (60,130) 作对照。
// 预期：吸附后的阵位不再落在陆格，且 WaterRegionAt 与舰船所在水域一致；水面目标
// 原样返回（samePos）。修复前直接下发陆格会变成“永远到不了”，舰船停在岸边水面、
// AI 认为没到位而每轮重下同一条指令。
// 覆盖：mapcfg.GetByName("pearl_harbor") + 手搭 battleSnapshot，调用 navigableDest，
// 用 IsLand / WaterRegionAt / samePos 断言两条分支。
func TestNavigableDestSnapsLandTargetToWater(t *testing.T) {
	mapCfg := mapcfg.GetByName("pearl_harbor")
	if mapCfg == nil {
		t.Fatal("pearl_harbor map missing")
	}
	ship := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 60, 130)
	misState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: mapCfg}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}
	snap := &battleSnapshot{misState: misState, byUID: map[string]*objUnit.BattleShip{ship.Uid: ship}}

	// (59,122) 是珍珠港岸线格，吸附后必须落在与舰船同一片水域
	landTarget := objPos.New(59, 122)
	if !mapCfg.Map.IsLand(landTarget.MX, landTarget.MY) {
		t.Fatal("前提：(59,122) 应为海岸格")
	}
	got := navigableDest(snap, ship, landTarget)
	if mapCfg.Map.IsLand(got.MX, got.MY) {
		t.Fatalf("阵位仍落在陆格上：%s", got.String())
	}
	if mapCfg.WaterRegionAt(got.MX, got.MY) != mapCfg.WaterRegionAt(ship.CurPos.MX, ship.CurPos.MY) {
		t.Fatalf("阵位吸附到了与舰船不连通的水域：%s", got.String())
	}

	// 水面目标保持原样
	seaTarget := objPos.New(60, 130)
	if samePos(got, seaTarget) {
		t.Fatal("用例前提有误：水面目标不应等于吸附结果")
	}
	if got := navigableDest(snap, ship, seaTarget); !samePos(got, seaTarget) {
		t.Fatalf("水面目标不应被改写：%s", got.String())
	}
}
