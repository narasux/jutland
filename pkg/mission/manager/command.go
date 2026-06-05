package manager

// 逐条执行指令（移动/炮击/雷击/建造）
func (m *MissionManager) executeInstructions() {
	m.instructionSet.ExecAll(m.state)
}
