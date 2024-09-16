package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

var (
	GB1024 = ebutil.NewImageWithColor(6, 14, colorx.DarkRed)
	GB460  = ebutil.NewImageWithColor(4, 9, colorx.Gold)
	GB406  = ebutil.NewImageWithColor(4, 8, colorx.Silver)
	GB356  = ebutil.NewImageWithColor(4, 6, colorx.Gold)
	GB305  = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	GB283  = ebutil.NewImageWithColor(3, 5, colorx.Gray)
	GB203  = ebutil.NewImageWithColor(3, 4, colorx.Gold)
	GB155  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB152  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB150  = ebutil.NewImageWithColor(2, 4, colorx.Gray)
	GB140  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB127  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB105  = ebutil.NewImageWithColor(2, 2, colorx.Gray)
)

var (
	TB533 = ebutil.NewImageWithColor(3, 20, colorx.DarkSilver)
	TB610 = ebutil.NewImageWithColor(4, 24, colorx.Silver)
)

var NotFount = ebutil.NewImageWithColor(10, 20, colorx.Red)

// GetShell 获取炮弹弹药图片
func GetShell(diameter int) *ebiten.Image {
	switch diameter {
	case 1024:
		return GB1024
	case 460:
		return GB460
	case 406:
		return GB406
	case 356:
		return GB356
	case 305:
		return GB305
	case 283:
		return GB283
	case 203:
		return GB203
	case 155:
		return GB155
	case 152:
		return GB152
	case 150:
		return GB150
	case 140:
		return GB140
	case 127:
		return GB127
	case 105:
		return GB105
	}
	// 找不到就暴露出来
	return NotFount
}

// GetTorpedo 获取鱼雷弹药图片
func GetTorpedo(diameter int) *ebiten.Image {
	switch diameter {
	case 533:
		return TB533
	case 610:
		return TB610
	}
	// 找不到就暴露出来
	return NotFount
}
