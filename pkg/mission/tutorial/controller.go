package tutorial

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// ExitKind 教学结束方式。
type ExitKind int

const (
	// ExitNone 仍在教学中。
	ExitNone ExitKind = iota
	// ExitCompleted 正常完成。
	ExitCompleted
	// ExitAborted 玩家主动退出。
	ExitAborted
)

// Controller 单个教学任务的步骤状态机。
type Controller struct {
	script           *Script
	stepIndex        int
	finished         bool
	exit             ExitKind
	restartRequested bool
	stepStartedAt    time.Time
	// sonarHeldSince 声呐步骤里 V 开始按住的时刻；松开即清零。
	sonarHeldSince time.Time
	baseline       snapshot
}

const tutorialStuckHintDelay = 12 * time.Second

// tutorialSonarHoldTime 声呐步骤要求连续按住 V 的时长。
// 声呐范围只在按住时显示，若按下第一帧就判定完成，玩家还没看清范围就已翻页。
const tutorialSonarHoldTime = 1500 * time.Millisecond

// tutorialMinStepTime 指引步骤的最短展示时间。
// 有些完成条件在同一帧就能满足（例如靠近即自动开火的战斗步骤），若立刻翻页，
// 玩家来不及读完卡片；统一给一个短暂的停留时间。
const tutorialMinStepTime = 2500 * time.Millisecond

// View 绘制层需要的只读教学视图。
type View struct {
	Visible   bool
	Finished  bool
	StepIndex int
	StepCount int
	Title     string
	Body      string
	StuckHint string
	Hint      Hint
	AllowSkip bool
	Mode      Mode
	Target    *objPos.MapPos
}

type snapshot struct {
	selected   map[string]objPos.MapPos
	groups     map[string]object.GroupID
	diveStates map[string]objUnit.DiveState
	status     state.MissionStatus
	zoom       int
	camera     objPos.MapPos
	// scoutUIDs 进入步骤时已在场的手动侦察机（含仍在滑跑的），只用来判断“本步骤又派出了一架”。
	scoutUIDs map[string]struct{}
	// airfieldProduction 进入步骤时己方机场的制造机型，用于判定玩家切换了生产机型。
	airfieldProduction map[string]string
	// tick 进入步骤时的模拟帧号，战况日志只统计此后的命中。
	tick              int64
	reinforceOncoming int
}

// New 为教学任务创建控制器，并进入第一步。
func New(mission string, ms *state.MissionState) (*Controller, error) {
	script, ok := ScriptForMission(mission)
	if !ok {
		return nil, fmt.Errorf("tutorial script not found for mission %q", mission)
	}
	controller := &Controller{script: script}
	controller.enterStep(0, ms)
	return controller, nil
}

// ExitRequested 返回是否应离开教学并回到教学列表。
func (c *Controller) ExitRequested() bool {
	return c.exit != ExitNone
}

// ExitKind 返回结束方式；未结束时为 ExitNone。
func (c *Controller) ExitKind() ExitKind {
	return c.exit
}

// Abort 让教学立即以“玩家主动退出”结束。
func (c *Controller) Abort() {
	if c.exit == ExitNone {
		c.exit = ExitAborted
	}
}

// RestartRequested 返回是否请求重玩当前章节。
func (c *Controller) RestartRequested() bool {
	return c.restartRequested
}

// StepIndex 返回当前步骤下标；完成时为步骤总数。
func (c *Controller) StepIndex() int {
	if c.finished {
		return len(c.script.Steps)
	}
	return c.stepIndex
}

// Finished 返回是否已走完所有步骤。
func (c *Controller) Finished() bool {
	return c.finished
}

// BlocksGameplay 当前帧是否需要独占输入并冻结战场。
func (c *Controller) BlocksGameplay() bool {
	if c.exit != ExitNone || c.finished {
		return true
	}
	step := c.currentStep()
	if step == nil {
		return true
	}
	return step.Mode == ModeInfo || step.Mode == ModeScripted
}

