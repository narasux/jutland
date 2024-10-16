package game

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/version"
)

type Game struct {
	// 游戏模式
	mode GameMode

	// 图像绘制器
	drawer *Drawer
	// 音频播放器
	player *audio.Player

	// 对象状态
	objStates *objStates

	curMission string
	// 任务管理
	missionMgr *manager.MissionManager
}

func New() *Game {
	g := &Game{
		mode:       GameModeStart,
		drawer:     NewDrawer(),
		player:     audio.NewPlayer(audio.Context),
		objStates:  nil,
		curMission: "Alpha",
		missionMgr: nil,
	}
	g.init()
	return g
}

// Update 核心方法，用于更新各资源状态
func (g *Game) Update() error {
	defer recoverAndLogThenExit()

	switch g.mode {
	case GameModeStart:
		return g.handleGameStart()
	case GameModeMenuSelect:
		return g.handleMenuSelect()
	case GameModeMissionSelect:
		return g.handleMissionSelect()
	case GameModeMissionLoading:
		return g.handleMissionLoading()
	case GameModeMissionStart:
		return g.handleMissionStart()
	case GameModeMissionRunning:
		return g.handleMissionRunning()
	case GameModeMissionSuccess:
		return g.handleMissionSuccess()
	case GameModeMissionFailed:
		return g.handleMissionFailed()
	case GameModeCollection:
		return g.handleGameCollection()
	case GameModeGameSetting:
		return g.handleGameSetting()
	case GameModeEnd:
		return g.handleGameEnd()
	default:
		log.Fatalf("unknown game mode: %d", g.mode)
	}
	return nil
}

// Draw 核心方法，用于在屏幕上绘制各资源
func (g *Game) Draw(screen *ebiten.Image) {
	defer recoverAndLogThenExit()

	switch g.mode {
	case GameModeStart:
		g.drawer.drawBackground(screen, bgImg.GameStart)
		g.drawer.drawGameTitle(screen)
	case GameModeMenuSelect:
		g.objStates.AutoUpdateMenuButtonStates(screen)
		g.drawer.drawBackground(screen, bgImg.GameMenu)
		g.drawer.drawGameMenu(screen, g.objStates.MenuButton)
	case GameModeMissionSelect:
		g.drawer.drawMissionSelect(screen, g.curMission)
		g.drawer.drawGameTip(screen, "选择任务...")
	case GameModeMissionLoading:
		g.drawer.drawBackground(screen, bgImg.MissionStart)
		g.drawer.drawGameTip(screen, "加载中...")
		g.objStates.LoadingInterface.Ready = true
	case GameModeMissionStart:
		g.drawer.drawBackground(screen, bgImg.MissionStart)
		g.drawer.drawGameTip(screen, "任务开始！")
	case GameModeMissionRunning:
		g.missionMgr.Draw(screen)
	case GameModeMissionSuccess:
		g.drawer.drawBackground(screen, bgImg.MissionSuccess)
		g.drawer.drawMissionResult(screen, "任务成功！", colorx.Green)
	case GameModeMissionFailed:
		g.drawer.drawBackground(screen, bgImg.MissionFailed)
		g.drawer.drawMissionResult(screen, "任务失败...", colorx.Red)
	case GameModeCollection:
		// TODO 功能待实现
		return
	case GameModeGameSetting:
		// TODO 功能待实现
		return
	case GameModeEnd:
		g.drawer.drawBackground(screen, bgImg.GameEnd)
		g.drawer.drawCredits(screen)
	default:
		log.Println("unknown game mode:", g.mode)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("VER %s FPS %0.2f", version.Version, ebiten.ActualFPS()))
}

// Layout 核心方法，用于设置窗口大小（全屏模式下无意义）
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// 游戏初始化
func (g *Game) init() {
	// 初始化菜单配置
	var fontSize float64 = 48
	g.objStates = &objStates{
		MenuButton: &menuButtonStates{
			MissionSelect: &menuButton{
				Text:     "任务选择",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				Mode:     GameModeMissionSelect,
			},
			Collection: &menuButton{
				Text:     "游戏图鉴",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				// Mode:     GameModeCollection,
				// TODO 临时调试
				Mode: GameModeMissionSuccess,
			},
			GameSetting: &menuButton{
				Text:     "游戏设置",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				// Mode:     GameModeGameSetting,
				// TODO 临时调试
				Mode: GameModeMissionFailed,
			},
			ExitGame: &menuButton{
				Text:     "退出游戏",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				Mode:     GameModeEnd,
			},
		},
		LoadingInterface: &loadingInterface{
			Ready: false,
		},
	}
}

// 游戏开始
func (g *Game) handleGameStart() error {
	// 播放游戏封面的 BGM
	g.player.Play(audioRes.NewGameStartBackground())
	// 任意下一按键触发后，切换模式，关闭 BGM
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}

// 菜单选择
func (g *Game) handleMenuSelect() error {
	g.player.Play(audioRes.NewMenuBackground())
	// 对于菜单按钮，如果 hover 则展示红色，点击则切换游戏模式
	for _, button := range []*menuButton{
		g.objStates.MenuButton.MissionSelect,
		g.objStates.MenuButton.Collection,
		g.objStates.MenuButton.GameSetting,
		g.objStates.MenuButton.ExitGame,
	} {
		if isHoverMenuButton(button) {
			// 仅首次移动会修改颜色 & 发声
			if button.Color != colorx.DarkRed {
				button.Color = colorx.DarkRed
				audio.PlayAudioToEnd(audioRes.NewMenuButtonHover())
			}
			// 左键点击按钮：切模式，播放音效，停止 BGM
			if isMouseButtonLeftJustPressed() {
				g.mode = button.Mode
				audio.PlayAudioToEnd(audioRes.NewMenuButtonClick())
				g.player.Close()
			}
		} else {
			button.Color = colorx.White
		}
	}
	return nil
}

// 游戏图鉴 TODO 功能待实现
func (g *Game) handleGameCollection() error {
	log.Println("work in progress")
	return ebiten.Termination
}

// 游戏设置 TODO 功能待实现
func (g *Game) handleGameSetting() error {
	log.Println("work in progress")
	return ebiten.Termination
}

// 游戏结束
func (g *Game) handleGameEnd() error {
	// 播放游戏结束的 BGM
	g.player.Play(audioRes.NewGameEndBackground())
	if isAnyNextInput() {
		return ebiten.Termination
	}
	return nil
}
