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
	GB381  = ebutil.NewImageWithColor(4, 7, colorx.Gold)
	GB356  = ebutil.NewImageWithColor(4, 6, colorx.Gold)
	GB343  = ebutil.NewImageWithColor(3, 6, colorx.White)
	GB305  = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	GB283  = ebutil.NewImageWithColor(3, 5, colorx.Gray)
	GB203  = ebutil.NewImageWithColor(3, 4, colorx.White)
	GB180  = ebutil.NewImageWithColor(2, 4, colorx.Gray)
	GB155  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB152  = ebutil.NewImageWithColor(2, 4, colorx.White)
	GB150  = ebutil.NewImageWithColor(2, 4, colorx.Gray)
	GB140  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB133  = ebutil.NewImageWithColor(2, 3, colorx.DarkBlue)
	GB130  = ebutil.NewImageWithColor(2, 3, colorx.White)
	GB127  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB120  = ebutil.NewImageWithColor(2, 3, colorx.DarkBlue)
	GB114  = ebutil.NewImageWithColor(2, 2, colorx.Silver)
	GB105  = ebutil.NewImageWithColor(2, 2, colorx.Gray)
	GB102  = ebutil.NewImageWithColor(2, 2, colorx.Silver)
	GB100  = ebutil.NewImageWithColor(2, 2, colorx.White)
	GB88   = ebutil.NewImageWithColor(2, 2, colorx.Gold)
	GB40   = ebutil.NewImageWithColor(1, 1, colorx.Gold)
	GB20   = ebutil.NewImageWithColor(1, 1, colorx.White)
	GB13   = ebutil.NewImageWithColor(1, 1, colorx.Gray)
)

var (
	TB450 = ebutil.NewImageWithColor(3, 16, colorx.Silver)
	TB533 = ebutil.NewImageWithColor(3, 20, colorx.DarkSilver)
	TB610 = ebutil.NewImageWithColor(4, 24, colorx.Silver)
	TB622 = ebutil.NewImageWithColor(4, 25, colorx.Gray)
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
	case 381:
		return GB381
	case 356:
		return GB356
	case 343:
		return GB343
	case 305:
		return GB305
	case 283:
		return GB283
	case 203:
		return GB203
	case 180:
		return GB180
	case 155:
		return GB155
	case 152:
		return GB152
	case 150:
		return GB150
	case 140:
		return GB140
	case 133:
		return GB133
	case 130:
		return GB130
	case 127:
		return GB127
	case 120:
		return GB120
	case 114:
		return GB114
	case 105:
		return GB105
	case 102:
		return GB102
	case 100:
		return GB100
	case 88:
		return GB88
	case 40:
		return GB40
	case 20:
		return GB20
	case 13:
		return GB13
	}
	// 找不到就暴露出来
	return NotFount
}

// GetTorpedo 获取鱼雷弹药图片
func GetTorpedo(diameter int) *ebiten.Image {
	switch diameter {
	case 540:
		return TB450
	case 533:
		return TB533
	case 610:
		return TB610
	case 622:
		return TB622
	}
	// 找不到就暴露出来
	return NotFount
}
