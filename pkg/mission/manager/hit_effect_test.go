package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestHitEffectTiersByCaliber 验证命中闪光按 ≤40mm / 41–155mm / >155mm 三档分档，尺寸与持续帧数随口径单调增大。
// 场景：分别用 20mm、127mm、460mm 三档炮弹命中舰体调用 hitEffect，并比较三档返回的 scale。
// 预期：三档都返回 KindFlash；持续帧数分别为 smallHitLife=20、mediumHitLife=26、heavyHitLife=32，scale>0 且小<中<大。
// 覆盖：hitEffectTier 的 40/155mm 分档边界、hitEffectCells / hitEffectLives 查表与舰体分支。
func TestHitEffectTiersByCaliber(t *testing.T) {
	testCases := []struct {
		name     string
		diameter int
		hitType  object.Type
		wantKind objExplosion.Kind
		wantLife int
	}{
		{
			name: "small caliber", diameter: 20, hitType: object.TypeShip,
			wantKind: objExplosion.KindFlash, wantLife: smallHitLife,
		},
		{
			name: "medium caliber", diameter: 127, hitType: object.TypeShip,
			wantKind: objExplosion.KindFlash, wantLife: mediumHitLife,
		},
		{
			name: "heavy caliber", diameter: 460, hitType: object.TypeShip,
			wantKind: objExplosion.KindFlash, wantLife: heavyHitLife,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bt := &objBullet.Bullet{HitObjType: testCase.hitType, Diameter: testCase.diameter}
			kind, scale, life, ok := hitEffect(bt)
			if !ok {
				t.Fatalf("hitEffect(%v, %dmm) = no effect, want one", testCase.hitType, testCase.diameter)
			}
			if kind != testCase.wantKind {
				t.Fatalf("kind = %v, want %v", kind, testCase.wantKind)
			}
			if life != testCase.wantLife {
				t.Fatalf("life = %d, want %d", life, testCase.wantLife)
			}
			if scale <= 0 {
				t.Fatalf("scale = %v, want positive", scale)
			}
		})
	}

	_, smallScale, _, _ := hitEffect(&objBullet.Bullet{HitObjType: object.TypeShip, Diameter: 20})
	_, mediumScale, _, _ := hitEffect(&objBullet.Bullet{HitObjType: object.TypeShip, Diameter: 127})
	_, heavyScale, _, _ := hitEffect(&objBullet.Bullet{HitObjType: object.TypeShip, Diameter: 460})
	if !(smallScale < mediumScale && mediumScale < heavyScale) {
		t.Fatalf("scales not increasing: %v %v %v", smallScale, mediumScale, heavyScale)
	}
}

// TestHedgehogHitEffectIsSmallerThanDepthCharge 验证刺猬弹使用更小的水柱 / 水花，避免 24 发齐投糊成一片白。
// 场景：刺猬弹一次同帧投 24 发；若每发沿用普通深弹的 1.6 格水柱 / 2.4 格水花，整片海面会被白色特效淹没。
// 预期：命中舰体为 KindTorpedoPlume 且 scale/life 小于普通深弹；落水为 KindSplash 且 scale/life 小于普通深弹。
// 覆盖：构造 WeaponName=UK/DC/24/Hedgehog 的 TypeDepthCharge，分别命中舰体与水面，断言 kind 与两组上限。
func TestHedgehogHitEffectIsSmallerThanDepthCharge(t *testing.T) {
	hit := &objBullet.Bullet{
		Type: objBullet.TypeDepthCharge, WeaponName: "UK/DC/24/Hedgehog",
		HitObjType: object.TypeShip, Diameter: 30,
	}
	kind, scale, life, ok := hitEffect(hit)
	if !ok || kind != objExplosion.KindTorpedoPlume {
		t.Fatalf("hedgehog hit effect = (%v, %v, %v, %v), want torpedo plume", kind, scale, life, ok)
	}
	if scale >= depthChargePlumeCells || life >= depthChargePlumeLife {
		t.Fatalf("hedgehog plume = (%.2f, %d), want smaller than (%.2f, %d)",
			scale, life, depthChargePlumeCells, depthChargePlumeLife)
	}

	miss := &objBullet.Bullet{
		Type: objBullet.TypeDepthCharge, WeaponName: "UK/DC/24/Hedgehog",
		HitObjType: object.TypeWater, Diameter: 30,
	}
	kind, scale, life, ok = hitEffect(miss)
	if !ok || kind != objExplosion.KindSplash {
		t.Fatalf("hedgehog splash effect = (%v, %v, %v, %v), want splash", kind, scale, life, ok)
	}
	if scale >= depthChargeSplashCells || life >= depthChargeSplashLife {
		t.Fatalf("hedgehog splash = (%.2f, %d), want smaller than (%.2f, %d)",
			scale, life, depthChargeSplashCells, depthChargeSplashLife)
	}
}

