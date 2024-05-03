package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

// 任务选择
func (g *Game) handleMissionSelect() error {
	g.player.Play(audioRes.NewMissionsBackground())
	// FIXME 目前没有任务可选，直接进入默认测试关卡
	if isAnyNextInput() {
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

// 任务运行 TODO 功能待实现
func (g *Game) handleMissionRunning() error {
	log.Println("work in progress")
	return ebiten.Termination
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
