package macro

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/state"
)

const (
	// battleReportEventLimit 是 HTML 战况记录保留的最近条数，完整记录另写 jsonl。
	battleReportEventLimit = 400
	// battleReportKillLimit 是 HTML 关键战果保留的最近条数。
	battleReportKillLimit = 500
)

// buildBattleReport 把任务结算数据整理成 HTML / JSON 可直接渲染的战报。
// 返回值第二项是完整战况，供 WriteReport 写成 battle-log.jsonl。
func buildBattleReport(
	ms *state.MissionState, side string, status state.MissionStatus, ticks int64,
) (*sim.BattleReport, []sim.BattleReportEvent) {
	if ms == nil {
		return nil, nil
	}
	ownPlayer, enemyPlayer := statePlayers(ms)
	own, enemy := ms.Stats.Side(ownPlayer), ms.Stats.Side(enemyPlayer)
	report := &sim.BattleReport{
		Side:      side,
		Status:    string(status),
		EndReason: string(ms.Stats.EndReason),
		Ticks:     ticks,
	}
	report.Cards = []sim.BattleReportCard{
		{
			Label: "建造舰船",
			Value: formatThousands(int64(own.ShipsBuilt)),
			Sub:   fmt.Sprintf("初始 %s 艘", formatThousands(int64(own.ShipsInitial))),
		},
		{
			Label: "舰艇损失",
			Value: formatThousands(int64(own.ShipsLost)),
			Sub:   fmt.Sprintf("%s 吨", formatThousands(int64(math.Round(own.SunkTonnage)))),
		},
		{
			Label: "敌方被击沉",
			Value: formatThousands(int64(enemy.ShipsLost)),
			Sub:   fmt.Sprintf("%s 吨", formatThousands(int64(math.Round(enemy.SunkTonnage)))),
		},
		{
			Label: "飞机损失",
			Value: formatThousands(int64(own.PlanesLost)),
			Sub:   fmt.Sprintf("敌方 %s 架", formatThousands(int64(enemy.PlanesLost))),
		},
	}
	report.Own = battleReportSide("我方", ms, own, true)
	report.Enemy = battleReportSide("敌方", ms, enemy, false)
	report.OwnShipRows = battleShipRows(own)
	report.EnemyShipRows = battleShipRows(enemy)
	report.OwnPlaneRows = battlePlaneRows(own)
	report.EnemyPlaneRows = battlePlaneRows(enemy)

	fullLog := make([]sim.BattleReportEvent, 0, len(ms.Stats.BattleLog))
	kills := make([]sim.BattleReportEvent, 0, 64)
	for _, entry := range ms.Stats.BattleLog {
		event := battleReportEvent(entry, ownPlayer)
		fullLog = append(fullLog, event)
		if event.Kill {
			kills = append(kills, event)
		}
	}
	report.EventsTotal = len(fullLog)
	report.KillsTotal = len(kills)
	report.Events, report.EventsOmitted = latestEvents(fullLog, battleReportEventLimit)
	report.Kills, report.KillsOmitted = latestEvents(kills, battleReportKillLimit)
	report.Trend = battleTrend(ms, ownPlayer, enemyPlayer)

	if ratio, ok := ms.Stats.ExchangeRatio(ownPlayer, enemyPlayer); ok {
		if math.IsInf(ratio, 1) {
			report.Exchange = "∞"
		} else {
			report.Exchange = fmt.Sprintf("%.2fx", ratio)
		}
	} else {
		report.Exchange = "—"
	}
	if rate, ok := own.SurvivalRate(); ok {
		report.Survival = fmt.Sprintf("%.1f%%", rate*100)
	} else {
		report.Survival = "—"
	}
	return report, fullLog
}

func statePlayers(ms *state.MissionState) (own, enemy faction.Player) {
	own, enemy = ms.Player.CurPlayer, ms.Player.CurEnemy
	if own == "" {
		own = faction.HumanAlpha
	}
	if enemy == "" {
		enemy = faction.ComputerAlpha
	}
	return own, enemy
}

