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
	LineTypeOutput LineType = iota
	LineTypeInput
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
		Font:          font.JetbrainsMono,
		Color:         colorx.Gold,
		Buffer:        []Line{{Text: "Welcome to use Jutland terminal! Type `help` to get started."}},
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
		fallthrough
	case "clear":
		t.Buffer = t.Buffer[:0]
	case "help":
		t.showHelpText()
	case "debug":
		t.showDebugText()
	default:
		// TODO 适当封装，提供前缀匹配的方法，不要直接 HasPrefix
		if strings.HasPrefix(cmd, ":set color") {
			// 修改终端颜色
			t.setColorByCmd(cmd)
		} else if strings.HasPrefix(cmd, ":set font") {
			// 修改终端字体
			t.setFontByCmd(cmd)
		} else {
			match := false
			// 其他情况下才进行匹配，先匹配普通秘籍，再匹配调试用秘籍
			for _, cheats := range [][]cheat.Cheat{cheat.Cheats, cheat.DebugCheats} {
				for _, c := range cheats {
					if c.Match(cmd) {
						t.Buffer = append(t.Buffer, Line{Text: c.Exec(misState), Type: LineTypeOutput})
						match = true
						break
					}
				}
			}
			// 无效的命令
			if !match {
				t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("command not found: %s", cmd)})
			}
			// 添加空行
			t.Buffer = append(t.Buffer, Line{Text: ""})
		}
	}
}

// 输出终端提示
func (t *Terminal) showHelpText() {
	tips := []string{
		"clear  -->  clear terminal buffer",
		":set font <type>  -->  change terminal font, suggestion: [regular, italic]",
		":set color <name>  -->  change terminal color, suggestion: [gold, cyan, yellow, skyblue, pink, silver]",
		":q | :wq | quit | exit  -->  exit terminal",
	}

	t.Buffer = append(t.Buffer, Line{Text: ""}, Line{Text: "Commands:"})
	for _, line := range tips {
		t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("• %s", line)})
	}

	t.Buffer = append(t.Buffer, Line{Text: ""}, Line{Text: "Game Cheats:"})
	for _, c := range cheat.Cheats {
		t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("• %s  -->  %s", c.String(), c.Desc())})
	}
	t.Buffer = append(t.Buffer, Line{Text: ""})
}

// 输出调试信息
func (t *Terminal) showDebugText() {
	t.Buffer = append(
		t.Buffer,
		Line{Text: "debug cheats only for developers"},
		Line{Text: ""},
		Line{Text: "Debug Commands:"},
	)
	for _, c := range cheat.DebugCheats {
		t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("• %s  -->  %s", c.String(), c.Desc())})
	}
	t.Buffer = append(t.Buffer, Line{Text: ""})
}

// 修改终端字体颜色
func (t *Terminal) setColorByCmd(cmd string) {
	clrName := strings.ReplaceAll(cmd, ":set color", "")

	if clr := colorx.GetColorByName(clrName); clr != nil {
		t.Color = clr
	} else {
		t.Buffer = append(t.Buffer, Line{Text: fmt.Sprintf("color not found: %s", clrName)})
	}
}

// 修改终端字体
func (t *Terminal) setFontByCmd(cmd string) {
	fontType := strings.ToLower(
		strings.Trim(strings.ReplaceAll(cmd, ":set font", ""), " "),
	)

	switch fontType {
	case "regular":
		t.Font = font.JetbrainsMono
	case "italic":
		t.Font = font.JetbrainsMonoItalic
	default:
		t.Font = font.JetbrainsMono
	}
}
