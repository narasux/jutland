package object

import (
	"image/color"

	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// Trail 尾流（战舰，鱼雷，炮弹）
type Trail struct {
	Pos   MapPos
	Shape textureImg.TrailShape
	// 当前尺寸 & 尺寸扩散速度
	CurSize       float64
	DiffusionRate float64
	// 当前生命值 & 生命值衰减速度
	CurLife           float64
	LifeReductionRate float64
	// 延迟时间
	Delay float64
	// 旋转角度
	Rotation float64
	// 颜色（nil 为默认白色）
	Color color.Color
}

// 创建尾流对象
func newTrail(
	pos MapPos,
	Shape textureImg.TrailShape,
	size, diffusionRate float64,
	life, lifeReductionRate float64,
	delay, rotation float64,
	clr color.Color,
) *Trail {
	return &Trail{
		Pos:               pos,
		Shape:             Shape,
		CurSize:           size,
		DiffusionRate:     diffusionRate,
		CurLife:           life,
		LifeReductionRate: lifeReductionRate,
		Delay:             delay,
		Rotation:          rotation,
		Color:             clr,
	}
}

// Update ...
func (t *Trail) Update() {
	if t.Delay > 0 {
		t.Delay -= t.LifeReductionRate
		return
	}
	t.CurSize += t.DiffusionRate
	t.CurLife -= t.LifeReductionRate
}

// IsAlive ...
func (t *Trail) IsAlive() bool {
	return t.CurLife > 0 && t.CurSize >= 1
}

// IsActive ...
func (t *Trail) IsActive() bool {
	return t.Delay <= 0 && t.IsAlive()
}
