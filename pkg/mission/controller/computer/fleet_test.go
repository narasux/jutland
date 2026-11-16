package computer

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestShipsBelongToOneOrder 验证一次编组把每艘舰只分进一个编队。
// 场景：默认五种策略全开，1 艘航母、1 艘战列舰、1 艘驱逐舰、1 艘护卫舰、
// 1 艘鱼雷艇面对 (6,0) 的敌战列舰，编组顺序为驻守 → 袭击 → 侦察 → 进攻。
// 预期：航母与战列舰进驻守、驱逐舰进袭击且目标为 foe、护卫舰与鱼雷艇进进攻，
// 没有撤退打断。同一艘舰被编进两组会收到两条互相冲突的航路。
// 覆盖：战列舰对舰 30、驱逐舰对舰 10 且机动 5；用 assertMembers 核对四组名单。
func TestShipsBelongToOneOrder(t *testing.T) {
	carrier := makeShip("cv", faction.ComputerAlpha, objUnit.ShipTypeAircraftCarrier, 0, 0)
	battleship := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 1, 0)
	battleship.CombatPower.AntiShip = 30
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 2, 0)
	destroyer.CombatPower.AntiShip = 10
	destroyer.CombatPower.Mobility = 5
	frigate := makeShip("ff", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 3, 0)
	torpedo := makeShip("tb", faction.ComputerAlpha, objUnit.ShipTypeTorpedoBoat, 4, 0)
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 6, 0)
	enemy.CombatPower.AntiShip = 10

	handler, _ := decide(
		[]*objUnit.BattleShip{carrier, battleship, destroyer, frigate, torpedo, enemy},
		[]*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 0),
		},
	)

	assertMembers(t, "garrison", handler.garrison, "bb", "cv")
	assertMembers(t, "raid", handler.raid, "dd")
	assertMembers(t, "attack", handler.attack, "ff", "tb")
	if len(handler.retreatOrder) != 0 {
		t.Fatalf("retreats = %v", handler.retreatOrder)
	}
	if handler.raid.targetUID != "foe" {
		t.Fatalf("raid target = %s", handler.raid.targetUID)
	}
}

// TestZeroAntiShipFallsBackToShipCount 验证双方对舰战力都是 0 时退回按艘数计分。
// 场景：3 艘对舰 0 的战列舰对上 1 艘同样对舰 0 的敌战列舰，艘数比 3 比 1，
// 驻守按 1.5 倍缺口只取 2 艘，剩下 1 艘编入进攻。
// 预期：garrison={a,b}、attack={c}、无撤退。若仍按 0 分比较，驻守会空、
// 全军会误判为没有威胁。
// 覆盖：makeShip 默认 CurHP/TotalHP=100 且不设 CombatPower.AntiShip，断言两组成员。
func TestZeroAntiShipFallsBackToShipCount(t *testing.T) {
	own := []*objUnit.BattleShip{
		makeShip("a", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 0, 0),
		makeShip("b", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 1, 0),
		makeShip("c", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 2, 0),
	}
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 3, 0)
	handler, _ := decide(append(own, enemy), []*objBuilding.ReinforcePoint{makePoint("rp", 0, 0, 0)})

	assertMembers(t, "garrison", handler.garrison, "a", "b")
	assertMembers(t, "attack", handler.attack, "c")
}

// TestGarrisonKeepsSurplusBattleshipsUnderDouble 验证驻守先补到 1.5 倍、战列舰最多 2 倍。
// 场景：6 艘对舰 4 的战列舰面对 1 艘对舰 10 的敌舰，1.5 倍线是 15 点、2 倍线是 20 点。
// 预期：takeGarrison 返回 5 艘：4 艘到 16 点，第 5 艘到 20 点恰好不超 2 倍；
// 混编时富余名额只给战列舰，20 点对舰的护卫舰不得进驻守。
// 覆盖：直接调用 takeGarrison，检查长度与 mixed[0].Uid=="bb"。
func TestGarrisonKeepsSurplusBattleshipsUnderDouble(t *testing.T) {
	enemy := groupPower{antiShip: 10, count: 1}
	battleships := []*objUnit.BattleShip{
		combatShip("a", objUnit.ShipTypeBattleShip, 4),
		combatShip("b", objUnit.ShipTypeBattleShip, 4),
		combatShip("c", objUnit.ShipTypeBattleShip, 4),
		combatShip("d", objUnit.ShipTypeBattleShip, 4),
		combatShip("e", objUnit.ShipTypeBattleShip, 4),
		combatShip("f", objUnit.ShipTypeBattleShip, 4),
	}
	got := takeGarrison(nil, battleships, enemy)
	if len(got) != 5 {
		t.Fatalf("garrison size = %d, want 5", len(got))
	}

	mixed := takeGarrison(nil, []*objUnit.BattleShip{
		combatShip("bb", objUnit.ShipTypeBattleShip, 20),
		combatShip("ff", objUnit.ShipTypeFrigate, 20),
	}, enemy)
	if len(mixed) != 1 || mixed[0].Uid != "bb" {
		t.Fatalf("mixed garrison = %v", uidsOf(mixed))
	}
}

