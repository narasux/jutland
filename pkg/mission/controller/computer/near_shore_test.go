package computer

import (
	"testing"
	"time"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestNearShoreAnchorFleetSettles 验证贴岸锚点的编队能各自吸附到水面阵位并停稳。
// 场景：回归验收 5——pearl_harbor 锚点 (60,124) 紧贴 y=122/123 海岸线，6 艘舰
// （2 战列舰、2 巡洋舰、2 驱逐舰）按 formationPos 间隔 2 格的环形阵位展开，并放一个
// 增援集结点迫使 AI 走锚点外环形阵位；舰船用真实量级参数（长 180、MaxSpeed=21/1200、
// 加速度 0.0002083、转向 0.375）。前提断言至少 1 个原始阵位落在陆格。
// 预期：推进 2600 拍后，每艘舰都在某个经 navigableDest 吸附后的阵位 2 格内，
// 中心点不在陆格，且最后 400 拍补发的移动指令不超过舰数；修复前阵位不吸附，
// 舰船长期顶岸、AI 每轮重下同一条指令。
// 覆盖：NewHandler + 手搭 MissionState/ReinforcePoint，循环 Handle/Exec 并每拍
// 让出 1ms 等异步寻路，最后用 navigableDest/formationPos/IsLand 断言到位。
func TestNearShoreAnchorFleetSettles(t *testing.T) {
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableFogOfWar = false
	t.Cleanup(func() { config.G = previous })

	mapCfg := mapcfg.GetByName("pearl_harbor")
	if mapCfg == nil {
		t.Fatal("pearl_harbor map missing")
	}
	// 锚点贴着珍珠港岸线（y=122/123 是海岸格），间隔 2 格的环形阵位必然有几个落在陆地上
	anchor := objPos.New(60, 124)

	handler := NewHandler(faction.ComputerAlpha)
	handler.anchor = anchor
	handler.anchorFixed = true

	kinds := []objUnit.ShipType{
		objUnit.ShipTypeBattleShip, objUnit.ShipTypeBattleShip,
		objUnit.ShipTypeCruiser, objUnit.ShipTypeCruiser,
		objUnit.ShipTypeDestroyer, objUnit.ShipTypeDestroyer,
	}
	ships := make([]*objUnit.BattleShip, 0, len(kinds))
	for idx, kind := range kinds {
		ship := makeShip(
			"ca-"+string(rune('a'+idx)), faction.ComputerAlpha, kind,
			anchor.MX+idx-2, anchor.MY+idx%3,
		)
		// 换成真实量级的机动参数，避免 makeShip 的 MaxSpeed=1 让位移失真
		ship.Length = 180
		ship.Width = 20
		ship.MaxSpeed = 21.0 / 1200
		ship.Acceleration = 0.00020833333333333335
		ship.RotateSpeed = 0.375
		ships = append(ships, ship)
	}

	misState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: mapCfg},
			FogOfWar:  false,
		},
		Arena: state.MissionArenaState{
			Ships:           map[string]*objUnit.BattleShip{},
			ReinforcePoints: map[string]*objBuilding.ReinforcePoint{},
		},
	}
	for _, ship := range ships {
		misState.Arena.Ships[ship.Uid] = ship
	}
	// 有增援集结点时 AI 不保留初始站位，改用锚点外的环形阵位——正是贴岸出问题的那条路径
	point := makePoint("rp", anchor.MX, anchor.MY, 4)
	misState.Arena.ReinforcePoints[point.Uid] = point

	// 用例前提：锚点附近确实有阵位落在陆地上
	landSlots := 0
	for idx := range ships {
		slot := formationPos(anchor, idx)
		if mapCfg.Map.IsLand(slot.MX, slot.MY) {
			landSlots++
		}
	}
	if landSlots == 0 {
		t.Fatal("用例前提不成立：锚点附近没有落在陆地上的阵位")
	}

	cur := map[string]instr.Instruction{}
	lateIssues := 0
	const totalTicks = 2600
	for tick := 0; tick < totalTicks; tick++ {
		misState.Core.SimTick = int64(tick)
		for uid, ins := range handler.Handle(cur, misState) {
			cur[uid] = ins
			if tick > totalTicks-400 {
				lateIssues++
			}
		}
		for uid, ins := range cur {
			_ = ins.Exec(misState)
			if ins.Executed() {
				delete(cur, uid)
			}
		}
		// 寻路结果由后台线程送达，等待期间让出主线程
		if len(cur) > 0 {
			time.Sleep(time.Millisecond)
		}
	}

	// 编队必须已经各自到位：每艘舰都在某个（吸附后的）阵位 2 格内，且没有压上岸
	snap := handler.snapshot(misState)
	for _, ship := range ships {
		best := 1e9
		for idx := range ships {
			slot := navigableDest(snap, ship, formationPos(anchor, idx))
			best = min(best, ship.CurPos.Distance(slot))
		}
		if best > 2.0 {
			t.Fatalf("舰 %s 未到位：距最近阵位 %.2f 格（pos=%s）", ship.Uid, best, ship.CurPos.String())
		}
		if mapCfg.Map.IsLand(ship.CurPos.MX, ship.CurPos.MY) {
			t.Fatalf("舰 %s 中心点压上了陆格：%s", ship.Uid, ship.CurPos.String())
		}
	}

	// 到位后不应再反复重下移动指令
	if lateIssues > len(ships) {
		t.Fatalf("到位后仍在反复下发移动指令：最后 400 拍共 %d 条（%d 艘舰）", lateIssues, len(ships))
	}
	t.Logf("近岸锚点编队：%d 个阵位落在陆地，%d 拍内全部到位，最后 400 拍补发指令 %d 条",
		landSlots, totalTicks, lateIssues)
}

