package manager

import (
	"math/rand"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

const antiTorpedoTestShell = "test/anti-torpedo-shell"

// antiTorpedoTestCore 返回一张全海测试地图，避免鱼雷航迹经过陆地分支。
func antiTorpedoTestCore() state.MissionCoreState {
	return state.MissionCoreState{MissionMD: metadata.MissionMetadata{
		MapCfg: &mapcfg.MapCfg{Map: mapcfg.MapData{"S"}},
	}}
}

// registerAntiTorpedoShell 注册反鱼雷测试炮弹并登记清理。
func registerAntiTorpedoShell(t *testing.T) {
	t.Helper()
	oldBullet, hadBullet := objBullet.Map[antiTorpedoTestShell]
	objBullet.Map[antiTorpedoTestShell] = &objBullet.Bullet{
		Name: antiTorpedoTestShell, Type: objBullet.TypeShell, Diameter: 40,
		Damage: 10, CriticalRate: 0,
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[antiTorpedoTestShell] = oldBullet
		} else {
			delete(objBullet.Map, antiTorpedoTestShell)
		}
	})
}

// newAntiTorpedoTestGun 造一门装填就绪、全向射界的 40mm 反鱼雷自动炮。
func newAntiTorpedoTestGun() *objUnit.Gun {
	return &objUnit.Gun{
		Name: "test/anti-torpedo-gun", BulletName: antiTorpedoTestShell,
		BulletCount: 1, ReloadTime: 0, Range: 3,
		BulletSpeed: 0.2, BulletSpread: 0,
		AntiAircraft: true, AntiTorpedo: true,
		LeftFiringArc:  objUnit.FiringArc{Start: 0, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 360},
	}
}

// TestInitDerivesAntiTorpedoForAutocannons 验证启动期只把「对空 + ≤40mm」的自动炮标记为可反鱼雷。
// 场景：真实配置已加载，US/40/56/4 是 40mm 博福斯，US/127/40/MK3 是 127mm 高平两用炮；
// 同时检查装备 40mm 的 porter 是否算出了反鱼雷最大射程。
// 预期：40mm 炮 AntiTorpedo=true、127mm 炮 AntiTorpedo=false，porter 的 MaxToTorpedoRange>0；
// 失败意味着口径推导把大口径炮也放进来，或舰船没有汇总反鱼雷射程。
// 覆盖：initGunMap 的 antiTorpedoMaxCaliber 分支与 initShipMap 的 MaxToTorpedoRange 汇总。
func TestInitDerivesAntiTorpedoForAutocannons(t *testing.T) {
	autocannon, ok := objUnit.GunMap["US/40/56/4"]
	if !ok {
		t.Fatal("US/40/56/4 not loaded")
	}
	if !autocannon.AntiTorpedo {
		t.Fatal("40mm autocannon should be able to engage torpedoes")
	}

	heavyGun, ok := objUnit.GunMap["US/127/40/MK3"]
	if !ok {
		t.Fatal("US/127/40/MK3 not loaded")
	}
	if heavyGun.AntiTorpedo {
		t.Fatal("127mm gun should not be able to engage torpedoes")
	}

	porter, ok := objUnit.ShipMap["porter"]
	if !ok {
		t.Fatal("porter not loaded")
	}
	if porter.Weapon.MaxToTorpedoRange <= 0 {
		t.Fatal("porter should have a positive anti-torpedo range")
	}
}