// TestAttackPressHysteresis 验证进攻组的压上/回撤带滞回，不随纸面兵力抖动。
// 场景：己方战列舰 150/200 血、敌方同为 150/200，逐轮改 AntiShip 与 CurHP：
// 先 99 对 100 不压上，100 对 100 压上，降到 40 仍压上（只看战损），
// 血量 120 低于 pressStop 0.85 倍才回撤，回撤后要 116 点且 173 血
// （两项都超 1.15 倍）才恢复。
// 预期：6 步结果依次为 未压上、压上、压上、回撤、保持回撤、恢复压上，与表一致；
// 失败说明阈值被单点跨越，队形会在接触边缘反复掉头。
// 覆盖：配置 Attack+Defend，按 i*regroupInterval 逐拍 Handle 并核对 attack 两字段。
func TestAttackPressHysteresis(t *testing.T) {
	useStrategies(t, config.AIStrategies{Attack: true, Defend: true})
	own := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 0, 0)
	own.TotalHP = 200
	own.CurHP = 150
	own.CombatPower.AntiShip = 99
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 80, 80)
	enemy.TotalHP = 200
	enemy.CurHP = 150
	enemy.CombatPower.AntiShip = 100
	point := makePoint("rp", 0, 0, 0)
	handler := NewHandler(faction.ComputerAlpha)

	steps := []struct {
		antiShip int
		hp       float64
		pressing bool
		fellBack bool
	}{
		{99, 150, false, false},
		{100, 150, true, false},
		// 纸面兵力掉到 0.4 倍也不再退出：AntiShip 之和被护航舰的鱼雷爆发主导，
		// 已经压上的小队只看真实战损。
		{40, 150, true, false},
		// 生命之和掉到对方的 0.85 倍以下才算打不动了。
		{40, 120, false, true},
		{100, 150, false, true},
		{116, 173, true, false},
	}
	for i, step := range steps {
		own.CombatPower.AntiShip = step.antiShip
		own.CurHP = step.hp
		handler.Handle(
			nil,
			newMission(
				int64(i)*regroupInterval,
				[]*objUnit.BattleShip{own, enemy},
				[]*objBuilding.ReinforcePoint{point},
			),
		)
		if handler.attack == nil {
			t.Fatalf("step %d attack order missing", i)
		}
		if handler.attack.pressing != step.pressing || handler.attack.fellBack != step.fellBack {
			t.Fatalf("step %d pressing=%v fellBack=%v, want %v %v",
				i, handler.attack.pressing, handler.attack.fellBack, step.pressing, step.fellBack)
		}
	}
}

// TestAttackKeepsPressingAfterTheScreenIsSunk 验证屏卫被清空后战列舰线仍继续压上。
// 场景：回归珍珠港 1941 的整队掉头——8 艘战列舰（202 点对舰 / 27.4 万生命）加
// 29 艘驱逐舰（493 点 / 6.6 万生命）对 15 艘日军舰艇（回归记录 431 点 / 30.6 万生命）；
// 第 0 拍全队在场并压上，第 regroupInterval=60 拍时 29 艘屏卫全部移除，只剩 8 艘
// 几乎完好的战列舰（生命之和仍为对方的 0.90 倍，纸面对舰比从 1.61 跌到 0.47）。
// 预期：即使纸面兵力比崩到 0.47，attack.pressing 仍为 true、fellBack=false，
// bb0 继续收到驶向目标 foe4 (64,8) 的 ShipMove；若按纸面兵力比撤退，整队会在
// 最不该掉头的时候回初始泊位。
// 覆盖：useStrategies(Attack+Defend)，按 fixture 逐艘填 AntiShip/HP，两次 Handle
// 后断言 attack 两字段与 ShipMove 航路字符串 "(64[64.00], 8[8.00])"。
func TestAttackKeepsPressingAfterTheScreenIsSunk(t *testing.T) {
	useStrategies(t, config.AIStrategies{Attack: true, Defend: true})

	var battleships []*objUnit.BattleShip
	for i, spec := range []struct {
		antiShip int
		hp       float64
	}{
		{25, 31700},
		{23, 35737},
		{26, 33190},
		{27, 37800},
		{27, 34946},
		{25, 31700},
		{26, 33190},
		{23, 35929},
	} {
		bb := makeShip(fmt.Sprintf("bb%d", i), faction.ComputerAlpha, objUnit.ShipTypeBattleShip, i, 0)
		bb.TypeAbbr = "WaterDrop"
		bb.TotalHP, bb.CurHP = spec.hp, spec.hp
		bb.CombatPower.AntiShip = spec.antiShip
		battleships = append(battleships, bb)
	}
	var screen []*objUnit.BattleShip
	for i := 0; i < 29; i++ {
		dd := makeShip(fmt.Sprintf("dd%02d", i), faction.ComputerAlpha, objUnit.ShipTypeDestroyer, i, 4)
		dd.TotalHP, dd.CurHP = 2286, 2286
		dd.CombatPower.AntiShip = 17
		screen = append(screen, dd)
	}
	var enemies []*objUnit.BattleShip
	for i, spec := range []struct {
		antiShip int
		hp       float64
	}{
		{44, 41300},
		{44, 41300},
		{31, 19500},
		{34, 21900},
		{45, 32105},
		{45, 32105},
		{21, 36600},
		{21, 36600},
		{23, 13200},
		{23, 13200},
		{16, 5925},
		{19, 2490},
		{19, 2490},
		{19, 2490},
		{19, 2490},
	} {
		foe := makeShip(fmt.Sprintf("foe%d", i), faction.HumanAlpha, objUnit.ShipTypeBattleShip, 60+i%5, 8+i/5)
		foe.TotalHP, foe.CurHP = spec.hp, spec.hp
		foe.CombatPower.AntiShip = spec.antiShip
		enemies = append(enemies, foe)
	}

	handler := NewHandler(faction.ComputerAlpha)
	all := append(append(append([]*objUnit.BattleShip{}, battleships...), screen...), enemies...)
	handler.Handle(nil, newMission(0, all, nil))
	if handler.attack == nil || !handler.attack.pressing {
		t.Fatalf("全队在场时应该压上: %+v", handler.attack)
	}

	// 屏卫被清空，只剩 8 艘几乎完好的战列舰。
	survivors := append(append([]*objUnit.BattleShip{}, battleships...), enemies...)
	got := handler.Handle(nil, newMission(regroupInterval, survivors, nil))
	if handler.attack == nil || !handler.attack.pressing || handler.attack.fellBack {
		t.Fatalf("屏卫沉没后仍应压上: %+v", handler.attack)
	}
	move, ok := got[instr.GenInstrUid(instr.NameShipMove, "bb0")].(*instr.ShipMove)
	if !ok {
		t.Fatalf("压上时应该继续朝目标下航路: %v", keys(got))
	}
	// 目标是离小队重心最近的一团里对舰战力最高的 foe4。
	if !strings.Contains(move.String(), "(64[64.00], 8[8.00])") {
		t.Fatalf("航路 = %v, want 目标 foe4 (64, 8)", move)
	}
}

