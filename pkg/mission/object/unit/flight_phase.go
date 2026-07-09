package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// PlaneFlightPhase 表示飞机的起降 / 巡航阶段。
type PlaneFlightPhase string

const (
	// PlaneFlightPhaseTakingOff 起飞直线爬升阶段。
	PlaneFlightPhaseTakingOff PlaneFlightPhase = "taking_off"
	// PlaneFlightPhaseCruising 巡航 / 交战阶段。
	PlaneFlightPhaseCruising PlaneFlightPhase = "cruising"
	// PlaneFlightPhaseLandingApproach 舰尾进近阶段。
	PlaneFlightPhaseLandingApproach PlaneFlightPhase = "landing_approach"
	// PlaneFlightPhaseLandingDeck 甲板滑跑回收阶段。
	PlaneFlightPhaseLandingDeck PlaneFlightPhase = "landing_deck"
)

const (
	carrierTakeoffStartOffsetRatio = 0
	carrierTakeoffLaneRatio        = 2.0
	carrierLandingApproachRatio    = 1.5
	carrierLandingSternOffsetRatio = -0.45

	planeLowAltitudeVisualScale = 0.5
	landingDeckStartVisualScale = 0.6

	takeoffInitialSpeedRatio   = 0.25
	landingApproachSpeedRatio  = 0.75
	landingDeckSpeedRatio      = 0.35
	phaseSpeedStepFallbackRate = 0.04
	phaseSpeedStepMaxRate      = 0.08
)

func carrierLengthInMapBlocks(ship *BattleShip) float64 {
	return max(ship.Length/constants.MapBlockSize, 0.1)
}

func gameSpeedMultiplier() float64 {
	if config.G == nil {
		return 1
	}
	return config.G.SpeedMultiplier
}

func carrierRelativePos(ship *BattleShip, offset float64) objPos.MapPos {
	radians := ship.CurRotation * math.Pi / 180
	return objPos.NewR(
		ship.CurPos.RX+math.Sin(radians)*offset,
		ship.CurPos.RY-math.Cos(radians)*offset,
	)
}

func carrierTakeoffStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierTakeoffStartOffsetRatio)
}

func carrierTakeoffEndPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(
		ship,
		carrierLengthInMapBlocks(ship)*(carrierTakeoffStartOffsetRatio+carrierTakeoffLaneRatio),
	)
}

func carrierLandingApproachPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(
		ship,
		-carrierLengthInMapBlocks(ship)*(0.5+carrierLandingApproachRatio),
	)
}

func carrierLandingDeckStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierLandingSternOffsetRatio)
}

func carrierLandingDeckEndPos(ship *BattleShip) objPos.MapPos {
	return ship.CurPos.Copy()
}

// StartTakeoff 初始化航母起飞阶段。
func (p *Plane) StartTakeoff(ship *BattleShip) {
	startPos := carrierTakeoffStartPos(ship)
	p.CurPos = startPos
	p.CurRotation = ship.CurRotation
	p.FlightPhase = PlaneFlightPhaseTakingOff
	p.FlightPhaseStartPos = startPos
	p.FlightPhaseEndPos = carrierTakeoffEndPos(ship)
	p.FlightVisualScaleStart = planeLowAltitudeVisualScale
	p.FlightVisualScaleEnd = 1
	p.CurSpeed = max(ship.CurSpeed, p.MaxSpeed*gameSpeedMultiplier()*takeoffInitialSpeedRatio)
}

// StartLandingApproach 初始化舰尾进近阶段。
func (p *Plane) StartLandingApproach(ship *BattleShip) {
	currentScale := p.VisualScaleMultiplier()
	p.CurAttackTarget = ""
	p.FlightPhase = PlaneFlightPhaseLandingApproach
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = carrierLandingApproachPos(ship)
	p.FlightVisualScaleStart = currentScale
	p.FlightVisualScaleEnd = landingDeckStartVisualScale
}

// StartLandingDeck 初始化甲板滑跑回收阶段。
func (p *Plane) StartLandingDeck(ship *BattleShip) {
	currentScale := p.VisualScaleMultiplier()
	p.FlightPhase = PlaneFlightPhaseLandingDeck
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(ship)
	p.FlightVisualScaleStart = currentScale
	p.FlightVisualScaleEnd = planeLowAltitudeVisualScale
}

// IsCruising 返回飞机是否处于可正常接敌 / 开火的巡航阶段。
func (p *Plane) IsCruising() bool {
	return p.FlightPhase == "" || p.FlightPhase == PlaneFlightPhaseCruising
}

// FinishTakeoff 切换到巡航阶段。
func (p *Plane) FinishTakeoff() {
	p.FlightPhase = PlaneFlightPhaseCruising
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
}

