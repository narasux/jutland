package mark

import (
	"image/color"
	"strconv"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
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
	ID       ID
	Pos      objPos.MapPos
	Img      *ebiten.Image
	Text     string
	FontSize float64
	Color    color.Color
	Life     int
}

// NewImg 创建图片类型标记
func NewImg(id ID, pos objPos.MapPos, img *ebiten.Image, life int) *Mark {
	return &Mark{ID: id, Pos: pos, Img: img, Life: life}
}

// DamageFlagText 伤害飘字一律取整，不把小数串送进纹理缓存。
func DamageFlagText(realDamage float64) string {
	return strconv.Itoa(int(realDamage))
}

// NewText 创建直接绘制的文字标记，不把字符串烤进纹理缓存。
func NewText(pos objPos.MapPos, text string, fontSize float64, clr color.Color, life int) *Mark {
	return &Mark{
		ID:       ID(uuid.New().String()),
		Pos:      pos,
		Text:     text,
		FontSize: fontSize,
		Color:    clr,
		Life:     life,
	}
}