func battleReportSide(
	label string, ms *state.MissionState, side state.BattleSideStats, ownSide bool,
) sim.BattleReportSide {
	funds := "无数据"
	if ownSide {
		end := ms.Stats.FundsStart + ms.Stats.FundsEarned - ms.Stats.FundsSpent
		funds = fmt.Sprintf(
			"%s → %s (+%s / -%s)",
			formatThousands(ms.Stats.FundsStart),
			formatThousands(end),
			formatThousands(ms.Stats.FundsEarned),
			formatThousands(ms.Stats.FundsSpent),
		)
	}
	return sim.BattleReportSide{
		Label: label,
		ShipsLine: fmt.Sprintf(
			"%s / %s / %s / %s",
			formatThousands(int64(side.ShipsInitial)),
			formatThousands(int64(side.ShipsBuilt)),
			formatThousands(int64(side.ShipsLost)),
			formatThousands(int64(side.RemainingShips())),
		),
		PlanesLine: fmt.Sprintf(
			"%s / %s / %s / %s",
			formatThousands(int64(side.PlanesInitial)),
			formatThousands(int64(side.PlanesBuilt)),
			formatThousands(int64(side.PlanesLost)),
			formatThousands(int64(side.RemainingPlanes())),
		),
		PlanesWithShip: formatThousands(int64(side.PlanesLostWithShip)),
		InitialTonnage: formatThousands(int64(math.Round(side.InitialTonnage))),
		SunkTonnage:    formatThousands(int64(math.Round(side.SunkTonnage))),
		ActiveTonnage:  formatThousands(int64(math.Round(side.SurvivingTonnage()))),
		DamageDealt:    formatThousands(int64(math.Round(side.DamageDealt))),
		Funds:          funds,
	}
}

func battleTrend(
	ms *state.MissionState, ownPlayer, enemyPlayer faction.Player,
) []sim.BattleReportTrendPoint {
	points := make([]sim.BattleReportTrendPoint, 0, len(ms.Stats.Timeline))
	for _, sample := range ms.Stats.Timeline {
		own := sample.TimelineSide(ownPlayer)
		enemy := sample.TimelineSide(enemyPlayer)
		points = append(points, sim.BattleReportTrendPoint{
			Minute:          int(sample.Tick / 60),
			OwnShips:        own.ShipsRemaining,
			EnemyShips:      enemy.ShipsRemaining,
			OwnPlanes:       own.PlanesRemaining,
			EnemyPlanes:     enemy.PlanesRemaining,
			OwnBuilt:        own.ShipsBuilt,
			OwnLost:         own.ShipsLost,
			EnemyBuilt:      enemy.ShipsBuilt,
			EnemyLost:       enemy.ShipsLost,
			OwnPlaneBuilt:   own.PlanesBuilt,
			OwnPlaneLost:    own.PlanesLost,
			EnemyPlaneBuilt: enemy.PlanesBuilt,
			EnemyPlaneLost:  enemy.PlanesLost,
		})
	}
	return points
}

func battleShipRows(side state.BattleSideStats) []sim.BattleReportDetailRow {
	rows := detailRows(side.ShipsInitialByClass, side.ShipsBuiltByClass, side.ShipsLostByClass)
	sort.Slice(rows, func(i, j int) bool {
		left, right := shipTonnage(rows[i].Name), shipTonnage(rows[j].Name)
		if left != right {
			return left > right
		}
		leftTotal := rows[i].Initial + rows[i].Built + rows[i].Lost
		rightTotal := rows[j].Initial + rows[j].Built + rows[j].Lost
		if leftTotal != rightTotal {
			return leftTotal > rightTotal
		}
		return rows[i].Name < rows[j].Name
	})
	for idx := range rows {
		rows[idx].Name = objUnit.GetShipDisplayName(rows[idx].Name)
	}
	return rows
}

func battlePlaneRows(side state.BattleSideStats) []sim.BattleReportDetailRow {
	rows := detailRows(side.PlanesInitialByType, side.PlanesBuiltByType, side.PlanesLostByType)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Lost != rows[j].Lost {
			return rows[i].Lost > rows[j].Lost
		}
		leftTotal := rows[i].Initial + rows[i].Built + rows[i].Lost
		rightTotal := rows[j].Initial + rows[j].Built + rows[j].Lost
		if leftTotal != rightTotal {
			return leftTotal > rightTotal
		}
		return rows[i].Name < rows[j].Name
	})
	for idx := range rows {
		rows[idx].Name = objUnit.GetPlaneDisplayName(rows[idx].Name)
	}
	return rows
}