// TestASWRocketHitEffectIsSmallerThanDepthCharge 验证火箭深弹使用自己的落点特效，不套用 300 磅深弹的大水花。
// 场景：250mm 65 式火箭深弹以 TypeRocket + AntiSubmarine 命中潜艇 / 落水；整枚弹若按 TypeDepthCharge
// 结算，会画出 1.6 格水柱 / 2.4 格水花，与 300mm 深弹混为一谈。
// 预期：命中舰体为 KindTorpedoPlume 且 scale/life 小于普通深弹；落水为 KindSplash 且 scale/life 小于
// 普通深弹、大于刺猬弹；水花仍压在舰船下层。
// 覆盖：hitEffect 的 IsASWRocket 分支、aswRocketPlumeCells/Life 与 aswRocketSplashCells/Life。
func TestASWRocketHitEffectIsSmallerThanDepthCharge(t *testing.T) {
	hit := &objBullet.Bullet{
		Type: objBullet.TypeRocket, AntiSubmarine: true,
		HitObjType: object.TypeShip, Diameter: 250,
	}
	kind, scale, life, ok := hitEffect(hit)
	if !ok || kind != objExplosion.KindTorpedoPlume {
		t.Fatalf("ASW rocket hit effect = (%v, %v, %v, %v), want torpedo plume", kind, scale, life, ok)
	}
	if scale >= depthChargePlumeCells || life >= depthChargePlumeLife {
		t.Fatalf("ASW rocket plume = (%.2f, %d), want smaller than (%.2f, %d)",
			scale, life, depthChargePlumeCells, depthChargePlumeLife)
	}

	miss := &objBullet.Bullet{
		Type: objBullet.TypeRocket, AntiSubmarine: true,
		HitObjType: object.TypeWater, Diameter: 250,
	}
	kind, scale, life, ok = hitEffect(miss)
	if !ok || kind != objExplosion.KindSplash {
		t.Fatalf("ASW rocket splash effect = (%v, %v, %v, %v), want splash", kind, scale, life, ok)
	}
	if scale >= depthChargeSplashCells || life >= depthChargeSplashLife {
		t.Fatalf("ASW rocket splash = (%.2f, %d), want smaller than (%.2f, %d)",
			scale, life, depthChargeSplashCells, depthChargeSplashLife)
	}
	if scale <= hedgehogSplashCells {
		t.Fatalf("ASW rocket splash = %.2f, want larger than hedgehog %.2f", scale, hedgehogSplashCells)
	}
	if kind.OverShips() {
		t.Fatal("ASW rocket splash is a water effect and should stay under ships")
	}
}

// TestHitEffectOnWaterAndLand 验证落水出水花、落陆出尘土，尺寸随口径增大，小口径防空弹两者都不生成。
// 场景：20/127/460mm 炮弹分别打在 TypeWater 与 TypeLand 上调用 hitEffect；另比较 127mm 与 460mm 的水花尺寸。
// 预期：20mm 落水、落陆均 ok=false；127/460mm 落水为 KindSplash、落陆为 KindDust，scale/life 为正，且 127mm 水花小于 460mm。
// 覆盖：hitEffect 的 TypeWater/TypeLand 分支、小口径档 tier==0 不生成、splashCells / dustCells 与 60 tick 的 splashLives / dustLives 查表。
func TestHitEffectOnWaterAndLand(t *testing.T) {
	testCases := []struct {
		name       string
		diameter   int
		hitType    object.Type
		wantKind   objExplosion.Kind
		wantEffect bool
	}{
		{name: "small caliber water", diameter: 20, hitType: object.TypeWater, wantEffect: false},
		{name: "small caliber land", diameter: 20, hitType: object.TypeLand, wantEffect: false},
		{
			name: "medium caliber water", diameter: 127, hitType: object.TypeWater,
			wantKind: objExplosion.KindSplash, wantEffect: true,
		},
		{
			name: "heavy caliber water", diameter: 460, hitType: object.TypeWater,
			wantKind: objExplosion.KindSplash, wantEffect: true,
		},
		{
			name: "medium caliber land", diameter: 127, hitType: object.TypeLand,
			wantKind: objExplosion.KindDust, wantEffect: true,
		},
		{
			name: "heavy caliber land", diameter: 460, hitType: object.TypeLand,
			wantKind: objExplosion.KindDust, wantEffect: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bt := &objBullet.Bullet{
				Type: objBullet.TypeShell, HitObjType: testCase.hitType, Diameter: testCase.diameter,
			}
			kind, scale, life, ok := hitEffect(bt)
			if ok != testCase.wantEffect {
				t.Fatalf(
					"hitEffect(%v, %dmm) spawned = %v, want %v",
					testCase.hitType,
					testCase.diameter,
					ok,
					testCase.wantEffect,
				)
			}
			if !ok {
				return
			}
			if kind != testCase.wantKind {
				t.Fatalf("kind = %v, want %v", kind, testCase.wantKind)
			}
			if scale <= 0 || life <= 0 {
				t.Fatalf("scale = %v, life = %d, want positive", scale, life)
			}
		})
	}

	waterShell := func(diameter int) *objBullet.Bullet {
		return &objBullet.Bullet{Type: objBullet.TypeShell, HitObjType: object.TypeWater, Diameter: diameter}
	}
	_, mediumScale, _, _ := hitEffect(waterShell(127))
	_, heavyScale, _, _ := hitEffect(waterShell(460))
	if mediumScale >= heavyScale {
		t.Fatalf("splash scales not increasing: %v %v", mediumScale, heavyScale)
	}
}

