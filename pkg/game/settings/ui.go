// 游戏设置界面，使用 ebitenui 实现，风格参考游戏图鉴（羊皮纸）
package settings

import (
	"fmt"
	stdimg "image"
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

var (
	colorPanelBg = color.RGBA{R: 24, G: 21, B: 18, A: 150} // 全屏暗色蒙层，提升羊皮纸上的文字对比度
	colorCardBg  = color.RGBA{R: 28, G: 25, B: 21, A: 235} // 设置卡片背景
	colorRowBg   = color.RGBA{R: 42, G: 37, B: 30, A: 210} // 单条设置项背景
	colorAccent  = color.RGBA{R: 212, G: 180, B: 112, A: 255}
	colorMuted   = color.RGBA{R: 190, G: 180, B: 160, A: 255}
	colorSelBg   = color.RGBA{R: 190, G: 157, B: 92, A: 255}
)

// 固定排版常量。
const (
	controlPadX   = 36
	strategyCount = 5

	// gameSettingsRows 是游戏页的设置行数；新增/删除行时必须同步，
	// 否则卡片高度仍按旧行数计算，最后一行会和页脚一起溢出。
	gameSettingsRows = 8
	maxSettingsRows  = gameSettingsRows
)

type speedOption struct {
	Label i18n.MessageID
	Value float64
}

type toggleOption struct {
	Label i18n.MessageID
	Value bool
}

// 速度选项定义
var speedOptions = []speedOption{
	{i18n.MsgSpeedSlow, config.SpeedSlowMultiplier},
	{i18n.MsgSpeedNormal, config.SpeedStandardMultiplier},
	{i18n.MsgSpeedFast, config.SpeedFastMultiplier},
}

// 开/关选项，关在前，便于默认关闭的选项显示为关。
var toggleOptions = []toggleOption{
	{i18n.MsgSettingsOff, false},
	{i18n.MsgSettingsOn, true},
}

type settingsTab int

const (
	tabGame settingsTab = iota
	tabAV
)

// settingsMetrics 是一次构建设置页时使用的像素排版参数，按屏幕高度等比缩放。
type settingsMetrics struct {
	scale       float64
	screenW     int
	screenH     int
	panelX      int
	panelY      int
	panelW      int
	panelH      int
	titleY      int
	tabsY       int
	footerY     int
	panelPad    int
	rowPadX     int
	rowH        int
	rowGap      int
	controlW    int
	controlH    int
	sliderW     int
	valueW      int
	valueGap    int
	handle      int
	titleH      int
	hintH       int
	sectionGap  int
	tabH        int
	footerH     int
	titleSize   int
	sectionSize int
	labelSize   int
	hintSize    int
	buttonSize  int
	valueSize   int
}

func newSettingsMetrics(screen layout.ScreenLayout, rows int) settingsMetrics {
	sw, sh := screen.Width, screen.Height
	if sw <= 0 || sh <= 0 {
		sw, sh = 1280, 720
	}
	rows = min(max(rows, 1), maxSettingsRows)
	scale := min(max(float64(sh)/900, 0.80), 1.30)

	margin := int(math.Round(64 * scale))
	if margin*2 > sw/3 {
		margin = sw / 6
	}
	panelW := min(sw-2*margin, int(math.Round(940*scale)))
	if panelW < 360 {
		panelW = max(sw-2*margin, 320)
	}
	panelX := (sw - panelW) / 2

	m := settingsMetrics{
		scale:       scale,
		screenW:     sw,
		screenH:     sh,
		panelX:      panelX,
		panelW:      panelW,
		panelPad:    int(math.Round(28 * scale)),
		rowPadX:     int(math.Round(18 * scale)),
		rowH:        int(math.Round(48 * scale)),
		rowGap:      int(math.Round(8 * scale)),
		controlW:    int(math.Round(230 * scale)),
		controlH:    int(math.Round(36 * scale)),
		valueW:      int(math.Round(52 * scale)),
		valueGap:    int(math.Round(12 * scale)),
		handle:      int(math.Round(14 * scale)),
		titleH:      int(math.Round(48 * scale)),
		hintH:       int(math.Round(18 * scale)),
		sectionGap:  int(math.Round(18 * scale)),
		tabH:        int(math.Round(40 * scale)),
		footerH:     int(math.Round(42 * scale)),
		titleSize:   int(math.Round(38 * scale)),
		sectionSize: int(math.Round(26 * scale)),
		labelSize:   int(math.Round(21 * scale)),
		hintSize:    int(math.Round(14 * scale)),
		buttonSize:  int(math.Round(19 * scale)),
		valueSize:   int(math.Round(17 * scale)),
	}
	m.handle = max(m.handle, 10)
	m.sliderW = max(m.controlW-m.valueW-m.valueGap, int(math.Round(80*scale)))

	m.panelH = m.panelPad*2 + m.hintH + m.sectionGap +
		rows*m.rowH + (rows-1)*m.rowGap

	gap := int(math.Round(16 * scale))
	blockH := m.titleH + gap + m.tabH + gap + m.panelH + gap + m.footerH
	startY := max(int(math.Round(24*scale)), (sh-blockH)/2)
	m.titleY = startY
	m.tabsY = m.titleY + m.titleH + gap
	m.panelY = m.tabsY + m.tabH + gap
	m.footerY = m.panelY + m.panelH + gap
	return m
}

// WindowHost 承载设置页的模态窗口，由游戏共享的 EbitenUI 主实例实现。
type WindowHost interface {
	AddWindow(*widget.Window) widget.RemoveWindowFunc
}

// UI 游戏设置 UI
type UI struct {
	container widget.Containerer
	metrics   settingsMetrics
	tab       settingsTab

	windowHost    WindowHost
	confirmWindow *widget.Window

	localValue         float64 // 本地副本，保存时才写回 config.G
	localLanguage      i18n.Language
	localBGMVolume     float64 // 背景音乐音量 0~1
	localEffectsVolume float64 // 战场音效音量 0~1
	localAI            config.AIStrategies
	localFog           bool
	localMinimap       bool
	localRam           bool
	localLand          bool
	// 命中特效 / 伤害数字 / 常显战舰状态：显示类开关，2026-10 从侧栏关卡设置迁入
	localHitEffects    bool
	localDamageNumbers bool
	localShipState     bool
	// 尾流长度 / 宽度 / 浓度自定义参数
	localWake config.WakeTrailSettings

	pendingRebuild bool
	backPressed    bool
}

// New 创建设置界面
func New() *UI {
	s := &UI{}
	s.loadFromSettings(config.G)
	s.applyAudioVolumes()
	s.buildUI()
	return s
}

// Draw 绘制设置界面
func (s *UI) Draw(screen *ebiten.Image) {
	s.drawParchmentBg(screen)
}

// Update 应用控件回调里排队的重建，避免在同一帧中断按钮释放事件。
func (s *UI) Update() {
	if !s.pendingRebuild {
		return
	}
	s.pendingRebuild = false
	s.buildUI()
}

// Container 返回由游戏共享 EbitenUI 主实例承载的设置容器。
func (s *UI) Container() widget.Containerer { return s.container }

// BackPressed 返回用户是否点了返回/取消
func (s *UI) BackPressed() bool { return s.backPressed }

// SetWindowHost 注入承载确认弹窗的 EbitenUI 主实例。
func (s *UI) SetWindowHost(host WindowHost) { s.windowHost = host }

// CancelConfirm 关闭恢复默认的确认弹窗；没有弹窗时返回 false。
func (s *UI) CancelConfirm() bool {
	if s.confirmWindow == nil {
		return false
	}
	s.closeConfirm()
	return true
}

// Reset 重置本地值（重新进入设置页时调用）
func (s *UI) Reset() {
	s.closeConfirm()
	s.loadFromSettings(config.G)
	s.applyAudioVolumes()
	s.tab = tabGame
	s.pendingRebuild = false
	s.buildUI()
}

// ReloadLanguage 按当前语言重建界面控件。
func (s *UI) ReloadLanguage() {
	s.localLanguage = i18n.CurrentLanguage()
	s.pendingRebuild = true
}

func (s *UI) loadFromSettings(settings *config.GameSettings) {
	if settings == nil {
		settings = config.NewDefaultGameSettings()
	}
	s.localValue = settings.SpeedMultiplier
	s.localLanguage = i18n.NormalizeLanguage(settings.Language)
	s.localBGMVolume = settings.ResolvedBGMVolume()
	s.localEffectsVolume = settings.ResolvedEffectsVolume()
	s.localAI = settings.AI
	s.localFog = settings.EnableFogOfWar
	s.localMinimap = settings.EnableMinimap
	s.localRam = settings.EnableRamming
	s.localLand = settings.EnableLandAirfield
	s.localHitEffects = settings.DisplayHitEffects
	s.localDamageNumbers = settings.DisplayDamageNumber
	s.localShipState = settings.ForceDisplayState
	s.localWake = settings.ResolvedWakeTrail()
	s.backPressed = false
}

// selectSpeed 选择一个速度倍率。
func (s *UI) selectSpeed(value float64) {
	s.localValue = value
}

func (s *UI) selectLanguage(value i18n.Language) {
	s.localLanguage = value
}

func fogSetting() bool {
	if config.G == nil {
		return false
	}
	return config.G.EnableFogOfWar
}

// minimapSetting 小地图默认开启，配置还没加载时也按开启处理。
func minimapSetting() bool {
	if config.G == nil {
		return true
	}
	return config.G.EnableMinimap
}

// hitEffectsSetting 命中特效默认开启，配置还没加载时也按开启处理。
func hitEffectsSetting() bool {
	if config.G == nil {
		return true
	}
	return config.G.DisplayHitEffects
}

// damageNumberSetting 伤害数字默认开启，配置还没加载时也按开启处理。
func damageNumberSetting() bool {
	if config.G == nil {
		return true
	}
	return config.G.DisplayDamageNumber
}

// displayStateSetting 常显战舰状态默认开启，配置还没加载时也按开启处理。
func displayStateSetting() bool {
	if config.G == nil {
		return true
	}
	return config.G.ForceDisplayState
}

// speedOptionIndex 返回当前 localValue 匹配的速度选项索引，不匹配时返回 -1
func (s *UI) speedOptionIndex() int {
	for i, opt := range speedOptions {
		if math.Abs(s.localValue-opt.Value) < 0.001 {
			return i
		}
	}
	return -1
}

// 绘制羊皮纸背景
func (s *UI) drawParchmentBg(screen *ebiten.Image) {
	bg := bgImg.MissionWindowParchment
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()
	scaleX := float64(screen.Bounds().Dx()) / float64(w)
	scaleY := float64(screen.Bounds().Dy()) / float64(h)
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(bg, opts)
}

func newFace(size int, source *text.GoTextFaceSource) *text.Face {
	face := text.Face(&text.GoTextFace{Source: source, Size: float64(size)})
	return &face
}

func newLabel(value string, face *text.Face, clr color.Color) *widget.Label {
	return widget.NewLabel(
		widget.LabelOpts.Text(value, face, &widget.LabelColor{Idle: clr, Disabled: clr}),
	)
}

// newWrappedLabel 按最大宽度换行，用于确认弹窗里可能较长的说明文案。
func newWrappedLabel(value string, face *text.Face, clr color.Color, maxWidth float64) *widget.Label {
	return widget.NewLabel(
		widget.LabelOpts.TextOpts(widget.TextOpts.MaxWidth(maxWidth)),
		widget.LabelOpts.Text(value, face, &widget.LabelColor{Idle: clr, Disabled: clr}),
	)
}

func (s *UI) buildUI() {
	s.metrics = newSettingsMetrics(layout.NewScreenLayout(), maxSettingsRows)

	titleFace := newFace(s.metrics.titleSize, font.LocalizedUI(font.Hang))
	labelFace := newFace(s.metrics.labelSize, font.LocalizedUI(font.Kai))
	hintFace := newFace(s.metrics.hintSize, font.LocalizedUI(font.Kai))
	buttonFace := newFace(s.metrics.buttonSize, font.LocalizedUI(font.Kai))
	valueFace := newFace(s.metrics.valueSize, font.LocalizedUI(font.Kai))
	// 普通下拉只显示当前语言，直接沿用当前语言字体；语言名下拉按语言分别选字体。
	comboFace := buttonFace

	// 下拉框宽度至少覆盖最长的关闭态文案，保证不同语言不截断。
	s.metrics.controlW = max(s.metrics.controlW, closedControlWidth(*comboFace, s.metrics.buttonSize))
	s.metrics.sliderW = max(
		s.metrics.controlW-s.metrics.valueW-s.metrics.valueGap,
		int(math.Round(80*s.metrics.scale)),
	)
	boxWidth := s.metrics.controlW
	boxHeight := s.metrics.controlH

	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	mainPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colorPanelBg)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	root.AddChild(mainPanel)
	mainPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
		StretchVertical:    true,
	}

	title := newLabel(i18n.Text(i18n.MsgSettingsTitle), titleFace, colorx.White)
	mainPanel.AddChild(title)
	title.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: s.metrics.panelX, Top: s.metrics.titleY},
	}

	tabs := s.buildTabs(buttonFace)
	mainPanel.AddChild(tabs)
	tabs.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: s.metrics.panelX, Top: s.metrics.tabsY},
	}

	panel := s.buildSectionPanel(hintFace, labelFace, comboFace, valueFace, boxWidth, boxHeight)
	mainPanel.AddChild(panel)
	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: s.metrics.panelX, Top: s.metrics.panelY},
	}

	footer := s.buildFooter(buttonFace)
	mainPanel.AddChild(footer)
	footer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: &widget.Insets{
			Right: max(s.metrics.screenW-s.metrics.panelX-s.metrics.panelW, 0),
			Top:   s.metrics.footerY,
		},
	}

	s.container = root
}