// TestStationPosStaysOnWater 验证近岸锚点的 stationPos 阵位落在同域水面上。
// 场景：pearl_harbor 锚点 (60,124)，战列舰 (60,130)，存在增援集结点（AI 使用
// 锚点外环形阵位）；原始 formationPos(anchor,0..7) 至少有 1 个陆格（前提断言）。
// 预期：stationPos 对 8 个槽位都返回 IsSea，且 WaterRegionAt 与舰船所在水域一致；
// 修复前阵位直接取锚点 + 螺旋偏移，贴岸锚点会给出陆格，舰船只能停在岸边水面、
// AI 认为没到位而反复重下。
// 覆盖：NewHandler 固定 anchor/anchorFixed，snapshot 后逐槽调用 stationPos，
// 用 IsSea 与 WaterRegionAt 断言。
func TestStationPosStaysOnWater(t *testing.T) {
	mapCfg := mapcfg.GetByName("pearl_harbor")
	if mapCfg == nil {
		t.Fatal("pearl_harbor map missing")
	}
	// 锚点贴着珍珠港岸线，间隔 2 格的阵位必然有几个落在陆地上
	anchor := objPos.New(60, 124)
	ship := makeShip("ca-bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 60, 130)
	ship.Length, ship.MaxSpeed, ship.RotateSpeed = 180, 21.0/1200, 0.375
	misState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: mapCfg}},
		Arena: state.MissionArenaState{
			Ships:           map[string]*objUnit.BattleShip{ship.Uid: ship},
			ReinforcePoints: map[string]*objBuilding.ReinforcePoint{},
		},
	}
	// 有增援集结点时 AI 用锚点外的环形阵位（不保留初始站位），阵位才可能落到岸上
	point := makePoint("rp", anchor.MX, anchor.MY, 4)
	misState.Arena.ReinforcePoints[point.Uid] = point
	handler := NewHandler(faction.ComputerAlpha)
	handler.anchor = anchor
	handler.anchorFixed = true
	snap := handler.snapshot(misState)

	shipRegion := mapCfg.WaterRegionAt(ship.CurPos.MX, ship.CurPos.MY)
	if shipRegion == 0 {
		t.Fatal("用例前提不成立：舰船应在水面上")
	}
	landSlots := 0
	for idx := 0; idx < 8; idx++ {
		if mapCfg.Map.IsLand(formationPos(anchor, idx).MX, formationPos(anchor, idx).MY) {
			landSlots++
		}
		slot := handler.stationPos(snap, ship, idx)
		if !mapCfg.Map.IsSea(slot.MX, slot.MY) {
			t.Fatalf("阵位 #%d 落在陆格上：%s", idx, slot.String())
		}
		if mapCfg.WaterRegionAt(slot.MX, slot.MY) != shipRegion {
			t.Fatalf("阵位 #%d 落在与舰船不连通的水域：%s", idx, slot.String())
		}
	}
	if landSlots == 0 {
		t.Fatal("用例前提不成立：锚点附近没有落在陆地上的阵位")
	}
	t.Logf("近岸锚点：%d/8 个原始阵位在陆地上，吸附后全部落在同域水面", landSlots)
}
