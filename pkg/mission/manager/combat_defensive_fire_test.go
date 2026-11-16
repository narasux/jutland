package manager

import (
	"testing"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestShipAttackPlaneReturnsFireAtAirborneEnemy 验证对舰机型飞行途中会用机炮/炮塔还击射程内的空中敌机。
// 场景：B-17G 在 (10,10)、CurAttackTarget="enemy-ship"，咬尾的 F6F-3 在正后方 (10,11) 且速度归零；对空机枪射程 1.25 格，FriendlyFire=false，跑一帧 updatePlaneWeaponFire。
// 预期：ForwardingBullets 中至少 1 发 Shooter=bomber.Uid 且 TargetObjType=TypePlane；B-17G 的炸弹与鱼雷均未 Released。
// 覆盖：updatePlaneDefensiveFire 的空中敌机扫描与还击，以及炸弹/鱼雷释放器不会对空投放（修复轰炸机纯挨打）。
func TestShipAttackPlaneReturnsFireAtAirborneEnemy(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	// 真实模板：B-17G（对舰水平轰炸机，尾部炮塔射界覆盖正后方）+ F6F-3 战斗机
	bomber := objUnit.NewPlane("B-17G", objPos.NewR(10, 10), 0, "", faction.HumanAlpha)
	bomber.CurAttackTarget = "enemy-ship"
	// 咬尾的敌机：位于轰炸机正后方 1 格，处于对空机枪射程（1.25 格）内；
	// 速度归零保证提前量预测点即当前位置，命中判定确定
	enemy := objUnit.NewPlane("F6F-3", objPos.NewR(10, 11), 0, "", faction.ComputerAlpha)
	enemy.CurSpeed = 0

	missionState := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{bomber.Uid: bomber, enemy.Uid: enemy},
		},
	}
	missionState.UI.GameOpts.FriendlyFire = false
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
	}

	manager.updatePlaneWeaponFire()

	defensive := 0
	for _, bt := range missionState.Arena.ForwardingBullets {
		if bt.Shooter == bomber.Uid && bt.TargetObjType == object.TypePlane {
			defensive++
		}
	}
	if defensive == 0 {
		t.Fatal("bomber should return fire at the tailing enemy fighter with its turrets")
	}
	// 自卫火力不能误投炸弹/鱼雷（释放器只能对舰与地面目标投放）
	for _, b := range bomber.Weapon.Bombs {
		if b.Released {
			t.Fatal("defensive fire must not release bombs at an airborne target")
		}
	}
	for _, tp := range bomber.Weapon.Torpedoes {
		if tp.Released {
			t.Fatal("defensive fire must not release torpedoes at an airborne target")
		}
	}
}

// TestShipAttackPlaneKeepsSilentOutOfRange 验证敌机在自卫对空射程外时不开火。
// 场景：B-17G 在 (10,10)，F6F-3 在正后方 4 格 (10,14)，速度归零，超过对空机枪 1.25 格射程。
// 预期：ForwardingBullets 为 0。
// 覆盖：updatePlaneDefensiveFire 的 MaxToPlaneRange 距离过滤。
func TestShipAttackPlaneKeepsSilentOutOfRange(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	bomber := objUnit.NewPlane("B-17G", objPos.NewR(10, 10), 0, "", faction.HumanAlpha)
	enemy := objUnit.NewPlane("F6F-3", objPos.NewR(10, 14), 0, "", faction.ComputerAlpha)
	enemy.CurSpeed = 0

	missionState := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{bomber.Uid: bomber, enemy.Uid: enemy},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
	}

	manager.updatePlaneWeaponFire()

	if len(missionState.Arena.ForwardingBullets) != 0 {
		t.Fatalf("forwarding bullets = %d, want 0 (enemy out of AA range)", len(missionState.Arena.ForwardingBullets))
	}
}