// HandleInput 处理教学卡上的输入；返回 true 表示这一帧不应再交给游戏逻辑。
func (c *Controller) HandleInput(ms *state.MissionState) bool {
	if c.exit != ExitNone {
		return true
	}
	layout := CalcLayout(
		float64(ms.View.Layout.Width), float64(ms.View.Layout.Height), ms.UI.TutorialCardCollapsed,
	)
	sx, sy := ebiten.CursorPosition()
	leftClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	rightClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	nextPressed := inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)
	escPressed := inpututil.IsKeyJustPressed(ebiten.KeyEscape)

	// 收起 / 展开把手优先处理：它只改变卡片形态，不推进步骤。
	if leftClick && layout.Toggle.Contains(sx, sy) {
		ms.UI.TutorialCardCollapsed = !ms.UI.TutorialCardCollapsed
		return true
	}
	if ms.UI.TutorialCardCollapsed {
		// 收起态只剩底部窄条：点它展开；回车/空格仍能推进信息卡与完成卡。
		if leftClick && layout.Card.Contains(sx, sy) {
			ms.UI.TutorialCardCollapsed = false
			return true
		}
		if c.finished {
			if nextPressed || escPressed {
				c.exit = ExitCompleted
			}
			return true
		}
		step := c.currentStep()
		if step == nil {
			c.exit = ExitAborted
			return true
		}
		if nextPressed && step.Mode != ModeGuided {
			c.advance(ms)
			return true
		}
		if step.Mode == ModeInfo && escPressed {
			c.exit = ExitAborted
			return true
		}
		// 信息卡仍冻结战场；操作步骤继续放行，玩家收起卡片后照常操作。
		return step.Mode == ModeInfo || step.Mode == ModeScripted
	}

	clickNext := leftClick && layout.Next.Contains(sx, sy)
	clickSkip := leftClick && layout.Skip.Contains(sx, sy)
	clickExit := leftClick && layout.Exit.Contains(sx, sy)

	if c.finished {
		if clickExit {
			// 已经走到完成卡，点右上角退出也视为完成本章。
			c.exit = ExitCompleted
			return true
		}
		if clickSkip {
			c.restartRequested = true
			return true
		}
		if clickNext || nextPressed || escPressed {
			c.exit = ExitCompleted
		}
		return true
	}

	step := c.currentStep()
	if step == nil {
		c.exit = ExitAborted
		return true
	}

	if clickExit || (step.Mode == ModeInfo && escPressed) {
		c.exit = ExitAborted
		return true
	}
	if clickSkip && step.AllowSkip {
		c.advance(ms)
		return true
	}

	switch step.Mode {
	case ModeInfo, ModeScripted:
		if clickNext || nextPressed {
			c.advance(ms)
		}
		return true
	case ModeFree:
		if clickNext || nextPressed {
			c.advance(ms)
			return true
		}
		return false
	case ModeGuided:
		// 卡片本身是 UI：点/右键在卡片范围内时不要让战场收到这次输入。
		if (leftClick || rightClick) && layout.Card.Contains(sx, sy) {
			return true
		}
		return false
	default:
		return false
	}
}

// Observe 在普通任务更新结束后比较状态，满足条件则进入下一步。
func (c *Controller) Observe(ms *state.MissionState) {
	if c.exit != ExitNone || c.finished {
		return
	}
	step := c.currentStep()
	if step == nil || step.Mode != ModeGuided {
		return
	}
	if time.Since(c.stepStartedAt) < tutorialMinStepTime {
		return
	}
	if c.completionMet(step.Completion, ms) {
		c.advance(ms)
	}
}

// View 返回当前步骤的只读展示数据。
func (c *Controller) View() View {
	if c.exit != ExitNone {
		return View{}
	}
	if c.finished {
		return View{
			Visible:   true,
			Finished:  true,
			StepIndex: len(c.script.Steps),
			StepCount: len(c.script.Steps),
			Title:     i18n.Text(i18n.MsgTutorialCompleted),
			Body:      i18n.Text(i18n.MsgTutorialCompletedBody),
			Mode:      ModeInfo,
		}
	}
	step := c.currentStep()
	if step == nil {
		return View{}
	}
	stuckHint := ""
	if step.Mode == ModeGuided && !c.stepStartedAt.IsZero() &&
		time.Since(c.stepStartedAt) >= tutorialStuckHintDelay {
		stuckHint = i18n.Text(i18n.MsgTutorialStuckHint)
	}
	return View{
		Visible:   true,
		StepIndex: c.stepIndex + 1,
		StepCount: len(c.script.Steps),
		Title:     step.Title.Value(),
		Body:      step.Body.Value(),
		StuckHint: stuckHint,
		Hint:      step.Hint,
		AllowSkip: step.AllowSkip,
		Mode:      step.Mode,
		Target:    step.Target,
	}
}