// TestHitEffectByAmmoType 验证鱼雷、炸弹、火箭与命中飞机的炮弹各自走专属表现，不重复叠加。
// 场景：533mm 鱼雷命中舰船、1100mm 炸弹落水、127mm 火箭落水、20/127/460mm 炮弹命中飞机，逐一调用 hitEffect。
// 预期：鱼雷返回 KindTorpedoPlume 且 OverShips 为 false；炸弹落水为 KindSplash 且 scale 大于 splashCells[2]=0.45；火箭与命中飞机的炮弹 ok=false。
// 覆盖：TypeTorpedo/TypeBomb/TypeRocket 专属分支、命中飞机不生成火光、鱼雷水柱压在舰船下层的绘制层级。
func TestHitEffectByAmmoType(t *testing.T) {
	kind, scale, life, ok := hitEffect(&objBullet.Bullet{
		Type: objBullet.TypeTorpedo, HitObjType: object.TypeShip, Diameter: 533,
	})
	if !ok || kind != objExplosion.KindTorpedoPlume {
		t.Fatalf("torpedo hit = (%v, %v), want torpedo plume", kind, ok)
	}
	if scale <= 0 || life <= 0 {
		t.Fatalf("torpedo scale = %v, life = %d, want positive", scale, life)
	}
	if kind.OverShips() {
		t.Fatal("torpedo plume is a water column and should stay under ships")
	}

	bombKind, bombScale, _, ok := hitEffect(&objBullet.Bullet{
		Type: objBullet.TypeBomb, HitObjType: object.TypeWater, Diameter: 1100,
	})
	if !ok || bombKind != objExplosion.KindSplash {
		t.Fatalf("bomb on water = (%v, %v), want splash", bombKind, ok)
	}
	splashScale := splashCells[2]
	if bombScale <= splashScale {
		t.Fatalf("bomb splash scale = %v, want larger than %v", bombScale, splashScale)
	}

	if _, _, _, ok := hitEffect(&objBullet.Bullet{
		Type: objBullet.TypeRocket, HitObjType: object.TypeWater, Diameter: 127,
	}); ok {
		t.Fatal("rocket airburst already has its own effect and should not spawn another")
	}

	// 命中飞机不出火光：防空弹幕太密，飞机被击落时本来就有自己的爆炸动画
	for _, diameter := range []int{20, 127, 460} {
		if _, _, _, ok := hitEffect(&objBullet.Bullet{
			Type: objBullet.TypeShell, HitObjType: object.TypePlane, Diameter: diameter,
		}); ok {
			t.Fatalf("plane hit with %dmm shell should not spawn an effect", diameter)
		}
	}
}