func (s *UI) buildTabs(face *text.Face) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(math.Round(10*s.metrics.scale))),
		)),
	)

	tabs := []struct {
		id    settingsTab
		label i18n.MessageID
	}{
		{tabGame, i18n.MsgSettingsTabGame},
		{tabAV, i18n.MsgSettingsTabAV},
	}
	widest := 0
	for _, tab := range tabs {
		widest = max(widest, int(text.Advance(i18n.Text(tab.label), *face)+0.5))
	}
	tabW := widest + int(math.Round(52*s.metrics.scale))
	pad := &widget.Insets{
		Left: int(math.Round(22 * s.metrics.scale)), Right: int(math.Round(22 * s.metrics.scale)),
		Top: int(math.Round(6 * s.metrics.scale)), Bottom: int(math.Round(6 * s.metrics.scale)),
	}

	for _, tab := range tabs {
		tab := tab
		selected := s.tab == tab.id
		idle := color.RGBA{R: 26, G: 23, B: 19, A: 235}
		hover := color.RGBA{R: 48, G: 42, B: 34, A: 255}
		pressed := color.RGBA{R: 18, G: 16, B: 14, A: 255}
		textColor := &widget.ButtonTextColor{
			Idle: colorx.White, Hover: colorx.Gold, Pressed: colorx.Silver,
			Disabled: colorMuted,
		}
		if selected {
			idle, hover, pressed = colorSelBg, colorSelBg, color.RGBA{R: 218, G: 190, B: 132, A: 255}
			textColor = &widget.ButtonTextColor{
				Idle: colorx.Black, Hover: colorx.Black, Pressed: colorx.Black,
				Disabled: colorMuted,
			}
		}
		button := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(tabW, s.metrics.tabH)),
			widget.ButtonOpts.Image(&widget.ButtonImage{
				Idle:    image.NewBorderedNineSliceColor(idle, colorAccent, 1),
				Hover:   image.NewBorderedNineSliceColor(hover, colorAccent, 1),
				Pressed: image.NewBorderedNineSliceColor(pressed, colorAccent, 1),
			}),
			widget.ButtonOpts.Text(i18n.Text(tab.label), face, textColor),
			widget.ButtonOpts.TextPadding(pad),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
				if s.tab == tab.id {
					return
				}
				s.tab = tab.id
				s.pendingRebuild = true
			}),
		)
		row.AddChild(button)
	}
	return row
}