func (c *Controller) currentStep() *Step {
	if c.stepIndex < 0 || c.stepIndex >= len(c.script.Steps) {
		return nil
	}
	return &c.script.Steps[c.stepIndex]
}

func (c *Controller) advance(ms *state.MissionState) {
	if c.stepIndex+1 >= len(c.script.Steps) {
		c.finished = true
		return
	}
	c.stepIndex++
	c.enterStep(c.stepIndex, ms)
}

func (c *Controller) enterStep(index int, ms *state.MissionState) {
	c.stepIndex = index
	c.stepStartedAt = time.Now()
	c.sonarHeldSince = time.Time{}
	if index < 0 || index >= len(c.script.Steps) {
		c.finished = true
		return
	}
	for _, setup := range c.script.Steps[index].Setup {
		applySetup(ms, setup)
	}
	c.baseline = capture(ms)
}

func applySetup(ms *state.MissionState, setup SetupAction) {
	switch setup {
	case SetupClearSelection:
		ms.Interaction.SelectedShips = []string{}
		ms.Interaction.FocusedShipUid = ""
		ms.Interaction.SelectedGroupID = object.GroupIDNone
		ms.Interaction.IsAreaSelecting = false
	case SetupSelectPlayerFleet:
		ms.Interaction.SelectedShips = ms.Interaction.SelectedShips[:0]
		for _, ship := range ms.Arena.OrderedShips() {
			if ship.BelongPlayer == ms.Player.CurPlayer && ship.CurHP > 0 {
				ms.Interaction.SelectedShips = append(ms.Interaction.SelectedShips, ship.Uid)
			}
		}
		if len(ms.Interaction.SelectedShips) > 0 {
			ms.Interaction.FocusedShipUid = ms.Interaction.SelectedShips[0]
		}
		ms.Interaction.SelectedGroupID = object.GroupIDNone
	case SetupOpenSidebar:
		ms.UI.SidebarExpanded = true
	case SetupOrderAirStrike:
		// 教学演示：替玩家给航母指定最近的敌舰，航母会自动派机攻击。
		for _, ship := range ms.Arena.Ships {
			if ship.BelongPlayer != ms.Player.CurPlayer ||
				ship.Type != objUnit.ShipTypeAircraftCarrier || !ship.Aircraft.HasPlane {
				continue
			}
			var target *objUnit.BattleShip
			bestDistance := math.MaxFloat64
			for _, enemy := range ms.Arena.Ships {
				if enemy.BelongPlayer == ms.Player.CurPlayer || enemy.CurHP <= 0 {
					continue
				}
				if distance := ship.CurPos.Distance(enemy.CurPos); distance < bestDistance {
					bestDistance, target = distance, enemy
				}
			}
			if target != nil {
				ship.Attack(target.Uid)
			}
		}
	case SetupDisarmEnemy:
		// 教学靶舰不还击：直接关闭敌方舰船的全部武器。
		for _, ship := range ms.Arena.Ships {
			if ship.BelongPlayer != ms.Player.CurPlayer {
				ship.DisableWeapon(objUnit.WeaponTypeAll)
			}
		}
	case SetupCloseSidebar:
		ms.UI.SidebarExpanded = false
	case SetupFocusAirfield:
		// 教学步骤专用：把镜头直接摆到己方机场，省得玩家自己找跑道。
		for _, airfield := range ms.Arena.Airfields {
			if airfield.BelongPlayer != ms.Player.CurPlayer {
				continue
			}
			ms.View.Camera.Target = nil
			ms.View.Camera.Pos.AssignRxy(
				airfield.Pos.RX-float64(ms.View.Camera.Width)/2,
				airfield.Pos.RY-float64(ms.View.Camera.Height)/2,
			)
			// 测试用的最小状态可能没有地图元数据，此时不做边界约束。
			borderX, borderY := 0.0, 0.0
			if ms.Core.MissionMD.MapCfg != nil {
				borderX, borderY = ms.CameraPosBorder()
			}
			ms.View.Camera.Pos.EnsureBorder(borderX, borderY)
			break
		}
	}
}

