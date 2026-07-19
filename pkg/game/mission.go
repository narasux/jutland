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
	return true
}

// 任务加载
func (g *Game) handleMissionLoading() error {
	g.player.PlayLazy(audioRes.NewMissionsBackground)

	// 确保先展示加载中的界面，再加载地图数据
	if !g.objStates.LoadingInterface.Ready {
		return nil
	}
	g.missionMgr = manager.New(g.curMission, g.ui)
	g.mode = GameModeMissionStart
	return nil
}

// 任务开始
func (g *Game) handleMissionStart() error {
	g.missionMgr.WarmupMapBlocks()
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
	g.player.PlayLazy(audioRes.NewMissionSuccess)
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}

// 任务失败
func (g *Game) handleMissionFailed() error {
	g.player.PlayLazy(audioRes.NewMissionFailed)
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}