func (s *UI) buildSectionPanel(
	hintFace, labelFace, comboFace, valueFace *text.Face,
	boxWidth, boxHeight int,
) *widget.Container {
	panel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(s.metrics.panelW, s.metrics.panelH)),
		widget.ContainerOpts.BackgroundImage(
			image.NewBorderedNineSliceColor(colorCardBg, colorAccent, 2),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// 页签已经标明分类，卡片内不再重复标题，只保留一行内容说明。
	hintMsg := i18n.MsgSettingsGameHint
	if s.tab == tabAV {
		hintMsg = i18n.MsgSettingsAVHint
	}

	hint := newLabel(i18n.Text(hintMsg), hintFace, colorMuted)
	panel.AddChild(hint)
	hint.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: s.metrics.panelPad, Top: s.metrics.panelPad},
	}

	rows := s.buildRows(labelFace, comboFace, valueFace, boxWidth, boxHeight)
	panel.AddChild(rows)
	rows.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding: &widget.Insets{
			Left: s.metrics.panelPad,
			Top:  s.metrics.panelPad + s.metrics.hintH + s.metrics.sectionGap,
		},
	}
	return panel
}

func (s *UI) buildRows(
	labelFace, comboFace, valueFace *text.Face, boxWidth, boxHeight int,
) *widget.Container {
	rows := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(s.metrics.rowGap),
		)),
	)

	switch s.tab {
	case tabAV:
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsBGMVolume,
				s.valueSlider(
					valueFace,
					0,
					1,
					s.localBGMVolume,
					formatPercent,
					func(v float64) {
						s.localBGMVolume = v
						// 拖动时立即试听，取消/返回时会由 Reset 恢复已保存音量。
						audio.SetBGMVolume(v)
					},
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsEffectsVolume,
				s.valueSlider(
					valueFace,
					0,
					1,
					s.localEffectsVolume,
					formatPercent,
					func(v float64) {
						s.localEffectsVolume = v
						audio.SetEffectsVolume(v)
					},
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsHitEffects,
				s.toggleCombo(
					s.localHitEffects,
					func(v bool) { s.localHitEffects = v },
					comboFace,
					boxWidth,
					boxHeight,
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsDamageNumbers,
				s.toggleCombo(
					s.localDamageNumbers,
					func(v bool) { s.localDamageNumbers = v },
					comboFace,
					boxWidth,
					boxHeight,
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsWakeLength,
				s.valueSlider(
					valueFace,
					config.WakeTrailLengthMin,
					config.WakeTrailLengthMax,
					s.localWake.Length,
					formatScale,
					func(v float64) { s.localWake.Length = v },
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsWakeWidth,
				s.valueSlider(
					valueFace,
					config.WakeTrailWidthMin,
					config.WakeTrailWidthMax,
					s.localWake.Width,
					formatScale,
					func(v float64) { s.localWake.Width = v },
				),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsWakeOpacity,
				s.valueSlider(
					valueFace,
					config.WakeTrailOpacityMin,
					config.WakeTrailOpacityMax,
					s.localWake.Opacity,
					formatPercent,
					func(v float64) { s.localWake.Opacity = v },
				),
				labelFace,
			),
		)
	default:
		rows.AddChild(s.settingRow(i18n.MsgSettingsSpeed, s.speedCombo(comboFace, boxWidth, boxHeight), labelFace))
		rows.AddChild(
			s.settingRow(i18n.MsgSettingsLanguage, s.languageCombo(boxWidth, boxHeight), labelFace),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsLandAirfield,
				s.toggleCombo(s.localLand, func(v bool) { s.localLand = v }, comboFace, boxWidth, boxHeight),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(i18n.MsgSettingsAI, s.strategyCombo(comboFace, boxWidth, boxHeight), labelFace),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsFogOfWar,
				s.toggleCombo(s.localFog, func(v bool) { s.localFog = v }, comboFace, boxWidth, boxHeight),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsMinimap,
				s.toggleCombo(s.localMinimap, func(v bool) { s.localMinimap = v }, comboFace, boxWidth, boxHeight),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsRamming,
				s.toggleCombo(s.localRam, func(v bool) { s.localRam = v }, comboFace, boxWidth, boxHeight),
				labelFace,
			),
		)
		rows.AddChild(
			s.settingRow(
				i18n.MsgSettingsShipState,
				s.toggleCombo(
					s.localShipState,
					func(v bool) { s.localShipState = v },
					comboFace,
					boxWidth,
					boxHeight,
				),
				labelFace,
			),
		)
	}

	return rows
}

func (s *UI) settingRow(
	labelMsg i18n.MessageID,
	control widget.PreferredSizeLocateableWidget,
	labelFace *text.Face,
) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			s.metrics.panelW-2*s.metrics.panelPad,
			s.metrics.rowH,
		)),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colorRowBg)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	label := newLabel(i18n.Text(labelMsg), labelFace, colorx.White)
	row.AddChild(label)
	label.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding:            &widget.Insets{Left: s.metrics.rowPadX},
	}
	row.AddChild(control)
	control.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding:            &widget.Insets{Right: s.metrics.rowPadX},
	}
	return row
}