func capture(ms *state.MissionState) snapshot {
	s := snapshot{
		selected:           make(map[string]objPos.MapPos, len(ms.Interaction.SelectedShips)),
		groups:             make(map[string]object.GroupID, len(ms.Interaction.SelectedShips)),
		diveStates:         make(map[string]objUnit.DiveState, len(ms.Interaction.SelectedShips)),
		scoutUIDs:          make(map[string]struct{}, len(ms.Arena.Planes)),
		airfieldProduction: make(map[string]string, len(ms.Arena.Airfields)),
		status:             ms.Core.MissionStatus,
		zoom:               ms.UI.GameOpts.Zoom,
		camera:             ms.View.Camera.Pos,
	}
	for _, uid := range ms.Interaction.SelectedShips {
		ship := ms.Arena.Ships[uid]
		if ship == nil {
			continue
		}
		s.selected[uid] = ship.CurPos
		s.groups[uid] = ship.GroupID
		s.diveStates[uid] = ship.Submarine.DiveState
	}
	for _, plane := range ms.Arena.Planes {
		if plane.ScoutManual && plane.CurHP > 0 {
			s.scoutUIDs[plane.Uid] = struct{}{}
		}
	}
	for _, airfield := range ms.Arena.Airfields {
		if airfield.BelongPlayer == ms.Player.CurPlayer {
			s.airfieldProduction[airfield.Uid] = airfield.CurProducing
		}
	}
	for _, rp := range ms.Arena.ReinforcePoints {
		if rp.BelongPlayer == ms.Player.CurPlayer {
			s.reinforceOncoming += len(rp.OncomingShips)
		}
	}
	return s
}

func (c *Controller) reinforceOncoming(ms *state.MissionState) int {
	total := 0
	for _, rp := range ms.Arena.ReinforcePoints {
		if rp.BelongPlayer == ms.Player.CurPlayer {
			total += len(rp.OncomingShips)
		}
	}
	return total
}

func (c *Controller) completionMet(completion Completion, ms *state.MissionState) bool {
	switch completion.Kind {
	case CompletionSelectedShips:
		minCount := completion.Min
		if minCount <= 0 {
			minCount = 1
		}
		return len(ms.Interaction.SelectedShips) >= minCount
	case CompletionShipMoved:
		threshold := completion.Distance
		if threshold <= 0 {
			threshold = 2
		}
		for _, uid := range ms.Interaction.SelectedShips {
			ship := ms.Arena.Ships[uid]
			prev, ok := c.baseline.selected[uid]
			if ship == nil || !ok {
				continue
			}
			if ship.CurPos.Distance(prev) >= threshold {
				return true
			}
		}
		return false
	case CompletionGroupAssigned:
		group := object.GroupID(completion.Group)
		if completion.Group == 0 {
			group = object.GroupID1
		}
		for _, uid := range ms.Interaction.SelectedShips {
			if ship := ms.Arena.Ships[uid]; ship != nil && ship.GroupID == group {
				return true
			}
		}
		return false
	case CompletionGroupSelected:
		group := object.GroupID(completion.Group)
		if completion.Group == 0 {
			group = object.GroupID1
		}
		return ms.Interaction.SelectedGroupID == group &&
			len(ms.Interaction.SelectedShips) > 0
	case CompletionDiveState:
		for _, uid := range ms.Interaction.SelectedShips {
			ship := ms.Arena.Ships[uid]
			prev, ok := c.baseline.diveStates[uid]
			if ship == nil || !ok {
				continue
			}
			if ship.Submarine.DiveState != prev && ship.Type == objUnit.ShipTypeSubmarine {
				return true
			}
		}
		return false
	case CompletionSonarVisible:
		return c.sonarHeldLongEnough(ms)
	case CompletionMapOpened:
		return ms.Core.MissionStatus == state.MissionInMap
	case CompletionMapReturned:
		return c.baseline.status == state.MissionInMap &&
			ms.Core.MissionStatus == state.MissionRunning
	case CompletionPaused:
		return ms.Core.MissionStatus == state.MissionPaused
	case CompletionResumed:
		return c.baseline.status == state.MissionPaused &&
			ms.Core.MissionStatus == state.MissionRunning
	case CompletionZoomChanged:
		return state.NormalizeZoom(ms.UI.GameOpts.Zoom) !=
			state.NormalizeZoom(c.baseline.zoom)
	case CompletionCameraMoved:
		return ms.View.Camera.Pos.Distance(c.baseline.camera) >= 0.5
	case CompletionScoutLaunched:
		// 本步骤新派出、且已经离舰的侦察机：刚下单时飞机还在甲板滑跑，
		// 若这一帧就判定完成，玩家还没看到起飞就翻页了。
		// 用 UID 而非数量，避免步骤开始时在空的侦察机返航后基线失效。
		for _, plane := range ms.Arena.Planes {
			if !plane.ScoutManual || plane.CurHP <= 0 || plane.IsOnGround() {
				continue
			}
			if _, existed := c.baseline.scoutUIDs[plane.Uid]; !existed {
				return true
			}
		}
		return false
	case CompletionReinforceOpened:
		return ms.Core.MissionStatus == state.MissionInBuilding
	case CompletionReinforceSummoned:
		return c.reinforceOncoming(ms) > c.baseline.reinforceOncoming
	case CompletionReinforceCanceled:
		return c.reinforceOncoming(ms) < c.baseline.reinforceOncoming
	case CompletionReinforceClosed:
		return c.baseline.status == state.MissionInBuilding &&
			ms.Core.MissionStatus == state.MissionRunning
	case CompletionSidebarExpanded:
		return ms.UI.SidebarExpanded
	case CompletionTerminalOpened:
		return ms.Core.MissionStatus == state.MissionInTerminal
	case CompletionAirfieldProduction:
		for _, airfield := range ms.Arena.Airfields {
			if airfield.BelongPlayer != ms.Player.CurPlayer {
				continue
			}
			if prev, ok := c.baseline.airfieldProduction[airfield.Uid]; ok && airfield.CurProducing != prev {
				return true
			}
		}
		return false
	case CompletionEnemyHit:
		return c.enemyHitSinceBaseline(ms, completion)
	case CompletionPlaneShotDown:
		return c.planeShotDownSinceBaseline(ms)
	case CompletionAirStrike:
		return c.airStrikeOrdered(ms)
	default:
		return false
	}
}