// TestHitEffectSpawnsOnShipHit 验证命中舰体后按开关生成命中闪光，且与伤害数字开关解耦。
// 场景：51.2×12.8 像素（0.4×0.1 格）靶舰在 (10,10)，10 伤害炮弹从 y=12 飞向 y=9；DisplayHitEffects=true、DisplayDamageNumber=false，推进 200 拍。
// 预期：Arena.Explosions 恰好 1 个且 Kind 为 KindFlash；伤害数字关闭不影响特效生成。
// 覆盖：updateShotBullets 伤害结算后的特效段、DisplayHitEffects 与 DisplayDamageNumber 两个开关的独立性。
func TestHitEffectSpawnsOnShipHit(t *testing.T) {
	target := newTestShip(10, 10)
	m := newShellImpactManager(target, newTestShell(0, 12, 9))
	m.state.UI.GameOpts.DisplayHitEffects = true
	m.state.UI.GameOpts.DisplayDamageNumber = false

	resolveShell(m, 200)

	if len(m.state.Arena.Explosions) != 1 {
		t.Fatalf("explosions = %d, want 1", len(m.state.Arena.Explosions))
	}
	if kind := m.state.Arena.Explosions[0].Kind; kind != objExplosion.KindFlash {
		t.Fatalf("explosion kind = %v, want light fireball", kind)
	}
}

// TestHitEffectToggleSuppressesSpawn 验证关闭 DisplayHitEffects 后不再生成命中特效，但伤害照常结算。
// 场景：同一 51.2×12.8 像素靶舰与 10 伤害炮弹，DisplayHitEffects=false，resolveShell 推进 200 拍。
// 预期：Explosions 为 0；target.CurHP 不再是 100（本发应扣到 90），证明关闭的只是表现层。
// 覆盖：命中特效段提前 return 不影响此前 hurtShip 的伤害结算。
func TestHitEffectToggleSuppressesSpawn(t *testing.T) {
	target := newTestShip(10, 10)
	m := newShellImpactManager(target, newTestShell(0, 12, 9))
	m.state.UI.GameOpts.DisplayHitEffects = false

	resolveShell(m, 200)

	if len(m.state.Arena.Explosions) != 0 {
		t.Fatalf("explosions = %d, want 0 when disabled", len(m.state.Arena.Explosions))
	}
	if target.CurHP == 100 {
		t.Fatal("damage should still be applied when hit effects are off")
	}
}

// TestFallSpawnsSplashOrDust 验证寿命耗尽的炮弹按落点格区分水花与尘土，而不是一律记成水面。
// 场景：127mm 炮弹速度 0、寿命 1，落点分别在 x=5.5（水）与 x=2.5（默认地图 "..L..." 的陆地列），跑一帧 updateShotBullets。
// 预期：两次各生成 1 个 Explosion，Kind 分别为 KindSplash 与 KindDust。
// 覆盖：bulletFallObjType 的 Map.IsLand 判定与落水/落陆特效生成。
func TestFallSpawnsSplashOrDust(t *testing.T) {
	testCases := []struct {
		name     string
		x        float64
		wantKind objExplosion.Kind
	}{
		{name: "water", x: 5.5, wantKind: objExplosion.KindSplash},
		{name: "land", x: 2.5, wantKind: objExplosion.KindDust},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bt := &objBullet.Bullet{
				Name: "test-fall-shell", Type: objBullet.TypeShell, Diameter: 127,
				CurPos: objPos.NewR(testCase.x, 0.5), TargetPos: objPos.NewR(testCase.x, 0.5),
				Life: 1, Speed: 0, TargetObjType: object.TypeShip,
			}
			manager := newBulletFallManager(bt)
			manager.state.UI.GameOpts.DisplayHitEffects = true

			manager.updateShotBullets()

			if len(manager.state.Arena.Explosions) != 1 {
				t.Fatalf("explosions = %d, want 1", len(manager.state.Arena.Explosions))
			}
			if kind := manager.state.Arena.Explosions[0].Kind; kind != testCase.wantKind {
				t.Fatalf("kind = %v, want %v", kind, testCase.wantKind)
			}
		})
	}
}

// TestHitEffectRespectsPerFrameCap 验证单帧命中特效数量有上限，防空弹幕不会一帧堆出上百个特效。
// 场景：构造 maxHitEffectsPerFrame+2（12）发 127mm 炮弹，落点按 1 格间隔排开以避开 0.25 格合并，全部落在 24×2 的水面地图上，跑一帧 updateShotBullets。
// 预期：Arena.Explosions 恰好 maxHitEffectsPerFrame=10 个。
// 覆盖：特效循环里的 spawned 单帧计数上限。
func TestHitEffectRespectsPerFrameCap(t *testing.T) {
	// 落点彼此拉开 1 格以上，避免被合并规则吃掉
	bullets := make([]*objBullet.Bullet, 0, maxHitEffectsPerFrame+2)
	for i := 0; i < maxHitEffectsPerFrame+2; i++ {
		x := float64(i) + 0.5
		bullets = append(bullets, &objBullet.Bullet{
			Name: "cap-shell", Type: objBullet.TypeShell, Diameter: 127,
			CurPos: objPos.NewR(x, 0.5), TargetPos: objPos.NewR(x, 0.5),
			Life: 1, Speed: 0, TargetObjType: object.TypeShip,
		})
	}
	m := newBulletFallManager(bullets[0])
	m.state.Arena.ForwardingBullets = bullets
	m.state.UI.GameOpts.DisplayHitEffects = true
	// 落点全部在水面
	m.state.Core.MissionMD.MapCfg.Map = mapcfg.MapData{"........................", "........................"}

	m.updateShotBullets()

	if len(m.state.Arena.Explosions) != maxHitEffectsPerFrame {
		t.Fatalf("explosions = %d, want %d", len(m.state.Arena.Explosions), maxHitEffectsPerFrame)
	}
}

