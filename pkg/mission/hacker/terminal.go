package hacker

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/hacker/cheat"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
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
func NewTerminal() *Terminal {
	misLayout := layout.NewScreenLayout()

	return &Terminal{
		ReservedLines: misLayout.Height / (terminalLineSpacing + terminalFontSize) / 5 * 4,
		LineSpacing:   terminalLineSpacing,
		FontSize:      terminalFontSize,
		Font:          font.OpenSansItalic,
		Color:         colorx.Gold,
	}
}

func (t *Terminal) CurInputString() string {
	cursor := lo.Ternary(time.Now().Unix()&1 == 0, "_", "")
	return inputPrefix + t.Input.String() + cursor
}

// Update ...
func (t *Terminal) Update(misState *state.MissionState) {
	keys := inpututil.AppendJustPressedKeys(nil)
	if len(keys) == 0 {
		return
	}
	for _, k := range keys {
		switch k {
		case ebiten.KeyEnter:
			cmd := t.Input.String()
			// 填充缓冲区
			t.Buffer = append(t.Buffer, Line{Text: cmd, Type: LineTypeInput})
			// 如果命令不为空，则执行 & 记录历史
			if cmd != "" {
				t.History = append(t.History, cmd)
				// 执行命令
				t.execCommand(misState, cmd)
				// 清理过长的缓冲区
				if len(t.Buffer) > t.ReservedLines {
					t.Buffer = t.Buffer[len(t.Buffer)-t.ReservedLines:]
				}
				// 重置输入行
				t.Input.Reset()
				t.HistoryIndex = 0
			}
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

// 执行命令
func (t *Terminal) execCommand(misState *state.MissionState, cmd string) {
	switch cmd {
	case ":q", ":wq", "exit", "quit":
		misState.MissionStatus = state.MissionRunning
		return
	case "clear":
		t.Buffer = t.Buffer[:0]
		return
	case "help":
		for _, c := range cheat.Cheats {
			t.Buffer = append(
				t.Buffer, Line{Text: fmt.Sprintf("%s: %s", c.String(), c.Desc()), Type: LineTypeOutput},
			)
		}
		return
	default:
		// TODO 适当封装，提供前缀匹配的方法，不要直接 HasPrefix
		// 修改终端颜色
		if strings.HasPrefix(cmd, ":set color") {
			t.setColorByCmd(cmd)
			return
		}
		// 其他情况下才进行匹配
		for _, c := range cheat.Cheats {
			if c.Match(cmd) {
				log := c.Exec(misState)
				t.Buffer = append(t.Buffer, Line{Text: log, Type: LineTypeOutput})
				return
			}
		}
	}
	// 还没有被执行的命令认为是无效的
	t.Buffer = append(
		t.Buffer, Line{Text: fmt.Sprintf("Command `%s` Not Effect", cmd), Type: LineTypeOutput},
	)
}

func (t *Terminal) setColorByCmd(cmd string) {
	clrName := strings.ReplaceAll(cmd, ":set color", "")

	if clr := colorx.GetColorByName(clrName); clr != nil {
		t.Color = clr
	} else {
		t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("Color `%s` Not Found", clrName), Type: LineTypeOutput})
	}
}
