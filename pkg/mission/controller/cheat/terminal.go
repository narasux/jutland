package cheat

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Terminal 作弊器终端
type Terminal struct {
	// 用户输入历史记录
	History []string
	// 缓冲区
	Buffer []string
	// 当前输入内容
	Input strings.Builder
}

// Update ...
func (t *Terminal) Update() {}

// HandleInput 处理输入
func (t *Terminal) HandleInput() error {
	keys := inpututil.AppendJustPressedKeys(nil)
	fmt.Println(keys)
	return nil
}

// NewTerminal 新建终端
func NewTerminal() *Terminal {
	return &Terminal{}
}
