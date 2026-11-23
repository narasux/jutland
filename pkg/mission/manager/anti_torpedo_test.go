package manager

import (
	"math/rand"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
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

// TestAntiTorpedoFireControlCooldownCapsVolleys 验证整舰反鱼雷火控每 1 秒只组织一次拦截，且一次只动用一座炮。
// 场景：测试舰带两门装填就绪的 40mm 反鱼雷炮，敌鱼雷在射程内直冲本舰；SimTick=1 时开火，
// SimTick=2 再开火，SimTick=1+ReloadTicks(1)=61 时第三次开火。
// 预期：第 1 拍只产生 1 发反鱼雷弹（两门炮只有一门开火），第 2 拍不产生，第 61 拍再产生 1 发；
// 失败意味着炮座数量仍能线性堆叠射速，战列舰会重新变成鱼雷免疫。
// 覆盖：FireAt 的 antiTorpedoFireInterval 冷却分支、fireOneMount 的单炮限制。
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

	objUnit.SetSimTick(1 + objUnit.ReloadTicks(objUnit.AntiTorpedoFireInterval))
	manager.updateShipWeaponFire()
	if got := countAntiTorpedoShells(); got != 2 {
		t.Fatalf("shells after cooldown = %d, want 2", got)
	}
}

// TestAntiTorpedoGunTargetsIncomingTorpedo 验证小口径自动炮会优先把来袭鱼雷选为反鱼雷目标。
// 场景：测试舰在 (50,50) 带一门 40mm 反鱼雷炮（射程 3），敌鱼雷在 (52,50) 以 270° 航向直冲本舰；
// 鱼雷航迹穿过舰心，属于真正的威胁。
// 预期：本帧只产生 1 发炮弹，TargetObjType=TypeTorpedo、TargetTorpedo 指向该鱼雷；
// 失败意味着反鱼雷炮仍在打舰船/飞机或没有把鱼雷选为目标。
// 覆盖：rebuildAntiTorpedoCandidates、nearestThreatTorpedo 的友军/射程/航迹判定，
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
// 覆盖：resolveDamage 的 antiTorpedoHitChance 小于判断分支，与命中用例形成真假两路。
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

// TestTorpedoOnCollisionCourse 验证只有朝本舰驶来且航迹穿过舰心附近的鱼雷才算威胁。
// 场景：本舰在原点，鱼雷分别在正西 2 格朝东（直冲）、正西 2 格朝西（远离）、
// 西北 2 格偏北 1 格朝东（航迹从舰心以北 1 格掠过）三种位置/航向；
// 横向阈值 antiTorpedoThreatCorridor=0.35。
// 预期：仅直冲航向返回 true；远离与横向偏移 1 格返回 false。
// 覆盖：表驱动 along<=0 与 cross>corridor 两条拒绝分支，以及命中走廊内的一路。
func TestTorpedoOnCollisionCourse(t *testing.T) {
	shipPos := objPos.NewR(0, 0)
	cases := []struct {
		name    string
		pos     objPos.MapPos
		rot     float64
		wantHit bool
	}{
		{name: "直接冲向本舰", pos: objPos.NewR(-2, 0), rot: 90, wantHit: true},
		{name: "背向本舰", pos: objPos.NewR(-2, 0), rot: 270, wantHit: false},
		{name: "横向偏移一格", pos: objPos.NewR(-2, -1), rot: 90, wantHit: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			torpedo := &objBullet.Bullet{CurPos: tc.pos, Rotation: tc.rot}
			if got := torpedoOnCollisionCourse(shipPos, torpedo); got != tc.wantHit {
				t.Fatalf("torpedoOnCollisionCourse(pos=%v, rot=%.0f) = %v, want %v",
					tc.pos, tc.rot, got, tc.wantHit)
			}
		})
	}
}
