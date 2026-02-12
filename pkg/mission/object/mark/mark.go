package mark

import (
	"image/color"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"

	objCommon "github.com/narasux/jutland/pkg/mission/object/common"
	"github.com/narasux/jutland/pkg/resources/font"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

type ID string

const (
	// IDTarget 目标标记
	IDTarget ID = "target"
	// IDLockOn 锁定标记
	IDLockOn ID = "lockOn"
	// IDAttack 攻击标记
	IDAttack ID = "attack"
)

// Mark 标记（如目标地点等，会存在一定时间后消失）
type Mark struct {
	ID   ID
	Pos  objCommon.MapPos
	Img  *ebiten.Image
	Life int
}

// NewImg 创建图片类型标记
func NewImg(id ID, pos objCommon.MapPos, img *ebiten.Image, life int) *Mark {
	return &Mark{ID: id, Pos: pos, Img: img, Life: life}
}

// NewText 创建文字类型标记
func NewText(pos objCommon.MapPos, text string, fontSize float64, clr color.Color, life int) *Mark {
	img := textureImg.GetText(text, font.Hang, fontSize, clr)
	return &Mark{ID: ID(uuid.New().String()), Pos: pos, Img: img, Life: life}
}
