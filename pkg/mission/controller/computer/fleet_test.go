package computer

import (
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

func TestAttackPressHysteresis(t *testing.T) {
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
		{85, 150, true, false},
		{84, 150, false, true},
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
	if handler.attack != nil || handler.counter == nil || !handler.counter.pressing || handler.counter.targetUID != "foe" {
		t.Fatalf("near counter = %+v", handler.counter)
	}
}

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

func TestFormationSpiral(t *testing.T) {
	origin := objPos.New(10, 10)
	if got := formationPos(origin, 0); !samePos(got, origin) {
		t.Fatalf("slot 0 = %s", got.String())
	}
	if got := formationPos(origin, 1); !samePos(got, objPos.New(8, 8)) {
		t.Fatalf("slot 1 = %s", got.String())
	}
}

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

// TestGarrisonKeepsInitialStationWithoutRallyPoint 锁定：没有增援集结点时，锚点就是
// 初始舰队自己的重心，AI 不该把任务布置好的初始阵型重排成锚点外的环形阵位——
// 航母这类长舰会被 2 格间距的环挤在一起、还各自转向，非常难看。
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