// TestAttackAloneNeverFallsBack 验证只勾进攻时劣势也保持压上、不回初始泊位。
// 场景：己方战列舰 bb 在 (0,0)，30000/30000 血、25 点对舰；敌方 4 艘同型战列舰
// 从 (60,0) 起每 2 格一艘；先只开 Attack，再开 Attack+Defend。
// 预期：只开进攻时 1 打 4 仍 attack.pressing=true，bb 收到驶向最近敌舰 (60,0) 的
// ShipMove；补上 Defend 后同一劣势变为 pressing=false，且不再下发 ShipMove。
// 覆盖：config.AIStrategies{Attack:true} 与 {Attack:true, Defend:true} 两次 decide，
// 断言 attack.pressing 与 NameShipMove 指令的航路字符串。
func TestAttackAloneNeverFallsBack(t *testing.T) {
	own := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 0, 0)
	own.TypeAbbr = "WaterDrop"
	own.TotalHP, own.CurHP = 30000, 30000
	own.CombatPower.AntiShip = 25
	var enemies []*objUnit.BattleShip
	for i := 0; i < 4; i++ {
		foe := makeShip(fmt.Sprintf("foe%d", i), faction.HumanAlpha, objUnit.ShipTypeBattleShip, 60+2*i, 0)
		foe.TotalHP, foe.CurHP = 30000, 30000
		foe.CombatPower.AntiShip = 25
		enemies = append(enemies, foe)
	}

	useStrategies(t, config.AIStrategies{Attack: true})
	got := func() map[string]instr.Instruction {
		handler, out := decide([]*objUnit.BattleShip{own, enemies[0], enemies[1], enemies[2], enemies[3]}, nil)
		if handler.attack == nil || !handler.attack.pressing {
			t.Fatalf("只有进攻时应该一直压上: %+v", handler.attack)
		}
		return out
	}()
	move, ok := got[instr.GenInstrUid(instr.NameShipMove, "bb")].(*instr.ShipMove)
	if !ok || !strings.Contains(move.String(), "(60[60.00], 0[0.00])") {
		t.Fatalf("航路 = %v, want 最近的敌舰 (60, 0)", got)
	}

	useStrategies(t, config.AIStrategies{Attack: true, Defend: true})
	handler, out := decide([]*objUnit.BattleShip{own, enemies[0], enemies[1], enemies[2], enemies[3]}, nil)
	if handler.attack == nil || handler.attack.pressing {
		t.Fatalf("勾了防守时劣势不该压上: %+v", handler.attack)
	}
	if _, ok := out[instr.GenInstrUid(instr.NameShipMove, "bb")]; ok {
		t.Fatalf("勾了防守时不该下压上航路: %v", keys(out))
	}
}

// TestNoRetreatInterruptWithoutDefend 验证未勾防守时不触发单舰撤退打断。
// 场景：己方战列舰 bb 只剩 30/100 血（低于 criticalHP=0.35）、25 点对舰，
// 两艘敌驱逐舰分别在 (2,0)、(4,0)，各 19 点对舰，落在 nearbyRadius=20 内。
// 预期：只开 Attack 时 retreatOrder 为空，bb 留在 attack；补上 Defend 后
// retreatOrder 恰为 ["bb"]。没有后方时把伤舰送回初始泊位等于送死。
// 覆盖：两次 decide 分别用 {Attack:true} 与 {Attack:true, Defend:true}，
// 断言 retreatOrder 长度/成员与 attack 成员。
func TestNoRetreatInterruptWithoutDefend(t *testing.T) {
	own := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 0, 0)
	own.CurHP = 30
	own.CombatPower.AntiShip = 25
	foes := []*objUnit.BattleShip{
		makeShip("dd1", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 2, 0),
		makeShip("dd2", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 4, 0),
	}
	foes[0].CombatPower.AntiShip = 19
	foes[1].CombatPower.AntiShip = 19
	ships := []*objUnit.BattleShip{own, foes[0], foes[1]}

	useStrategies(t, config.AIStrategies{Attack: true})
	handler, _ := decide(ships, nil)
	if len(handler.retreatOrder) != 0 {
		t.Fatalf("只有进攻时不该有撤退打断: %v", handler.retreatOrder)
	}
	assertMembers(t, "attack", handler.attack, "bb")

	useStrategies(t, config.AIStrategies{Attack: true, Defend: true})
	handler, _ = decide(ships, nil)
	if len(handler.retreatOrder) != 1 || handler.retreatOrder[0] != "bb" {
		t.Fatalf("勾了防守时受伤的战列舰应该回撤: %v", handler.retreatOrder)
	}
}

