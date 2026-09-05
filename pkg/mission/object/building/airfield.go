package building

import (
	"time"

	"github.com/google/uuid"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 机场没有实体停机坪：飞机与航母机库一致，以库存（Groups[].CurCount）形式
// 待命；起飞时直接在跑道起点刷新并沿跑道滑跑，返航着舰后直接入库。

const (
	// minRunwayLength 跑道最小长度（格）
	minRunwayLength = 2.0
	// minRunwayWidth 跑道最小宽度（格）
	minRunwayWidth = 0.4
)

// AirfieldProduction 单机型的生产状态。
type AirfieldProduction struct {
	// ElapsedMs 已累计的生产时长（毫秒），暂停期间不累计
	ElapsedMs int64
	// LastTickedAt 上次计时时刻（毫秒时间戳），用于增量累计
	LastTickedAt int64
}

// Airfield 陆地机场
type Airfield struct {
	Uid string
	// 位置（地图格坐标，跑道中心）
	Pos objPos.MapPos
	// 跑道朝向（度）
	Rotation float64
	// 跑道长度（格）
	RunwayLength float64
	// 跑道宽度（格）
	RunwayWidth float64
	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 警戒起飞半径（格），敌方进入半径内时自动起飞迎战
	AlertRadius float64
	// 机库（与航母共用结构；起飞在跑道刷新、着舰直接入库）
	Aircraft objUnit.ShipAircraft

	// 各机型生产状态（Key: 机型名称）
	Production map[string]*AirfieldProduction
	// 各机型累计损失数量（Key: 机型名称）
	Losses map[string]int64
}

var _ objUnit.AircraftBase = (*Airfield)(nil)

// NewAirfield 创建陆地机场。
// 起飞点设于跑道起点（后端）左右两舷各一，滑跑整条跑道后离地爬升；
// 降落沿跑道中线进近、跑道中点回收，最终进近段 3 个跑道长。
// 编组的目标类型与库存数量在此初始化。
func NewAirfield(
	pos objPos.MapPos,
	rotation, runwayLength, runwayWidth float64,
	belongPlayer faction.Player,
	alertRadius, takeOffTime float64,
	groups []objUnit.PlaneGroup,
) *Airfield {
	airfield := &Airfield{
		Uid:          uuid.NewString(),
		Pos:          pos,
		Rotation:     rotation,
		RunwayLength: max(runwayLength, minRunwayLength),
		RunwayWidth:  max(runwayWidth, minRunwayWidth),
		BelongPlayer: belongPlayer,
		AlertRadius:  max(alertRadius, 0),
		Production:   map[string]*AirfieldProduction{},
		Losses:       map[string]int64{},
	}
	airfield.Aircraft = objUnit.ShipAircraft{
		TakeOffTime: max(takeOffTime, 0),
		Groups:      groups,
	}
	for idx := range airfield.Aircraft.Groups {
		// 机型不存在时 GetPlaneTargetObjType 会直接报错退出（配置校验兜底）
		airfield.Aircraft.Groups[idx].TargetType = objUnit.GetPlaneTargetObjType(
			airfield.Aircraft.Groups[idx].Name,
		)
	}
	airfield.Aircraft.HasPlane = len(airfield.Aircraft.Groups) > 0

	// 程序合成甲板模板：双起飞点并排，支持双机并行起飞；
	// forward=1 即跑道起点（后端），滑跑整条跑道后离地
	deck := &objUnit.CarrierDeck{
		Name: "AirfieldRunway",
		TakeoffPoints: []objUnit.TakeoffPoint{
			{Forward: 1, Lateral: -0.15, RunLength: 1},
			{Forward: 1, Lateral: 0.15, RunLength: 1},
		},
		Landing: objUnit.LandingConfig{
			Mode:           objUnit.LandingModeDeck,
			Forward:        0.5,
			Lateral:        0,
			ApproachAngle:  0,
			ApproachLength: 3,
		},
	}
	deck.Validate()
	airfield.Aircraft.ResolveDeck(deck)
	return airfield
}

// ---- AircraftBase 接口实现（静态基地） ----
func (a *Airfield) BaseUid() string { return a.Uid }

// BasePos 基地中心位置。
func (a *Airfield) BasePos() objPos.MapPos { return a.Pos }

// BaseRotation 基地朝向（跑道朝向，恒定不变）。
func (a *Airfield) BaseRotation() float64 { return a.Rotation }

// BaseLength 基地长度（跑道长度，换算为与舰船一致的资源像素量纲）。
func (a *Airfield) BaseLength() float64 { return a.RunwayLength * constants.MapBlockSize }

// BaseWidth 基地宽度（跑道宽度）。
func (a *Airfield) BaseWidth() float64 { return a.RunwayWidth * constants.MapBlockSize }

// BaseSpeed 静态基地速度恒为 0。
func (a *Airfield) BaseSpeed() float64 { return 0 }

// BaseBelongPlayer 基地所属阵营。
func (a *Airfield) BaseBelongPlayer() faction.Player { return a.BelongPlayer }

// BaseAircraft 机库。
func (a *Airfield) BaseAircraft() *objUnit.ShipAircraft { return &a.Aircraft }

// StockPlane 一架飞机完工入库（不生成实体，与航母机库一致）。
func (a *Airfield) StockPlane(name string) {
	for idx := range a.Aircraft.Groups {
		group := &a.Aircraft.Groups[idx]
		if group.Name == name && group.CurCount < group.MaxCount {
			// 非指针需要通过索引修改
			group.CurCount++
			return
		}
	}
}

// RecordLoss 记录一架飞机损失（库存已在起飞时扣减，这里仅累计统计）。
func (a *Airfield) RecordLoss(planeName string) {
	a.Losses[planeName]++
}

// ---- 自动生产 ----

// Update 推进自动生产：资金充足时按机型独立计时，完成后返回完工机型列表
// （由调用方扣款并调用 StockPlane 入库）。
func (a *Airfield) Update(curFunds int64) []string {
	completed := []string{}
	timeNow := time.Now().UnixMilli()
	for idx := range a.Aircraft.Groups {
		group := &a.Aircraft.Groups[idx]
		// 满编：无需生产，清理状态
		if group.CurCount >= group.MaxCount {
			delete(a.Production, group.Name)
			continue
		}
		fundsCost, timeCost := objUnit.GetPlaneCost(group.Name)
		producing := a.Production[group.Name]
		// 资金不足：暂停计时，保留现有进度
		if curFunds < fundsCost {
			if producing != nil {
				producing.LastTickedAt = timeNow
			}
			continue
		}
		// 尚未开工：开始计时
		if producing == nil {
			a.Production[group.Name] = &AirfieldProduction{LastTickedAt: timeNow}
			continue
		}
		// 累计生产时长（暂停期间不计时）
		if producing.LastTickedAt > 0 {
			producing.ElapsedMs += timeNow - producing.LastTickedAt
		}
		producing.LastTickedAt = timeNow
		// 生产完成：清理状态并交由调用方扣款、入库
		if producing.ElapsedMs >= timeCost*1e3 {
			delete(a.Production, group.Name)
			completed = append(completed, group.Name)
		}
	}
	return completed
}

// ProductionProgress 返回当前生产进度百分比（0-100，取各机型最大值）。
func (a *Airfield) ProductionProgress() int {
	process := 0
	for name, producing := range a.Production {
		_, timeCost := objUnit.GetPlaneCost(name)
		if timeCost <= 0 {
			continue
		}
		process = max(process, int(float64(producing.ElapsedMs)/float64(timeCost*1e3)*100))
	}
	return min(process, 100)
}

// CanAlertLaunch 是否允许警戒起飞（配置了警戒半径且机库有编组）。
func (a *Airfield) CanAlertLaunch() bool {
	return a.AlertRadius > 0 && a.Aircraft.HasPlane
}