// TestShipAttackPlaneIgnoresFriendlyPlanes 验证己方飞机不会成为自卫火力目标。
// 场景：B-17G 在 (10,10)，同阵营 F6F-3 在 (10,11)、距离 1 格且在射程内，速度归零。
// 预期：ForwardingBullets 为 0（不做友军误伤）。
// 覆盖：updatePlaneDefensiveFire 的 BelongPlayer 敌我判定。
func TestShipAttackPlaneIgnoresFriendlyPlanes(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	bomber := objUnit.NewPlane("B-17G", objPos.NewR(10, 10), 0, "", faction.HumanAlpha)
	friendly := objUnit.NewPlane("F6F-3", objPos.NewR(10, 11), 0, "", faction.HumanAlpha)
	friendly.CurSpeed = 0

	missionState := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{bomber.Uid: bomber, friendly.Uid: friendly},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
	}

	manager.updatePlaneWeaponFire()

	if len(missionState.Arena.ForwardingBullets) != 0 {
		t.Fatalf("forwarding bullets = %d, want 0 (no friendly fire)", len(missionState.Arena.ForwardingBullets))
	}
}

// TestShipAttackPlaneEngagesSingleTargetPerFrame 验证每帧至多集火一个自卫目标。
// 场景：B-17G 在 (10,10)，正前方 (10,9) 与正后方 (10,11) 各有一架速度归零的 F6F-3，均在 1.25 格射程内；跑一帧 updatePlaneWeaponFire。
// 预期：至少一批自卫弹，且以第一发归属为准，所有自卫弹目标点距同一架敌机 ≤0.35 格、距另一架 >0.35 格（两机相距 2 格、散布 ≤0.12 格）。
// 覆盖：updatePlaneDefensiveFire 中 eachPlane 命中即返回，保证一帧只交战一个目标；过滤 Shooter=bomber.Uid 排除敌机还击。
func TestShipAttackPlaneEngagesSingleTargetPerFrame(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	bomber := objUnit.NewPlane("B-17G", objPos.NewR(10, 10), 0, "", faction.HumanAlpha)
	enemyAhead := objUnit.NewPlane("F6F-3", objPos.NewR(10, 9), 0, "", faction.ComputerAlpha)
	enemyAhead.CurSpeed = 0
	enemyBehind := objUnit.NewPlane("F6F-3", objPos.NewR(10, 11), 0, "", faction.ComputerAlpha)
	enemyBehind.CurSpeed = 0

	missionState := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{
				bomber.Uid: bomber, enemyAhead.Uid: enemyAhead, enemyBehind.Uid: enemyBehind,
			},
		},
	}
	missionState.UI.GameOpts.FriendlyFire = false
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
	}

	manager.updatePlaneWeaponFire()

	// 敌机也会开火还击轰炸机，按 Shooter 过滤出轰炸机的自卫火力
	targets := []objPos.MapPos{}
	for _, bt := range missionState.Arena.ForwardingBullets {
		if bt.Shooter == bomber.Uid && bt.TargetObjType == object.TypePlane {
			targets = append(targets, bt.TargetPos)
		}
	}
	if len(targets) == 0 {
		t.Fatal("bomber should engage one of the two in-range enemies")
	}

	// 两架敌机相距 2 格，机枪散布半径 <=0.12 格：以第一发弹的归属为准，
	// 所有自卫弹的目标点必须集中同一架敌机附近
	aheadPos, behindPos := objPos.NewR(10, 9), objPos.NewR(10, 11)
	engagingAhead := targets[0].Distance(aheadPos) <= targets[0].Distance(behindPos)
	for _, tp := range targets {
		own, other := behindPos, aheadPos
		if engagingAhead {
			own, other = aheadPos, behindPos
		}
		if tp.Distance(own) > 0.35 || tp.Distance(other) <= 0.35 {
			t.Fatalf("defensive fire spread across multiple targets: bullet target %s", tp.String())
		}
	}
}
