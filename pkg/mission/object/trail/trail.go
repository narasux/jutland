package trail

import (
	"image/color"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// Trail 尾流（战舰，鱼雷，炮弹）
type Trail struct {
	Pos   objPos.MapPos
	Shape textureImg.TrailShape
	// OwnerUid 标识持续生成该尾流的对象；舰船尾流停船后需要统一淡出。
	OwnerUid string `json:"-"`
	// BelongPlayer 是产生这条尾流的玩家，迷雾下据此决定要不要画。
	BelongPlayer faction.Player `json:"-"`
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

	stopFade bool
}

var trailPool []*Trail

// New 创建尾流对象。优先复用已经消亡的实例。
func New(
	pos objPos.MapPos,
	Shape textureImg.TrailShape,
	size, diffusionRate float64,
	life, lifeReductionRate float64,
	delay, rotation float64,
	clr color.Color,
) *Trail {
	var t *Trail
	if n := len(trailPool); n > 0 {
		t = trailPool[n-1]
		trailPool = trailPool[:n-1]
	} else {
		t = &Trail{}
	}
	*t = Trail{
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
	return t
}

// Release 把消亡尾迹放回池里。调用后不能再使用这只指针。
func Release(t *Trail) {
	if t == nil {
		return
	}
	*t = Trail{}
	trailPool = append(trailPool, t)
}

// Update ...
func (t *Trail) Update() {
	if t.Delay > 0 {
		t.Delay -= t.LifeReductionRate * config.G.SpeedMultiplier
		return
	}
	t.CurSize += t.DiffusionRate * config.G.SpeedMultiplier
	t.CurLife -= t.LifeReductionRate * config.G.SpeedMultiplier
}

// BeginStopFade 让舰船停下后残留的整组尾流在指定帧数内同步淡出。
func (t *Trail) BeginStopFade(frames float64) {
	if t.stopFade || t.CurLife <= 0 {
		return
	}
	t.stopFade = true
	if frames <= 0 {
		t.CurLife = 0
		return
	}
	t.LifeReductionRate = max(t.LifeReductionRate, t.CurLife/frames)
}

// IsAlive ...
func (t *Trail) IsAlive() bool {
	return t.CurLife > 0 && t.CurSize >= 1
}

// IsActive ...
func (t *Trail) IsActive() bool {
	return t.Delay <= 0 && t.IsAlive()
}