// UpdateTakeoff 推进起飞阶段，返回是否已经进入巡航。
func (p *Plane) UpdateTakeoff(mapCfg *mapcfg.MapCfg) bool {
	if p.FlightPhase != PlaneFlightPhaseTakingOff {
		return p.IsCruising()
	}
	if p.CurHP <= 0 {
		return false
	}

	p.accelerateTo(p.MaxSpeed * gameSpeedMultiplier())
	p.forward(mapCfg, p.CurRotation, p.CurSpeed)
	if p.CurPos.Near(p.FlightPhaseEndPos, max(0.2, p.CurSpeed*1.5)) {
		p.FinishTakeoff()
		return true
	}
	return false
}

// UpdateLandingApproach 推进舰尾进近阶段，返回是否到达进近点。
func (p *Plane) UpdateLandingApproach(mapCfg *mapcfg.MapCfg, ship *BattleShip) bool {
	if p.CurHP <= 0 {
		return false
	}

	queuePos := carrierLandingApproachPos(ship)
	deckStartPos := carrierLandingDeckStartPos(ship)
	targetPos := queuePos
	switchingToDeckStart := !p.FlightPhaseEndPos.Near(deckStartPos, 0.05) &&
		p.CurPos.Near(queuePos, max(0.35, p.CurSpeed*2.5))
	if p.FlightPhaseEndPos.Near(deckStartPos, 0.05) || switchingToDeckStart {
		targetPos = deckStartPos
	}
	if switchingToDeckStart {
		currentScale := p.VisualScaleMultiplier()
		p.FlightPhaseStartPos = p.CurPos.Copy()
		p.FlightVisualScaleStart = currentScale
		p.FlightVisualScaleEnd = landingDeckStartVisualScale
	}

	p.FlightPhaseEndPos = targetPos
	executePlaneMovement(p, mapCfg, targetPos, p.approachSpeed(p.MaxSpeed*gameSpeedMultiplier()*landingApproachSpeedRatio))
	return targetPos.Near(deckStartPos, 0.05) && p.CurPos.Near(deckStartPos, max(0.25, p.CurSpeed*1.5))
}

// UpdateLandingDeck 推进甲板滑跑阶段，返回是否完成回收。
func (p *Plane) UpdateLandingDeck(mapCfg *mapcfg.MapCfg, ship *BattleShip) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(ship)
	executePlaneMovement(
		p,
		mapCfg,
		p.FlightPhaseEndPos,
		p.approachSpeed(max(ship.CurSpeed, p.MaxSpeed*gameSpeedMultiplier()*landingDeckSpeedRatio)),
	)
	return p.CurPos.Near(p.FlightPhaseEndPos, max(0.25, p.CurSpeed*1.5))
}

// VisualScaleMultiplier 返回当前起降阶段相对常规飞机绘制比例的倍率。
func (p *Plane) VisualScaleMultiplier() float64 {
	if p.IsCruising() {
		return 1
	}

	progress := p.FlightPhaseProgress()
	if p.FlightVisualScaleStart <= 0 && p.FlightVisualScaleEnd <= 0 {
		return 1
	}
	return p.FlightVisualScaleStart + (p.FlightVisualScaleEnd-p.FlightVisualScaleStart)*progress
}

// FlightPhaseProgress 返回当前阶段在起止点之间的进度，范围 [0, 1]。
func (p *Plane) FlightPhaseProgress() float64 {
	total := p.FlightPhaseStartPos.Distance(p.FlightPhaseEndPos)
	if total <= 0.001 {
		return 1
	}
	remaining := p.CurPos.Distance(p.FlightPhaseEndPos)
	return max(0, min(1, 1-remaining/total))
}

func (p *Plane) accelerateTo(targetSpeed float64) {
	p.CurSpeed = moveSpeedToward(p.CurSpeed, targetSpeed, p.phaseSpeedStep(targetSpeed))
}

func (p *Plane) approachSpeed(targetSpeed float64) float64 {
	return moveSpeedToward(p.CurSpeed, targetSpeed, p.phaseSpeedStep(targetSpeed))
}

func (p *Plane) phaseSpeedStep(targetSpeed float64) float64 {
	maxSpeed := max(p.MaxSpeed*gameSpeedMultiplier(), targetSpeed)
	if maxSpeed <= 0 {
		return 0
	}
	step := p.Acceleration * gameSpeedMultiplier()
	if step <= 0 {
		step = maxSpeed * phaseSpeedStepFallbackRate
	}
	return min(step, maxSpeed*phaseSpeedStepMaxRate)
}

func moveSpeedToward(curSpeed, targetSpeed, step float64) float64 {
	if step <= 0 {
		return targetSpeed
	}
	if curSpeed < targetSpeed {
		return min(targetSpeed, curSpeed+step)
	}
	return max(targetSpeed, curSpeed-step)
}

func (p *Plane) forward(mapCfg *mapcfg.MapCfg, rotation, speed float64) {
	nextPos := p.CurPos.Copy()
	nextPos.AddRx(math.Sin(rotation*math.Pi/180) * speed)
	nextPos.SubRy(math.Cos(rotation*math.Pi/180) * speed)
	if mapCfg != nil {
		nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	}
	p.CurPos = nextPos
	p.RemainRange -= speed
}