// TestRaidPrefersLightCarrierThenAbandonsWhenOutgunned 验证优先打轻护航航母、劣势时放弃。
// 场景：50 点对舰的驱逐舰面对 (80,80) 的航母（40 点对舰）和 2 格外仅 5 点对舰的
// 护航舰，航母属轻护航，优先级高于 (30,0) 残血驱逐舰与 (0,60) 孤立驱逐舰。
// 预期：首选 cv；换成 10 点对舰的驱逐舰时，目标旁 20 格内 31 点对舰达 1.5 倍即
// 放弃袭击并转侦察；只开 Attack+Raid 时再退回进攻。失败意味着袭击队会硬冲护航圈。
// 覆盖：默认策略下断言 raid.targetUID=="cv"，放弃后 scout/attack 成员为 "dd"。
func TestRaidPrefersLightCarrierThenAbandonsWhenOutgunned(t *testing.T) {
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	destroyer.CombatPower.AntiShip = 50
	carrier := makeShip("cv", faction.HumanAlpha, objUnit.ShipTypeAircraftCarrier, 80, 80)
	carrier.CombatPower.AntiShip = 40
	escort := makeShip("escort", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 82, 80)
	escort.CombatPower.AntiShip = 5
	wounded := makeShip("hurt", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 30, 0)
	wounded.CurHP = 20
	wounded.TotalHP = 100
	isolated := makeShip("lone", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 0, 60)

	handler, _ := decide(
		[]*objUnit.BattleShip{destroyer, carrier, escort, wounded, isolated},
		[]*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 0),
		},
	)
	if handler.raid == nil || handler.raid.targetUID != "cv" {
		t.Fatalf("raid = %+v, want carrier", handler.raid)
	}

	outgunned := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	outgunned.CombatPower.AntiShip = 10
	weak := makeShip("hurt", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 50, 50)
	weak.CurHP = 10
	weak.CombatPower.AntiShip = 1
	strong := makeShip("strong", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 52, 50)
	strong.CombatPower.AntiShip = 30
	ships := []*objUnit.BattleShip{outgunned, weak, strong}
	point := []*objBuilding.ReinforcePoint{makePoint("rp", 0, 0, 0)}
	handler, _ = decide(ships, point)
	if handler.raid != nil {
		t.Fatalf("raid = %+v, want abandon", handler.raid)
	}
	assertMembers(t, "scout", handler.scout, "dd")

	useStrategies(t, config.AIStrategies{Attack: true, Raid: true})
	handler, _ = decide(ships, point)
	if handler.raid != nil || handler.scout != nil {
		t.Fatalf("raid=%v scout=%v", handler.raid, handler.scout)
	}
	assertMembers(t, "attack", handler.attack, "dd")
}

// TestRaidPicksNearestInTheSameTier 验证同一优先级内按离编队重心最近选袭击目标。
// 场景：驱逐舰在 (0,0)，两艘敌驱逐舰分别在 (10,0) 与 (40,0)，相距 30 格、
// 都超过 isolatedRadius=15，因此同属「孤立」档，只能靠距离分胜负。
// 预期：raid.targetUID=="near"；失败说明同档内退化成按 uid 或遍历顺序取目标。
// 覆盖：集结点放在 (200,200) 只用于确定锚点，不影响袭击目标选择。
func TestRaidPicksNearestInTheSameTier(t *testing.T) {
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	near := makeShip("near", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 10, 0)
	far := makeShip("far", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 40, 0)
	handler, _ := decide([]*objUnit.BattleShip{destroyer, near, far}, []*objBuilding.ReinforcePoint{
		makePoint("rp", 200, 200, 0),
	})
	if handler.raid == nil || handler.raid.targetUID != "near" {
		t.Fatalf("raid target = %+v", handler.raid)
	}
}

