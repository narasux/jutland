package game

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/collection"
	"github.com/narasux/jutland/pkg/common/types"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/game/settings"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/manager"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
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
	// 设置界面
	settingUI *settings.UI
	// 游戏图鉴界面
	collectionUI *collection.UI
	// 全游戏唯一的 EbitenUI 主实例
	ui          *ebitenui.UI
	emptyUIRoot widget.Containerer
}

func New() *Game {
	config.G.Language = string(i18n.SetLanguage(config.G.Language))
	emptyUIRoot := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	g := &Game{
		mode:        GameModeStart,
		drawer:      NewDrawer(),
		player:      audio.NewPlayer(audio.Context),
		objStates:   nil,
		curMission:  "Alpha",
		missionMgr:  nil,
		settingUI:   settings.New(),
		emptyUIRoot: emptyUIRoot,
		ui: &ebitenui.UI{
			Container:           emptyUIRoot,
			DisableDefaultFocus: true,
		},
	}
	g.collectionUI = collection.New()
	g.init()
	return g
}

// Update 核心方法，用于更新各资源状态
func (g *Game) Update() error {
	defer recoverAndLogThenExit()
	if g.handleUIEscape() {
		g.syncUIContainer()
		return nil
	}
	g.syncUIContainer()
	// 任务运行页由战术侧栏承载 EbitenUI，避免同一 Tick 重复更新全局输入。
	if g.mode != GameModeMissionRunning {
		g.ui.Update()
	}

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
		g.drawer.drawMissionSelect(screen, g.curMission, g.objStates)
	case GameModeMissionLoading:
		g.drawer.drawBackground(screen, bgImg.MissionStart)
		g.drawer.drawGameTip(screen, i18n.Text(i18n.MsgLoading))
		g.objStates.LoadingInterface.Ready = true
	case GameModeMissionStart:
		g.drawer.drawBackground(screen, bgImg.MissionStart)
		g.drawer.drawGameTip(screen, i18n.Text(i18n.MsgMissionStarted))
		g.objStates.LoadingInterface.MissionStartDrawn = true
	case GameModeMissionRunning:
		g.missionMgr.Draw(screen)
	case GameModeMissionSuccess:
		g.drawer.drawBackground(screen, bgImg.MissionSuccess)
		g.drawer.drawMissionResult(screen, i18n.Text(i18n.MsgMissionSuccess), colorx.Green)
	case GameModeMissionFailed:
		g.drawer.drawBackground(screen, bgImg.MissionFailed)
		g.drawer.drawMissionResult(screen, i18n.Text(i18n.MsgMissionFailed), colorx.Red)
	case GameModeCollection:
		g.drawer.drawBackground(screen, bgImg.MissionWindow)
		g.collectionUI.Draw(screen)
	case GameModeGameSetting:
		g.settingUI.Draw(screen)
	case GameModeEnd:
		g.drawer.drawBackground(screen, bgImg.GameEnd)
		g.drawer.drawCredits(screen)
	default:
		log.Println("unknown game mode:", g.mode)
	}

	if g.mode != GameModeMissionRunning {
		g.syncUIContainer()
		g.ui.Draw(screen)
	}
	if g.mode == GameModeCollection {
		g.collectionUI.DrawOverlay(screen)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("VER %s FPS %0.2f", version.Version, ebiten.ActualFPS()))
}

func (g *Game) syncUIContainer() {
	if g.mode == GameModeMissionRunning {
		return
	}
	target := g.emptyUIRoot
	switch g.mode {
	case GameModeCollection:
		target = g.collectionUI.Container()
	case GameModeGameSetting:
		target = g.settingUI.Container()
	}
	if target != nil && g.ui.Container != target {
		g.ui.Container = target
	}
}

func (g *Game) handleUIEscape() bool {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return false
	}
	switch g.mode {
	case GameModeCollection:
		g.mode = GameModeMenuSelect
		g.player.Close()
		return true
	case GameModeGameSetting:
		g.settingUI.Reset()
		g.mode = GameModeMenuSelect
		return true
	default:
		return false
	}
}

