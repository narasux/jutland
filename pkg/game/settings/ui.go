// 游戏设置界面，使用 ebitenui 实现，风格参考游戏图鉴（羊皮纸）
package settings

import (
	stdimg "image"
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

var colorPanelBg = color.RGBA{R: 48, G: 40, B: 30, A: 100} // 半透明深色面板

// 文字尺寸
const (
	titleFontSize  = 40
	labelFontSize  = 24
	buttonFontSize = 22

	controlWidth = 140
)

type speedOption struct {
	Label i18n.MessageID
	Value float64
}

// 速度选项定义
var speedOptions = []speedOption{
	{i18n.MsgSpeedVerySlow, 0.25},
	{i18n.MsgSpeedSlow, 0.50},
	{i18n.MsgSpeedNormal, 1.00},
	{i18n.MsgSpeedFast, 2.00},
	{i18n.MsgSpeedVeryFast, 4.00},
}

// UI 游戏设置 UI
type UI struct {
	container     widget.Containerer
	localValue    float64 // 本地副本，保存时才写回 config.G
	localLanguage i18n.Language
	backPressed   bool
}

// New 创建设置界面
func New() *UI {
	s := &UI{
		localValue:    config.G.SpeedMultiplier,
		localLanguage: i18n.NormalizeLanguage(config.G.Language),
	}
	s.buildUI()
	return s
}

// Draw 绘制设置界面
func (s *UI) Draw(screen *ebiten.Image) {
	s.drawParchmentBg(screen)
}

// Container 返回由游戏共享 EbitenUI 主实例承载的设置容器。
func (s *UI) Container() widget.Containerer { return s.container }

// BackPressed 返回用户是否点了返回/取消
func (s *UI) BackPressed() bool { return s.backPressed }

// Reset 重置本地值（重新进入设置页时调用）
func (s *UI) Reset() {
	s.localValue = config.G.SpeedMultiplier
	s.localLanguage = i18n.NormalizeLanguage(config.G.Language)
	s.backPressed = false
	s.buildUI()
}

// ReloadLanguage 按当前语言重建界面控件。
func (s *UI) ReloadLanguage() {
	s.localLanguage = i18n.CurrentLanguage()
	s.buildUI()
}

// selectSpeed 选择一个速度倍率。
func (s *UI) selectSpeed(value float64) {
	s.localValue = value
}

func (s *UI) selectLanguage(value i18n.Language) {
	s.localLanguage = value
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

func (s *UI) buildUI() {
	s.backPressed = false

	// 构建主题字体（ebitenui 需要 *text.Face 即指向接口的指针）
	_titleFace := text.Face(&text.GoTextFace{Source: font.LocalizedUI(font.Hang), Size: titleFontSize})
	_labelFace := text.Face(&text.GoTextFace{Source: font.LocalizedUI(font.Kai), Size: labelFontSize})
	_buttonFace := text.Face(&text.GoTextFace{Source: font.LocalizedUI(font.Kai), Size: buttonFontSize})
	titleFace := &_titleFace
	labelFace := &_labelFace
	buttonFace := &_buttonFace

	// 构建 NineSlice 图像
	panelBg := image.NewNineSliceColor(colorPanelBg)
	btnIdleImg := image.NewNineSliceColor(colorx.Black)
	btnHoverImg := image.NewNineSliceColor(colorx.DarkSilver)
	btnPressedImg := image.NewNineSliceColor(colorx.Black)

	// 普通按钮样式
	normalBtnImage := &widget.ButtonImage{
		Idle:    btnIdleImg,
		Hover:   btnHoverImg,
		Pressed: btnPressedImg,
	}
	normalBtnTextColor := &widget.ButtonTextColor{
		Idle:     colorx.White,
		Disabled: color.RGBA{R: 120, G: 110, B: 100, A: 255},
		Hover:    colorx.Gold,
		Pressed:  colorx.DarkSilver,
	}

	// ====== 标题 ======
	titleLabel := widget.NewLabel(
		widget.LabelOpts.Text(
			i18n.Text(i18n.MsgSettingsTitle),
			titleFace,
			&widget.LabelColor{Idle: colorx.White, Disabled: colorx.White},
		),
	)

	// ====== 速度倍率标签 ======
	speedLabel := widget.NewLabel(
		widget.LabelOpts.Text(
			i18n.Text(i18n.MsgSettingsSpeed),
			labelFace,
			&widget.LabelColor{Idle: colorx.White, Disabled: colorx.White},
		),
	)

	// ====== 速度倍率下拉框 ======
	selectedIndex := s.speedOptionIndex()
	speedEntries := make([]any, len(speedOptions))
	for idx := range speedOptions {
		speedEntries[idx] = speedOptions[idx]
	}
	if selectedIndex < 0 {
		selectedIndex = 2
	}
	speedCombo := newSettingsCombo(
		speedEntries,
		speedOptions[selectedIndex],
		buttonFace,
		func(entry any) string { return i18n.Text(entry.(speedOption).Label) + "  ▾" },
		func(entry any) string { return i18n.Text(entry.(speedOption).Label) },
		func(entry any) { s.selectSpeed(entry.(speedOption).Value) },
	)

	languageLabel := widget.NewLabel(
		widget.LabelOpts.Text(
			i18n.Text(i18n.MsgSettingsLanguage),
			labelFace,
			&widget.LabelColor{Idle: colorx.White, Disabled: colorx.White},
		),
	)
	languages := i18n.SupportedLanguages()
	languageEntries := make([]any, len(languages))
	for idx := range languages {
		languageEntries[idx] = languages[idx]
	}
	// 语言列表同时包含中英文，固定使用具备完整拉丁与汉字字形的字体。
	languageFaceValue := text.Face(&text.GoTextFace{Source: font.Kai, Size: buttonFontSize})
	languageCombo := newSettingsCombo(
		languageEntries,
		s.localLanguage,
		&languageFaceValue,
		func(entry any) string { return entry.(i18n.Language).NativeName() + "  ▾" },
		func(entry any) string { return entry.(i18n.Language).NativeName() },
		func(entry any) { s.selectLanguage(entry.(i18n.Language)) },
	)

	// ====== 按钮栏 ======
	saveBtn := widget.NewButton(
		widget.ButtonOpts.Image(normalBtnImage),
		widget.ButtonOpts.Text(i18n.Text(i18n.MsgSettingsSave), buttonFace, normalBtnTextColor),
		widget.ButtonOpts.TextPadding(&widget.Insets{Left: 24, Right: 24, Top: 8, Bottom: 8}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.backPressed = true
			config.G.SpeedMultiplier = s.localValue
			config.G.Language = string(s.localLanguage)
			_ = config.SaveGameSettings()
		}),
	)

	cancelBtn := widget.NewButton(
		widget.ButtonOpts.Image(normalBtnImage),
		widget.ButtonOpts.Text(i18n.Text(i18n.MsgSettingsCancel), buttonFace, normalBtnTextColor),
		widget.ButtonOpts.TextPadding(&widget.Insets{Left: 24, Right: 24, Top: 8, Bottom: 8}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			s.backPressed = true
		}),
	)

	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(40),
		)),
	)
	buttonRow.AddChild(saveBtn)
	buttonRow.AddChild(cancelBtn)

	// ====== 顶部内容（标题 + 速度倍率 + 选项按钮） ======
	topContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(32),
		)),
	)
	topContent.AddChild(titleLabel)
	topContent.AddChild(speedLabel)
	topContent.AddChild(speedCombo)
	topContent.AddChild(languageLabel)
	topContent.AddChild(languageCombo)

	// ====== 底部内容（操作按钮 + 提示） ======
	bottomContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(16),
		)),
	)
	bottomContent.AddChild(buttonRow)

	// ====== 主面板（全屏填充，Anchor 布局） ======
	mainPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(panelBg),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	mainPanel.AddChild(topContent)
	topData := widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: 160, Top: 120},
	}
	topWidget := topContent.GetWidget()
	topWidget.LayoutData = topData

	mainPanel.AddChild(bottomContent)
	bottomData := widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding:            &widget.Insets{Right: 160, Bottom: 100},
	}
	bottomWidget := bottomContent.GetWidget()
	bottomWidget.LayoutData = bottomData

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	rootContainer.AddChild(mainPanel)
	rootData := widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
		StretchVertical:    true,
	}
	rootWidget := mainPanel.GetWidget()
	rootWidget.LayoutData = rootData

	s.container = rootContainer
}

