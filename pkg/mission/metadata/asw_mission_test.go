package metadata

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 引擎把 initCameraPos 作为初始视野左上角（见 state.Camera.Pos 与 CameraPosToScreen），
// 不是视野中心。这里按 1280x720、默认 1x 缩放的可见范围验证 ASWTest 的 P1 镜头
// 确实覆盖两艘玩家初始舰，防止再次出现「开局看不到自己的船」。
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

// 反潜关卡的双方开场不应贴脸：最近敌我舰至少相隔 50 格（约 6.4km），
// 给玩家留出展开、下潜和观察各舰状态的时间。
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
