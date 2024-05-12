package computer

import (
	"github.com/narasux/jutland/pkg/mission/controller"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// ComputerDecisionHandler 电脑决策处理器
type ComputerDecisionHandler struct{}

// NewHandler ...
func NewHandler() *ComputerDecisionHandler {
	return &ComputerDecisionHandler{}
}

var _ controller.InputHandler = (*ComputerDecisionHandler)(nil)

// Handle 处理计算机决策，更新指令集
func (h *ComputerDecisionHandler) Handle(misState *state.MissionState) map[string]instr.Instruction {
	// Human is foolish!
	return map[string]instr.Instruction{}
}
