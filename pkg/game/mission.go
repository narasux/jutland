package game

import (
	"log"

	"github.com/narasux/jutland/pkg/mission"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 任务选择
func (g *Game) handleMissionSelect() error {
	g.player.Play(audioRes.NewMissionsBackground())
	// FIXME 目前没有任务可选，直接进入默认测试关卡
	if isAnyNextInput() {
		g.missionMgr = mission.NewManager(metadata.MissionDefault)
		g.mode = GameModeMissionStart
		g.player.Close()
	}
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
