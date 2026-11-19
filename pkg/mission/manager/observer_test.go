package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/observer"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestObserverReceivesShotHitSinkAndLoss 验证 observer 能从开火、命中、击沉、损失四条路径拿到归因字段。
// 场景：TEST/GB/127 炮弹由 Attacker 对 100HP 的 Target 发射并造成 1000 伤害；随后一艘航母被击沉，
// 一架 F4F 被空对空 30 基础伤害击落；observer 在每条路径上都必须收到带 UID/武器/目标的事件。
// 预期：Shot 有 weapon/target/shell；Hit 的 applied=1000、effective=100、overkill=900、Killed=true；
// Sink.PlanesLost 保留 RecordShipLost 清空前的 2；Loss.ShotDown=true 且目标 UID 为 p。
// 覆盖：Recorder 接入 SetObserver，调用 observeShots/hurtShip/observeSink/hurtPlane/observeLoss，
// 断言四类事件的数量与关键字段；不启动 Ebiten 窗口。
func TestObserverReceivesShotHitSinkAndLoss(t *testing.T) {
	m := &MissionManager{state: &state.MissionState{}}
	m.state.Stats = state.NewBattleStats()
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)

	target := &objUnit.BattleShip{
		Uid: "t", Name: "Target", Type: objUnit.ShipTypeBattleShip,
		CurHP: 100, BelongPlayer: faction.ComputerAlpha, Tonnage: 50000,
	}
	bt := &objBullet.Bullet{
		Name: "TEST/GB/127/1940", Type: objBullet.TypeShell, Diameter: 127, Damage: 1000,
		Shooter: "a", ShooterName: "Attacker", ShooterType: string(objUnit.ShipTypeDestroyer),
		ShooterObjType: object.TypeShip, BelongPlayer: faction.HumanAlpha,
		TargetObjType: object.TypeShip, TargetUID: "t", WeaponName: "5in/38",
		FireDistance: 30, FireRangeRatio: 0.5,
	}
	m.observeShots([]*objBullet.Bullet{bt})
	m.hurtShip(target, bt)

	require.Len(t, recorder.Shots, 1)
	require.Equal(t, "Attacker", recorder.Shots[0].ShooterName)
	require.Equal(t, "5in/38", recorder.Shots[0].WeaponName)
	require.Equal(t, "t", recorder.Shots[0].TargetUID)
	require.Equal(t, "shell", recorder.Shots[0].AmmoType)
	require.False(t, recorder.Shots[0].ShooterIsPlane)

	require.Len(t, recorder.Hits, 1)
	hit := recorder.Hits[0]
	require.Equal(t, "t", hit.TargetUID)
	require.Equal(t, "direct", hit.Kind)
	require.Equal(t, 1000.0, hit.AppliedDamage)
	require.Equal(t, 100.0, hit.EffectiveDamage)
	require.Equal(t, 900.0, hit.Overkill)
	require.Equal(t, 100.0, hit.HPBefore)
	require.Equal(t, 0.0, hit.HPAfter)
	require.True(t, hit.Killed)
	require.False(t, hit.FriendlyFire)

	m.observeSink(target, 2)
	require.Len(t, recorder.Sinks, 1)
	require.Equal(t, "t", recorder.Sinks[0].TargetUID)
	require.Equal(t, 2, recorder.Sinks[0].PlanesLost)
	require.Equal(t, 50000.0, recorder.Sinks[0].Tonnage)

	plane := &objUnit.Plane{
		Uid: "p", Name: "F4F", Type: objUnit.PlaneTypeFighter,
		CurHP: 30, TotalHP: 30, BelongPlayer: faction.ComputerAlpha, BelongShip: "cv",
	}
	airBullet := &objBullet.Bullet{
		Name: "TEST/GB/20/1940", Type: objBullet.TypeShell, Diameter: 20, Damage: 30,
		Shooter: "a", ShooterName: "Attacker", ShooterType: string(objUnit.PlaneTypeFighter),
		ShooterObjType: object.TypePlane, BelongPlayer: faction.HumanAlpha,
		TargetObjType: object.TypePlane, TargetUID: "p", WeaponName: "20mm",
	}
	m.hurtPlane(plane, airBullet)
	m.observeLoss(plane)

	require.Len(t, recorder.Hits, 2)
	require.True(t, recorder.Hits[1].TargetIsPlane)
	require.True(t, recorder.Hits[1].Killed)
	require.Len(t, recorder.Losses, 1)
	require.Equal(t, "p", recorder.Losses[0].TargetUID)
	require.True(t, recorder.Losses[0].ShotDown)
}