func (s *UI) speedCombo(face *text.Face, boxWidth, boxHeight int) *widget.ListComboButton {
	selectedIndex := s.speedOptionIndex()
	if selectedIndex < 0 {
		selectedIndex = 1
	}
	speedEntries := make([]any, len(speedOptions))
	for idx := range speedOptions {
		speedEntries[idx] = speedOptions[idx]
	}
	return newSettingsCombo(
		speedEntries,
		speedOptions[selectedIndex],
		face,
		boxWidth,
		boxHeight,
		func(entry any) string { return i18n.Text(entry.(speedOption).Label) },
		func(entry any) string { return i18n.Text(entry.(speedOption).Label) },
		func(entry any) { s.selectSpeed(entry.(speedOption).Value) },
	)
}

func (s *UI) languageCombo(boxWidth, boxHeight int) *widget.ComboButton {
	border := colorAccent
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewBorderedNineSliceColor(color.RGBA{R: 20, G: 22, B: 25, A: 250}, border, 2),
		Hover:   image.NewBorderedNineSliceColor(color.RGBA{R: 45, G: 42, B: 35, A: 255}, colorx.Silver, 2),
		Pressed: image.NewBorderedNineSliceColor(color.RGBA{R: 12, G: 14, B: 17, A: 255}, colorx.Silver, 2),
	}
	textColor := &widget.ButtonTextColor{
		Idle: colorx.White, Hover: colorx.White, Pressed: colorx.White,
		Disabled: colorMuted,
	}
	menu := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewBorderedNineSliceColor(color.RGBA{R: 13, G: 15, B: 18, A: 255}, border, 2),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Top: 4, Bottom: 4}),
		)),
	)
	var combo *widget.ComboButton
	for _, lang := range i18n.SupportedLanguages() {
		lang := lang
		selected := lang == s.localLanguage
		rowColor := &widget.ButtonTextColor{
			Idle: colorx.White, Hover: colorx.Gold, Pressed: colorx.Gold,
			Disabled: colorMuted,
		}
		if selected {
			rowColor.Idle = colorx.Gold
		}
		row := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(boxWidth, boxHeight)),
			widget.ButtonOpts.Image(menuRowImage(selected)),
			widget.ButtonOpts.Text(lang.NativeName(), s.languageNameFace(lang), rowColor),
			widget.ButtonOpts.TextPadding(&widget.Insets{Left: 18, Right: 18, Top: 8, Bottom: 8}),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
				s.selectLanguage(lang)
				if combo != nil {
					combo.ContentVisible = false
				}
				s.pendingRebuild = true
			}),
		)
		menu.AddChild(row)
	}
	combo = widget.NewComboButton(
		widget.ComboButtonOpts.MaxContentHeight(boxHeight*5),
		widget.ComboButtonOpts.Content(menu),
		widget.ComboButtonOpts.ButtonOpts(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(boxWidth, boxHeight),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{MaxWidth: boxWidth, MaxHeight: boxHeight}),
			),
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(s.localLanguage.NativeName(), s.languageNameFace(s.localLanguage), textColor),
			widget.ButtonOpts.TextPadding(&widget.Insets{Left: 18, Right: 18, Top: 8, Bottom: 8}),
		),
	)
	return combo
}

