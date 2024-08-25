package cheat

import (
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 终端字体大小
const terminalFontSize = 24

// 终端行间距
const terminalLineSpacing = 6

// 输入行前缀
const inputPrefix = ">>> "

type LineType int

const (
	LineTypeInput LineType = iota
	LineTypeOutput
)

// 缓冲区行
type Line struct {
	Text string
	Type LineType
}

// String ...
func (l *Line) String() string {
	return lo.Ternary(l.Type == LineTypeInput, inputPrefix, "") + l.Text
}

// Terminal 作弊器终端
type Terminal struct {
	// 窗口大小
	ReservedLines int
	// 行间距
	LineSpacing float64
	// 字体大小
	FontSize float64
	// 字体
	Font *text.GoTextFaceSource
	// 颜色
	Color color.Color
	// 用户输入历史记录
	History []string
	// 历史命令索引
	HistoryIndex int
	// 缓冲区
	Buffer []Line
	// 当前输入内容
	Input strings.Builder
}

// NewTerminal 新建终端
func NewTerminal(windowHeight int) *Terminal {
	return &Terminal{
		ReservedLines: windowHeight / (terminalLineSpacing + terminalFontSize) / 5 * 4,
		LineSpacing:   terminalLineSpacing,
		FontSize:      terminalFontSize,
		Font:          font.OpenSansItalic,
		Color:         colorx.Green,
	}
}

func (t *Terminal) CurInputString() string {
	cursor := lo.Ternary(time.Now().Unix()&1 == 0, "_", "")
	return inputPrefix + t.Input.String() + cursor
}

// Update ...
func (t *Terminal) Update() {
	keys := inpututil.AppendJustPressedKeys(nil)
	if len(keys) == 0 {
		return
	}
	for _, k := range keys {
		switch k {
		case ebiten.KeyEnter:
			// 回车换行 & 执行命令
			cmd := t.Input.String()
			t.History = append(t.History, cmd)
			t.Buffer = append(t.Buffer, Line{Text: cmd, Type: LineTypeInput})
			t.exec(cmd)
			t.cleanBuffer()
			t.Input.Reset()
		case ebiten.KeyBackspace:
			if t.Input.Len() > 0 {
				str := t.Input.String()
				str = str[0 : len(str)-1]
				t.Input.Reset()
				t.Input.WriteString(str)
			}
		case ebiten.KeyDelete:
			t.Input.Reset()
		case ebiten.KeyArrowUp:
			t.HistoryIndex++
			t.fillHistory()
		case ebiten.KeyArrowDown:
			t.HistoryIndex--
			t.fillHistory()
		default:
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				t.Input.WriteString(keyWithShiftCharMap[k])
			} else {
				t.Input.WriteString(keyCharMap[k])
			}
		}
	}
	return
}

// 填充历史记录
func (t *Terminal) fillHistory() {
	t.Input.Reset()
	historyLength := len(t.History)

	if t.HistoryIndex <= 0 || historyLength == 0 {
		t.HistoryIndex = 0
		return
	}
	t.HistoryIndex = min(t.HistoryIndex, historyLength)
	t.Input.WriteString(t.History[historyLength-t.HistoryIndex])
}

// 清理过多的缓冲区行
func (t *Terminal) cleanBuffer() {
	if len(t.Buffer) > t.ReservedLines {
		t.Buffer = t.Buffer[len(t.Buffer)-t.ReservedLines:]
	}
}

// FIXME 执行命令
func (t *Terminal) exec(cmd string) {
	t.Buffer = append(t.Buffer, Line{Text: "echo -> " + cmd, Type: LineTypeOutput})
}
