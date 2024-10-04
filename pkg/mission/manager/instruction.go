package manager

import (
	"log"
	"sync"

	"github.com/samber/lo"

	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// InstructionSet 指令集
type InstructionSet struct {
	sync.RWMutex
	// 指令集合 key 为 objUid + instrName
	// 注：同一对象，只能有一个同名指令（如：战舰不能有两个目标位置）
	instructions map[string]instr.Instruction
}

// NewInstructionSet ...
func NewInstructionSet() *InstructionSet {
	return &InstructionSet{
		instructions: map[string]instr.Instruction{},
	}
}

// Add 添加指令
func (s *InstructionSet) Add(instr instr.Instruction) {
	s.Lock()
	defer s.Unlock()
	s.instructions[instr.Uid()] = instr
}

// Assign 批量添加指令（覆盖合并）
func (s *InstructionSet) Assign(instructions map[string]instr.Instruction) {
	s.Lock()
	defer s.Unlock()
	s.instructions = lo.Assign(s.instructions, instructions)
}

// Remove 删除指令
func (s *InstructionSet) Remove(uid string) {
	s.Lock()
	defer s.Unlock()
	delete(s.instructions, uid)
}

// RemoveExecuted 删除已执行的指令
func (s *InstructionSet) RemoveExecuted() {
	s.Lock()
	defer s.Unlock()
	s.instructions = lo.PickBy(s.instructions, func(key string, instruction instr.Instruction) bool {
		return !instruction.Executed()
	})
}

// Items 获取指令集
func (s *InstructionSet) Items() map[string]instr.Instruction {
	s.RLock()
	defer s.RUnlock()
	return s.instructions
}

// ExecAll 执行所有指令
func (s *InstructionSet) ExecAll(state *state.MissionState) {
	s.RLock()
	defer s.RUnlock()
	for _, i := range s.instructions {
		if err := i.Exec(state); err != nil {
			log.Printf("Instruction %s exec error: %s\n", i.String(), err)
		}
	}
}
