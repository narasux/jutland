package object

import (
	"image/color"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
)

type MarkType int

const (
	// MarkTypeImg 图片标记
	MarkTypeImg MarkType = iota
	// MarkTypeText 文本标记
	MarkTypeText
)

type MarkID string

const (
	// MarkIDTarget 目标标记
	MarkIDTarget MarkID = "target"
)

// Mark 标记（如目标地点等，会存在一定时间后消失）
type Mark struct {
	ID       MarkID
	Pos      MapPos
	Type     MarkType
	Img      *ebiten.Image
	Text     string
	FontSize float64
	Color    color.Color
	Life     int
}

// NewImgMark ...
func NewImgMark(pos MapPos, img *ebiten.Image, life int) *Mark {
	return &Mark{ID: MarkIDTarget, Pos: pos, Type: MarkTypeImg, Img: img, Life: life}
}

// NewTextMark ...
func NewTextMark(pos MapPos, text string, fontSize float64, clr color.Color, life int) *Mark {
	return &Mark{
		ID:  MarkID(uuid.New().String()),
		Pos: pos, Type: MarkTypeText,
		Text: text, FontSize: fontSize,
		Color: clr, Life: life,
	}
}