// languageNameFace 返回语言自称应使用的字体：简体中文楷体、繁体中文 jf open 粉圓、英文 JetBrains、俄文 Golos、日文 Zen Kaku。
func (s *UI) languageNameFace(lang i18n.Language) *text.Face {
	return newFace(s.metrics.buttonSize, font.ForLanguage(lang, font.Kai))
}

func (s *UI) toggleCombo(
	selected bool,
	apply func(bool),
	face *text.Face,
	boxWidth, boxHeight int,
) *widget.ListComboButton {
	choice := toggleOptions[0]
	if selected {
		choice = toggleOptions[1]
	}
	entries := make([]any, len(toggleOptions))
	for idx := range toggleOptions {
		entries[idx] = toggleOptions[idx]
	}
	return newSettingsCombo(
		entries,
		choice,
		face,
		boxWidth,
		boxHeight,
		func(entry any) string { return i18n.Text(entry.(toggleOption).Label) },
		func(entry any) string { return i18n.Text(entry.(toggleOption).Label) },
		func(entry any) { apply(entry.(toggleOption).Value) },
	)
}

func (s *UI) valueSlider(
	face *text.Face,
	minValue, maxValue, current float64,
	format func(float64) string,
	apply func(float64),
) *widget.Container {
	valueLabel := newLabel(format(current), face, colorAccent)
	valueLabel.GetWidget().LayoutData = widget.RowLayoutData{
		Position: widget.RowLayoutPositionCenter,
		MaxWidth: s.metrics.valueW,
	}
	valueLabel.GetWidget().MinWidth = s.metrics.valueW
	valueLabel.GetWidget().MinHeight = s.metrics.controlH

	handle := &widget.ButtonImage{
		Idle:    image.NewBorderedNineSliceColor(colorAccent, colorx.White, 1),
		Hover:   image.NewBorderedNineSliceColor(colorx.White, colorAccent, 1),
		Pressed: image.NewBorderedNineSliceColor(colorAccent, colorx.Black, 1),
	}
	track := &widget.SliderTrackImage{
		Idle: image.NewBorderedNineSliceColor(
			color.RGBA{R: 45, G: 40, B: 33, A: 255}, color.RGBA{R: 130, G: 112, B: 82, A: 255}, 1,
		),
		Hover: image.NewBorderedNineSliceColor(
			color.RGBA{R: 62, G: 54, B: 42, A: 255}, colorAccent, 1,
		),
	}
	slider := widget.NewSlider(
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(s.metrics.sliderW, s.metrics.handle),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
				MaxWidth: s.metrics.sliderW,
			}),
		),
		widget.SliderOpts.MinMax(
			int(math.Round(minValue*100)),
			int(math.Round(maxValue*100)),
		),
		widget.SliderOpts.InitialCurrent(int(math.Round(current*100))),
		widget.SliderOpts.MinHandleSize(s.metrics.handle),
		widget.SliderOpts.FixedHandleSize(s.metrics.handle),
		widget.SliderOpts.Images(track, handle),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			value := float64(args.Current) / 100
			apply(value)
			valueLabel.Label = format(value)
		}),
	)

	control := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			s.metrics.sliderW+s.metrics.valueGap+s.metrics.valueW,
			s.metrics.controlH,
		)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(s.metrics.valueGap),
		)),
	)
	control.AddChild(slider)
	control.AddChild(valueLabel)
	return control
}

