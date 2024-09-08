package controller

import (
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 输入处理器
type InputHandler interface {
	Handle(curInstructions map[string]instr.Instruction, misState *state.MissionState) map[string]instr.Instruction
}
