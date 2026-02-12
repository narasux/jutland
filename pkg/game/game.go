package game

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pkg/browser"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/common/types"
	"github.com/narasux/jutland/pkg/mission/manager"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
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

	curShipName string
}

func New() *Game {
	g := &Game{
		mode:        GameModeStart,
		drawer:      NewDrawer(),
		player:      audio.NewPlayer(audio.Context),
		objStates:   nil,
		curMission:  "Alpha",
		missionMgr:  nil,
		curShipName: "lowa",
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
	case GameModeMissionLoading:
		g.drawer.drawBackground(screen, bgImg.MissionStart)
		g.drawer.drawGameTip(screen, "Loading...")
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
		g.drawer.drawBackground(screen, bgImg.MissionWindow)
		g.drawer.drawCollection(screen, g.curShipName, g.objStates.RefLinks)
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
				Mode:     GameModeCollection,
			},
			GameSetting: &menuButton{
				Text:     "游戏设置",
				FontSize: fontSize,
				Font:     font.Hang,
				Color:    colorx.White,
				// Mode:     GameModeGameSetting,
				// TODO 临时调试
				Mode: GameModeMissionSuccess,
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
				// 注：任务关卡选择延续菜单的 BGM
				if g.mode != GameModeMissionSelect {
					g.player.Close()
				}
				audio.PlayAudioToEnd(audioRes.NewMenuButtonClick())
			}
		} else {
			button.Color = colorx.White
		}
	}
	return nil
}

// 游戏图鉴
func (g *Game) handleGameCollection() error {
	// Esc 键返回菜单
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.mode = GameModeMenuSelect
		g.player.Close()
		return nil
	}
	// 随机选个军港 BGM（如果没播放完，会跳过的）
	newHarborFuncs := []func() types.AudioStream{
		audioRes.NewHarborUS,
		audioRes.NewHarborJP,
		audioRes.NewHarborUK,
		audioRes.NewHarborGEM,
		audioRes.NewHarborFR,
		audioRes.NewHarborNeutral,
	}
	g.player.Play(newHarborFuncs[rand.Intn(len(newHarborFuncs))]())

	allShipNames := objUnit.GetAllShipNames()
	// 上下左右方向键 / 鼠标滚轮选择战舰
	shipIndex := lo.IndexOf(allShipNames, g.curShipName)
	_, wheelY := ebiten.Wheel()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) ||
		wheelY > 0 {
		shipIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) ||
		wheelY < 0 {
		shipIndex++
	}
	shipCount := len(allShipNames)
	shipIndex = (shipIndex + shipCount) % shipCount

	g.curShipName = allShipNames[shipIndex]
	g.objStates.RefLinks = nil

	if ref := objRef.GetReference(g.curShipName); ref != nil {
		misLayout := layout.NewScreenLayout()
		xOffset, yOffset := float64(misLayout.Width/3*2)+200, float64(misLayout.Height/5*3)+250

		// 生成跳转链接点击区域
		for idx, link := range ref.Links {
			g.objStates.RefLinks = append(g.objStates.RefLinks, &refLink{
				fmt.Sprintf("[%d] %s", idx+1, link.Name), link.URL,
				xOffset, yOffset + float64(idx*45),
				24, font.Hang, colorx.White,
				float64(len(link.Name)) * 12, 30,
			})
		}

		for _, link := range g.objStates.RefLinks {
			if isHoverRefLink(link) {
				link.Color = colorx.SkyBlue
				// 左键点击按钮：浏览器打开链接
				if isMouseButtonLeftJustPressed() {
					_ = browser.OpenURL(link.URL)
				}
			} else {
				link.Color = colorx.White
			}
		}
	}
	return nil
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