// sonarHeldLongEnough 判断 V 是否已连续按住足够久；松开立即清零，短暂误触不算完成。
func (c *Controller) sonarHeldLongEnough(ms *state.MissionState) bool {
	if !ms.UI.ShowSonarRanges {
		c.sonarHeldSince = time.Time{}
		return false
	}
	if c.sonarHeldSince.IsZero() {
		c.sonarHeldSince = time.Now()
		return false
	}
	return time.Since(c.sonarHeldSince) >= tutorialSonarHoldTime
}

// battleLogSince 返回进入步骤之后新增的战况记录。
// 战况日志按时间追加，从尾部往前找到第一条越过基线的记录即可。
func (c *Controller) battleLogSince(ms *state.MissionState) []state.BattleLogEntry {
	log := ms.Stats.BattleLog
	start := len(log)
	for start > 0 && log[start-1].Tick > c.baseline.tick {
		start--
	}
	if start == len(log) {
		return nil
	}
	return log[start:]
}

// enemyHitSinceBaseline 判断步骤开始后我方是否命中敌方舰船。
// weapon / actor 为空表示不筛选；命中与击沉都算。
func (c *Controller) enemyHitSinceBaseline(ms *state.MissionState, completion Completion) bool {
	player := ms.Player.CurPlayer
	for _, entry := range c.battleLogSince(ms) {
		if entry.ActorPlayer != player || entry.TargetPlayer == player || entry.TargetIsPlane {
			continue
		}
		if entry.Action != state.BattleLogShipHit && entry.Action != state.BattleLogShipSunk {
			continue
		}
		if completion.Weapon != "" && entry.Hit.WeaponType != completion.Weapon {
			continue
		}
		switch completion.Actor {
		case "ship":
			if entry.ActorIsPlane {
				continue
			}
		case "plane":
			if !entry.ActorIsPlane {
				continue
			}
		}
		return true
	}
	return false
}

// planeShotDownSinceBaseline 判断步骤开始后我方是否击落过敌机。
func (c *Controller) planeShotDownSinceBaseline(ms *state.MissionState) bool {
	player := ms.Player.CurPlayer
	for _, entry := range c.battleLogSince(ms) {
		if entry.Action != state.BattleLogPlaneShotDown {
			continue
		}
		if entry.ActorPlayer != player || !entry.TargetIsPlane || entry.TargetPlayer == player {
			continue
		}
		return true
	}
	return false
}

// airStrikeOrdered 判断我方是否有舰载机已受命攻击敌方舰船。
func (c *Controller) airStrikeOrdered(ms *state.MissionState) bool {
	player := ms.Player.CurPlayer
	for _, plane := range ms.Arena.Planes {
		if plane.BelongPlayer != player || plane.CurAttackTarget == "" {
			continue
		}
		if target := ms.Arena.Ships[plane.CurAttackTarget]; target != nil &&
			target.BelongPlayer != player && target.CurHP > 0 {
			return true
		}
	}
	return false
}
