package game

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"

	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	"github.com/narasux/jutland/pkg/resources/background"
	"github.com/narasux/jutland/pkg/resources/colorx"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

type Game struct {
	// 游戏模式
	mode GameMode

	// 图像绘制器
	drawer *Drawer
	// 音频播放器
	player *AudioPlayer

	// 对象状态
	objStates *objStates
}

func New() *Game {
	g := &Game{
		mode:   GameModeStart,
		drawer: NewDrawer(),
		player: NewAudioPlayer(audio.NewContext(SampleRate)),
	}
	g.init()
	return g
}

// Update 核心方法，用于更新各资源状态
// TODO 抽象成 Updater 类?
func (g *Game) Update() error {
	//log.Println("In update with mode: ", g.mode)

	switch g.mode {
	case GameModeStart:
		g.player.Play(audioRes.NewGameStartBackground())
		if isAnyNextInput() {
			g.mode = GameModeMenuSelect
			g.player.Close()
		}
	case GameModeMenuSelect:
		g.player.Play(audioRes.NewMenuBackground())
		// 对于菜单按钮，如果 hover 则展示红色，点击则切换游戏模式
		for _, button := range []*menuButton{
			g.objStates.MenuButton.MissionSelect,
			g.objStates.MenuButton.ShipCollection,
			g.objStates.MenuButton.GameSetting,
			g.objStates.MenuButton.ExitGame,
		} {
			if isHoverMenuButton(button) {
				// 仅首次移动会修改颜色 & 发声
				if button.Color != colorx.DarkRed {
					button.Color = colorx.DarkRed
					PlayAudioToEnd(g.player.ctx, audioRes.NewMenuButtonHover())
				}
				// 点击按钮：切模式，播放音效，停止 BGM
				if isMouseButtonLeftJustPressed() {
					g.mode = button.Mode
					PlayAudioToEnd(g.player.ctx, audioRes.NewMenuButtonClick())
					g.player.Close()
				}
			} else {
				button.Color = colorx.White
			}
		}
	case GameModeMissionSelect:
		return nil
	case GameModeMissionStart:
		return nil
	case GameModeMissionRunning:
		return nil
	case GameModeMissionSuccess:
		g.player.Play(audioRes.NewMissionSuccess())
		if isAnyNextInput() {
			g.mode = GameModeMenuSelect
		}
	case GameModeMissionFailed:
		g.player.Play(audioRes.NewMissionFailed())
		if isAnyNextInput() {
			g.mode = GameModeMenuSelect
		}
	case GameModeShipCollection:
		// TODO 实现功能
		return ebiten.Termination
	case GameModeGameSetting:
		// TODO 实现功能
		return ebiten.Termination
	case GameModeEnd:
		// 退出游戏
		g.player.Play(audioRes.NewGameEndBackground())
		if isAnyNextInput() {
			return ebiten.Termination
		}
	default:
		log.Fatalf("unknown game mode: %d", g.mode)
	}
	return nil
}

// Draw 核心方法，用于在屏幕上绘制各资源
func (g *Game) Draw(screen *ebiten.Image) {
	//log.Println("In draw with mode: ", g.mode)

	switch g.mode {
	case GameModeStart:
		g.drawer.drawBackground(screen, background.GameStartImg)
		g.drawer.drawGameTitle(screen, "日 德 兰 海 战")
	case GameModeMenuSelect:
		g.objStates.AutoUpdateMenuButtonStates(screen)
		g.drawer.drawBackground(screen, background.GameMenuImg)
		g.drawer.drawGameMenu(screen, g.objStates.MenuButton)
	case GameModeMissionSelect:
		return
	case GameModeMissionStart:
		return
	case GameModeMissionRunning:
		return
	case GameModeMissionSuccess:
		g.drawer.drawBackground(screen, background.MissionSuccessImg)
	case GameModeMissionFailed:
		g.drawer.drawBackground(screen, background.MissionFailedImg)
	case GameModeShipCollection:
		return
	case GameModeGameSetting:
		return
	case GameModeEnd:
		g.drawer.drawBackground(screen, background.GameEndImg)
		g.drawer.drawGameTitle(screen, "祝君武运昌隆！")
	default:
		log.Printf("unknown game mode: %d", g.mode)
	}

	ebutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f GameMode: %d", ebiten.ActualFPS(), g.mode))
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
			ShipCollection: &menuButton{
				Text:     "战舰图鉴",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				//Mode:     GameModeShipCollection,
				// TODO 临时调试
				Mode: GameModeMissionSuccess,
			},
			GameSetting: &menuButton{
				Text:     "游戏设置",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				//Mode:     GameModeGameSetting,
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
	}
}