// TestHitEffectMergesSalvoOnSameTarget 验证同帧落在同一舰体附近的齐射只生成一个特效，避免叠成大亮斑。
// 场景：靶舰 HP 拉到 100000，8 发同参数 127mm 炮弹同帧命中同一舰体 (10,10)。
// 预期：Explosions 恰好 1 个（落点间距小于 hitEffectMergeDistance=0.25 格被合并）。
// 覆盖：hitEffectMerged 的 0.25 格距离判定与 spawnedPos 去重。
func TestHitEffectMergesSalvoOnSameTarget(t *testing.T) {
	target := newTestShip(10, 10)
	target.TotalHP, target.CurHP = 100000, 100000
	bullets := make([]*objBullet.Bullet, 0, 8)
	for i := 0; i < 8; i++ {
		bullets = append(bullets, newTestShell(0, 12, 9))
	}
	m := newShellImpactManager(target, bullets[0])
	m.state.Arena.ForwardingBullets = bullets
	m.state.UI.GameOpts.DisplayHitEffects = true

	resolveShell(m, 200)

	if len(m.state.Arena.Explosions) != 1 {
		t.Fatalf("explosions = %d, want 1 merged effect for a salvo on one hull", len(m.state.Arena.Explosions))
	}
}

// TestHitEffectRespectsLiveCap 验证同时存在的特效达到上限后不再追加。
// 场景：Arena.Explosions 预置 maxLiveEffects=200 个元素，再让一发 10 伤害炮弹命中靶舰并推进 200 拍。
// 预期：Explosions 仍为 200 个，不生成第 201 个。
// 覆盖：特效循环的 len(Arena.Explosions) >= maxLiveEffects 存活上限。
func TestHitEffectRespectsLiveCap(t *testing.T) {
	target := newTestShip(10, 10)
	m := newShellImpactManager(target, newTestShell(0, 12, 9))
	m.state.UI.GameOpts.DisplayHitEffects = true
	m.state.Arena.Explosions = make([]*objExplosion.Explosion, maxLiveEffects)

	resolveShell(m, 200)

	if len(m.state.Arena.Explosions) != maxLiveEffects {
		t.Fatalf("explosions = %d, want %d", len(m.state.Arena.Explosions), maxLiveEffects)
	}
}

// TestWaterSplashSkippedInsideHull 验证落在舰体内的「落水」不再画水花，避免白沫浮在舰上。
// 场景：127mm 炮弹速度 0、寿命 1，在 51.2×12.8 像素靶舰正中 (10,10) 寿命耗尽，落点类型被记成 TypeWater。
// 预期：先确认 bt.HitObjType 为 TypeWater，但 Explosions 为 0（insideAnyHull 跳过水花/尘土）。
// 覆盖：炮弹寿命耗尽落点判定与 insideAnyHull 舰体包含判断的回归。
func TestWaterSplashSkippedInsideHull(t *testing.T) {
	target := newTestShip(10, 10)
	// 寿命 1 + 速度 0：本帧在舰体正中"寿命耗尽"，落点类型是水
	bt := &objBullet.Bullet{
		Name: "expire-in-hull", Type: objBullet.TypeShell, Diameter: 127,
		CurPos: objPos.NewR(10, 10), TargetPos: objPos.NewR(10, 10),
		Life: 1, Speed: 0, TargetObjType: object.TypeShip,
	}
	m := newShellImpactManager(target, bt)
	m.state.UI.GameOpts.DisplayHitEffects = true

	m.updateShotBullets()

	if bt.HitObjType != object.TypeWater {
		t.Fatalf("hit type = %v, want water", bt.HitObjType)
	}
	if len(m.state.Arena.Explosions) != 0 {
		t.Fatalf("explosions = %d, want 0 (splash inside a hull)", len(m.state.Arena.Explosions))
	}
}
