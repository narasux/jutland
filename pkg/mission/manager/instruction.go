package manager

import (
	"log"
	"sort"
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
	// order 是稳定执行顺序；变更时置脏，ExecAll 才重新排序。
	order      []string
	orderDirty bool
}

// NewInstructionSet 创建空指令集
func NewInstructionSet() *InstructionSet {
	return &InstructionSet{
		instructions: map[string]instr.Instruction{},
		orderDirty:   true,
	}
}

// Add 添加指令
func (s *InstructionSet) Add(instr instr.Instruction) {
	s.Lock()
	defer s.Unlock()
	s.instructions[instr.Uid()] = instr
	s.orderDirty = true
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
	s.orderDirty = true
}

// Remove 删除指令
func (s *InstructionSet) Remove(uid string) {
	s.Lock()
	defer s.Unlock()
	delete(s.instructions, uid)
	s.orderDirty = true
}

// RemoveExecuted 删除已执行的指令
func (s *InstructionSet) RemoveExecuted() {
	s.Lock()
	defer s.Unlock()
	for uid, instruction := range s.instructions {
		if instruction.Executed() {
			delete(s.instructions, uid)
			s.orderDirty = true
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

// Get 按 uid 取指令，不存在时返回 nil。
func (s *InstructionSet) Get(uid string) instr.Instruction {
	s.RLock()
	defer s.RUnlock()
	return s.instructions[uid]
}

// ExecAll 按稳定 key 顺序执行所有指令。
// 指令的 Exec 只读写 MissionState，不会回调 InstructionSet，因此这里用写锁
// 重建 order 是安全的；相比每拍复制 map 再排序，只在 Add/Remove 后重排。
func (s *InstructionSet) ExecAll(state *state.MissionState) {
	s.Lock()
	defer s.Unlock()
	if s.orderDirty {
		s.order = s.order[:0]
		for uid := range s.instructions {
			s.order = append(s.order, uid)
		}
		sort.Strings(s.order)
		s.orderDirty = false
	}
	for _, uid := range s.order {
		i := s.instructions[uid]
		if i == nil {
			continue
		}
		if err := i.Exec(state); err != nil {
			log.Printf("Instruction %s exec error: %s\n", i.String(), err)
		}
	}
}
