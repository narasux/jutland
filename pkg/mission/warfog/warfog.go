package warfog

// FogOfWar 战争迷雾状态
type FogOfWar struct {
	// 已探索区域网格（每个格子是否被探索过）
	// true = 已探索，false = 未探索
	ExploredGrid [][]bool
	// 当前视野网格（动态计算，基于友方单位位置）
	// true = 在视野内，false = 在视野外
	VisibleGrid [][]bool
	// 地图尺寸（单位：地图格数）
	MapWidth  int
	MapHeight int
	// BlackSheepWall 激活状态
	// true = 全图可见，false = 正常迷雾
	BlackSheepWallActive bool
	// 脏标记：标记是否需要重新计算视野
	dirty bool
}

// NewFogOfWar 创建新的迷雾状态
func NewFogOfWar(width, height int) *FogOfWar {
	exploredGrid := make([][]bool, width)
	visibleGrid := make([][]bool, width)
	for i := range exploredGrid {
		exploredGrid[i] = make([]bool, height)
		visibleGrid[i] = make([]bool, height)
	}

	return &FogOfWar{
		ExploredGrid:         exploredGrid,
		VisibleGrid:          visibleGrid,
		MapWidth:             width,
		MapHeight:            height,
		BlackSheepWallActive: false,
		dirty:                true,
	}
}

// IsExplored 检查指定格子是否已被探索
func (f *FogOfWar) IsExplored(x, y int) bool {
	if !f.isValidCell(x, y) {
		return false
	}
	return f.ExploredGrid[x][y]
}

// IsVisible 检查指定格子是否在当前视野内
func (f *FogOfWar) IsVisible(x, y int) bool {
	// BlackSheepWall 激活时，所有格子都可见
	if f.BlackSheepWallActive {
		return true
	}

	if !f.isValidCell(x, y) {
		return false
	}
	return f.VisibleGrid[x][y]
}

// GetFogState 获取指定格子的迷雾状态
func (f *FogOfWar) GetFogState(x, y int) FogState {
	if f.BlackSheepWallActive {
		return FogStateVisible
	}

	if f.IsVisible(x, y) {
		return FogStateVisible
	}

	if f.IsExplored(x, y) {
		return FogStateExplored
	}

	return FogStateUnexplored
}

// MarkDirty 标记需要重新计算视野
func (f *FogOfWar) MarkDirty() {
	f.dirty = true
}

// IsDirty 检查是否需要更新视野
func (f *FogOfWar) IsDirty() bool {
	return f.dirty
}

// isValidCell 检查坐标是否有效
func (f *FogOfWar) isValidCell(x, y int) bool {
	return x >= 0 && x < f.MapWidth && y >= 0 && y < f.MapHeight
}