// Layout 核心方法，用于设置窗口大小（全屏模式下无意义）
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// 游戏初始化
func (g *Game) init() {
	g.initMenuButtons()
	g.objStates.LoadingInterface = &loadingInterface{Ready: false}
}

func (g *Game) initMenuButtons() {
	// 初始化菜单配置
	if g.objStates == nil {
		g.objStates = &objStates{}
	}
	menuFont := font.Hang
	if i18n.CurrentLanguage() == i18n.LanguageEnglish {
		menuFont = font.OpenSans
	}
	g.objStates.MenuButton = &menuButtonStates{
		MissionSelect: &menuButton{
			Text:     i18n.Text(i18n.MsgMenuMissionSelect),
			FontSize: menuFontSize,
			Font:     menuFont,
			Color:    colorx.White,
			Mode:     GameModeMissionSelect,
		},
		Collection: &menuButton{
			Text:     i18n.Text(i18n.MsgMenuCollection),
			FontSize: menuFontSize,
			Font:     menuFont,
			Color:    colorx.White,
			Mode:     GameModeCollection,
		},
		GameSetting: &menuButton{
			Text:     i18n.Text(i18n.MsgMenuSettings),
			FontSize: menuFontSize,
			Font:     menuFont,
			Color:    colorx.White,
			Mode:     GameModeGameSetting,
		},
		ExitGame: &menuButton{
			Text:     i18n.Text(i18n.MsgMenuExit),
			FontSize: menuFontSize,
			Font:     menuFont,
			Color:    colorx.White,
			Mode:     GameModeEnd,
		},
	}
}

// 游戏开始
func (g *Game) handleGameStart() error {
	// 播放游戏封面的 BGM
	g.player.PlayLazy(audioRes.NewGameStartBackground)
	// 任意下一按键触发后，切换模式，关闭 BGM
	if isAnyNextInput() {
		g.mode = GameModeMenuSelect
		g.player.Close()
	}
	return nil
}

// 菜单选择
func (g *Game) handleMenuSelect() error {
	g.player.PlayLazy(audioRes.NewMenuBackground)
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
				nextMode := button.Mode
				// 注：任务关卡选择延续菜单的 BGM
				if nextMode != GameModeMissionSelect {
					g.player.Close()
				}
				g.mode = nextMode
				if nextMode == GameModeCollection {
					g.startCollectionBGM()
				}
				g.syncUIContainer()
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
	g.collectionUI.Update()
	g.syncUIContainer()
	return nil
}

func (g *Game) startCollectionBGM() {
	newHarborFuncs := []func() types.AudioStream{
		audioRes.NewHarborUS,
		audioRes.NewHarborJP,
		audioRes.NewHarborUK,
		audioRes.NewHarborGEM,
		audioRes.NewHarborFR,
		audioRes.NewHarborNeutral,
	}
	g.player.PlayLazy(newHarborFuncs[rand.Intn(len(newHarborFuncs))])
}

// 游戏设置
func (g *Game) handleGameSetting() error {
	if g.settingUI.BackPressed() {
		g.applyLanguage(config.G.Language)
		g.settingUI.Reset()
		g.mode = GameModeMenuSelect
	}
	g.syncUIContainer()
	return nil
}

func (g *Game) applyLanguage(value string) {
	lang := i18n.SetLanguage(value)
	config.G.Language = string(lang)
	g.initMenuButtons()
	g.settingUI.ReloadLanguage()
	g.collectionUI.ReloadLanguage()
}

// 游戏结束
func (g *Game) handleGameEnd() error {
	// 播放游戏结束的 BGM
	g.player.PlayLazy(audioRes.NewGameEndBackground)
	if isAnyNextInput() {
		return ebiten.Termination
	}
	return nil
}
