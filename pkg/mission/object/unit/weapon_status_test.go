package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

// TestShipWeaponReloadStatusUsesNextReadyMount 验证主炮装填状态取“最快就绪的那门”，并正确报告禁用。
// 场景：3 门主炮装填进度不同，单位面板要显示最近可开火的一门及倒计时，而不是第一门/平均值。
// 预期：Equipped=3、Ready=1、RemainingMillis=1000、Progress=0.5、Disabled=true。
// 覆盖：构造 3 门不同 ReloadStartAt 的主炮，调用 ReloadStatus(now=10000) 断言字段。
func TestShipWeaponReloadStatusUsesNextReadyMount(t *testing.T) {
	now := int64(10_000)
	weapon := ShipWeapon{
		MainGunDisabled: true,
		MainGuns: []*Gun{
			{ReloadStartAt: 0, ReloadTime: 1},
			{ReloadStartAt: 9_000, ReloadTime: 2},
			{ReloadStartAt: 8_000, ReloadTime: 4},
		},
	}

	status := weapon.ReloadStatus(WeaponTypeMainGun, now)
	if status.Equipped != 3 || status.Ready != 1 {
		t.Fatalf("equipment status = %+v, want 1/3 ready", status)
	}
	if status.RemainingMillis != 1_000 {
		t.Fatalf("remaining = %d, want 1000", status.RemainingMillis)
	}
	if math.Abs(status.Progress-0.5) > 1e-9 {
		t.Fatalf("progress = %v, want 0.5", status.Progress)
	}
	if !status.Disabled {
		t.Fatal("disabled weapon category reported as enabled")
	}
}

// TestTorpedoReloadStatusAccountsForShotIntervalAndReload 验证鱼雷装填状态同时考虑发射间隔与整管装填。
// 场景：一座发射器刚打完 4 根进入间隔，另一座在整管装填；面板应显示最近可再发射的时间。
// 预期：Ready=0、RemainingMillis=500、Progress=0.5。
// 覆盖：构造两座不同 LatestFireAt/ReloadTime 的发射器，调用 ReloadStatus(now=10000)。
func TestTorpedoReloadStatusAccountsForShotIntervalAndReload(t *testing.T) {
	now := int64(10_000)
	weapon := ShipWeapon{Torpedoes: []*TorpedoLauncher{
		{
			ReloadStartAt: 0,
			ReloadTime:    2,
			LatestFireAt:  9_500,
			ShotInterval:  1,
			BulletCount:   4,
		},
		{
			ReloadStartAt: 9_000,
			ReloadTime:    5,
			LatestFireAt:  0,
			ShotInterval:  1,
			BulletCount:   4,
		},
	}}

	status := weapon.ReloadStatus(WeaponTypeTorpedo, now)
	if status.Ready != 0 || status.RemainingMillis != 500 {
		t.Fatalf("status = %+v, want next launcher in 500ms", status)
	}
	if math.Abs(status.Progress-0.5) > 1e-9 {
		t.Fatalf("progress = %v, want 0.5", status.Progress)
	}
}

// TestRocketReloadStatusUsesGroupInterval 验证火箭炮按“组间隔”而不是单发间隔计算下一次可发射时间。
// 场景：6 发 3 组、组间隔 4 秒、已打 2 发；面板应显示下一组剩余 3 秒，而不是 0.5 秒单发间隔。
// 预期：RemainingMillis=3000、Progress=0.25。
// 覆盖：构造带 ShotInterval/GroupInterval 的 RocketLauncher，调用 ReloadStatus(now=10000)。
func TestRocketReloadStatusUsesGroupInterval(t *testing.T) {
	weapon := ShipWeapon{Rockets: []*RocketLauncher{{
		RocketCount:           6,
		GroupCount:            3,
		ShotInterval:          0.5,
		GroupInterval:         4,
		ReloadTime:            8,
		ReloadStartAt:         0,
		LatestFireAt:          9_000,
		ShotCountBeforeReload: 2,
	}}}
	status := weapon.ReloadStatus(WeaponTypeRocket, 10_000)
	if status.RemainingMillis != 3_000 {
		t.Fatalf("remaining = %d, want 3000", status.RemainingMillis)
	}
	if math.Abs(status.Progress-0.25) > 1e-9 {
		t.Fatalf("progress = %v, want 0.25", status.Progress)
	}
}

// TestReloadStatusAllReady 验证全部武器就绪时状态为满进度、零倒计时。
// 场景：副炮刚装填完/从未开火，面板应显示 1/1、100%、无倒计时，而不是残余进度。
// 预期：Ready=1、Progress=1、RemainingMillis=0。
// 覆盖：构造单门 ReloadStartAt=0/ReloadTime=1 的副炮，调用 ReloadStatus(now=10000)。
func TestReloadStatusAllReady(t *testing.T) {
	weapon := ShipWeapon{SecondaryGuns: []*Gun{{ReloadStartAt: 0, ReloadTime: 1}}}
	status := weapon.ReloadStatus(WeaponTypeSecondaryGun, 10_000)
	if status.Ready != 1 || status.Progress != 1 || status.RemainingMillis != 0 {
		t.Fatalf("status = %+v, want fully ready", status)
	}
}

