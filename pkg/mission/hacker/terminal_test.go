package hacker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// TestTerminalFillHistoryWalksBackAndForward 验证上下键浏览历史时索引边界与输入内容。
// 场景：玩家在终端按 ↑ 回看命令、按 ↓ 前进；历史为空/索引为负时不能 panic 或残留旧输入。
// 预期：索引 1 取最近一条 "debug"，2 取 "clear"，超过长度夹到最旧 "help"；索引 <=0 清空输入。
// 覆盖：构造三条历史后依次设置 HistoryIndex，调用 fillHistory 并断言 Input.String()。
func TestTerminalFillHistoryWalksBackAndForward(t *testing.T) {
	terminal := &Terminal{History: []string{"help", "clear", "debug"}}

	terminal.HistoryIndex = 1
	terminal.fillHistory()
	require.Equal(t, "debug", terminal.Input.String())

	terminal.HistoryIndex = 2
	terminal.fillHistory()
	require.Equal(t, "clear", terminal.Input.String())

	terminal.HistoryIndex = 99
	terminal.fillHistory()
	require.Equal(t, "help", terminal.Input.String())

	terminal.HistoryIndex = -1
	terminal.fillHistory()
	require.Empty(t, terminal.Input.String())
}

// TestTerminalExecCommandHandlesLocalAndUnknownCommands 验证 clear/help/:set color/:set font/未知命令的分支。
// 场景：玩家输入 clear 清屏、help 查命令、:set color pink 改颜色、:set font italic 改字体，或输入拼错命令。
// 预期：clear 清空 Buffer；help 追加 "Commands:"；颜色/字体指针被替换；未知命令追加 command not found 与空行。
// 覆盖：零值 Terminal 上逐个调用 execCommand，检查 Buffer 文本、Color、Font 指针。
func TestTerminalExecCommandHandlesLocalAndUnknownCommands(t *testing.T) {
	ms := &state.MissionState{}
	terminal := &Terminal{Buffer: []Line{{Text: "old"}}}

	terminal.execCommand(ms, "clear")
	require.Empty(t, terminal.Buffer)

	terminal.execCommand(ms, "help")
	require.NotEmpty(t, terminal.Buffer)
	foundHelp := false
	for _, line := range terminal.Buffer {
		if line.Text == "Commands:" {
			foundHelp = true
			break
		}
	}
	require.True(t, foundHelp, "help should append command list")

	terminal.execCommand(ms, ":set color pink")
	require.Equal(t, colorx.Pink, terminal.Color)

	terminal.execCommand(ms, ":set font italic")
	require.Equal(t, font.JetbrainsMonoItalic, terminal.Font)

	terminal.execCommand(ms, "definitely-not-a-command")
	foundError := false
	for _, line := range terminal.Buffer {
		if line.Text == "command not found: definitely-not-a-command" {
			foundError = true
			break
		}
	}
	require.True(t, foundError, "unknown command should append an error line")
}

// TestTerminalLineStringAddsInputPrefix 验证输入行带 ">>> " 前缀，输出行保持原样。
// 场景：终端缓冲区用 Line.Type 区分玩家输入与命令输出；渲染/复制时依赖 String 加前缀。
// 预期：LineTypeInput → ">>> help"；LineTypeOutput → "plain"。
// 覆盖：两种 LineType 各构造一条 Line，直接比较 String 返回值。
func TestTerminalLineStringAddsInputPrefix(t *testing.T) {
	require.Equal(t, ">>> help", (&Line{Text: "help", Type: LineTypeInput}).String())
	require.Equal(t, "plain", (&Line{Text: "plain", Type: LineTypeOutput}).String())
}
