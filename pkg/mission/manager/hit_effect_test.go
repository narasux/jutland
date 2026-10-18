package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// 命中闪光按口径分三档，尺寸随口径单调增大。
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

// 落水出水花、落陆出尘土，尺寸随口径增大；小口径防空弹两者都不生成。
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
				t.Fatalf("hitEffect(%v, %dmm) spawned = %v, want %v", testCase.hitType, testCase.diameter, ok, testCase.wantEffect)
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

// 鱼雷与炸弹走专属表现，火箭空爆不重复生成。
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

// 开关打开时命中舰体生成火球，特效与伤害数字互不影响。
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

// 关闭命中特效后不再生成，但伤害结算照旧。
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

// 落水与落陆的弹药分别生成水花与尘土。
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

// 单帧生成数量有上限，防空弹幕下不会一次性堆出上百个特效。
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

// 同一帧内落点接近的命中（一轮齐射打在同一条船上）只出一个特效，
// 否则十几个相同闪光会叠成一团大亮斑。
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

// 同时存在的特效到达上限后不再生成新的。
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

// 落点在舰体内的"落水"不出水花：舰体上泛白沫看着像水花浮在舰上。
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
