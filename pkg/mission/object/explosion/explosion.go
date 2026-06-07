package explosion

import (
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// Explosion 表示短生命周期的局部爆炸效果。
type Explosion struct {
	Pos      objPos.MapPos
	Rotation float64
	Age      int
	Life     int
}

// NewRocket 创建火箭弹空爆效果，复用较轻的飞机爆炸帧序列
func NewRocket(pos objPos.MapPos, rotation float64) *Explosion {
	return &Explosion{
		Pos:      pos,
		Rotation: rotation,
		Age:      0,
		Life:     textureImg.MaxPlaneExplodeState,
	}
}

// Update 推进爆炸动画一帧
func (e *Explosion) Update() {
	e.Age++
}

// IsAlive 返回爆炸动画是否仍需绘制
func (e *Explosion) IsAlive() bool {
	return e.Age < e.Life
}

// FrameHP 返回可传给现有爆炸贴图接口的反向帧索引
func (e *Explosion) FrameHP() float64 {
	frame := textureImg.MaxPlaneExplodeState - 1 - e.Age
	if frame < 0 {
		frame = 0
	}
	return float64(frame)
}
