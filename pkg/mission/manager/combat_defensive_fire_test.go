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

// 轰炸机自卫防空火力：对舰机型飞行途中遭遇射程内的空中敌机时，
// 机炮/炮塔应开火还击（此前对空开火目标只来自 AttackObjType，
// 轰炸机在空中纯挨打）。
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

// 敌机在对空火力射程之外时不应开火。
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

// 己方飞机不应成为自卫火力目标。
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

// 每帧至多集火一个自卫目标：前后各有一架敌机在射程内（机颚/颊部炮塔
// 覆盖正前方，机背/机腹/机尾炮塔覆盖正后方），本帧的自卫火力必须全部
// 来自其中一架，另一架等下一帧再交战。
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
