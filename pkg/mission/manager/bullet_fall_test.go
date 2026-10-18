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

// 寿命耗尽的弹药按落点格判定落水还是落陆，而不是一律记成水面；
// 且必须进入到达列表，表现层才有机会画水花 / 尘土。
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

// 吊射弹药越过瞄准点后的落点同样区分陆水（此前无条件记成水面）。
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
