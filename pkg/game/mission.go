package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 任务选择
func (g *Game) handleMissionSelect() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.selectMissionCategory(otherMissionCategory(g.curMissionCategory))
		return nil
	}

	ui := g.objStates.MissionSelectUI
	if ui != nil && isMouseButtonLeftJustPressed() {
		if isHoverArea(ui.ClassicButton) {
			g.selectMissionCategory(metadata.MissionCategoryClassic)
			return nil
		}
		if isHoverArea(ui.TestButton) {
			g.selectMissionCategory(metadata.MissionCategoryTest)
			return nil
		}
		if isHoverArea(ui.SideP1Button) && g.selectMissionPlayerSide(faction.SideP1) {
			return nil
		}
		if isHoverArea(ui.SideP2Button) && g.selectMissionPlayerSide(faction.SideP2) {
			return nil
		}
	}

	missions := metadata.AvailableMissions(g.curMissionCategory)
	offset := 0

	// 左右方向键选择关卡
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		offset--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		offset++
	}

	// 鼠标滚轮切换
	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		offset--
	} else if wheelY < 0 {
		offset++
	}

	// 鼠标点击左右箭头
	if ui != nil {
		if isHoverArea(ui.LeftArrow) && isMouseButtonLeftJustPressed() {
			offset--
		}
		if isHoverArea(ui.RightArrow) && isMouseButtonLeftJustPressed() {
			offset++
		}
	}

	g.curMission = cycleMission(missions, g.curMission, offset)
	g.normalizeMissionPlayerSide()

	// 确定：Enter 键或点击「开始任务」
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.startMissionLoading()
	} else if ui != nil && isHoverArea(ui.StartButton) && isMouseButtonLeftJustPressed() {
		g.startMissionLoading()
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.mode = GameModeMenuSelect
	} else if ui != nil && isHoverArea(ui.BackButton) && isMouseButtonLeftJustPressed() {
		g.mode = GameModeMenuSelect
	}
	return nil
}

func otherMissionCategory(category metadata.MissionCategory) metadata.MissionCategory {
	if category == metadata.MissionCategoryTest {
		return metadata.MissionCategoryClassic
	}
	return metadata.MissionCategoryTest
}

func cycleMission(missions []string, current string, offset int) string {
	if len(missions) == 0 {
		return current
	}
	index := lo.IndexOf(missions, current)
	if index < 0 {
		index = 0
	}
	index = (index + offset%len(missions) + len(missions)) % len(missions)
	return missions[index]
}

// selectMissionCategory 切换任务分类，并落到该分类的第一关。
func (g *Game) selectMissionCategory(category metadata.MissionCategory) bool {
	if category == g.curMissionCategory && g.curMission != "" {
		return false
	}
	missions := metadata.AvailableMissions(category)
	if len(missions) == 0 {
		return false
	}
	g.curMissionCategory = category
	g.curMission = missions[0]
	g.normalizeMissionPlayerSide()
	return true
}

// normalizeMissionPlayerSide 将当前选择修正为任务可用的 P1/P2。
func (g *Game) normalizeMissionPlayerSide() {
	if g.curMission == "" {
		g.curMissionPlayerSide = faction.SideP1
		return
	}
	g.curMissionPlayerSide = metadata.Get(g.curMission).NormalizePlayerSide(g.curMissionPlayerSide)
}

// currentMissionPlayerSide 返回本次进入关卡使用的配置阵营。
func (g *Game) currentMissionPlayerSide() faction.Side {
	g.normalizeMissionPlayerSide()
	return g.curMissionPlayerSide
}

// selectMissionPlayerSide 记录玩家选择的配置阵营。
func (g *Game) selectMissionPlayerSide(side faction.Side) bool {
	if !metadata.Get(g.curMission).HasPlayerSide(side) {
		return false
	}
	g.curMissionPlayerSide = side
	return true
}

// 任务加载
func (g *Game) handleMissionLoading() error {
	g.player.PlayLazy(audioRes.NewMissionsBackground)

	// 确保先展示加载中的界面，再加载地图数据
	if !g.objStates.LoadingInterface.Ready {
		return nil
	}
	if g.missionMgr == nil {
		g.missionMgr = manager.New(g.curMission, g.currentMissionPlayerSide())
	}
	if !g.missionMgr.WarmupMapBlocks() {
		return nil
	}
	g.mode = GameModeMissionRunning
	return nil
}

// startMissionLoading 进入关卡加载状态，并重置加载界面与完成音效状态。
func (g *Game) startMissionLoading() {
	g.objStates.LoadingInterface.Reset()
	if g.missionMgr != nil {
		g.missionMgr.ClearBattleStats()
	}
	g.missionMgr = nil
	g.mode = GameModeMissionLoading
	g.player.Close()
}

