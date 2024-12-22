package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

var (
	gb1024 = ebutil.NewImageWithColor(6, 14, colorx.DarkRed)
	gb510  = ebutil.NewImageWithColor(5, 10, colorx.Gold)
	gb500  = ebutil.NewImageWithColor(5, 10, colorx.Yellow)
	gb460  = ebutil.NewImageWithColor(4, 9, colorx.Gold)
	gb457  = ebutil.NewImageWithColor(4, 9, colorx.Silver)
	gb406  = ebutil.NewImageWithColor(4, 8, colorx.Silver)
	gb381  = ebutil.NewImageWithColor(4, 7, colorx.Gold)
	gb356  = ebutil.NewImageWithColor(4, 6, colorx.Gold)
	gb343  = ebutil.NewImageWithColor(3, 6, colorx.White)
	gb305  = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	gb283  = ebutil.NewImageWithColor(3, 5, colorx.Gray)
	gb203  = ebutil.NewImageWithColor(3, 4, colorx.White)
	gb200  = ebutil.NewImageWithColor(3, 4, colorx.Gold)
	gb180  = ebutil.NewImageWithColor(2, 4, colorx.Gray)
	gb155  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	gb152  = ebutil.NewImageWithColor(2, 4, colorx.White)
	gb150  = ebutil.NewImageWithColor(2, 4, colorx.Gray)
	gb140  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	gb133  = ebutil.NewImageWithColor(2, 3, colorx.DarkBlue)
	gb130  = ebutil.NewImageWithColor(2, 3, colorx.White)
	gb127  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	gb120  = ebutil.NewImageWithColor(2, 3, colorx.DarkBlue)
	gb114  = ebutil.NewImageWithColor(2, 2, colorx.Silver)
	gb105  = ebutil.NewImageWithColor(2, 2, colorx.Gray)
	gb102  = ebutil.NewImageWithColor(2, 2, colorx.Silver)
	gb100  = ebutil.NewImageWithColor(2, 2, colorx.White)
	gb88   = ebutil.NewImageWithColor(2, 2, colorx.Gold)
	gb76   = ebutil.NewImageWithColor(2, 2, colorx.White)
	gb57   = ebutil.NewImageWithColor(1, 2, colorx.Yellow)
	gb40   = ebutil.NewImageWithColor(1, 1, colorx.Gold)
	gb37   = ebutil.NewImageWithColor(1, 1, colorx.DarkBlue)
	gb25   = ebutil.NewImageWithColor(1, 1, colorx.Gold)
	gb20   = ebutil.NewImageWithColor(1, 1, colorx.White)
	gb13   = ebutil.NewImageWithColor(1, 1, colorx.Gray)
	// gb12 其实就是 12.7mm，懒得改成 float，就这样吧 :D
	gb12 = ebutil.NewImageWithColor(1, 1, colorx.White)
)

var (
	tb450 = ebutil.NewImageWithColor(3, 16, colorx.Silver)
	tb533 = ebutil.NewImageWithColor(3, 20, colorx.DarkSilver)
	tb610 = ebutil.NewImageWithColor(4, 24, colorx.Silver)
	tb622 = ebutil.NewImageWithColor(4, 25, colorx.Gray)
)

var NotFount = ebutil.NewImageWithColor(10, 20, colorx.Red)

// GetShell 获取炮弹弹药图片
func GetShell(diameter int) *ebiten.Image {
	switch diameter {
	case 1024:
		return gb1024
	case 510:
		return gb510
	case 500:
		return gb500
	case 460:
		return gb460
	case 457:
		return gb457
	case 406:
		return gb406
	case 381:
		return gb381
	case 356:
		return gb356
	case 343:
		return gb343
	case 305:
		return gb305
	case 283:
		return gb283
	case 203:
		return gb203
	case 200:
		return gb200
	case 180:
		return gb180
	case 155:
		return gb155
	case 152:
		return gb152
	case 150:
		return gb150
	case 140:
		return gb140
	case 133:
		return gb133
	case 130:
		return gb130
	case 127:
		return gb127
	case 120:
		return gb120
	case 114:
		return gb114
	case 105:
		return gb105
	case 102:
		return gb102
	case 100:
		return gb100
	case 88:
		return gb88
	case 76:
		return gb76
	case 57:
		return gb57
	case 40:
		return gb40
	case 37:
		return gb37
	case 25:
		return gb25
	case 20:
		return gb20
	case 13:
		return gb13
	case 12:
		return gb12
	}
	// 找不到就暴露出来
	return NotFount
}

// GetTorpedo 获取鱼雷弹药图片
func GetTorpedo(diameter int) *ebiten.Image {
	switch diameter {
	case 450:
		return tb450
	case 533:
		return tb533
	case 610:
		return tb610
	case 622:
		return tb622
	}
	// 找不到就暴露出来
	return NotFount
}
