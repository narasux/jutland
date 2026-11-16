package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// newBulletFallManager 造一个带小地图的管理器：x=2 是海岸，x=5 是海面。
func newBulletFallManager(bt *objBullet.Bullet) *MissionManager {
	return &MissionManager{
		state: &state.MissionState{
			Core: state.MissionCoreState{
				MissionMD: metadata.MissionMetadata{
					MapCfg: &mapcfg.MapCfg{Map: mapcfg.MapData{"..L...", "......"}},
				},
			},
			Arena: state.MissionArenaState{
				ForwardingBullets: []*objBullet.Bullet{bt},
			},
		},
	}
}

// TestBulletFallOnLandOrWater 验证寿命耗尽的弹药按落点格判定落陆/落水并移入到达列表。
// 场景：小地图 "..L..."，127mm 炮弹 Life=1、Speed=0，分别位于海岸 x=2.5 与海面 x=5.5（y=0.5）。
// 预期：HitObjType 分别为 TypeLand/TypeWater；ForwardingBullets 清空，表现层才有机会画尘土/水花。
// 覆盖：updateShotBullets 的 Life<=0 分支与 bulletFallObjType 的 IsLand 判定，两个子用例逐项断言。
func TestBulletFallOnLandOrWater(t *testing.T) {
	testCases := []struct {
		name string
		x    float64
		want object.Type
	}{
		{name: "coast", x: 2.5, want: object.TypeLand},
		{name: "sea", x: 5.5, want: object.TypeWater},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Life=1 会在本帧 Forward 后归零，Speed=0 保证落点不移动
			bt := &objBullet.Bullet{
				Name: "test-fall-shell", Type: objBullet.TypeShell, Diameter: 127,
				CurPos: objPos.NewR(testCase.x, 0.5), TargetPos: objPos.NewR(testCase.x, 0.5),
				Life: 1, Speed: 0, TargetObjType: object.TypeShip,
			}
			manager := newBulletFallManager(bt)

			manager.updateShotBullets()

			if bt.HitObjType != testCase.want {
				t.Fatalf("HitObjType = %v, want %v", bt.HitObjType, testCase.want)
			}
			if len(manager.state.Arena.ForwardingBullets) != 0 {
				t.Fatalf(
					"forwarding bullets = %d, want 0 (arrived bullet must leave the in-flight list)",
					len(manager.state.Arena.ForwardingBullets),
				)
			}
		})
	}
}

// TestBulletPassedAimFallOnLandOrWater 验证吊射越过瞄准点后的落点同样区分陆水。
// 场景：203mm 炮弹 Plunge=1（危险界 0.2/0.7≈0.286 格）、Rotation=90、瞄准点 x=0.5，弹体在海岸 x=2.5 与海面 x=5.5。
// 预期：PassedAim 成立后 HitObjType 分别为 TypeLand/TypeWater 并被移出在飞列表；此前该分支无条件记 TypeWater。
// 覆盖：updateShotBullets 的 PassedAim 分支与 bulletFallObjType，两个落点各断言一次。
func TestBulletPassedAimFallOnLandOrWater(t *testing.T) {
	testCases := []struct {
		name string
		x    float64
		want object.Type
	}{
		{name: "coast", x: 2.5, want: object.TypeLand},
		{name: "sea", x: 5.5, want: object.TypeWater},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Plunge=1 时危险界约 0.29 格；Rotation=90 让「越过瞄准点」只看 RX 差
			bt := &objBullet.Bullet{
				Name: "test-fall-shell", Type: objBullet.TypeShell, Diameter: 203,
				CurPos: objPos.NewR(testCase.x, 0.5), TargetPos: objPos.NewR(0.5, 0.5),
				Life: 5, Speed: 0, Rotation: 90, Plunge: 1, TargetObjType: object.TypeShip,
			}
			manager := newBulletFallManager(bt)

			manager.updateShotBullets()

			if bt.HitObjType != testCase.want {
				t.Fatalf("HitObjType = %v, want %v", bt.HitObjType, testCase.want)
			}
			if len(manager.state.Arena.ForwardingBullets) != 0 {
				t.Fatalf("forwarding bullets = %d, want 0", len(manager.state.Arena.ForwardingBullets))
			}
		})
	}
}
