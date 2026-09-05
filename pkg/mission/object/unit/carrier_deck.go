package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
)

// LandingMode 着舰方式
type LandingMode string

const (
	// LandingModeDeck 甲板回收
	LandingModeDeck LandingMode = "deck"
	// LandingModeSea 舷侧着水回收（水上飞机降落到舰侧水面）
	LandingModeSea LandingMode = "sea"
)

const (
	// defaultApproachLength 是未配置时的最终直线进近段长度（舰长倍数）。
	defaultApproachLength = 3.0
	// minApproachLength 限制最终进近段过短导致着舰动画不可读。
	minApproachLength = 1.0
	// defaultTakeoffRunLength 是未配置时的滑跑距离（舰长倍数）。
	defaultTakeoffRunLength = 0.5
)

// CarrierDeck 航母起降甲板模板（configs/carrier_decks.json5）。
// 纵向位置使用距舰艏的舰长比例（0=舰艏，1=舰尾）；
// 横向位置使用舰宽比例（0=中线，正=右舷，|0.5| 为舷边，>0.5 为舰外水面）。
type CarrierDeck struct {
	// Name 模板名称
	Name string `json:"name"`
	// TakeoffPoints 起飞点列表（多弹射器并行且独享，按配置顺序选用）
	TakeoffPoints []TakeoffPoint `json:"takeoffPoints"`
	// Landing 降落配置
	Landing LandingConfig `json:"landing"`
}

// TakeoffPoint 起飞点（弹射器或滑跑起点）
type TakeoffPoint struct {
	// Forward 滑跑起点距舰艏的舰长比例
	Forward float64 `json:"forward"`
	// Lateral 滑跑起点横向舰宽比例
	Lateral float64 `json:"lateral"`
	// RunLength 滑跑距离（舰长倍数），飞过该距离后升空
	RunLength float64 `json:"runLength"`
	// LaunchAngle 弹射航向相对舰体航向的偏转角（度），正=右舷
	LaunchAngle float64 `json:"launchAngle"`
	// PlaneTypes 服务机型白名单（fighter/dive_bomber/level_bomber/torpedo_bomber），空 = 任意机型
	PlaneTypes []PlaneType `json:"planeTypes"`
	// TakeOffTime 该点独立冷却（秒），<=0 时使用舰船的 takeOffTime
	TakeOffTime float64 `json:"takeOffTime"`
}

// LandingConfig 降落配置
type LandingConfig struct {
	// Mode 着舰方式：deck 甲板 / sea 舷侧水面
	Mode LandingMode `json:"mode"`
	// Forward 着舰回收点距舰艏的舰长比例
	Forward float64 `json:"forward"`
	// Lateral 着舰回收点横向舰宽比例（斜角甲板时偏离中线）
	Lateral float64 `json:"lateral"`
	// ApproachAngle 进近方向相对舰体航向的夹角（度），正=右舷偏转
	ApproachAngle float64 `json:"approachAngle"`
	// ApproachLength 最终直线进近段长度（舰长倍数）
	ApproachLength float64 `json:"approachLength"`
}

// DeckMap 保存按名称索引的甲板模板。
var DeckMap = map[string]*CarrierDeck{}

// Validate 校验并修正模板字段；结构性缺失由加载方 log.Fatal，这里只兜底数值。
func (d *CarrierDeck) Validate() {
	d.validate()
}

// validate 校验并修正模板字段。
func (d *CarrierDeck) validate() {
	for idx := range d.TakeoffPoints {
		point := &d.TakeoffPoints[idx]
		point.Forward = max(0, min(1, point.Forward))
		point.Lateral = max(-0.5, min(0.5, point.Lateral))
		if point.RunLength <= 0 {
			point.RunLength = defaultTakeoffRunLength
		}
		point.TakeOffTime = max(0, point.TakeOffTime)
	}

	if d.Landing.Mode != LandingModeSea {
		d.Landing.Mode = LandingModeDeck
	}
	d.Landing.Forward = max(0, min(1, d.Landing.Forward))
	if d.Landing.ApproachLength <= 0 {
		d.Landing.ApproachLength = defaultApproachLength
	}
	d.Landing.ApproachLength = max(minApproachLength, d.Landing.ApproachLength)
}

// carrierWidthInMapBlocks 将基地宽度资源像素长度换算为地图坐标长度。
func carrierWidthInMapBlocks(base AircraftBase) float64 {
	return max(base.BaseWidth()/constants.MapBlockSize, 0.1)
}

// landingConfig 返回解析后的降落配置；未配置模板时回退到甲板中线降落。
func (sa *ShipAircraft) landingConfig() LandingConfig {
	if sa.deck != nil {
		return sa.deck.Landing
	}
	return LandingConfig{
		Mode:           LandingModeDeck,
		Forward:        0.5,
		Lateral:        0,
		ApproachAngle:  0,
		ApproachLength: defaultApproachLength,
	}
}

// landingOnWater 返回舰船是否使用舷侧着水回收。
func (sa *ShipAircraft) landingOnWater() bool {
	return sa.landingConfig().Mode == LandingModeSea
}

// landingConfigForSlot 返回指定回收槽位的降落配置。
// 着水回收按进近通道左右交替选择舷侧水面；甲板回收仍使用模板原值。
func (sa *ShipAircraft) landingConfigForSlot(slot int) LandingConfig {
	cfg := sa.landingConfig()
	if cfg.Mode != LandingModeSea {
		return cfg
	}
	offset := math.Abs(cfg.Lateral)
	if offset < 0.5 {
		offset = 1.1
	}
	if landingLaneOffsetRatio(slot) < 0 {
		cfg.Lateral = -offset
	} else {
		cfg.Lateral = offset
	}
	return cfg
}

// takeoffLandingPos 将着舰回收点配置换算为基地局部地图坐标。
func takeoffLandingPos(base AircraftBase, landing LandingConfig) carrierLocalOffset {
	return carrierLocalOffset{
		forward: carrierLengthInMapBlocks(base) * (0.5 - landing.Forward),
		lateral: carrierWidthInMapBlocks(base) * landing.Lateral,
	}
}

// landingApproachTangent 返回进近方向的航母局部单位向量。
func landingApproachTangent(landing LandingConfig) carrierLocalOffset {
	radians := landing.ApproachAngle * math.Pi / 180
	return carrierLocalOffset{forward: math.Cos(radians), lateral: math.Sin(radians)}
}