// TestRetreatRules 验证单舰撤退的三段血量与速度/火力判定。
// 场景：对舰 10、航速 1 的敌驱逐舰贴近；己方血比 0.7 算健康、0.5 算受伤、
// 0.1 算危急；受伤段要求对方火力更强且自己跑不掉，危急段要求对方还有健康舰。
// 预期：70 血不撤；50 血航速 2 不撤、降到 1 撤；10 血面对 2 艘 10 血 4 点对舰的
// 敌舰不撤，其中一艘改成 50 血后才撤；身边无敌人时危急舰也撤。
// 覆盖：直接调用 shouldRetreat，逐一核对 healthyHP=0.7、criticalHP=0.35 的边界。
func TestRetreatRules(t *testing.T) {
	nearby := []*objUnit.BattleShip{makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 1, 0)}
	nearby[0].CombatPower.AntiShip = 10
	nearby[0].MaxSpeed = 1

	healthy := makeShip("own", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	healthy.CurHP = 70
	healthy.CombatPower.AntiShip = 1
	if shouldRetreat(healthy, nearby) {
		t.Fatal("healthy ship retreated")
	}

	wounded := makeShip("own", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	wounded.CurHP = 50
	wounded.CombatPower.AntiShip = 1
	wounded.MaxSpeed = 2
	if shouldRetreat(wounded, nearby) {
		t.Fatal("faster wounded ship retreated")
	}
	wounded.MaxSpeed = 1
	if !shouldRetreat(wounded, nearby) {
		t.Fatal("slower wounded ship stayed")
	}

	critical := makeShip("own", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	critical.CurHP = 10
	critical.CombatPower.AntiShip = 10
	criticalFoes := []*objUnit.BattleShip{
		makeShip("a", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 1, 0),
		makeShip("b", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 2, 0),
	}
	criticalFoes[0].CurHP = 10
	criticalFoes[0].CombatPower.AntiShip = 4
	criticalFoes[1].CurHP = 10
	criticalFoes[1].CombatPower.AntiShip = 4
	if shouldRetreat(critical, criticalFoes) {
		t.Fatal("critical ship left a fight it still wins")
	}
	criticalFoes[0].CurHP = 50
	if !shouldRetreat(critical, criticalFoes) {
		t.Fatal("critical ship stayed against a healthy enemy")
	}
	if !shouldRetreat(critical, nil) {
		t.Fatal("critical ship stayed with nobody nearby")
	}
}

// TestRetreatOnStationStaysPut 验证驻守中的舰就地不动，其余舰退到主力舰尾后 6 格。
// 场景：锚点 (4,4) 已固定、驻守成员只有驱逐舰 dd，dd 正好在锚点上；
// 另一艘驱逐舰 away 没有站位，最近的主力舰在 (20,4)、航向 0（正北）。
// 预期：dd 的撤退点是 (4,4) 本身；away 的撤退点是 (20,10)，即主力舰正后方 6 格。
// 覆盖：手搭只含 own/byUID 的 battleSnapshot，调用 retreatAnchorFor(..., anchorOK=true)。
func TestRetreatOnStationStaysPut(t *testing.T) {
	handler := NewHandler(faction.ComputerAlpha)
	handler.anchor = objPos.New(4, 4)
	handler.anchorFixed = true
	handler.garrison = &fleetOrder{members: []string{"dd"}}
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 4, 4)
	capital := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 20, 4)
	snap := &battleSnapshot{
		own:   []*objUnit.BattleShip{destroyer, capital},
		byUID: map[string]*objUnit.BattleShip{destroyer.Uid: destroyer, capital.Uid: capital},
	}
	got := handler.retreatAnchorFor(destroyer, snap, handler.anchor, true)
	if !samePos(got, objPos.New(4, 4)) {
		t.Fatalf("on-station retreat = %s", got.String())
	}

	away := makeShip("away", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	snap.own = append(snap.own, away)
	got = handler.retreatAnchorFor(away, snap, handler.anchor, true)
	if !samePos(got, objPos.New(20, 10)) {
		t.Fatalf("astern retreat = %s, want (20, 10)", got.String())
	}
}

// TestSummonFollowsDeficitAndSkipsOccupiedQueue 验证召唤按缺口选舰并跳过忙碌队列。
// 场景：8 个子用例覆盖近敌选建造时间短、无战列舰时选防空高、优先袭击舰、
// 跳过不可召唤的货船、护卫舰兜底、支援舰与鱼雷艇不可召唤、队列被占用时不重复、
// 多集结点时后一个能看见前一个刚选的舰（缺口已补）。
// 预期：每个用例只对目标集结点下 1 条召唤且舰名符合偏好，忙碌或缺口已补时返回空。
// 失败意味着增援会重复排队或选错舰型。
// 覆盖：makeTemplate 注册 slow/fast/cl/ff/bb/dd/cargo/tb，用 assertSummon 校验舰名。
func TestSummonFollowsDeficitAndSkipsOccupiedQueue(t *testing.T) {
	t.Run("shorter build while enemies are close", func(t *testing.T) {
		useTemplates(
			t,
			makeTemplate("slow", objUnit.ShipTypeBattleShip, 80, 0, 0, 0, 100),
			makeTemplate("fast", objUnit.ShipTypeBattleShip, 10, 0, 0, 0, 10),
		)
		enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 2, 0)
		enemy.CombatPower.AntiShip = 100
		_, got := decide([]*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 2, "slow", "fast"),
		})
		assertSummon(t, got, "rp", "fast")
	})

	t.Run("highest anti-air when no battleship is available", func(t *testing.T) {
		useTemplates(
			t,
			makeTemplate("cl", objUnit.ShipTypeCruiser, 10, 1, 0, 0, 1),
			makeTemplate("ff", objUnit.ShipTypeFrigate, 1, 8, 0, 0, 50),
		)
		enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 2, 0)
		enemy.CombatPower.AntiShip = 50
		_, got := decide([]*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 2, "cl", "ff"),
		})
		assertSummon(t, got, "rp", "ff")
	})

	t.Run("raid ship before another battleship", func(t *testing.T) {
		useTemplates(
			t,
			makeTemplate("bb", objUnit.ShipTypeBattleShip, 20, 0, 0, 0, 10),
			makeTemplate("dd", objUnit.ShipTypeDestroyer, 5, 0, 8, 3, 10),
		)
		enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 100, 100)
		_, got := decide([]*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 2, "bb", "dd"),
		})
		assertSummon(t, got, "rp", "dd")
	})

	t.Run("falls through to the next hull", func(t *testing.T) {
		useTemplates(
			t,
			makeTemplate("cargo", objUnit.ShipTypeCargo, 0, 0, 0, 0, 1),
			makeTemplate("dd", objUnit.ShipTypeDestroyer, 4, 0, 0, 0, 1),
		)
		first := makeShip("a", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 100, 100)
		second := makeShip("b", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 105, 100)
		_, got := decide([]*objUnit.BattleShip{first, second}, []*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 2, "cargo", "dd"),
		})
		assertSummon(t, got, "rp", "dd")
	})

	t.Run("frigate is the last attack filler", func(t *testing.T) {
		useTemplates(t, makeTemplate("ff", objUnit.ShipTypeFrigate, 2, 0, 0, 0, 1))
		_, got := decide(nil, []*objBuilding.ReinforcePoint{makePoint("rp", 0, 0, 2, "ff")})
		assertSummon(t, got, "rp", "ff")
	})

	t.Run("support ships and torpedo boats are not summoned", func(t *testing.T) {
		useTemplates(
			t,
			makeTemplate("cargo", objUnit.ShipTypeCargo, 0, 0, 0, 0, 1),
			makeTemplate("tb", objUnit.ShipTypeTorpedoBoat, 9, 0, 0, 0, 1),
		)
		_, got := decide(nil, []*objBuilding.ReinforcePoint{
			makePoint("rp", 0, 0, 2, "cargo", "tb"),
		})
		if len(got) != 0 {
			t.Fatalf("summons = %v", summonNames(got))
		}
	})

	t.Run("a busy queue is left alone", func(t *testing.T) {
		useTemplates(t, makeTemplate("bb", objUnit.ShipTypeBattleShip, 20, 0, 0, 0, 10))
		enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 2, 0)
		enemy.CombatPower.AntiShip = 10
		point := makePoint("rp", 0, 0, 3, "bb")
		point.OncomingShips = []*objBuilding.OncomingShip{{Name: "bb"}}
		_, got := decide([]*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{point})
		if len(got) != 0 {
			t.Fatalf("summons = %v", summonNames(got))
		}
	})

	t.Run("the next point sees the ship just chosen", func(t *testing.T) {
		useTemplates(t, makeTemplate("bb", objUnit.ShipTypeBattleShip, 20, 0, 0, 0, 10))
		enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 2, 0)
		enemy.CombatPower.AntiShip = 10
		_, got := decide([]*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{
			makePoint("a", 0, 0, 2, "bb"),
			makePoint("b", 500, 500, 2, "bb"),
		})
		assertSummon(t, got, "a", "bb")
		if _, _, ok := findSummon(got, "b"); ok {
			t.Fatal("second point summoned after the deficit was filled")
		}
	})
}