// TestObserverNilKeepsGamePathNoop 验证未接 observer 时四条路径都不 panic 且不改变目标状态。
// 场景：普通游戏路径不调用 SetObserver；炮弹仍要正常结算命中，只是不产生事件。
// 预期：hurtShip 仍把目标打到 0、设置 SunkByHit；observeShots/observeSink/observeLoss 可安全调用。
// 覆盖：不设置 observer，构造一个 10HP 目标与 20 伤害炮弹，调用四条观察路径并断言目标状态。
func TestObserverNilKeepsGamePathNoop(t *testing.T) {
	m := &MissionManager{state: &state.MissionState{}}
	m.state.Stats = state.NewBattleStats()
	ship := &objUnit.BattleShip{
		Uid: "t", Name: "Target", Type: objUnit.ShipTypeDestroyer,
		CurHP: 10, BelongPlayer: faction.ComputerAlpha,
	}
	bt := &objBullet.Bullet{
		Name: "TEST/GB/127/1940", Type: objBullet.TypeShell, Diameter: 127, Damage: 20,
		Shooter: "a", ShooterName: "Attacker", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, TargetObjType: object.TypeShip,
	}
	m.observeShots([]*objBullet.Bullet{bt})
	m.hurtShip(ship, bt)
	m.observeSink(ship, 0)
	m.observeLoss(&objUnit.Plane{Uid: "p", Name: "F4F"})

	require.Zero(t, ship.CurHP)
	require.True(t, ship.SunkByHit)
}

// TestObserverHitKindAndFriendlyFire 验证 AoE 命中类型与友军伤害标记会传到 observer。
// 场景：深弹/落点爆炸会对每个目标单独结算，报告必须能区分 direct 与 depth_charge，且不能把友伤算成敌方战果。
// 预期：同一阵营目标收到 Kind="depth_charge"、FriendlyFire=true、AppliedDamage=10；
// 若这两个字段丢失，伤害归因和友伤审计都会失真。
// 覆盖：直接调用 hurtShip 并设置 bt.HitKind="depth_charge"，双方 BelongPlayer 都为 HumanAlpha，断言 Hit 字段。
func TestObserverHitKindAndFriendlyFire(t *testing.T) {
	m := &MissionManager{state: &state.MissionState{}}
	m.state.Stats = state.NewBattleStats()
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)

	ship := &objUnit.BattleShip{
		Uid: "friendly", Name: "Friendly", Type: objUnit.ShipTypeDestroyer,
		CurHP: 100, BelongPlayer: faction.HumanAlpha,
	}
	bt := &objBullet.Bullet{
		Name: "TEST/DC/300/1940", Type: objBullet.TypeDepthCharge, Damage: 10,
		Shooter: "a", ShooterName: "Attacker", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, TargetObjType: object.TypeShip,
		WeaponName: "depth-charge", HitKind: "depth_charge",
	}
	m.hurtShip(ship, bt)

	require.Len(t, recorder.Hits, 1)
	hit := recorder.Hits[0]
	require.Equal(t, "depth_charge", hit.Kind)
	require.True(t, hit.FriendlyFire)
	require.Equal(t, 10.0, hit.AppliedDamage)
	require.False(t, hit.Killed)
}

// TestObserverReceivesSinkAndLossFromUpdateLoops 验证 updateMissionShips / updateMissionPlanes 真正调用了观察者。
// 场景：只测 observeSink/observeLoss 会漏掉“忘了在消亡循环里调用”的回归；这里让 0HP 单位走真实移除路径。
// 预期：舰船移除后 Recorder.Sinks 有 1 条且 TargetUID="s"；飞机移除后 Losses 有 1 条且 TargetUID="p"、
// ShotDown=false；失败说明 observer 只接了直接伤害路径，没有覆盖战损统计。
// 覆盖：构造最小 Arena（Ships/Planes 各 1 个 0HP 单位，位置在相机外避免音频），调用两个 update 循环。
func TestObserverReceivesSinkAndLossFromUpdateLoops(t *testing.T) {
	m := &MissionManager{state: &state.MissionState{}}
	m.state.Stats = state.NewBattleStats()
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)

	ship := &objUnit.BattleShip{
		Uid: "s", Name: "Sinking", Type: objUnit.ShipTypeDestroyer,
		CurHP: 0, BelongPlayer: faction.ComputerAlpha, Tonnage: 2000,
		CurPos: objPos.NewR(1000, 1000),
	}
	plane := &objUnit.Plane{
		Uid: "p", Name: "LostPlane", Type: objUnit.PlaneTypeFighter,
		CurHP: 0, BelongPlayer: faction.ComputerAlpha,
	}
	m.state.Arena.Ships = map[string]*objUnit.BattleShip{"s": ship}
	m.state.Arena.Planes = map[string]*objUnit.Plane{"p": plane}

	m.updateMissionShips()
	m.updateMissionPlanes()

	require.Len(t, recorder.Sinks, 1)
	require.Equal(t, "s", recorder.Sinks[0].TargetUID)
	require.Equal(t, 2000.0, recorder.Sinks[0].Tonnage)
	require.Len(t, recorder.Losses, 1)
	require.Equal(t, "p", recorder.Losses[0].TargetUID)
	require.False(t, recorder.Losses[0].ShotDown)
}