// TestAntiTorpedoFireControlCooldownCapsVolleys 验证整舰反鱼雷火控按舰种间隔只组织一次拦截，且一次只动用一座炮。
// 场景：测试舰（未标注舰种，用基准档 1 秒）带两门装填就绪的 40mm 反鱼雷炮，敌鱼雷在射程内；
// SimTick=1 时开火，SimTick=2 再开火，SimTick=1+ReloadTicks(1)=61 时第三次开火。
// 预期：第 1 拍只产生 1 发反鱼雷弹（两门炮只有一门开火），第 2 拍不产生，第 61 拍再产生 1 发；
// 失败意味着炮座数量仍能线性堆叠射速，大舰会重新变成鱼雷免疫。
// 覆盖：BattleShip.FireAt 的 DefaultAntiTorpedoProfile 冷却分支、fireOneMount 的单炮限制。
func TestAntiTorpedoFireControlCooldownCapsVolleys(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })
	objUnit.SetSimTick(0)
	t.Cleanup(func() { objUnit.SetSimTick(0) })

	registerAntiTorpedoShell(t)
	shooter := &objUnit.BattleShip{
		Uid: "shooter", BelongPlayer: faction.HumanAlpha,
		CurPos: objPos.NewR(50, 50), CurRotation: 0,
		TotalHP: 1000, CurHP: 1000,
		Weapon: objUnit.ShipWeapon{
			AntiAircraftGuns: []*objUnit.Gun{
				newAntiTorpedoTestGun(), newAntiTorpedoTestGun(),
			},
			MaxToTorpedoRange: 3,
			MaxToPlaneRange:   3,
		},
	}
	torpedo := &objBullet.Bullet{
		Name: "test/incoming-torpedo", Type: objBullet.TypeTorpedo,
		CurPos: objPos.NewR(52, 50), Rotation: 270, Speed: 0.1,
		Life: 100, BelongPlayer: faction.ComputerAlpha,
	}
	manager := &MissionManager{headless: true, state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{shooter.Uid: shooter},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: []*objBullet.Bullet{torpedo},
		},
	}}
	countAntiTorpedoShells := func() int {
		count := 0
		for _, bt := range manager.state.Arena.ForwardingBullets {
			if bt.Type == objBullet.TypeShell && bt.TargetObjType == object.TypeTorpedo {
				count++
			}
		}
		return count
	}

	objUnit.SetSimTick(1)
	manager.updateShipWeaponFire()
	if got := countAntiTorpedoShells(); got != 1 {
		t.Fatalf("first volley shells = %d, want 1", got)
	}

	objUnit.SetSimTick(2)
	manager.updateShipWeaponFire()
	if got := countAntiTorpedoShells(); got != 1 {
		t.Fatalf("shells during cooldown = %d, want 1", got)
	}

	objUnit.SetSimTick(1 + objUnit.ReloadTicks(objUnit.DefaultAntiTorpedoProfile(shooter.Type).FireInterval))
	manager.updateShipWeaponFire()
	if got := countAntiTorpedoShells(); got != 2 {
		t.Fatalf("shells after cooldown = %d, want 2", got)
	}
}

// TestAntiTorpedoGunTargetsIncomingTorpedo 验证小口径自动炮会优先把射程内的敌方鱼雷选为反鱼雷目标。
// 场景：测试舰在 (50,50) 带一门 40mm 反鱼雷炮（射程 3），敌鱼雷在 (52,50) 以 270° 航向直冲本舰。
// 预期：本帧只产生 1 发炮弹，TargetObjType=TypeTorpedo、TargetTorpedo 指向该鱼雷；
// 失败意味着反鱼雷炮仍在打舰船/飞机或没有把鱼雷选为目标。
// 覆盖：rebuildAntiTorpedoCandidates、nearestEnemyTorpedo 的敌我与距离判定，
// 以及 FireTargets.AntiTorpedo -> BattleShip.FireAt -> Gun.Fire 的目标传递链。
func TestAntiTorpedoGunTargetsIncomingTorpedo(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	registerAntiTorpedoShell(t)
	shooter := &objUnit.BattleShip{
		Uid: "shooter", BelongPlayer: faction.HumanAlpha,
		CurPos: objPos.NewR(50, 50), CurRotation: 0,
		TotalHP: 1000, CurHP: 1000,
		Weapon: objUnit.ShipWeapon{
			AntiAircraftGuns:  []*objUnit.Gun{newAntiTorpedoTestGun()},
			MaxToTorpedoRange: 3,
			MaxToPlaneRange:   3,
		},
	}
	torpedo := &objBullet.Bullet{
		Name: "test/incoming-torpedo", Type: objBullet.TypeTorpedo,
		CurPos: objPos.NewR(52, 50), Rotation: 270, Speed: 0.1,
		Life: 100, BelongPlayer: faction.ComputerAlpha,
	}

	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{shooter.Uid: shooter},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: []*objBullet.Bullet{torpedo},
		},
	}}
	manager.updateShipWeaponFire()

	bullets := manager.state.Arena.ForwardingBullets
	shellCount := 0
	for _, bt := range bullets {
		if bt.Type == objBullet.TypeShell {
			shellCount++
		}
	}
	if shellCount != 1 {
		t.Fatalf("shell count = %d, want 1", shellCount)
	}
	for _, shell := range bullets {
		if shell.Type != objBullet.TypeShell {
			continue
		}
		if shell.TargetObjType != object.TypeTorpedo {
			t.Fatalf("shell target type = %v, want torpedo", shell.TargetObjType)
		}
		if shell.TargetTorpedo != torpedo {
			t.Fatalf("shell target torpedo = %p, want %p", shell.TargetTorpedo, torpedo)
		}
	}
}

