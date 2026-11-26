package tutorial

import (
	"fmt"
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
	baseline         snapshot
}

const tutorialStuckHintDelay = 12 * time.Second

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
	selected          map[string]objPos.MapPos
	groups            map[string]object.GroupID
	diveStates        map[string]objUnit.DiveState
	status            state.MissionStatus
	zoom              int
	camera            objPos.MapPos
	scoutCount        int
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
	layout := CalcLayout(float64(ms.View.Layout.Width), float64(ms.View.Layout.Height))
	sx, sy := ebiten.CursorPosition()
	leftClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	rightClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	nextPressed := inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)
	escPressed := inpututil.IsKeyJustPressed(ebiten.KeyEscape)
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
	case SetupCloseSidebar:
		ms.UI.SidebarExpanded = false
	}
}

func capture(ms *state.MissionState) snapshot {
	s := snapshot{
		selected:   make(map[string]objPos.MapPos, len(ms.Interaction.SelectedShips)),
		groups:     make(map[string]object.GroupID, len(ms.Interaction.SelectedShips)),
		diveStates: make(map[string]objUnit.DiveState, len(ms.Interaction.SelectedShips)),
		status:     ms.Core.MissionStatus,
		zoom:       ms.UI.GameOpts.Zoom,
		camera:     ms.View.Camera.Pos,
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
			s.scoutCount++
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
		return ms.UI.ShowSonarRanges
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
		count := 0
		for _, plane := range ms.Arena.Planes {
			if plane.ScoutManual && plane.CurHP > 0 {
				count++
			}
		}
		return count > c.baseline.scoutCount
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
	default:
		return false
	}
}
