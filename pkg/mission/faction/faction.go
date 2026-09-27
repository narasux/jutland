// faction 游戏阵营
package faction

type Player string

const (
	// HumanAlpha 人类玩家一
	HumanAlpha Player = "HA"
	// HumanBeta 人类玩家二
	HumanBeta Player = "HB"
	// ComputerAlpha 电脑 AI 玩家一
	ComputerAlpha Player = "CA"
	// ComputerBeta 电脑 AI 玩家二
	ComputerBeta Player = "CB"
)

// Side 任务配置中的中立阵营，不直接参与运行时控制。
type Side string

const (
	// SideP1 任务配置阵营一
	SideP1 Side = "P1"
	// SideP2 任务配置阵营二
	SideP2 Side = "P2"
)

// IsValid 判断是否为支持的任务配置阵营。
func (s Side) IsValid() bool {
	return s == SideP1 || s == SideP2
}

// RuntimePlayer 将任务配置阵营映射为运行时玩家。
// 玩家选择的阵营映射为 HA，另一侧映射为 CA。
func (s Side) RuntimePlayer(selected Side) Player {
	if s == selected {
		return HumanAlpha
	}
	return ComputerAlpha
}
