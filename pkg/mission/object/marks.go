package object

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/images/texture"
)

type MarkType string

const (
	// MarkTypeTargetPos 目标地点
	MarkTypeTargetPos MarkType = "TargetPos"
)

// Mark 标记（如目标地点等，会存在一定时间后消失）
type Mark struct {
	Pos  MapPos
	Img  *ebiten.Image
	Life int
}

// NewMark ...
func NewMark(t MarkType, pos MapPos) *Mark {
	if t == MarkTypeTargetPos {
		return &Mark{Pos: pos, Img: texture.TargetPosImg, Life: 20}
	}
	return nil
}