// TestAnyReloadedSkipsDisabledAndReloadingGuns 验证 AnyReloaded 只为“未禁用且已装填”的炮返回 true。
// 场景：AI/玩家按下开火键时先问 AnyReloaded；若把装填中或整组禁用的炮算作可开火，会空放/绕过禁用。
// 预期：装填中 false；MainGunDisabled=true false；就绪且未禁用 true。
// 覆盖：固定 SimTick=10，分别构造三种 ShipWeapon 调用 AnyReloaded。
func TestAnyReloadedSkipsDisabledAndReloadingGuns(t *testing.T) {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	SetSimTick(10)
	reloading := &Gun{ReloadTime: 100, ReloadStartTick: 10, AntiShip: true}
	if (&ShipWeapon{MainGuns: []*Gun{reloading}}).AnyReloaded() {
		t.Fatal("reloading gun reported ready")
	}
	ready := &Gun{ReloadTime: 0, ReloadStartAt: 0, AntiShip: true}
	if (&ShipWeapon{MainGunDisabled: true, MainGuns: []*Gun{ready}}).AnyReloaded() {
		t.Fatal("disabled group reported ready")
	}
	if !(&ShipWeapon{MainGuns: []*Gun{ready}}).AnyReloaded() {
		t.Fatal("ready gun was skipped")
	}
}

// TestTorpedoReloadStatusKeepsUnfiredTubesReady 验证从未发射过的鱼雷管按“已就绪”处理，
// 不按 0 号时间戳算出一段虚假装填倒计时。
// 场景：3 根鱼雷管，前两根从未发射（ReloadStartTick/LatestFireTick 均为 0），
// 第三根在 tick 600 发射并进入 ReloadTime=25s 的整管装填；面板传入的是拍（tick 时钟），
// 若把 0 号时间戳当装填起点，会得到 1 拍只走 1 毫秒的假进度。
// 预期：now=660 拍时 Equipped=3、Ready=2、第三根剩余 1440 拍=24,000ms、Progress=0.04；
// now=1200 拍时剩余 15,000ms；now=2100 拍时 3 根全部就绪、Progress=1。
// 覆盖：构造三根鱼雷管，分别以 660/1200/2100 调用 ReloadStatus(WeaponTypeTorpedo)，断言 Ready/剩余时间/进度。
func TestTorpedoReloadStatusKeepsUnfiredTubesReady(t *testing.T) {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	weapon := ShipWeapon{Torpedoes: []*TorpedoLauncher{
		{ReloadTime: 25, ShotInterval: 1, BulletCount: 1},
		{ReloadTime: 25, ShotInterval: 1, BulletCount: 1},
		{ReloadTime: 25, ShotInterval: 1, BulletCount: 1, ReloadStartTick: 600, LatestFireTick: 600},
	}}

	status := weapon.ReloadStatus(WeaponTypeTorpedo, 660)
	if status.Ready != 2 || status.Equipped != 3 {
		t.Fatalf("status = %+v, want 2/3 ready", status)
	}
	if status.RemainingMillis != 24_000 {
		t.Fatalf("remaining = %dms, want 24000", status.RemainingMillis)
	}
	if math.Abs(status.Progress-0.04) > 1e-9 {
		t.Fatalf("progress = %v, want 0.04", status.Progress)
	}

	status = weapon.ReloadStatus(WeaponTypeTorpedo, 1_200)
	if status.RemainingMillis != 15_000 {
		t.Fatalf("remaining = %dms, want 15000", status.RemainingMillis)
	}

	status = weapon.ReloadStatus(WeaponTypeTorpedo, 2_100)
	if status.Ready != 3 || status.RemainingMillis != 0 || status.Progress != 1 {
		t.Fatalf("status = %+v, want all tubes ready", status)
	}
}

// TestRocketReloadStatusKeepsUnfiredLaunchersReady 验证未发射的火箭发射器按“已就绪”处理，
// 同样不应显示装填倒计时。
// 场景：1 座火箭发射器（6 发分 3 组、单发间隔 0.5s、组间隔 4s、整座装填 8s），
// 但 ReloadStartTick/LatestFireTick/ShotCountBeforeReload 全为 0，表示一发都没打过。
// 预期：ReloadStatus(WeaponTypeRocket, 120) 返回 Ready=1、RemainingMillis=0、Progress=1，
// 而不是凭空显示 8s 的整座装填进度。
// 覆盖：构造未发射的 RocketLauncher，调用一次 ReloadStatus 并断言三个字段。
func TestRocketReloadStatusKeepsUnfiredLaunchersReady(t *testing.T) {
	weapon := ShipWeapon{Rockets: []*RocketLauncher{{
		RocketCount:   6,
		GroupCount:    3,
		ShotInterval:  0.5,
		GroupInterval: 4,
		ReloadTime:    8,
	}}}

	status := weapon.ReloadStatus(WeaponTypeRocket, 120)
	if status.Ready != 1 || status.RemainingMillis != 0 || status.Progress != 1 {
		t.Fatalf("status = %+v, want ready launcher", status)
	}
}