// TestAntiTorpedoHitDestroysTorpedo 验证炮弹穿过鱼雷命中框且概率通过时，鱼雷被标记击毁并退出战场。
// 场景：炮弹从 (50.5,50) 以 1 格/拍向西飞，鱼雷在 (50.1,50) 以 0.1 格/拍向西航行；
// 随机源 seed=9 的首个 Float64 约 0.0036，低于 antiTorpedoHitChance=0.10。
// 预期：鱼雷 HitObjType=TypeTorpedo，炮弹也以 TypeTorpedo 到达，ForwardingBullets 清空；
// 失败意味着拦截判定、概率门或已命中弹药清理链路有一处失效。
// 覆盖：resolveDamage 的 TypeTorpedo 分支、TargetTorpedo 指针结算、外循环对已命中目标的提前到达处理。
func TestAntiTorpedoHitDestroysTorpedo(t *testing.T) {
	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(9)))
	t.Cleanup(restoreRand)

	torpedo := &objBullet.Bullet{
		Name: "test/intercepted-torpedo", Type: objBullet.TypeTorpedo,
		CurPos: objPos.NewR(50.1, 50), Rotation: 270, Speed: 0.1,
		Life: 100, BelongPlayer: faction.ComputerAlpha,
	}
	shell := &objBullet.Bullet{
		Name: antiTorpedoTestShell, Type: objBullet.TypeShell, Diameter: 40,
		CurPos: objPos.NewR(50.5, 50), TargetPos: objPos.NewR(50, 50),
		Rotation: 270, Speed: 1, Life: 10,
		TargetObjType: object.TypeTorpedo, TargetTorpedo: torpedo,
		Shooter: "shooter", ShooterObjType: object.TypeShip, BelongPlayer: faction.HumanAlpha,
	}
	manager := &MissionManager{headless: true, state: &state.MissionState{
		Core: antiTorpedoTestCore(),
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: []*objBullet.Bullet{torpedo, shell},
		},
	}}
	manager.updateShotBullets()

	if torpedo.HitObjType != object.TypeTorpedo {
		t.Fatalf("torpedo hit type = %v, want torpedo", torpedo.HitObjType)
	}
	if shell.HitObjType != object.TypeTorpedo {
		t.Fatalf("shell hit type = %v, want torpedo", shell.HitObjType)
	}
	if len(manager.state.Arena.ForwardingBullets) != 0 {
		t.Fatalf("forwarding bullets = %d, want 0", len(manager.state.Arena.ForwardingBullets))
	}
}

// TestAntiTorpedoProbabilityGateKeepsTorpedo 验证概率门未通过时炮弹不会击毁鱼雷。
// 场景：与命中用例相同的几何关系，但随机源 seed=1 的首个 Float64 约 0.6047，高于 0.10。
// 预期：鱼雷 HitObjType 仍为 TypeNone，炮弹与鱼雷都继续留在 ForwardingBullets；
// 失败意味着概率门被绕过，反鱼雷会变成必中。
// 覆盖：resolveDamage 的 antiTorpedoBaseHitChance 小于判断分支，与命中用例形成真假两路。
func TestAntiTorpedoProbabilityGateKeepsTorpedo(t *testing.T) {
	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(1)))
	t.Cleanup(restoreRand)

	torpedo := &objBullet.Bullet{
		Name: "test/surviving-torpedo", Type: objBullet.TypeTorpedo,
		CurPos: objPos.NewR(50.1, 50), Rotation: 270, Speed: 0.1,
		Life: 100, BelongPlayer: faction.ComputerAlpha,
	}
	shell := &objBullet.Bullet{
		Name: antiTorpedoTestShell, Type: objBullet.TypeShell, Diameter: 40,
		CurPos: objPos.NewR(50.5, 50), TargetPos: objPos.NewR(50, 50),
		Rotation: 270, Speed: 1, Life: 10,
		TargetObjType: object.TypeTorpedo, TargetTorpedo: torpedo,
		Shooter: "shooter", ShooterObjType: object.TypeShip, BelongPlayer: faction.HumanAlpha,
	}
	manager := &MissionManager{headless: true, state: &state.MissionState{
		Core: antiTorpedoTestCore(),
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: []*objBullet.Bullet{torpedo, shell},
		},
	}}
	manager.updateShotBullets()

	if torpedo.HitObjType != object.TypeNone {
		t.Fatalf("torpedo hit type = %v, want none", torpedo.HitObjType)
	}
	if len(manager.state.Arena.ForwardingBullets) != 2 {
		t.Fatalf("forwarding bullets = %d, want 2", len(manager.state.Arena.ForwardingBullets))
	}
}

