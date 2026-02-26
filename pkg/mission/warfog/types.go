package warfog

// FogState 迷雾状态
type FogState int

const (
	// FogStateUnexplored 未探索区域（完全不可见）
	FogStateUnexplored FogState = iota
	// FogStateExplored 已探索但不在当前视野内（半透明遮罩）
	FogStateExplored
	// FogStateVisible 当前视野内（完全可见）
	FogStateVisible
)

// Alpha 返回迷雾状态对应的透明度（0.0 = 完全透明，1.0 = 完全不透明）
func (s FogState) Alpha() float64 {
	switch s {
	case FogStateUnexplored:
		return 1.0 // 100% 黑色遮罩
	case FogStateExplored:
		return 0.5 // 50% 黑色遮罩
	case FogStateVisible:
		return 0.0 // 无遮罩
	default:
		return 1.0
	}
}
