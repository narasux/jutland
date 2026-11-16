package metadata

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestASWTestInitialCameraShowsPlayerShips 验证 ASWTest 的 P1 初始镜头确实覆盖两艘玩家初始舰。
// 场景：引擎把 initCameraPos 当作视野左上角而非中心（见 state.Camera.Pos/CameraPosToScreen）；1280×720、默认 1x 缩放时可见约 10×5.625 格，P1 镜头为 (24,32)，两舰在 (30,34)/(30,37)。
// 预期：每艘 P1 舰的 RX/RY 都落在 [cam.RX, cam.RX+10]×[cam.RY, cam.RY+5.625] 内，且 P1 舰数恰为 2；失败会再次出现“开局看不到自己的船”。
// 覆盖：Get("ASWTest") 后调用 CameraPosForPlayerSide(SideP1)，遍历 InitShips 过滤 SideP1 并逐舰断言四边界与计数。
func TestASWTestInitialCameraShowsPlayerShips(t *testing.T) {
	const (
		// 默认缩放 1x 时，1280x720 屏幕大约显示 10x5.6 格。
		viewWidth  = 1280.0 / 128.0
		viewHeight = 720.0 / 128.0
	)

	md := Get("ASWTest")
	cam := md.CameraPosForPlayerSide(faction.SideP1)

	playerShips := 0
	for _, init := range md.InitShips {
		if init.BelongSide != faction.SideP1 {
			continue
		}
		playerShips++
		require.GreaterOrEqual(t, init.Pos.RX, cam.RX, "ship %s is left of the initial camera", init.ShipName)
		require.LessOrEqual(t, init.Pos.RX, cam.RX+viewWidth, "ship %s is right of the initial camera", init.ShipName)
		require.GreaterOrEqual(t, init.Pos.RY, cam.RY, "ship %s is above the initial camera", init.ShipName)
		require.LessOrEqual(t, init.Pos.RY, cam.RY+viewHeight, "ship %s is below the initial camera", init.ShipName)
	}
	require.Equal(t, 2, playerShips, "ASWTest 应有两艘 P1 初始舰")
}

// TestASWMissionsKeepOpeningSeparation 验证反潜关卡双方开场不贴脸，留出展开、下潜和观察时间。
// 场景：对 ASWTest 与 SubmarineTest 收集 P1/P2 全部初始舰位，计算最近敌我舰的 MapPos 欧氏距离；两关实际最近约 60~71 格（约 7.7~9.1km）。
// 预期：最近距离 ≥50 格（约 6.4km）；低于阈值说明任务布阵把双方放进开局即交火的距离。
// 覆盖：表驱动两个任务名，t.Run 子测试内双重循环取 math.Min，再用 require.GreaterOrEqual 断言。
func TestASWMissionsKeepOpeningSeparation(t *testing.T) {
	const minSeparation = 50.0
	for _, mission := range []string{"ASWTest", "SubmarineTest"} {
		t.Run(mission, func(t *testing.T) {
			md := Get(mission)
			var player, enemy []objPos.MapPos
			for _, init := range md.InitShips {
				switch init.BelongSide {
				case faction.SideP1:
					player = append(player, init.Pos)
				case faction.SideP2:
					enemy = append(enemy, init.Pos)
				}
			}
			require.NotEmpty(t, player)
			require.NotEmpty(t, enemy)

			nearest := math.Inf(1)
			for _, mine := range player {
				for _, foe := range enemy {
					nearest = math.Min(nearest, mine.Distance(foe))
				}
			}
			require.GreaterOrEqual(
				t, nearest, minSeparation,
				"%s 最近敌我舰仅相距 %.1f 格", mission, nearest,
			)
		})
	}
}
