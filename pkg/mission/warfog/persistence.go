package warfog

// FogOfWarSaveData 迷雾保存数据
type FogOfWarSaveData struct {
	ExploredGrid         [][]bool `json:"exploredGrid"`
	BlackSheepWallActive bool     `json:"blackSheepWallActive"`
}

// ToSaveData 转换为保存数据
func (f *FogOfWar) ToSaveData() FogOfWarSaveData {
	return FogOfWarSaveData{
		ExploredGrid:         f.ExploredGrid,
		BlackSheepWallActive: f.BlackSheepWallActive,
	}
}

// LoadFromSaveData 从保存数据加载
func LoadFromSaveData(data FogOfWarSaveData, width, height int) *FogOfWar {
	fog := NewFogOfWar(width, height)
	fog.ExploredGrid = data.ExploredGrid
	fog.BlackSheepWallActive = data.BlackSheepWallActive
	fog.dirty = true
	return fog
}