func formatScale(value float64) string { return fmt.Sprintf("%.2f×", value) }

func formatPercent(value float64) string { return fmt.Sprintf("%.0f%%", value*100) }

func (s *UI) buildFooter(face *text.Face) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(math.Round(14*s.metrics.scale))),
		)),
	)
	row.AddChild(s.footerButton(i18n.MsgSettingsReset, face, false, func() {
		s.openResetConfirm()
	}))
	row.AddChild(s.footerButton(i18n.MsgSettingsCancel, face, false, func() {
		s.backPressed = true
	}))
	row.AddChild(s.footerButton(i18n.MsgSettingsSave, face, true, func() {
		s.save()
	}))
	return row
}

func (s *UI) footerButton(
	labelMsg i18n.MessageID,
	face *text.Face,
	primary bool,
	clicked func(),
) *widget.Button {
	idle := color.RGBA{R: 26, G: 23, B: 19, A: 240}
	hover := color.RGBA{R: 48, G: 42, B: 34, A: 255}
	pressed := color.RGBA{R: 18, G: 16, B: 14, A: 255}
	textColor := &widget.ButtonTextColor{
		Idle: colorx.White, Hover: colorx.Gold, Pressed: colorx.Silver, Disabled: colorMuted,
	}
	if primary {
		idle, hover, pressed = colorSelBg, color.RGBA{
			R: 218,
			G: 190,
			B: 132,
			A: 255,
		}, color.RGBA{
			R: 160,
			G: 132,
			B: 76,
			A: 255,
		}
		textColor = &widget.ButtonTextColor{
			Idle: colorx.Black, Hover: colorx.Black, Pressed: colorx.Black, Disabled: colorMuted,
		}
	}
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			int(math.Round(120*s.metrics.scale)),
			s.metrics.footerH,
		)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewBorderedNineSliceColor(idle, colorAccent, 1),
			Hover:   image.NewBorderedNineSliceColor(hover, colorAccent, 1),
			Pressed: image.NewBorderedNineSliceColor(pressed, colorAccent, 1),
		}),
		widget.ButtonOpts.Text(i18n.Text(labelMsg), face, textColor),
		widget.ButtonOpts.TextPadding(&widget.Insets{
			Left: int(math.Round(20 * s.metrics.scale)), Right: int(math.Round(20 * s.metrics.scale)),
			Top: int(math.Round(8 * s.metrics.scale)), Bottom: int(math.Round(8 * s.metrics.scale)),
		}),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			clicked()
		}),
	)
}

// openResetConfirm 打开恢复默认的二次确认弹窗。没有窗口宿主时直接执行恢复。
func (s *UI) openResetConfirm() {
	if s.confirmWindow != nil {
		return
	}
	if s.windowHost == nil {
		s.restoreDefaults()
		return
	}

	screen := layout.NewScreenLayout()
	sw, sh := screen.Width, screen.Height
	if sw <= 0 || sh <= 0 {
		sw, sh = 1280, 720
	}
	scale := s.metrics.scale
	if scale <= 0 {
		scale = 1
	}
	pad := int(math.Round(26 * scale))
	gap := int(math.Round(16 * scale))
	maxCardW := sw - int(math.Round(48*scale))
	if maxCardW < 240 {
		maxCardW = sw - 16
	}
	cardW := min(int(math.Round(560*scale)), maxCardW)

	titleFace := newFace(s.metrics.sectionSize, font.LocalizedUI(font.Hang))
	labelFace := newFace(s.metrics.labelSize, font.LocalizedUI(font.Kai))
	buttonFace := newFace(s.metrics.buttonSize, font.LocalizedUI(font.Kai))

	card := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(cardW, 0)),
		widget.ContainerOpts.BackgroundImage(
			image.NewBorderedNineSliceColor(colorCardBg, colorAccent, 2),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{
				Left: pad, Right: pad, Top: pad, Bottom: pad,
			}),
			widget.RowLayoutOpts.Spacing(gap),
		)),
	)
	card.AddChild(newLabel(i18n.Text(i18n.MsgSettingsResetTitle), titleFace, colorAccent))
	card.AddChild(newWrappedLabel(
		i18n.Text(i18n.MsgSettingsResetMessage),
		labelFace,
		colorx.White,
		float64(cardW-2*pad),
	))

	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(math.Round(14*s.metrics.scale))),
		)),
	)
	buttonRow.AddChild(s.footerButton(i18n.MsgSettingsCancel, buttonFace, false, func() {
		s.closeConfirm()
	}))
	buttonRow.AddChild(s.footerButton(i18n.MsgSettingsConfirmReset, buttonFace, true, func() {
		s.restoreDefaults()
	}))
	buttonArea := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(cardW-2*pad, s.metrics.footerH),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	buttonArea.AddChild(buttonRow)
	buttonRow.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	card.AddChild(buttonArea)

	overlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(color.RGBA{A: 170}),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	overlay.AddChild(card)
	card.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	win := widget.NewWindow(
		widget.WindowOpts.Contents(overlay),
		widget.WindowOpts.Modal(),
		widget.WindowOpts.CloseMode(widget.NONE),
		widget.WindowOpts.Location(stdimg.Rect(0, 0, sw, sh)),
	)
	s.confirmWindow = win
	s.windowHost.AddWindow(win)
}