// antiTorpedoTestManager 构造只含一艘舰与给定鱼雷的 manager，并预收集鱼雷候选。
func antiTorpedoTestManager(ship *objUnit.BattleShip, torpedoes ...*objBullet.Bullet) *MissionManager {
	m := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: torpedoes,
		},
	}}
	m.rebuildAntiTorpedoCandidates()
	return m
}

// TestNearestEnemyTorpedoIgnoresTrackAndFriend 验证反鱼雷择敌只看敌我与距离，不再判断航迹是否指向本舰。
// 场景：本舰在 (50,50)、反鱼雷射程 3 格、全向炮座，敌方鱼雷分别放在正东 2 格，航向 270°（直冲本舰）、
// 90°（背离本舰）、0°（从舰艏前擦过）；另测己方鱼雷在正东 1 格、敌方鱼雷在正东 4 格（射程外）、
// 两条射程内敌雷分别位于 2 格与 3 格，以及只留舰艏射界时正后方 1 格的近雷与正东 2 格的远雷。
// 预期：三种航迹都返回该鱼雷；己方鱼雷与射程外鱼雷返回 nil；两条敌雷时返回 2 格那条；
// 只留舰艏射界时跳过正后方近雷、选中正东远雷；
// 失败意味着择敌又绑回「航迹穿过本舰」，或射界外的雷能占住火控反复空转。
// 覆盖：nearestEnemyTorpedo 的友军过滤、射程上界（等号内）、最近距离更新、航迹无关性，
// 以及 CanBearAntiTorpedo 的射界预筛。
func TestNearestEnemyTorpedoIgnoresTrackAndFriend(t *testing.T) {
	newShip := func(gun *objUnit.Gun) *objUnit.BattleShip {
		return &objUnit.BattleShip{
			Uid: "shooter", BelongPlayer: faction.HumanAlpha, CurPos: objPos.NewR(50, 50),
			Weapon: objUnit.ShipWeapon{
				AntiAircraftGuns: []*objUnit.Gun{gun}, MaxToTorpedoRange: 3,
			},
		}
	}
	ship := newShip(newAntiTorpedoTestGun())
	newTorpedo := func(pos objPos.MapPos, rot float64, player faction.Player) *objBullet.Bullet {
		return &objBullet.Bullet{
			Name: "test/track-torpedo", Type: objBullet.TypeTorpedo,
			CurPos: pos, Rotation: rot, Speed: 0.1, Life: 100, BelongPlayer: player,
		}
	}

	for _, tc := range []struct {
		name string
		rot  float64
	}{
		{name: "直冲本舰", rot: 270},
		{name: "背离本舰", rot: 90},
		{name: "从舰艏前方擦过", rot: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			torpedo := newTorpedo(objPos.NewR(52, 50), tc.rot, faction.ComputerAlpha)
			m := antiTorpedoTestManager(ship, torpedo)
			if got := m.nearestEnemyTorpedo(ship); got != torpedo {
				t.Fatalf("nearestEnemyTorpedo = %v, want the %s torpedo", got, tc.name)
			}
		})
	}

	t.Run("友军鱼雷不拦", func(t *testing.T) {
		friendly := newTorpedo(objPos.NewR(51, 50), 270, faction.HumanAlpha)
		m := antiTorpedoTestManager(ship, friendly)
		if got := m.nearestEnemyTorpedo(ship); got != nil {
			t.Fatalf("nearestEnemyTorpedo = %v, want nil for friendly torpedo", got)
		}
	})

	t.Run("射程外不打", func(t *testing.T) {
		far := newTorpedo(objPos.NewR(54, 50), 270, faction.ComputerAlpha)
		m := antiTorpedoTestManager(ship, far)
		if got := m.nearestEnemyTorpedo(ship); got != nil {
			t.Fatalf("nearestEnemyTorpedo = %v, want nil beyond anti-torpedo range", got)
		}
	})

	t.Run("取最近的敌雷", func(t *testing.T) {
		near := newTorpedo(objPos.NewR(52, 50), 270, faction.ComputerAlpha)
		far := newTorpedo(objPos.NewR(53, 50), 90, faction.ComputerAlpha)
		m := antiTorpedoTestManager(ship, far, near)
		if got := m.nearestEnemyTorpedo(ship); got != near {
			t.Fatalf("nearestEnemyTorpedo = %v, want the nearer torpedo", got)
		}
	})

	t.Run("射界外的最近雷跳过", func(t *testing.T) {
		// 只在舰艏左右各 90° 有射界：正后方的雷转不过去，不能占住火控。
		forwardOnly := newAntiTorpedoTestGun()
		forwardOnly.LeftFiringArc = objUnit.FiringArc{Start: 1, End: 90}
		forwardOnly.RightFiringArc = objUnit.FiringArc{Start: 270, End: 359}
		guarded := newShip(forwardOnly)
		astern := newTorpedo(objPos.NewR(50, 51), 0, faction.ComputerAlpha)
		ahead := newTorpedo(objPos.NewR(52, 50), 270, faction.ComputerAlpha)
		m := antiTorpedoTestManager(guarded, astern, ahead)
		if got := m.nearestEnemyTorpedo(guarded); got != ahead {
			t.Fatalf("nearestEnemyTorpedo = %v, want the bearable forward torpedo", got)
		}
	})
}