// TestDisabledStrategiesAreNotUsed 验证未勾选的策略不会产生对应编队。
// 场景：同一支 5 艘己方舰队分别用 Attack+Defend 与 Attack+Raid 两套配置编组，
// 前者没有 Raid/Scout/Counter 编队，后者没有 Defend，驻守组必须为空。
// 预期：第一套下 raid/scout/counter 均为 nil、attack={dd,ff,tb}；
// 第二套下 garrison=nil、raid={dd}、attack={bb,ff,tb}。
// 覆盖：航母 cv、战列舰 30 点对舰、驱逐舰 10 点、护卫舰与鱼雷艇各 1 艘，敌舰在 (6,0)。
func TestDisabledStrategiesAreNotUsed(t *testing.T) {
	carrier := makeShip("cv", faction.ComputerAlpha, objUnit.ShipTypeAircraftCarrier, 0, 0)
	battleship := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 1, 0)
	battleship.CombatPower.AntiShip = 30
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 2, 0)
	destroyer.CombatPower.AntiShip = 10
	frigate := makeShip("ff", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 3, 0)
	torpedo := makeShip("tb", faction.ComputerAlpha, objUnit.ShipTypeTorpedoBoat, 4, 0)
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 6, 0)
	enemy.CombatPower.AntiShip = 10
	ships := []*objUnit.BattleShip{carrier, battleship, destroyer, frigate, torpedo, enemy}
	point := []*objBuilding.ReinforcePoint{makePoint("rp", 0, 0, 0)}

	useStrategies(t, config.AIStrategies{Attack: true, Defend: true})
	handler, _ := decide(ships, point)
	if handler.raid != nil || handler.scout != nil || handler.counter != nil {
		t.Fatalf("raid=%v scout=%v counter=%v", handler.raid, handler.scout, handler.counter)
	}
	assertMembers(t, "attack", handler.attack, "dd", "ff", "tb")

	useStrategies(t, config.AIStrategies{Attack: true, Raid: true})
	handler, _ = decide(ships, point)
	if handler.garrison != nil {
		t.Fatalf("garrison = %v", handler.garrison.members)
	}
	assertMembers(t, "attack", handler.attack, "bb", "ff", "tb")
	assertMembers(t, "raid", handler.raid, "dd")
}