// closeConfirm 关闭确认弹窗，底层设置页保持原状。
func (s *UI) closeConfirm() {
	if s.confirmWindow == nil {
		return
	}
	s.confirmWindow.Close()
	s.confirmWindow = nil
}

// restoreDefaults 把全部设置恢复为默认值并立即保存，随后关闭设置页。
func (s *UI) restoreDefaults() {
	s.closeConfirm()
	defaults := config.NewDefaultGameSettings()
	config.G = defaults
	s.loadFromSettings(defaults)
	s.applyAudioVolumes()
	_ = config.SaveGameSettings()
	s.backPressed = true
}

func (s *UI) save() {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	config.G.SpeedMultiplier = s.localValue
	config.G.Language = string(s.localLanguage)
	config.G.SetBGMVolume(s.localBGMVolume)
	config.G.SetEffectsVolume(s.localEffectsVolume)
	config.G.AI = s.localAI
	config.G.EnableFogOfWar = s.localFog
	config.G.EnableMinimap = s.localMinimap
	config.G.EnableRamming = s.localRam
	config.G.EnableLandAirfield = s.localLand
	config.G.DisplayHitEffects = s.localHitEffects
	config.G.DisplayDamageNumber = s.localDamageNumbers
	config.G.ForceDisplayState = s.localShipState
	config.G.WakeTrail = s.localWake.Normalize()
	s.applyAudioVolumes()
	_ = config.SaveGameSettings()
	s.backPressed = true
}

// applyAudioVolumes 把本地音量同步到音频层；进入/重置/保存设置时调用。
func (s *UI) applyAudioVolumes() {
	audio.SetBGMVolume(s.localBGMVolume)
	audio.SetEffectsVolume(s.localEffectsVolume)
}

type strategyOption struct {
	label i18n.MessageID
	flag  *bool
}

func closedControlWidth(comboFace text.Face, buttonSize int) int {
	labels := []string{"5/5"}
	for _, option := range speedOptions {
		labels = append(labels, i18n.Text(option.Label))
	}
	widest := 0
	for _, label := range labels {
		width := int(text.Advance(label, comboFace) + 0.5)
		if width > widest {
			widest = width
		}
	}
	for _, lang := range i18n.SupportedLanguages() {
		face := *newFace(buttonSize, font.ForLanguage(lang, font.Kai))
		widest = max(widest, int(text.Advance(lang.NativeName(), face)+0.5))
	}
	return widest + controlPadX
}

func (s *UI) strategyCombo(face *text.Face, boxWidth, boxHeight int) *widget.ComboButton {
	options := []strategyOption{
		{i18n.MsgAIStrategyAttack, &s.localAI.Attack},
		{i18n.MsgAIStrategyDefend, &s.localAI.Defend},
		{i18n.MsgAIStrategyCounter, &s.localAI.Counter},
		{i18n.MsgAIStrategyRaid, &s.localAI.Raid},
		{i18n.MsgAIStrategyScout, &s.localAI.Scout},
	}
	border := colorAccent
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewBorderedNineSliceColor(color.RGBA{R: 20, G: 22, B: 25, A: 250}, border, 2),
		Hover:   image.NewBorderedNineSliceColor(color.RGBA{R: 45, G: 42, B: 35, A: 255}, colorx.Silver, 2),
		Pressed: image.NewBorderedNineSliceColor(color.RGBA{R: 12, G: 14, B: 17, A: 255}, colorx.Silver, 2),
	}
	textColor := &widget.ButtonTextColor{
		Idle: colorx.White, Hover: colorx.White, Pressed: colorx.White,
		Disabled: colorMuted,
	}
	rowOff := menuRowImage(false)
	rowOn := menuRowImage(true)
	menu := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewBorderedNineSliceColor(color.RGBA{R: 13, G: 15, B: 18, A: 255}, border, 2),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Top: 4, Bottom: 4}),
		)),
	)
	var combo *widget.ComboButton
	for _, option := range options {
		option := option
		rowColor := &widget.ButtonTextColor{
			Idle: colorx.White, Hover: colorx.Gold, Pressed: colorx.Gold,
			Disabled: colorMuted,
		}
		var row *widget.Button
		row = widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(boxWidth, boxHeight)),
			widget.ButtonOpts.Image(menuRowImage(*option.flag)),
			widget.ButtonOpts.Text(i18n.Text(option.label), face, rowColor),
			widget.ButtonOpts.TextPadding(&widget.Insets{Left: 18, Right: 18, Top: 8, Bottom: 8}),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
				*option.flag = !*option.flag
				if *option.flag {
					row.SetImage(rowOn)
					rowColor.Idle = colorx.Gold
				} else {
					row.SetImage(rowOff)
					rowColor.Idle = colorx.White
				}
				combo.SetLabel(s.strategySummary())
			}),
		)
		if *option.flag {
			rowColor.Idle = colorx.Gold
		}
		menu.AddChild(row)
	}
	combo = widget.NewComboButton(
		widget.ComboButtonOpts.MaxContentHeight(boxHeight*5),
		widget.ComboButtonOpts.Content(menu),
		widget.ComboButtonOpts.ButtonOpts(
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(boxWidth, boxHeight),
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{MaxWidth: boxWidth, MaxHeight: boxHeight}),
			),
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(s.strategySummary(), face, textColor),
			widget.ButtonOpts.TextPadding(&widget.Insets{Left: 18, Right: 18, Top: 12, Bottom: 12}),
		),
	)
	return combo
}