// newAntiTorpedoProbeGun 造一门按真实博福斯四联打 4 发齐射的反鱼雷炮。
func newAntiTorpedoProbeGun() *objUnit.Gun {
	gun := newAntiTorpedoTestGun()
	gun.BulletCount = 4
	// 真实博福斯四联有散布，探针也带上，否则每发弹都压在命中框中心。
	gun.BulletSpread = 20
	return gun
}

// antiTorpedoProbeShip 构造带两门 40mm 四联反鱼雷炮的测试舰，舰种可切换。
// 两门炮是为了让 0.5 秒火控间隔的第二次拦截真的有炮可用（单炮会卡在装填边界）。
func antiTorpedoProbeShip(uid string, shipType objUnit.ShipType, player faction.Player) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: uid, Type: shipType, BelongPlayer: player,
		CurPos: objPos.NewR(50, 150), CurRotation: 0,
		TotalHP: 100000, CurHP: 100000,
		Weapon: objUnit.ShipWeapon{
			AntiAircraftGuns:  []*objUnit.Gun{newAntiTorpedoProbeGun(), newAntiTorpedoProbeGun()},
			MaxToTorpedoRange: 3, MaxToPlaneRange: 3,
		},
	}
}

// antiTorpedoInterceptTrials 跑 trials 次「Mk15 533mm 鱼雷从 8 格外直冲静止测试舰」，
// 统计鱼雷被机炮拦下的次数。随机源每次按固定种子重设，结果可复现。
func antiTorpedoInterceptTrials(t *testing.T, shipType objUnit.ShipType, trials int) int {
	t.Helper()
	return antiTorpedoInterceptTrialsFor(t, shipType, "US/TB/533/1935/Mk15", object.TypeShip, trials)
}

// antiTorpedoInterceptTrialsFor 按指定鱼雷模板与投放来源跑 trials 次直冲测试。
// 种子固定，同一组参数重复运行结果一致，便于把「航空雷 vs 舰载雷」的差值当断言。
func antiTorpedoInterceptTrialsFor(
	t *testing.T, shipType objUnit.ShipType, torpName string, shooterType object.Type, trials int,
) int {
	t.Helper()
	intercepted := 0
	for i := 0; i < trials; i++ {
		intercepted += runAntiTorpedoInterceptTrialFor(t, shipType, torpName, shooterType, int64(i*7919+13))
	}
	return intercepted
}

