package layout

import "github.com/hajimehoshi/ebiten/v2"

/*
区域划分示意图：

+-----------------------------------------------------------+----------+-----
|                                                           |          |
|                                                           |   small  |  1/5
|                                                           |    map   |
|                                                           |          |
|                                                           +----------+-----
|                                                           |          |
|                          camera                           |   menu   |
|                                                           |          |
|                                                           |          | 4/5
|                                                           |          |
|                                                           |          |
|                                                           |          |
|                                                           |          |
|                                                           |          |
+-----------------------------------------------------------+----------+-----
|                            6/7                            |    1/7   |

*/

// ScreenPos 屏幕坐标
type ScreenPos struct {
	SX int
	SY int
}

// Screen 关卡屏幕
type ScreenLayout struct {
	// 屏幕宽高（一般是屏幕分辨率）
	Width  int
	Height int

	// 组件元数据
	Camera  ScreenCameraLayout
	Console ScreenConsoleLayout
}

// NewScreenLayout ...
func NewScreenLayout() ScreenLayout {
	width, height := ebiten.Monitor().Size()
	// 游戏地图左 6/7, 右 1/7 是小地图 & 菜单
	sepX := width / 7 * 6
	// 控制台上 1/5 是小地图，下 4/5 是菜单
	sepY := height / 5

	return ScreenLayout{
		Width:  width,
		Height: height,
		Camera: ScreenCameraLayout{
			Width:   sepX,
			Height:  height,
			TopLeft: ScreenPos{0, 0},
		},
		Console: ScreenConsoleLayout{
			Width:   width - sepX,
			Height:  height,
			TopLeft: ScreenPos{sepX, 0},
			SmallMap: ScreenSmallMapLayout{
				// 四周留白各 8 像素
				Width:   width - sepX - 16,
				Height:  sepY - 16,
				TopLeft: ScreenPos{sepX + 8, 8},
			},
			Menu: ScreenMenuLayout{
				// 左右 + 底部留白各 8 像素，顶部不需要，因为小地图已经留白
				Width:   width - sepX - 16,
				Height:  height - sepY - 8,
				TopLeft: ScreenPos{sepX + 8, sepY},
			},
		},
	}
}

// ScreenCameraLayout 游戏地图
type ScreenCameraLayout struct {
	Width   int
	Height  int
	TopLeft ScreenPos
}

// ScreenConsoleLayout 控制台
type ScreenConsoleLayout struct {
	Width   int
	Height  int
	TopLeft ScreenPos

	SmallMap ScreenSmallMapLayout
	Menu     ScreenMenuLayout
}

// ScreenSmallMapLayout 小地图
type ScreenSmallMapLayout struct {
	Width   int
	Height  int
	TopLeft ScreenPos
}

// ScreenMenuLayout 菜单
type ScreenMenuLayout struct {
	Width   int
	Height  int
	TopLeft ScreenPos
}