// TestCounterWaitsUntilEnemiesReachTheAnchor 验证反击只守锚点附近 20 格，不主动出击。
// 场景：只开 Counter，己方战列舰 20 点对舰守 (0,0)，敌战列舰先从 (80,0) 到 8 格处。
// 预期：80 格外时 counter 存在但不压上、靶标为空，也不生成进攻组；
// 进到 8 格后 counter.pressing=true、targetUID=="foe"，attack 仍为 nil。
// 覆盖：config.AIStrategies{Counter:true} 下两次 decide，核对 counter 字段。
func TestCounterWaitsUntilEnemiesReachTheAnchor(t *testing.T) {
	own := makeShip("bb", faction.ComputerAlpha, objUnit.ShipTypeBattleShip, 0, 0)
	own.CombatPower.AntiShip = 20
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 80, 0)
	enemy.CombatPower.AntiShip = 10
	point := []*objBuilding.ReinforcePoint{makePoint("rp", 0, 0, 0)}
	useStrategies(t, config.AIStrategies{Counter: true})

	handler, _ := decide([]*objUnit.BattleShip{own, enemy}, point)
	if handler.attack != nil || handler.counter == nil || handler.counter.pressing {
		t.Fatalf("far counter attack=%v counter=%+v", handler.attack, handler.counter)
	}

	enemy.CurPos = objPos.New(8, 0)
	handler, _ = decide([]*objUnit.BattleShip{own, enemy}, point)
	if handler.attack != nil || handler.counter == nil || !handler.counter.pressing ||
		handler.counter.targetUID != "foe" {
		t.Fatalf("near counter = %+v", handler.counter)
	}
}

// TestScoutStopsShortOfTheEnemy 验证侦察舰停在敌舰外侧 scoutStandoff=15 格处。
// 场景：只开 Scout，驱逐舰 dd 在 (0,10)，敌驱逐舰在 (50,10) 相距 50 格，
// standoffPos 从己方重心朝敌舰方向取 15 格。
// 预期：dd 的 ShipMove 目标是 (35,10)，即 50 减 15 的位置，保持约定的观察间距。
// 覆盖：TypeAbbr="WaterDrop" 让侦察舰走直线指令，断言 String 含 (35[35.00], 10[10.00])。
func TestScoutStopsShortOfTheEnemy(t *testing.T) {
	destroyer := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 10)
	destroyer.TypeAbbr = "WaterDrop"
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 50, 10)
	useStrategies(t, config.AIStrategies{Scout: true})
	_, got := decide([]*objUnit.BattleShip{destroyer, enemy}, []*objBuilding.ReinforcePoint{
		makePoint("rp", 0, 0, 0),
	})
	move, ok := got[instr.GenInstrUid(instr.NameShipMove, "dd")].(*instr.ShipMove)
	if !ok || !strings.Contains(move.String(), "(35[35.00], 10[10.00])") {
		t.Fatalf("scout move = %v", got)
	}
}

// TestFormationSpiral 验证锚点环形阵位的 0 号与 1 号槽位坐标。
// 场景：阵位按 formationStep=2 的方环螺旋展开，0 号占锚点本身，
// 1 号落在第一环西北角，偏移 (-1,-1) 再乘 2 格。
// 预期：formationPos((10,10),0)==(10,10)、formationPos((10,10),1)==(8,8)。
// 覆盖：直接调用 formationPos，用 samePos 做整数格比较。
func TestFormationSpiral(t *testing.T) {
	origin := objPos.New(10, 10)
	if got := formationPos(origin, 0); !samePos(got, origin) {
		t.Fatalf("slot 0 = %s", got.String())
	}
	if got := formationPos(origin, 1); !samePos(got, objPos.New(8, 8)) {
		t.Fatalf("slot 1 = %s", got.String())
	}
}

// TestGarrisonChaseAndHold 验证驻守航母只追锚点 12 格内的近敌，否则回阵位。
// 场景：默认策略下航母 cv 在 (0,0)、敌驱逐舰在 (5,0)，满足离锚点 ≤12 格、
// 离本舰 ≤20 格，属于可追击范围；随后把敌舰挪到 (40,0) 超出 chaseAnchorRadius。
// 预期：第一次决定给 cv 下 (5,0) 的追航线；第二次 cv 已在阵位、敌舰又太远，
// 不产生任何指令。失败意味着航母会脱离驻守去追全图目标。
// 覆盖：TypeAbbr="WaterDrop" 让 cv 走 ShipMove，断言航路字符串与第二次的空指令集。
func TestGarrisonChaseAndHold(t *testing.T) {
	carrier := makeShip("cv", faction.ComputerAlpha, objUnit.ShipTypeAircraftCarrier, 0, 0)
	carrier.TypeAbbr = "WaterDrop"
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 5, 0)
	_, got := decide([]*objUnit.BattleShip{carrier, enemy}, []*objBuilding.ReinforcePoint{
		makePoint("rp", 0, 0, 0),
	})
	move, ok := got[instr.GenInstrUid(instr.NameShipMove, "cv")].(*instr.ShipMove)
	if !ok || !strings.Contains(move.String(), "(5[5.00], 0[0.00])") {
		t.Fatalf("chase = %v", got)
	}

	carrier.CurPos = objPos.New(0, 0)
	enemy.CurPos = objPos.New(40, 0)
	_, got = decide([]*objUnit.BattleShip{carrier, enemy}, []*objBuilding.ReinforcePoint{
		makePoint("rp", 0, 0, 0),
	})
	if len(got) != 0 {
		t.Fatalf("hold issued %v", got)
	}
}

func newMission(
	tick int64, ships []*objUnit.BattleShip, points []*objBuilding.ReinforcePoint,
) *state.MissionState {
	misState := &state.MissionState{}
	misState.Core.SimTick = tick
	misState.Arena.Ships = map[string]*objUnit.BattleShip{}
	for _, ship := range ships {
		misState.Arena.Ships[ship.Uid] = ship
	}
	misState.Arena.ReinforcePoints = map[string]*objBuilding.ReinforcePoint{}
	for _, point := range points {
		misState.Arena.ReinforcePoints[point.Uid] = point
	}
	return misState
}