// runAntiTorpedoInterceptTrial 跑单次直冲测试，返回鱼雷是否被拦下。
func runAntiTorpedoInterceptTrial(t *testing.T, shipType objUnit.ShipType, seed int64) int {
	t.Helper()
	return runAntiTorpedoInterceptTrialFor(t, shipType, "US/TB/533/1935/Mk15", object.TypeShip, seed)
}

// runAntiTorpedoInterceptTrialFor 用指定鱼雷模板与投放来源跑单次直冲测试，返回是否被拦下。
func runAntiTorpedoInterceptTrialFor(
	t *testing.T, shipType objUnit.ShipType, torpName string, shooterType object.Type, seed int64,
) int {
	t.Helper()
	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(seed)))
	defer restoreRand()

	torpTpl, ok := objBullet.Map[torpName]
	if !ok {
		t.Fatalf("torpedo template %q not loaded", torpName)
	}
	torpedo := &objBullet.Bullet{
		Name: torpTpl.Name, Type: objBullet.TypeTorpedo,
		Diameter: torpTpl.Diameter, Damage: torpTpl.Damage, CriticalRate: torpTpl.CriticalRate,
		CurPos: objPos.NewR(58, 150), TargetPos: objPos.NewR(50, 150),
		Rotation: 270, Speed: 0.075, Life: 400,
		BelongPlayer: faction.ComputerAlpha, ShooterObjType: shooterType,
		TargetObjType: object.TypeShip,
	}
	shooter := antiTorpedoProbeShip("probe", shipType, faction.HumanAlpha)
	manager := &MissionManager{headless: true, state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{shooter.Uid: shooter},
			Planes:            map[string]*objUnit.Plane{},
			ForwardingBullets: []*objBullet.Bullet{torpedo},
		},
	}}
	manager.state.UI.GameMarks = map[objMark.ID]*objMark.Mark{}
	manager.state.Core = antiTorpedoTestCore()

	for tick := int64(1); tick < 400; tick++ {
		objUnit.SetSimTick(tick)
		manager.updateShipWeaponFire()
		manager.updateShotBullets()
		if torpedo.HitObjType == object.TypeTorpedo {
			return 1
		}
		if torpedo.HitObjType == object.TypeShip || torpedo.CurPos.RX < 40 {
			return 0
		}
	}
	return 0
}

// TestAntiTorpedoDestroyerOutperformsBattleship 验证同样炮座配置下驱逐舰的反鱼雷拦截率明显高于战列舰。
// 场景：两艘只有舰种不同的测试舰（各两门 40mm 四联反鱼雷炮、射程 3 格），
// 敌 Mk15 533mm 鱼雷从 8 格外直冲本舰，驱逐舰与战列舰各用同一组 120 个固定种子跑一遍。
// 预期：驱逐舰拦截率至少比战列舰高 15 个百分点，驱逐档落在 15%~55%、主力舰档落在 1%~15%；
// 失败意味着舰种分档失效——驱逐舰又会「遇到鱼雷等死」，或战列舰重新变成鱼雷免疫。
// 覆盖：DefaultAntiTorpedoProfile 的驱逐 / 主力舰两档，经 FireAt 冷却与单发概率、
// 真实 updateShipWeaponFire / updateShotBullets 结算成拦截率。
func TestAntiTorpedoDestroyerOutperformsBattleship(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})
	registerAntiTorpedoShell(t)

	const trials = 120
	destroyerHits := antiTorpedoInterceptTrials(t, objUnit.ShipTypeDestroyer, trials)
	battleshipHits := antiTorpedoInterceptTrials(t, objUnit.ShipTypeBattleShip, trials)
	destroyerRate := float64(destroyerHits) / trials
	battleshipRate := float64(battleshipHits) / trials

	if destroyerRate <= battleshipRate+0.15 {
		t.Fatalf("驱逐舰拦截率 %.1f%% 没有明显高于战列舰 %.1f%%",
			100*destroyerRate, 100*battleshipRate)
	}
	if destroyerRate < 0.15 || destroyerRate > 0.55 {
		t.Fatalf("驱逐舰拦截率 %.1f%% 不在 15%%~55%% 的预期区间（既不该等死也不该免疫）",
			100*destroyerRate)
	}
	if battleshipRate < 0.01 || battleshipRate > 0.15 {
		t.Fatalf("战列舰拦截率 %.1f%% 不在 1%%~15%% 的预期区间（只该是最后一道近防）",
			100*battleshipRate)
	}
}

