package sim

// BattleReportCard 是战报总览顶部的一张摘要卡，字段均已格式化。
type BattleReportCard struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Sub   string `json:"sub,omitempty"`
}

// BattleReportSide 是战报中单一阵营的汇总，数量行与吨位/资金均为展示文本。
type BattleReportSide struct {
	Label          string `json:"label"`
	ShipsLine      string `json:"shipsLine"`      // 初始 / 建造 / 损失 / 在役
	PlanesLine     string `json:"planesLine"`     // 初始 / 建造 / 损失 / 在役
	PlanesWithShip string `json:"planesWithShip"` // 其中随舰沉没的飞机
	InitialTonnage string `json:"initialTonnage"` // 初始吨位
	SunkTonnage    string `json:"sunkTonnage"`    // 损失吨位
	ActiveTonnage  string `json:"activeTonnage"`  // 在役吨位
	DamageDealt    string `json:"damageDealt"`    // 造成伤害
	Funds          string `json:"funds"`          // 资金流水
}

// BattleReportDetailRow 是舰艇 / 飞机明细页的一行。
type BattleReportDetailRow struct {
	Name      string `json:"name"`
	Initial   int    `json:"initial"`
	Built     int    `json:"built"`
	Lost      int    `json:"lost"`
	Remaining int    `json:"remaining"`
}

// BattleReportTrendPoint 是一个战况采样点，供 HTML 图表使用。
type BattleReportTrendPoint struct {
	Minute          int `json:"minute"`
	OwnShips        int `json:"ownShips"`
	EnemyShips      int `json:"enemyShips"`
	OwnPlanes       int `json:"ownPlanes"`
	EnemyPlanes     int `json:"enemyPlanes"`
	OwnBuilt        int `json:"ownBuilt"`
	OwnLost         int `json:"ownLost"`
	EnemyBuilt      int `json:"enemyBuilt"`
	EnemyLost       int `json:"enemyLost"`
	OwnPlaneBuilt   int `json:"ownPlaneBuilt"`
	OwnPlaneLost    int `json:"ownPlaneLost"`
	EnemyPlaneBuilt int `json:"enemyPlaneBuilt"`
	EnemyPlaneLost  int `json:"enemyPlaneLost"`
}

// BattleReportEvent 是一条可读的战况记录；Summary 与游戏内战报的文案一致。
type BattleReportEvent struct {
	Tick        int64   `json:"tick"`
	Time        string  `json:"time"`
	Side        string  `json:"side,omitempty"`
	SideClass   string  `json:"sideClass,omitempty"`
	Actor       string  `json:"actor,omitempty"`
	ActorSide   string  `json:"actorSide,omitempty"`
	Weapon      string  `json:"weapon,omitempty"`
	Action      string  `json:"action"`
	Target      string  `json:"target,omitempty"`
	TargetSide  string  `json:"targetSide,omitempty"`
	Summary     string  `json:"summary"`
	Damage      float64 `json:"damage,omitempty"`
	TotalDamage float64 `json:"totalDamage,omitempty"`
	Count       int     `json:"count,omitempty"`
	Tonnage     float64 `json:"tonnage,omitempty"`
	Kill        bool    `json:"kill,omitempty"`
}

// BattleReport 是 F3 战报在 HTML / JSON 中的完整视图。
// 它对应游戏结算页的「总览 / 舰艇 / 飞机 / 战况」四类内容，不包含 UI 布局。
type BattleReport struct {
	Side      string `json:"side"`   // 场景玩家侧 P1 / P2
	Status    string `json:"status"` // success / failed / running
	EndReason string `json:"endReason,omitempty"`
	Ticks     int64  `json:"ticks"`
	Note      string `json:"note,omitempty"` // 多 seed 时标注明细来源

	Cards []BattleReportCard `json:"cards"`
	Own   BattleReportSide   `json:"own"`
	Enemy BattleReportSide   `json:"enemy"`

	OwnShipRows    []BattleReportDetailRow `json:"ownShipRows"`
	EnemyShipRows  []BattleReportDetailRow `json:"enemyShipRows"`
	OwnPlaneRows   []BattleReportDetailRow `json:"ownPlaneRows"`
	EnemyPlaneRows []BattleReportDetailRow `json:"enemyPlaneRows"`

	Kills         []BattleReportEvent `json:"kills"`
	KillsTotal    int                 `json:"killsTotal"`
	KillsOmitted  int                 `json:"killsOmitted"`
	Events        []BattleReportEvent `json:"events"`
	EventsTotal   int                 `json:"eventsTotal"`
	EventsOmitted int                 `json:"eventsOmitted"`

	Exchange string                   `json:"exchange"`
	Survival string                   `json:"survival"`
	Trend    []BattleReportTrendPoint `json:"trend,omitempty"`
}
