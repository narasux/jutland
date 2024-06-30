package game

import (
	"log"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 任务选择
func (g *Game) handleMissionSelect() error {
	g.player.Play(audioRes.NewMissionsBackground())
	if isAnyNextInput() {
		// FIXME 目前没有任务可选，直接点击进入默认关卡
		g.curMission = "default"
		g.mode = GameModeMissionLoading
	}
	return nil
}

// 任务加载
func (g *Game) handleMissionLoading() error {
	g.missionMgr = manager.New(g.curMission)
	g.mode = GameModeMissionStart
	audio.PlayAudioToEnd(audioRes.NewMissionLoaded())
	return nil
}

// 任务开始
func (g *Game) handleMissionStart() error {
	if isAnyNextInput() {
		g.mode = GameModeMissionRunning
		g.player.Close()
	}
	return nil
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
