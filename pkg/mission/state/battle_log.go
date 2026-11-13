package state

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// BattleLogAction 战况记录动作。
type BattleLogAction string

const (
	// BattleLogShipHit 舰艇命中舰艇。
	BattleLogShipHit BattleLogAction = "shipHit"
	// BattleLogShipSunk 舰艇被击沉。
	BattleLogShipSunk BattleLogAction = "shipSunk"
	// BattleLogPlaneHit 飞机被命中但未损失。
	BattleLogPlaneHit BattleLogAction = "planeHit"
	// BattleLogPlaneShotDown 飞机被击落。
	BattleLogPlaneShotDown BattleLogAction = "planeShotDown"
	// BattleLogPlaneLost 飞机因非战斗原因损失（坠毁 / 随舰沉没外的损耗）。
	BattleLogPlaneLost BattleLogAction = "planeLost"
	// BattleLogPlaneWithShip 舰载机随母舰沉没。
	BattleLogPlaneWithShip BattleLogAction = "planeWithShip"
)

// BattleReportEventRows 战况页每屏可显示的行数，也是滚轮一次翻页的范围基准。
const BattleReportEventRows = 13

const (
	// battleLogMaxEntries 内存中最多保留的战况条数，防止超长局无限增长。
	battleLogMaxEntries = 50000
	// battleLogTrimBatch 超出上限后按批裁剪，避免每发命中都复制整份日志。
	battleLogTrimBatch = 4096
	// battleLogHitWindowTicks 同一攻击者对同一目标的连续命中合并时间窗（1 秒）。
	battleLogHitWindowTicks int64 = 60
)

// BattleLogEntry 一条战况记录，例如「赤城 击沉 亚利桑那」。
// 名称保存配置名，绘制时再按当前语言解析显示名。
type BattleLogEntry struct {
	Tick int64

	ActorPlayer  faction.Player
	ActorName    string
	ActorType    string
	ActorIsPlane bool

	TargetPlayer  faction.Player
	TargetName    string
	TargetType    string
	TargetIsPlane bool

	Action  BattleLogAction
	Count   int
	Tonnage float64
	// Hit 命中武器与实际伤害；击沉 / 击落时 Damage 为最后一击伤害
	Hit objUnit.HitInfo
	// TotalDamage 目标被击沉 / 击落时的累计承伤
	TotalDamage float64
}

// IsKill 是否为击沉或击落。
func (e BattleLogEntry) IsKill() bool {
	return e.Action == BattleLogShipSunk || e.Action == BattleLogPlaneShotDown
}

// FilterBattleLog 按详情页筛选返回可见战况；全部时直接返回原切片。
func FilterBattleLog(log []BattleLogEntry, filter BattleReportEventFilter) []BattleLogEntry {
	if filter != BattleReportEventFilterKills {
		return log
	}
	filtered := make([]BattleLogEntry, 0, min(len(log), 256))
	for _, entry := range log {
		if entry.IsKill() {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FilteredBattleLogCount 返回筛选后的条数，供滚动上限使用而不分配列表。
func FilteredBattleLogCount(log []BattleLogEntry, filter BattleReportEventFilter) int {
	if filter != BattleReportEventFilterKills {
		return len(log)
	}
	count := 0
	for _, entry := range log {
		if entry.IsKill() {
			count++
		}
	}
	return count
}

// RecordBattleLog 追加一条战况记录。
// 连续命中 / 连续击落同一目标时合并计数，避免高射速武器刷屏。
func (s *BattleStats) RecordBattleLog(entry BattleLogEntry) {
	if entry.Count <= 0 {
		entry.Count = 1
	}
	window := int64(0)
	switch entry.Action {
	case BattleLogShipHit, BattleLogPlaneHit:
		window = battleLogHitWindowTicks
	}
	if window > 0 {
		if n := len(s.BattleLog); n > 0 {
			last := &s.BattleLog[n-1]
			if last.Action == entry.Action &&
				last.ActorPlayer == entry.ActorPlayer && last.ActorName == entry.ActorName &&
				last.ActorType == entry.ActorType && last.ActorIsPlane == entry.ActorIsPlane &&
				last.TargetPlayer == entry.TargetPlayer && last.TargetName == entry.TargetName &&
				last.TargetType == entry.TargetType && last.TargetIsPlane == entry.TargetIsPlane &&
				last.Hit.WeaponType == entry.Hit.WeaponType &&
				last.Hit.Diameter == entry.Hit.Diameter &&
				last.Hit.Caliber == entry.Hit.Caliber &&
				last.Hit.Weight == entry.Hit.Weight &&
				entry.Tick >= last.Tick && entry.Tick-last.Tick <= window {
				last.Count += entry.Count
				last.Hit.Damage += entry.Hit.Damage
				last.Tick = entry.Tick
				s.BattleLogVersion++
				return
			}
		}
	}

	s.BattleLog = append(s.BattleLog, entry)
	s.BattleLogVersion++
	if len(s.BattleLog) >= battleLogMaxEntries+battleLogTrimBatch {
		n := copy(s.BattleLog, s.BattleLog[len(s.BattleLog)-battleLogMaxEntries:])
		s.BattleLog = s.BattleLog[:n]
	}
}