func detailRows(initial, built, lost map[string]int) []sim.BattleReportDetailRow {
	byName := map[string]*sim.BattleReportDetailRow{}
	ensure := func(name string) *sim.BattleReportDetailRow {
		row := byName[name]
		if row == nil {
			row = &sim.BattleReportDetailRow{Name: name}
			byName[name] = row
		}
		return row
	}
	for name, count := range initial {
		ensure(name).Initial += count
	}
	for name, count := range built {
		ensure(name).Built += count
	}
	for name, count := range lost {
		ensure(name).Lost += count
	}
	rows := make([]sim.BattleReportDetailRow, 0, len(byName))
	for _, row := range byName {
		row.Remaining = max(0, row.Initial+row.Built-row.Lost)
		if row.Initial+row.Built+row.Lost+row.Remaining == 0 {
			continue
		}
		rows = append(rows, *row)
	}
	return rows
}

func shipTonnage(name string) float64 {
	if ship, ok := objUnit.ShipMap[name]; ok {
		return ship.Tonnage
	}
	return 0
}

// latestEvents 返回最新在前的战况切片，并给出因上限被省略的旧记录数。
func latestEvents(events []sim.BattleReportEvent, limit int) ([]sim.BattleReportEvent, int) {
	if len(events) == 0 || limit <= 0 {
		return nil, len(events)
	}
	start := max(0, len(events)-limit)
	out := make([]sim.BattleReportEvent, 0, len(events)-start)
	for idx := len(events) - 1; idx >= start; idx-- {
		out = append(out, events[idx])
	}
	return out, start
}

func battleReportEvent(entry state.BattleLogEntry, own faction.Player) sim.BattleReportEvent {
	actor := battleCombatantText(entry.ActorName, entry.ActorType, entry.ActorIsPlane)
	target := battleCombatantText(entry.TargetName, entry.TargetType, entry.TargetIsPlane)
	event := sim.BattleReportEvent{
		Tick:        entry.Tick,
		Time:        state.BattleReportDurationText(entry.Tick),
		Actor:       actor,
		ActorSide:   battleSideLabel(entry.ActorPlayer, own),
		Weapon:      battleWeaponText(entry),
		Action:      battleActionText(entry),
		Target:      target,
		TargetSide:  battleSideLabel(entry.TargetPlayer, own),
		Damage:      entry.Hit.Damage,
		TotalDamage: entry.TotalDamage,
		Count:       entry.Count,
		Tonnage:     entry.Tonnage,
		Kill:        entry.IsKill(),
	}
	sidePlayer := entry.ActorPlayer
	if entry.Action == state.BattleLogPlaneLost || entry.ActorName == "" {
		sidePlayer = entry.TargetPlayer
	}
	event.Side, event.SideClass = battleSideText(sidePlayer, own)
	event.Summary = battleEventSummary(entry, event)
	return event
}

// battleEventSummary 生成与游戏内战报一致的可读文案；时间由表格单独展示。
func battleEventSummary(entry state.BattleLogEntry, event sim.BattleReportEvent) string {
	if entry.Action == state.BattleLogPlaneWithShip {
		return fmt.Sprintf(
			"%s 随舰损失飞机 %s 架",
			event.Actor,
			formatThousands(int64(entry.Count)),
		)
	}
	if entry.Action == state.BattleLogShipSelfHit {
		return fmt.Sprintf(
			"%s %s 反作用自伤 %s",
			event.Actor,
			event.Weapon,
			battleDamageText(entry.Hit.Damage),
		)
	}
	parts := make([]string, 0, 4)
	if event.Actor != "" {
		parts = append(parts, event.Actor)
	} else if event.Side != "" {
		parts = append(parts, event.Side)
	}
	if event.Weapon != "" {
		parts = append(parts, event.Weapon)
	}
	parts = append(parts, event.Action)
	if event.Target != "" {
		parts = append(parts, event.Target)
	}
	text := strings.Join(parts, " ")

	if entry.Hit.WeaponType != "" || entry.Hit.Damage > 0 {
		text += "，造成 " + battleDamageText(entry.Hit.Damage) + " 伤害"
	}
	if entry.TotalDamage > entry.Hit.Damage+0.5 {
		text += "（累计 " + battleDamageText(entry.TotalDamage) + "）"
	}
	if entry.Count > 1 {
		text += "，共 " + formatThousands(int64(entry.Count)) + " 次"
	}
	if entry.Tonnage > 0 {
		text += "，吨位 " + formatThousands(int64(math.Round(entry.Tonnage))) + " 吨"
	}
	return text
}

