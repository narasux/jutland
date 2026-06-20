package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// isAnyNextInput 判定是否为下一步的输入（空格/回车/鼠标左键）
func isAnyNextInput() bool {
	if isKeySpaceJustPressed() || isKeyEnterJustPressed() || isMouseButtonLeftJustPressed() {
		return true
	}
	return false
}

// isHoverMenuButton 判定鼠标是否在菜单按钮上
func isHoverMenuButton(button *menuButton) bool {
	r := image.Rectangle{
		Min: image.Point{X: int(button.PosX), Y: int(button.PosY)},
		Max: image.Point{X: int(button.PosX + button.Width), Y: int(button.PosY + button.Height)},
	}
	return image.Pt(ebiten.CursorPosition()).In(r)
}

// 判定是否按下空格键
func isKeySpaceJustPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeySpace)
}

// 判定是否按下回车键
func isKeyEnterJustPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEnter)
}

// 判定是否按下鼠标左键
func isMouseButtonLeftJustPressed() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}