func (s *UI) strategySummary() string {
	selected := 0
	for _, option := range []strategyOption{
		{i18n.MsgAIStrategyAttack, &s.localAI.Attack},
		{i18n.MsgAIStrategyDefend, &s.localAI.Defend},
		{i18n.MsgAIStrategyCounter, &s.localAI.Counter},
		{i18n.MsgAIStrategyRaid, &s.localAI.Raid},
		{i18n.MsgAIStrategyScout, &s.localAI.Scout},
	} {
		if option.flag != nil && *option.flag {
			selected++
		}
	}
	return fmt.Sprintf("%d/%d", selected, strategyCount)
}

func menuRowImage(selected bool) *widget.ButtonImage {
	if selected {
		on := image.NewNineSliceColor(color.RGBA{R: 76, G: 65, B: 47, A: 255})
		return &widget.ButtonImage{Idle: on, Hover: on, Pressed: on}
	}
	idle := image.NewNineSliceColor(color.RGBA{R: 20, G: 22, B: 25, A: 250})
	hover := image.NewNineSliceColor(color.RGBA{R: 45, G: 42, B: 35, A: 255})
	return &widget.ButtonImage{Idle: idle, Hover: hover, Pressed: hover}
}

func newSettingsCombo(
	entries []any,
	selected any,
	face *text.Face,
	boxWidth, boxHeight int,
	buttonLabel func(any) string,
	entryLabel func(any) string,
	selectedHandler func(any),
) *widget.ListComboButton {
	border := colorAccent
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewBorderedNineSliceColor(color.RGBA{R: 20, G: 22, B: 25, A: 250}, border, 2),
		Hover:   image.NewBorderedNineSliceColor(color.RGBA{R: 45, G: 42, B: 35, A: 255}, colorx.Silver, 2),
		Pressed: image.NewBorderedNineSliceColor(color.RGBA{R: 12, G: 14, B: 17, A: 255}, colorx.Silver, 2),
	}
	textColor := &widget.ButtonTextColor{
		Idle: colorx.White, Hover: colorx.White, Pressed: colorx.White,
		Disabled: colorMuted,
	}
	listImage := &widget.ScrollContainerImage{
		Idle: image.NewBorderedNineSliceColor(color.RGBA{R: 13, G: 15, B: 18, A: 255}, border, 2),
		Mask: image.NewNineSliceColor(colorx.White),
	}

	// 选项数量少无需滚动条，配置透明滑块使其不可见
	transparent := image.NewNineSliceColor(color.RGBA{A: 0})
	sliderHandleImg := &widget.ButtonImage{
		Idle:    transparent,
		Hover:   transparent,
		Pressed: transparent,
	}
	sliderTrackImg := &widget.SliderTrackImage{
		Idle:  transparent,
		Hover: transparent,
	}
	fixedHandleSize := 0

	return widget.NewListComboButton(
		widget.ListComboButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(boxWidth, boxHeight),
		),
		widget.ListComboButtonOpts.Entries(entries),
		widget.ListComboButtonOpts.InitialEntry(selected),
		widget.ListComboButtonOpts.ButtonParams(&widget.ButtonParams{
			Image:       buttonImage,
			TextPadding: &widget.Insets{Left: 18, Right: 18, Top: 8, Bottom: 8},
			MinSize:     &stdimg.Point{X: boxWidth, Y: boxHeight},
		}),
		widget.ListComboButtonOpts.Text(face, nil, textColor),
		widget.ListComboButtonOpts.ListParams(&widget.ListParams{
			MinSize:              &stdimg.Point{X: boxWidth, Y: 0},
			ScrollContainerImage: listImage,
			ScrollContainerPadding: &widget.Insets{
				Left: 2, Right: 2, Top: 2, Bottom: 2,
			},
			EntryFace: face,
			EntryColor: &widget.ListEntryColor{
				Unselected:                 colorx.White,
				Selected:                   colorx.Silver,
				DisabledUnselected:         colorx.Gray,
				DisabledSelected:           colorx.Gray,
				SelectingBackground:        color.RGBA{R: 76, G: 65, B: 47, A: 255},
				SelectedBackground:         color.RGBA{R: 48, G: 43, B: 34, A: 255},
				FocusedBackground:          color.RGBA{R: 58, G: 51, B: 40, A: 255},
				SelectingFocusedBackground: color.RGBA{R: 88, G: 73, B: 50, A: 255},
				SelectedFocusedBackground:  color.RGBA{R: 65, G: 56, B: 42, A: 255},
				DisabledSelectedBackground: color.RGBA{R: 28, G: 28, B: 27, A: 255},
			},
			EntryTextPadding: &widget.Insets{Left: 18, Right: 18, Top: 10, Bottom: 10},
			Slider: &widget.SliderParams{
				HandleImage:     sliderHandleImg,
				TrackImage:      sliderTrackImg,
				FixedHandleSize: &fixedHandleSize,
			},
		}),
		widget.ListComboButtonOpts.EntryLabelFunc(buttonLabel, entryLabel),
		widget.ListComboButtonOpts.EntrySelectedHandler(
			func(args *widget.ListComboButtonEntrySelectedEventArgs) {
				selectedHandler(args.Entry)
			},
		),
		widget.ListComboButtonOpts.MaxContentHeight(boxHeight*5),
	)
}