// TestAntiTorpedoAirTorpedoIsEasierThanShipTorpedo 验证同战斗部下航空鱼雷比舰载鱼雷更容易被机炮拦下。
// 场景：Mk13 570mm 鱼雷（4800 伤害）分别按飞机投放与舰载发射两种来源，从 8 格外直冲同一艘
// 驱逐舰测试舰，各用同一组 120 个固定种子跑一遍；两种来源只差 TorpedoDetonationFactor 的 1.5 倍。
// 预期：航空雷拦截率至少比舰载雷高 5 个百分点，航空雷不超过 55%、舰载雷不低于 5%，
// 即航空雷更好拦但仍有突防机会；失败意味着投放修正没生效，或航空雷被放大到接近必拦。
// 覆盖：TorpedoDetonationFactor 的 ShooterObjType==TypePlane 分支，经真实
// updateShipWeaponFire / updateShotBullets 结算成拦截率。
func TestAntiTorpedoAirTorpedoIsEasierThanShipTorpedo(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})
	registerAntiTorpedoShell(t)

	const (
		trials   = 120
		mk13Name = "US/TB/570/1938/Mk13"
	)
	shipHits := antiTorpedoInterceptTrialsFor(t, objUnit.ShipTypeDestroyer, mk13Name, object.TypeShip, trials)
	airHits := antiTorpedoInterceptTrialsFor(t, objUnit.ShipTypeDestroyer, mk13Name, object.TypePlane, trials)
	shipRate := float64(shipHits) / trials
	airRate := float64(airHits) / trials

	// 用于平衡观测：-v 时打印实测拦截率，调参后可直接对比 docs/anti-torpedo/README.md。
	t.Logf("驱逐舰拦截率：航空雷 %.1f%% / 舰载雷 %.1f%%", 100*airRate, 100*shipRate)

	if airRate < shipRate+0.05 {
		t.Fatalf("航空雷拦截率 %.1f%% 没有明显高于舰载雷 %.1f%%", 100*airRate, 100*shipRate)
	}
	if airRate > 0.55 {
		t.Fatalf("航空雷拦截率 %.1f%% 过高，航空雷不该接近必拦", 100*airRate)
	}
	if shipRate < 0.05 {
		t.Fatalf("舰载雷拦截率 %.1f%% 过低，驱逐舰不该对鱼雷无还手之力", 100*shipRate)
	}
}

// TestAntiTorpedoVolleyStampsShipFireControl 验证反鱼雷炮弹会按发射舰舰种写入单发引爆概率。
// 场景：驱逐舰与战列舰各带同一门 40mm 反鱼雷炮，敌鱼雷在 2 格内，各自 FireAt 一轮。
// 预期：炮弹 AntiTorpedoChance 分别为 0.08 与 0.03；失败意味着命中结算拿不到舰种差异，
// 所有舰船的反鱼雷强度会重新拉平。
// 覆盖：BattleShip.FireAt 的 profile 写入分支与 Gun.Fire -> FireTargets.AntiTorpedo 链路。
func TestAntiTorpedoVolleyStampsShipFireControl(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })
	registerAntiTorpedoShell(t)

	for _, tc := range []struct {
		shipType objUnit.ShipType
		want     float64
	}{
		{objUnit.ShipTypeDestroyer, 0.08},
		{objUnit.ShipTypeBattleShip, 0.03},
	} {
		shooter := antiTorpedoProbeShip("probe", tc.shipType, faction.HumanAlpha)
		torpedo := &objBullet.Bullet{
			Name: "test/incoming-torpedo", Type: objBullet.TypeTorpedo,
			CurPos: objPos.NewR(52, 150), Rotation: 270, Speed: 0.1,
			Life: 100, BelongPlayer: faction.ComputerAlpha,
		}
		target := &objUnit.TorpedoTarget{Bullet: torpedo}
		bullets := shooter.FireAt(objUnit.FireTargets{AntiTorpedo: target})
		if len(bullets) == 0 {
			t.Fatalf("%s 没有打出反鱼雷弹", tc.shipType)
		}
		for _, bt := range bullets {
			if bt.AntiTorpedoChance != tc.want {
				t.Fatalf("%s 炮弹 AntiTorpedoChance = %v, want %v",
					tc.shipType, bt.AntiTorpedoChance, tc.want)
			}
		}
	}
}