// TestUpdateMissionShipsProcessesInUIDOrder 验证消亡循环按 UID 顺序记账，而不是 map 随机序。
// 场景：同一拍两艘 0HP 舰 b/a 同时移除；若遍历 map，observer 的 Sinks 顺序会在同 seed 下漂移。
// 预期：Sinks[0].TargetUID="a"、Sinks[1].TargetUID="b"；失败说明有序切片没有用于消亡循环。
// 覆盖：Arena.Ships 以 b/a（及 a/b 两种插入）建 map，调用 updateMissionShips，断言 Sinks 顺序。
func TestUpdateMissionShipsProcessesInUIDOrder(t *testing.T) {
	for _, order := range [][]string{{"b", "a"}, {"a", "b"}} {
		m := &MissionManager{state: &state.MissionState{}}
		m.state.Stats = state.NewBattleStats()
		recorder := &observer.Recorder{}
		m.SetObserver(recorder)
		m.state.Arena.Ships = map[string]*objUnit.BattleShip{}
		for _, uid := range order {
			m.state.Arena.Ships[uid] = &objUnit.BattleShip{
				Uid: uid, Name: uid, Type: objUnit.ShipTypeDestroyer,
				CurHP: 0, BelongPlayer: faction.ComputerAlpha,
				CurPos: objPos.NewR(1000, 1000),
			}
		}
		m.updateMissionShips()
		require.Len(t, recorder.Sinks, 2)
		require.Equal(t, "a", recorder.Sinks[0].TargetUID)
		require.Equal(t, "b", recorder.Sinks[1].TargetUID)
	}
}

// BenchmarkObserveShotsNil 测量未接 observer 时逐发检查的额外开销，要求不产生分配。
// 场景：游戏默认路径每帧都会走 updateShipWeaponFire/updatePlaneWeaponFire，不能因空 observer 变慢。
// 预期：循环内只做 nil 分支；ReportAllocs 应显示 0 allocs/op。
// 覆盖：10 发炮弹切片重复调用 observeShots，分别测试 nil observer。
func BenchmarkObserveShotsNil(b *testing.B) {
	m := &MissionManager{state: &state.MissionState{}}
	bullets := make([]*objBullet.Bullet, 10)
	for i := range bullets {
		bullets[i] = &objBullet.Bullet{Name: "TEST/GB/127/1940", Type: objBullet.TypeShell}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.observeShots(bullets)
	}
}

// BenchmarkObserveShotsRecorder 测量接 observer 后逐发写事件的成本，作为 Macro 采样预算参考。
// 场景：headless 模拟启用 Recorder，事件写入切片会产生分配；需要知道每发量级。
// 预期：基准输出 ns/op 与 allocs/op，供后续 p99/采样预算对比，不设置硬阈值。
// 覆盖：10 发炮弹重复调用 observeShots，observer 为内存 Recorder。
func BenchmarkObserveShotsRecorder(b *testing.B) {
	m := &MissionManager{state: &state.MissionState{}}
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)
	bullets := make([]*objBullet.Bullet, 10)
	for i := range bullets {
		bullets[i] = &objBullet.Bullet{
			Name: "TEST/GB/127/1940", Type: objBullet.TypeShell,
			Shooter: "a", ShooterName: "Attacker", ShooterObjType: object.TypeShip,
			TargetUID: "t", TargetObjType: object.TypeShip, WeaponName: "5in/38",
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.Reset()
		m.observeShots(bullets)
	}
}