func battleCombatantText(name, kind string, isPlane bool) string {
	if name == "" {
		return ""
	}
	if isPlane {
		if kind != "" {
			return objUnit.PlaneType(kind).ToDisplay() + " " + objUnit.GetPlaneDisplayName(name)
		}
		return objUnit.GetPlaneDisplayName(name)
	}
	if kind != "" {
		return objUnit.ShipType(kind).ToDisplay() + " " + objUnit.GetShipDisplayName(name)
	}
	return objUnit.GetShipDisplayName(name)
}

func battleActionText(entry state.BattleLogEntry) string {
	switch entry.Action {
	case state.BattleLogShipHit, state.BattleLogPlaneHit:
		return "命中"
	case state.BattleLogShipSelfHit:
		return "自伤"
	case state.BattleLogShipSunk:
		if entry.ActorName == "" {
			return "沉没"
		}
		return "击沉"
	case state.BattleLogPlaneShotDown:
		return "击落"
	case state.BattleLogPlaneLost:
		return "损失"
	case state.BattleLogPlaneWithShip:
		return "随舰损失"
	default:
		return string(entry.Action)
	}
}

func battleWeaponText(entry state.BattleLogEntry) string {
	caliber := entry.Hit.Caliber
	if caliber == "" && entry.Hit.Diameter > 0 {
		caliber = strconv.Itoa(entry.Hit.Diameter)
	}
	switch entry.Hit.WeaponType {
	case string(objBullet.TypeShell):
		switch {
		case entry.ActorIsPlane:
			if caliber != "" {
				if entry.Hit.Diameter < 20 {
					return caliber + "mm 机枪"
				}
				return caliber + "mm 机炮"
			}
		case entry.TargetIsPlane:
			if caliber != "" {
				return caliber + "mm 防空炮"
			}
		default:
			if caliber != "" {
				return caliber + "mm 舰炮"
			}
		}
	case string(objBullet.TypeTorpedo):
		if caliber != "" {
			return caliber + "mm 鱼雷"
		}
	case string(objBullet.TypeBomb):
		if entry.Hit.Weight > 0 {
			return strconv.Itoa(entry.Hit.Weight) + "kg 炸弹"
		}
		return "炸弹"
	case string(objBullet.TypeRocket):
		// 火箭深弹按 rocket 类型飞行，靠发射器名与普通火箭区分
		if strings.Contains(entry.Hit.WeaponName, "Type65") {
			return "反潜火箭"
		}
		return "火箭弹"
	case string(objBullet.TypeDepthCharge):
		switch {
		case strings.Contains(entry.Hit.WeaponName, "Hedgehog"),
			strings.Contains(entry.Hit.WeaponName, "Mk10"):
			return "刺猬弹"
		default:
			return "深水炸弹"
		}
	case string(objBullet.TypeRam):
		return "撞击"
	case string(objBullet.TypeLaser):
		return "激光"
	}
	if entry.Hit.WeaponName != "" {
		return entry.Hit.WeaponName
	}
	return ""
}

func battleSideLabel(player, own faction.Player) string {
	if player == "" {
		return ""
	}
	if player == own {
		return "我方"
	}
	return "敌方"
}

func battleSideText(player, own faction.Player) (label, class string) {
	switch battleSideLabel(player, own) {
	case "我方":
		return "我方", "side-own"
	case "敌方":
		return "敌方", "side-enemy"
	default:
		return "", ""
	}
}

// battleDamageText 与游戏内战报一致：小伤害保留小数，其他取整加千位分隔。
func battleDamageText(value float64) string {
	if value > 0 && value < 10 && value != math.Trunc(value) {
		precision := 1
		if value < 1 {
			precision = 2
		}
		return strconv.FormatFloat(value, 'f', precision, 64)
	}
	return formatThousands(int64(math.Round(value)))
}

func formatThousands(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	digits := strconv.FormatInt(value, 10)
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	var builder strings.Builder
	builder.Grow(len(digits) + len(digits)/3 + len(sign))
	builder.WriteString(sign)
	builder.WriteString(digits[:first])
	for idx := first; idx < len(digits); idx += 3 {
		builder.WriteByte(',')
		builder.WriteString(digits[idx : idx+3])
	}
	return builder.String()
}
