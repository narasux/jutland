package game

import (
	"log"

	"github.com/narasux/jutland/pkg/mission"
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
		g.player.Close()
	}
	return nil
}

// 任务加载
func (g *Game) handleMissionLoading() error {
	g.player.Play(audioRes.NewMissionLoadingBackground())
	g.missionMgr = mission.NewManager(g.curMission)
	g.mode = GameModeMissionStart
	return nil
}

// 任务启动
func (g *Game) handleMissionStart() error {
	g.player.Play(audioRes.NewMissionStartBackground())
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
		log.Fatalf("failed to update mission: %s", err)
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