// 任务运行
func (g *Game) handleMissionRunning() error {
	// 等关卡至少实际绘制一帧后再播放提示音，避免声音先于关卡画面出现。
	if g.objStates.LoadingInterface.MissionRunningDrawn &&
		!g.objStates.LoadingInterface.LoadedAudioPlayed {
		audio.PlayAudioToEnd(audioRes.NewMissionLoaded())
		g.objStates.LoadingInterface.LoadedAudioPlayed = true
	}
	status, err := g.missionMgr.Update()
	if err != nil {
		log.Fatal("failed to update mission: ", err)
	}
	if status == state.MissionSuccess {
		g.enterMissionSplash(GameModeMissionSuccess)
	} else if status == state.MissionFailed {
		g.enterMissionSplash(GameModeMissionFailed)
	}
	return nil
}

// resultSplashMinTicks 胜负全屏图至少展示的帧数，避免战斗末尾的点击直接跳过。
const resultSplashMinTicks = 30

// enterMissionSplash 进入胜负全屏图阶段，并记住成功 / 失败用于后续战报背景。
func (g *Game) enterMissionSplash(mode GameMode) {
	g.resultMode = mode
	g.resultSplashTicks = 0
	g.mode = mode
}

// 任务成功：先展示全屏胜利图，再进入战役总结。
func (g *Game) handleMissionSuccess() error {
	g.player.PlayLazy(audioRes.NewMissionSuccess)
	return g.handleMissionSplash()
}

// 任务失败：先展示全屏失败图，再进入战役总结。
func (g *Game) handleMissionFailed() error {
	g.player.PlayLazy(audioRes.NewMissionFailed)
	return g.handleMissionSplash()
}

// handleMissionSplash 等待玩家点击后跳转到战役总结。
func (g *Game) handleMissionSplash() error {
	// 防御：管理器缺失时退回旧行为，避免空指针。
	if g.missionMgr == nil {
		if isMouseButtonLeftJustPressed() {
			g.mode = GameModeMenuSelect
			g.player.Close()
		}
		return nil
	}
	if g.resultSplashTicks < resultSplashMinTicks {
		g.resultSplashTicks++
		return nil
	}
	if isMouseButtonLeftJustPressed() {
		g.mode = GameModeMissionReport
	}
	return nil
}

// missionResultAction 结算页上的一次显式操作。
type missionResultAction int

const (
	missionResultActionNone missionResultAction = iota
	missionResultActionBack
	missionResultActionRetry
)

// handleMissionReport 处理战役总结页的鼠标操作：
// 点击页签切换总览 / 舰艇 / 飞机，点击按钮返回任务选择或重新挑战。
// 结算流程不提供键盘快捷键。
func (g *Game) handleMissionReport() {
	// 防御：管理器缺失时退回旧行为，避免空指针。
	if g.missionMgr == nil {
		if isMouseButtonLeftJustPressed() {
			g.mode = GameModeMenuSelect
			g.player.Close()
		}
		return
	}
	sx, sy := ebiten.CursorPosition()
	if _, dy := ebiten.Wheel(); dy != 0 {
		g.missionMgr.ScrollBattleReportByWheel(dy, sx)
	}
	if g.missionMgr.UpdateBattleReportScrollDrag(sx, sy) {
		return
	}
	if !isMouseButtonLeftJustPressed() {
		return
	}
	layout := g.missionMgr.BattleReportLayout(true)
	if g.missionMgr.HandleBattleReportEventFilterClick(sx, sy) {
		return
	}
	if page, ok := layout.PageAt(sx, sy); ok {
		g.missionMgr.SetBattleReportPage(page)
		return
	}
	switch missionResultClickAction(layout, sx, sy) {
	case missionResultActionBack:
		g.leaveMissionResult()
	case missionResultActionRetry:
		g.startMissionLoading()
	}
}

// leaveMissionResult 返回任务选择。
func (g *Game) leaveMissionResult() {
	if g.missionMgr != nil {
		g.missionMgr.ClearBattleStats()
	}
	g.mode = GameModeMenuSelect
	g.player.Close()
}

// missionResultClickAction 把鼠标点击位置映射为结算页操作。
func missionResultClickAction(report state.BattleReportLayout, x, y int) missionResultAction {
	switch {
	case report.BackButton.Contains(x, y):
		return missionResultActionBack
	case report.RetryButton.Contains(x, y):
		return missionResultActionRetry
	default:
		return missionResultActionNone
	}
}
