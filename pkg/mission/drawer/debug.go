package drawer

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// 绘制调试用坐标
func (d *Drawer) drawDebugIndex(screen *ebiten.Image, ms *state.MissionState) {
	for x := 0; x < ms.Camera.Width; x++ {
		for y := 0; y < ms.Camera.Height; y++ {
			mapX, mapY := ms.Camera.Pos.MX+x, ms.Camera.Pos.MY+y
			ebutil.DebugPrintAt(
				screen, fmt.Sprintf("\n(%d,%d)", mapX, mapY), x*constants.MapBlockSize, y*constants.MapBlockSize,
			)
		}
	}
}
