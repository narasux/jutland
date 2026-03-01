package state

// DebugFlags 调试标志
type DebugFlags struct {
	// DamageColorByTeam 区分敌我伤害颜色
	DamageColorByTeam bool
	// ShowCursorPosObjInfo 显示光标悬停对象信息
	ShowCursorPosObjInfo bool
	// ShowPlaneHP 显示飞机生命值（当前 HP / 总 HP）
	ShowPlaneHP bool
}
