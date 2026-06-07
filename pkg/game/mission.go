package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 任务选择
func (g *Game) handleMissionSelect() error {
	missions := metadata.AvailableMissions()
	curIndex := lo.IndexOf(missions, g.curMission)

	ui := g.objStates.MissionSelectUI

	// 左右方向键选择关卡
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		curIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		curIndex++
	}

	// 鼠标滚轮切换
	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		curIndex--
	} else if wheelY < 0 {
		curIndex++
	}

	// 鼠标点击左右箭头
	if ui != nil {
		if isHoverArea(ui.LeftArrow) && isMouseButtonLeftJustPressed() {
			curIndex--
		}
		if isHoverArea(ui.RightArrow) && isMouseButtonLeftJustPressed() {
			curIndex++
		}
	}

	curIndex = (curIndex + len(missions)) % len(missions)
	g.curMission = missions[curIndex]

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

// 任务加载
func (g *Game) handleMissionLoading() error {
	g.player.Play(audioRes.NewMissionsBackground())

	// 确保先展示加载中的界面，再加载地图数据
	if !g.objStates.LoadingInterface.Ready {
		return nil
	}
	g.missionMgr = manager.New(g.curMission)
	g.mode = GameModeMissionStart
	return nil
}

// 任务开始
func (g *Game) handleMissionStart() error {
	if g.objStates.LoadingInterface.MissionStartDrawn &&
		!g.objStates.LoadingInterface.LoadedAudioPlayed {
		audio.PlayAudioToEnd(audioRes.NewMissionLoaded())
		g.objStates.LoadingInterface.LoadedAudioPlayed = true
	}
	if isAnyNextInput() {
		g.mode = GameModeMissionRunning
		// TODO 支持关卡内 BGM
		// g.player.Close()
	}
	return nil
}

// startMissionLoading 进入关卡加载状态，并重置加载界面与完成音效状态。
func (g *Game) startMissionLoading() {
	g.objStates.LoadingInterface.Reset()
	g.mode = GameModeMissionLoading
	g.player.Close()
}

// 任务运行
func (g *Game) handleMissionRunning() error {
	status, err := g.missionMgr.Update()
	if err != nil {
		log.Fatal("failed to update mission: ", err)
	}
	if status == state.MissionSuccess {
		g.mode = GameModeMissionSuccess
	} else if status == state.MissionFailed {
		g.mode = GameModeMissionFailed
	}
	return nil
}

// 任务成功
func (g *Game) handleMissionSuccess() error {
	g.player.Play(audioRes.NewMissionSuccess())
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}

// 任务失败
func (g *Game) handleMissionFailed() error {
	g.player.Play(audioRes.NewMissionFailed())
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}
