package manager

import (
	"log"
	"sync"

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

// NewInstructionSet 创建空指令集
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
	if len(instructions) == 0 {
		return
	}
	s.Lock()
	defer s.Unlock()
	for uid, instruction := range instructions {
		s.instructions[uid] = instruction
	}
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
	for uid, instruction := range s.instructions {
		if instruction.Executed() {
			delete(s.instructions, uid)
		}
	}
}

// Items 获取指令集
func (s *InstructionSet) Items() map[string]instr.Instruction {
	s.RLock()
	defer s.RUnlock()
	return s.instructions
}

// Exists 相同 ID 的指令是否存在
func (s *InstructionSet) Exists(uid string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.instructions[uid]
	return ok
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
