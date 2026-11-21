package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestRocketLauncherSalvoFiresWholeLoadAtOnce 验证 salvoSize 配置能把整轮弹药同帧齐投。
// 场景：刺猬弹这类反潜武器一次齐投 24 枚，若仍按逐发间隔投放，会失去“覆盖一个圆”的设计意图。
// 预期：salvoSize=5、rocketCount=5 时一次 Fire 返回 5 枚同类型深弹，装填计数归零并进入 reload；
// 紧接着第二次 Fire 返回 0；salvoSize=0 时恢复每次 1 枚。
// 覆盖：注册测试深弹模板，构造全向射界的发射器与正前方 1.5 格的潜艇目标，分别断言 5 发、0 发、1 发。
func TestRocketLauncherSalvoFiresWholeLoadAtOnce(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-salvo-depth-charge"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{
		Type: objBullet.TypeDepthCharge, Damage: 450,
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
	})

	newLauncher := func(salvo int) *RocketLauncher {
		return &RocketLauncher{
			Name: "test-hedgehog", BulletName: bulletName,
			RocketCount: 5, SalvoSize: salvo,
			GroupCount: 1, ReloadTime: 30,
			Range: 2, BulletSpread: 80, BulletSpeed: 0.1,
			ProximityRadius: 0.35, BlastRadius: 0.35,
			AntiSubmarine:  true,
			LeftFiringArc:  FiringArc{Start: 180, End: 360},
			RightFiringArc: FiringArc{Start: 0, End: 180},
		}
	}
	shooter := &BattleShip{
		Uid: "shooter", Type: ShipTypeDestroyer, Name: "test-dd",
		CurHP: 100, CurPos: objPos.NewR(10, 10), CurRotation: 90,
		BelongPlayer: faction.HumanAlpha,
		Length:       100, Width: 10,
	}
	target := &BattleShip{
		Uid: "target", Type: ShipTypeSubmarine, Name: "test-ss",
		CurHP: 100, CurPos: objPos.NewR(11.5, 10),
		BelongPlayer: faction.ComputerAlpha,
	}

	SetSimTick(100)
	salvo := newLauncher(5)
	bullets := salvo.Fire(shooter, target)
	require.Len(t, bullets, 5, "齐投应一次返回整轮 5 枚")
	for _, bt := range bullets {
		require.Equal(t, objBullet.TypeDepthCharge, bt.Type)
	}
	require.Zero(t, salvo.ShotCountBeforeReload, "整轮投完后应重置装填计数")
	require.Positive(t, salvo.ReloadStartTick, "整轮投完后应进入 reload")
	require.Empty(t, salvo.Fire(shooter, target), "reload 期间不应继续投弹")

	single := newLauncher(0)
	require.Len(t, single.Fire(shooter, target), 1, "未配置 salvoSize 时保持每次 1 枚")
}