func newSettingsCombo(
	entries []any,
	selected any,
	face *text.Face,
	buttonLabel func(any) string,
	entryLabel func(any) string,
	selectedHandler func(any),
) *widget.ListComboButton {
	border := color.RGBA{R: 212, G: 180, B: 112, A: 255}
	buttonImage := &widget.ButtonImage{
		Idle:    image.NewBorderedNineSliceColor(color.RGBA{R: 20, G: 22, B: 25, A: 250}, border, 2),
		Hover:   image.NewBorderedNineSliceColor(color.RGBA{R: 45, G: 42, B: 35, A: 255}, colorx.Silver, 2),
		Pressed: image.NewBorderedNineSliceColor(color.RGBA{R: 12, G: 14, B: 17, A: 255}, colorx.Silver, 2),
	}
	textColor := &widget.ButtonTextColor{
		Idle: colorx.White, Hover: colorx.White, Pressed: colorx.White,
		Disabled: color.RGBA{R: 120, G: 110, B: 100, A: 255},
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
		widget.ListComboButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(controlWidth, 52)),
		widget.ListComboButtonOpts.Entries(entries),
		widget.ListComboButtonOpts.InitialEntry(selected),
		widget.ListComboButtonOpts.ButtonParams(&widget.ButtonParams{
			Image:       buttonImage,
			TextPadding: &widget.Insets{Left: 18, Right: 18, Top: 12, Bottom: 12},
		}),
		widget.ListComboButtonOpts.Text(face, nil, textColor),
		widget.ListComboButtonOpts.ListParams(&widget.ListParams{
			MinSize:              &stdimg.Point{X: controlWidth, Y: 0},
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
		widget.ListComboButtonOpts.MaxContentHeight(240),
	)
}