func decide(
	ships []*objUnit.BattleShip,
	points []*objBuilding.ReinforcePoint,
) (*ComputerDecisionHandler, map[string]instr.Instruction) {
	handler := NewHandler(faction.ComputerAlpha)
	return handler, handler.Handle(nil, newMission(0, ships, points))
}

func makeShip(uid string, player faction.Player, shipType objUnit.ShipType, x, y int) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: uid, Name: uid, BelongPlayer: player, Type: shipType,
		CurPos: objPos.New(x, y), CurHP: 100, TotalHP: 100, MaxSpeed: 1,
	}
}

func combatShip(uid string, shipType objUnit.ShipType, antiShip int) *objUnit.BattleShip {
	ship := makeShip(uid, faction.ComputerAlpha, shipType, 0, 0)
	ship.CombatPower.AntiShip = antiShip
	return ship
}

func makeTemplate(
	name string, shipType objUnit.ShipType, antiShip, antiAir, mobility, projection int, timeCost int64,
) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Name: name, Type: shipType, TimeCost: timeCost, TotalHP: 100, CurHP: 100,
		CombatPower: objUnit.CombatPowerInfo{
			AntiShip: antiShip, AntiAir: antiAir, Mobility: mobility, Projection: projection,
		},
	}
}

func makePoint(uid string, x, y, maxOncoming int, names ...string) *objBuilding.ReinforcePoint {
	return &objBuilding.ReinforcePoint{
		Uid: uid, BelongPlayer: faction.ComputerAlpha, RallyPos: objPos.New(x, y),
		MaxOncomingShip: maxOncoming, ProvidedShipNames: names,
	}
}

func useStrategies(t *testing.T, strategies config.AIStrategies) {
	t.Helper()
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.AI = strategies
	t.Cleanup(func() { config.G = previous })
}

func useTemplates(t *testing.T, templates ...*objUnit.BattleShip) {
	t.Helper()
	for _, template := range templates {
		if _, ok := objUnit.ShipMap[template.Name]; ok {
			t.Fatalf("template %s already registered", template.Name)
		}
		objUnit.ShipMap[template.Name] = template
	}
	t.Cleanup(func() {
		for _, template := range templates {
			delete(objUnit.ShipMap, template.Name)
		}
	})
}

func assertMembers(t *testing.T, label string, order *fleetOrder, want ...string) {
	t.Helper()
	got := memberIDs(order)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func memberIDs(order *fleetOrder) []string {
	if order == nil {
		return nil
	}
	out := append([]string{}, order.members...)
	sort.Strings(out)
	return out
}

func uidsOf(ships []*objUnit.BattleShip) []string {
	out := make([]string, len(ships))
	for i, ship := range ships {
		out[i] = ship.Uid
	}
	return out
}

func assertSummon(t *testing.T, instructions map[string]instr.Instruction, point, name string) {
	t.Helper()
	gotName, _, ok := findSummon(instructions, point)
	if !ok || gotName != name {
		t.Fatalf("point %s summon = %q, want %s (%v)", point, gotName, name, summonNames(instructions))
	}
}

func findSummon(instructions map[string]instr.Instruction, point string) (string, string, bool) {
	for _, instruction := range instructions {
		name, from, ok := summonedShip(instruction)
		if ok && from == point {
			return name, from, true
		}
	}
	return "", "", false
}

func summonedShip(instruction instr.Instruction) (string, string, bool) {
	rest, ok := strings.CutPrefix(instruction.String(), "summon ")
	if !ok {
		return "", "", false
	}
	name, point, ok := strings.Cut(rest, " from reinforce point ")
	return name, point, ok
}

func summonNames(instructions map[string]instr.Instruction) []string {
	names := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		name, point, ok := summonedShip(instruction)
		if ok {
			names = append(names, point+":"+name)
		}
	}
	sort.Strings(names)
	return names
}

// TestGarrisonKeepsInitialStationWithoutRallyPoint 验证无增援集结点时保留任务初始阵型。
// 场景：两艘航母 cv1 (10,10)、cv2 (20,10) 编成初始舰队，Arena.ReinforcePoints
// 为空；连续 31 拍（tick 0..30）调用 Handle。
// 预期：每一拍都不产生任何指令，舰船不被重排成锚点外 formationStep=2 的环形阵位；
// 否则 128 长的航母会被 2 格间距的环挤在一起并各自转向。
// 覆盖：NewHandler + newMission(无 points) 循环调用 Handle，逐拍断言返回指令集为空。
func TestGarrisonKeepsInitialStationWithoutRallyPoint(t *testing.T) {
	carriers := []*objUnit.BattleShip{
		makeShip("cv1", faction.ComputerAlpha, objUnit.ShipTypeAircraftCarrier, 10, 10),
		makeShip("cv2", faction.ComputerAlpha, objUnit.ShipTypeAircraftCarrier, 20, 10),
	}
	handler := NewHandler(faction.ComputerAlpha)
	for tick := int64(0); tick <= 30; tick++ {
		got := handler.Handle(nil, newMission(tick, carriers, nil))
		if len(got) != 0 {
			t.Fatalf("tick %d: 初始舰队被重新排阵: %v", tick, got)
		}
	}
}
