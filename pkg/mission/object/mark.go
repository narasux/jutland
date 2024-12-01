package object

import (
	"image/color"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/font"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

type MarkID string

const (
	// MarkIDTarget 目标标记
	MarkIDTarget MarkID = "target"
	// MarkIDLockOn 锁定标记
	MarkIDLockOn MarkID = "lockOn"
	// MarkIDAttack 攻击标记
	MarkIDAttack MarkID = "attack"
)

// Mark 标记（如目标地点等，会存在一定时间后消失）
type Mark struct {
	ID   MarkID
	Pos  MapPos
	Img  *ebiten.Image
	Life int
}

// NewImgMark ...
func NewImgMark(id MarkID, pos MapPos, img *ebiten.Image, life int) *Mark {
	return &Mark{ID: id, Pos: pos, Img: img, Life: life}
}

// NewTextMark ...
func NewTextMark(pos MapPos, text string, fontSize float64, clr color.Color, life int) *Mark {
	img := textureImg.GetText(text, font.Hang, fontSize, clr)
	return &Mark{ID: MarkID(uuid.New().String()), Pos: pos, Img: img, Life: life}
}
