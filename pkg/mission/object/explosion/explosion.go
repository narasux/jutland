package explosion

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// Kind 命中特效种类，决定取哪套帧序列、播放节奏与绘制层级。
type Kind int

const (
	// KindFlash 命中闪光（effects/flash）：炮弹命中舰体 / 飞机，按口径放大
	KindFlash Kind = iota
	// KindSplash 落水水花（effects/splash）
	KindSplash
	// KindDust 落陆尘土（effects/dust）
	KindDust
	// KindTorpedoPlume 鱼雷命中舰体的水下爆炸水柱（复用 splash 放大）
	KindTorpedoPlume
	// KindRocketAirburst 火箭 / 近炸引信空爆（复用飞机爆炸帧，倒序播放）
	KindRocketAirburst
)

// OverShips 返回该种类特效是否应绘制在舰船之上。
// 只有命中闪光需要在舰体之上：命中点就在舰体表面，画在舰体下会被完全遮住。
// 水花、尘土与鱼雷水柱都在海面 / 地面，压在舰船之下才自然
// （它们都比舰身宽，露在舷外的部分照样看得见）。
func (k Kind) OverShips() bool {
	return k == KindFlash
}

// RocketAirburstSize 火箭空爆的目标直径（地图格）：复用被压到 1/4（64px）的飞机爆炸帧。
const RocketAirburstSize = 0.8

// Explosion 表示短生命周期的局部爆炸效果。
type Explosion struct {
	Pos      objPos.MapPos
	Rotation float64
	// Kind 特效种类
	Kind Kind
	// Age 已经推进的 tick 数
	Age int
	// Life 总 tick 数，Age >= Life 后不再绘制
	Life int
	// Size 特效目标直径（地图格）：粒子按它铺开，绘制时再乘 MapBlockSize 与相机 ZoomScale
	Size float64
	// Seed 决定粒子的角度 / 距离 / 透明度分布。
	// 用确定性散列而不是全局随机数：命中特效不能影响战斗随机序列。
	Seed         uint32
	BelongPlayer faction.Player
}

// NewHitEffect 创建命中特效。
func NewHitEffect(
	kind Kind,
	pos objPos.MapPos,
	rotation, size float64,
	life int,
	seed uint32,
	owner faction.Player,
) *Explosion {
	if life < 1 {
		life = 1
	}
	return &Explosion{
		Pos:          pos,
		Rotation:     rotation,
		Kind:         kind,
		Age:          0,
		Life:         life,
		Size:         size,
		Seed:         seed,
		BelongPlayer: owner,
	}
}

// NewRocket 创建火箭弹空爆效果，复用较轻的飞机爆炸帧序列
func NewRocket(pos objPos.MapPos, rotation float64, owner faction.Player) *Explosion {
	return NewHitEffect(
		KindRocketAirburst, pos, rotation,
		RocketAirburstSize, textureImg.MaxPlaneExplodeState, 0, owner,
	)
}

// Update 推进爆炸动画一帧
func (e *Explosion) Update() {
	e.Age++
}

// IsAlive 返回爆炸动画是否仍需绘制
func (e *Explosion) IsAlive() bool {
	return e.Age < e.Life
}

// Frame 返回火箭空爆使用的序列帧号（0 起，正向）。
func (e *Explosion) Frame() int {
	if e.Age <= 0 {
		return 0
	}
	return e.Age
}
