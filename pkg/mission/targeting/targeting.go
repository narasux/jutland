// Package targeting 提供与任务状态解耦的异步目标规划。
package targeting

import (
	"math"
	"sort"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

const (
	// ReplanIntervalTicks 两次后台规划之间的最小模拟帧间隔。
	ReplanIntervalTicks int64 = 30
	// PlanTTLTicks 无事件时计划的最长有效时间。
	PlanTTLTicks int64 = 180
	// FleetClusterRadius 敌舰空间聚类距离（地图格）。
	FleetClusterRadius = 20.0
	// MaxQueueSize 每个基地、每种目标类型最多保留的目标数。
	MaxQueueSize = 12

	shipBalanceWeight   = 0.35
	shipValueWeight     = 0.30
	shipDistanceWeight  = 0.25
	shipThreatWeight    = 0.10
	airBalanceWeight    = 0.40
	airThreatWeight     = 0.35
	airDistanceWeight   = 0.25
	coverageLoadPenalty = 0.12
)

// Point 是目标规划使用的地图坐标，避免调度器依赖具体对象类型。
type Point struct {
	X float64
	Y float64
}

// Group 描述基地当前可出动的机型编组。
type Group struct {
	Name       string
	TargetType object.Type
	Available  int64
	Range      float64
}

// Base 是可出动飞机并对敌分配目标的机场或航母。
type Base struct {
	UID    string
	Player faction.Player
	Pos    Point
	Groups []Group
}

// Plane 是当前在场飞机的目标规划快照。
type Plane struct {
	UID        string
	BaseUID    string
	Player     faction.Player
	TargetType object.Type
	TargetUID  string
	Pos        Point
}

// EnemyShip 是对舰调度使用的敌舰快照。
type EnemyShip struct {
	UID    string
	Player faction.Player
	Pos    Point
	Value  float64
}

// EnemyPlane 是对空调度使用的敌机快照。
type EnemyPlane struct {
	UID      string
	Player   faction.Player
	Pos      Point
	Threat   float64
	Airborne bool
}

// Snapshot 是主线程在单个模拟帧创建的只读调度输入。
type Snapshot struct {
	Tick        int64
	Bases       []Base
	Planes      []Plane
	EnemyShips  []EnemyShip
	EnemyPlanes []EnemyPlane
}

// TargetRef 是基地目标队列中的一项。
type TargetRef struct {
	UID        string
	FleetID    string
	TargetType object.Type
	Score      float64
}

// Plan 是后台返回给主线程的不可变计划。
type Plan struct {
	Revision     int64
	ComputedTick int64
	ExpiresTick  int64
	BaseQueues   map[string]map[object.Type][]TargetRef
}

type fleetCluster struct {
	ID     string
	Player faction.Player
	Ships  []EnemyShip
}

// BuildPlan 计算基地级目标队列。该函数只读取 Snapshot，可安全在后台执行。
func BuildPlan(snapshot Snapshot) Plan {
	plan := Plan{
		Revision:     snapshot.Tick,
		ComputedTick: snapshot.Tick,
		ExpiresTick:  snapshot.Tick + PlanTTLTicks,
		BaseQueues:   make(map[string]map[object.Type][]TargetRef, len(snapshot.Bases)),
	}
	for _, base := range snapshot.Bases {
		plan.BaseQueues[base.UID] = map[object.Type][]TargetRef{
			object.TypePlane: {},
			object.TypeShip:  {},
		}
	}

	activeCounts := activeTargetTypeCounts(snapshot.Planes)
	appendShipQueues(snapshot, activeCounts, &plan)
	appendAirQueues(snapshot, activeCounts, &plan)
	return plan
}

// appendShipQueues 为所有具备对舰能力的基地生成舰队级目标队列。
// 先做全局覆盖，保证每个舰队至少被一个基地负责，再让每个基地独立填充。
func appendShipQueues(snapshot Snapshot, activeCounts map[object.Type]map[string]int, plan *Plan) {
	clusters := clusterEnemyShips(snapshot.EnemyShips)
	ownedByBase := make(map[string][]fleetCluster)
	coverageLoad := make(map[string]int)
	assigned := assignedFleetCounts(snapshot.Planes, clusters)
	for _, cluster := range clusters {
		candidateBases := make([]Base, 0)
		for _, base := range snapshot.Bases {
			if base.Player == cluster.Player || queueLimit(base, object.TypeShip, activeCounts[object.TypeShip]) == 0 {
				continue
			}
			if minShipDistance(base.Pos, cluster.Ships) > baseReachableRange(base, object.TypeShip) {
				continue
			}
			candidateBases = append(candidateBases, base)
		}
		if len(candidateBases) == 0 {
			continue
		}

		sort.SliceStable(candidateBases, func(i, j int) bool {
			return candidateBases[i].UID < candidateBases[j].UID
		})
		bestBaseUID := ""
		bestScore := math.Inf(-1)
		for _, base := range candidateBases {
			if len(plan.BaseQueues[base.UID][object.TypeShip]) >=
				queueLimit(base, object.TypeShip, activeCounts[object.TypeShip]) {
				continue
			}
			score := shipClusterScore(base, cluster, assigned) -
				coverageLoadPenalty*float64(coverageLoad[base.UID])
			if score > bestScore || (score == bestScore && base.UID < bestBaseUID) {
				bestScore = score
				bestBaseUID = base.UID
			}
		}
		if bestBaseUID == "" {
			continue
		}
		ownedByBase[bestBaseUID] = append(ownedByBase[bestBaseUID], cluster)
		coverageLoad[bestBaseUID]++
	}

	for baseUID, owned := range ownedByBase {
		var base Base
		for _, candidate := range snapshot.Bases {
			if candidate.UID == baseUID {
				base = candidate
				break
			}
		}
		for _, cluster := range owned {
			appendShipRef(plan, base, cluster, assigned)
		}
	}

	// 覆盖分配只负责保证每个舰队至少进入一个基地队列。所有有舰船
	// 打击能力且目标可达的基地仍需独立填充，否则未拿到舰队归属的
	// 航母 / 机场会永远得不到对舰目标，只能反复起飞战斗机。
	for _, base := range snapshot.Bases {
		if queueLimit(base, object.TypeShip, activeCounts[object.TypeShip]) == 0 {
			continue
		}
		reachable := make([]fleetCluster, 0, len(clusters))
		rangeLimit := baseReachableRange(base, object.TypeShip)
		for _, cluster := range clusters {
			if cluster.Player != base.Player && minShipDistance(base.Pos, cluster.Ships) <= rangeLimit {
				reachable = append(reachable, cluster)
			}
		}
		fillShipQueue(activeCounts, plan, base, reachable, assigned)
	}
}

// fillShipQueue 按舰队评分填充单个基地的对舰队列。
// 队列已有覆盖项时从舰队第二艘舰开始补充，避免重复写入代表舰。
func fillShipQueue(
	activeCounts map[object.Type]map[string]int,
	plan *Plan,
	base Base,
	clusters []fleetCluster,
	assigned map[string]int,
) {
	if base.UID == "" {
		return
	}

	limit := queueLimit(base, object.TypeShip, activeCounts[object.TypeShip])
	rangeLimit := baseReachableRange(base, object.TypeShip)
	queue := plan.BaseQueues[base.UID][object.TypeShip]
	if len(queue) >= limit {
		return
	}

	sort.SliceStable(clusters, func(i, j int) bool {
		left := shipClusterScore(base, clusters[i], assigned)
		right := shipClusterScore(base, clusters[j], assigned)
		if left == right {
			return clusters[i].ID < clusters[j].ID
		}
		return left > right
	})

	// 已有队列项时，首舰已经由舰队覆盖阶段写入；没有归属的基地
	// 则从代表舰开始填充。
	startShipIdx := 1
	if len(queue) == 0 {
		startShipIdx = 0
	}
	for shipIdx := startShipIdx; len(queue) < limit; shipIdx++ {
		added := false
		for _, cluster := range clusters {
			if shipIdx >= len(cluster.Ships) {
				continue
			}
			ship := cluster.Ships[shipIdx]
			if distance(base.Pos, ship.Pos) > rangeLimit {
				continue
			}
			queue = append(queue, TargetRef{
				UID:        ship.UID,
				FleetID:    cluster.ID,
				TargetType: object.TypeShip,
				Score:      shipClusterScore(base, cluster, assigned),
			})
			added = true
			if len(queue) >= limit {
				break
			}
		}
		if !added {
			break
		}
	}
	plan.BaseQueues[base.UID][object.TypeShip] = queue
}

// appendAirQueues 为每个基地生成对空目标队列。
// 敌机按分配均衡、威胁和距离评分，同一敌机被越多基地选中，后续得分越低。
func appendAirQueues(snapshot Snapshot, activeCounts map[object.Type]map[string]int, plan *Plan) {
	assignedInPlan := make(map[string]int)
	for _, plane := range snapshot.Planes {
		if plane.TargetType == object.TypePlane && plane.TargetUID != "" {
			assignedInPlan[plane.TargetUID]++
		}
	}
	for _, base := range snapshot.Bases {
		limit := queueLimit(base, object.TypePlane, activeCounts[object.TypePlane])
		if limit == 0 {
			continue
		}

		candidates := make([]EnemyPlane, 0, len(snapshot.EnemyPlanes))
		for _, enemy := range snapshot.EnemyPlanes {
			if enemy.Player != base.Player && enemy.Airborne &&
				distance(base.Pos, enemy.Pos) <= baseReachableRange(base, object.TypePlane) {
				candidates = append(candidates, enemy)
			}
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].UID < candidates[j].UID
		})

		queue := plan.BaseQueues[base.UID][object.TypePlane]
		for len(queue) < limit && len(candidates) > 0 {
			bestIdx := -1
			bestScore := math.Inf(-1)
			for idx, enemy := range candidates {
				score := airPlaneScore(base, enemy, assignedInPlan[enemy.UID])
				if score > bestScore || (score == bestScore && (bestIdx < 0 || enemy.UID < candidates[bestIdx].UID)) {
					bestScore = score
					bestIdx = idx
				}
			}
			if bestIdx < 0 {
				break
			}
			enemy := candidates[bestIdx]
			queue = append(queue, TargetRef{
				UID:        enemy.UID,
				TargetType: object.TypePlane,
				Score:      bestScore,
			})
			assignedInPlan[enemy.UID]++
		}
		plan.BaseQueues[base.UID][object.TypePlane] = queue
	}
}

// activeTargetTypeCounts 统计每个基地当前在空的飞机数量，键为目标类型。
func activeTargetTypeCounts(planes []Plane) map[object.Type]map[string]int {
	counts := map[object.Type]map[string]int{
		object.TypePlane: {},
		object.TypeShip:  {},
	}
	for _, plane := range planes {
		if plane.BaseUID == "" || plane.TargetType == object.TypeNone {
			continue
		}
		counts[plane.TargetType][plane.BaseUID]++
	}
	return counts
}

// queueLimit 返回基地某目标类型的队列容量。
// 容量不会超过 MaxQueueSize，也不会超过在空飞机与库存飞机之和。
func queueLimit(base Base, targetType object.Type, active map[string]int) int {
	total := active[base.UID]
	for _, group := range base.Groups {
		if group.TargetType == targetType {
			total += max(0, int(group.Available))
		}
	}
	return min(MaxQueueSize, total)
}

// baseReachableRange 返回基地某目标类型的规划航程。
// 取当前有库存机型中的最大 Range，使长航程机型不会被同基地短腿机型过滤掉；
// 实际起飞时仍会按目标距离校验具体机型。
func baseReachableRange(base Base, targetType object.Type) float64 {
	maxRange := 0.0
	for _, group := range base.Groups {
		if group.TargetType != targetType || group.Available <= 0 || group.Range <= 0 {
			continue
		}
		maxRange = max(maxRange, group.Range)
	}
	if maxRange <= 0 {
		return math.Inf(1)
	}
	return maxRange
}

// clusterEnemyShips 按阵营和空间距离把敌舰聚合成舰队。
// 同一阵营内距离不超过 FleetClusterRadius 的舰船属于同一连通分量。
func clusterEnemyShips(ships []EnemyShip) []fleetCluster {
	sortedShips := append([]EnemyShip(nil), ships...)
	sort.SliceStable(sortedShips, func(i, j int) bool {
		if sortedShips[i].Player == sortedShips[j].Player {
			return sortedShips[i].UID < sortedShips[j].UID
		}
		return sortedShips[i].Player < sortedShips[j].Player
	})

	visited := make([]bool, len(sortedShips))
	clusters := make([]fleetCluster, 0)
	for idx := range sortedShips {
		if visited[idx] {
			continue
		}
		queue := []int{idx}
		visited[idx] = true
		cluster := fleetCluster{Player: sortedShips[idx].Player}
		for len(queue) > 0 {
			curIdx := queue[0]
			queue = queue[1:]
			ship := sortedShips[curIdx]
			cluster.Ships = append(cluster.Ships, ship)
			for nextIdx := range sortedShips {
				if visited[nextIdx] || sortedShips[nextIdx].Player != ship.Player {
					continue
				}
				if distance(ship.Pos, sortedShips[nextIdx].Pos) <= FleetClusterRadius {
					visited[nextIdx] = true
					queue = append(queue, nextIdx)
				}
			}
		}
		sort.SliceStable(cluster.Ships, func(i, j int) bool {
			if cluster.Ships[i].Value == cluster.Ships[j].Value {
				return cluster.Ships[i].UID < cluster.Ships[j].UID
			}
			return cluster.Ships[i].Value > cluster.Ships[j].Value
		})
		cluster.ID = "fleet:" + cluster.Ships[0].UID
		clusters = append(clusters, cluster)
	}

	sort.SliceStable(clusters, func(i, j int) bool {
		left, right := fleetValue(clusters[i]), fleetValue(clusters[j])
		if left == right {
			return clusters[i].ID < clusters[j].ID
		}
		return left > right
	})
	return clusters
}

// appendShipRef 把一个舰队的可达代表舰写入基地队列。
// 代表舰优先选择舰队内价值最高且在基地航程内的舰船。
func appendShipRef(
	plan *Plan,
	base Base,
	cluster fleetCluster,
	assigned map[string]int,
) {
	if len(cluster.Ships) == 0 {
		return
	}
	rangeLimit := baseReachableRange(base, object.TypeShip)
	ship := cluster.Ships[0]
	for _, candidate := range cluster.Ships {
		if distance(base.Pos, candidate.Pos) <= rangeLimit {
			ship = candidate
			break
		}
	}
	if distance(base.Pos, ship.Pos) > rangeLimit {
		return
	}
	queue := plan.BaseQueues[base.UID][object.TypeShip]
	queue = append(queue, TargetRef{
		UID:        ship.UID,
		FleetID:    cluster.ID,
		TargetType: object.TypeShip,
		Score:      shipClusterScore(base, cluster, assigned),
	})
	plan.BaseQueues[base.UID][object.TypeShip] = queue
}

// assignedFleetCounts 统计当前在空飞机对各舰队的追击数量，用于分配均衡评分。
func assignedFleetCounts(planes []Plane, clusters []fleetCluster) map[string]int {
	shipFleetIDs := make(map[string]string)
	for _, cluster := range clusters {
		for _, ship := range cluster.Ships {
			shipFleetIDs[ship.UID] = cluster.ID
		}
	}
	counts := make(map[string]int, len(clusters))
	for _, plane := range planes {
		if fleetID, ok := shipFleetIDs[plane.TargetUID]; ok {
			counts[fleetID]++
		}
	}
	return counts
}

// shipClusterScore 计算舰队对基地的综合攻击价值。
// 评分由分配均衡、舰队价值、基地距离和威胁四部分组成。
func shipClusterScore(base Base, cluster fleetCluster, assigned map[string]int) float64 {
	distanceScore := 1 / (1 + minShipDistance(base.Pos, cluster.Ships)/FleetClusterRadius)
	balanceScore := 1 / (1 + float64(assigned[cluster.ID]))
	return shipBalanceWeight*balanceScore +
		shipValueWeight*clamp01(fleetValue(cluster)) +
		shipDistanceWeight*distanceScore +
		shipThreatWeight*clamp01(fleetThreat(cluster, base))
}

// airPlaneScore 计算单架敌机对基地的对空威胁价值。
// 评分由分配均衡、敌机战力和基地距离三部分组成。
func airPlaneScore(base Base, enemy EnemyPlane, assigned int) float64 {
	distanceScore := 1 / (1 + distance(base.Pos, enemy.Pos)/FleetClusterRadius)
	balanceScore := 1 / (1 + float64(assigned))
	return airBalanceWeight*balanceScore +
		airThreatWeight*clamp01(enemy.Threat) +
		airDistanceWeight*distanceScore
}

// fleetValue 返回舰队价值，取舰队中最高价值的单舰。
func fleetValue(cluster fleetCluster) float64 {
	value := 0.0
	for _, ship := range cluster.Ships {
		value = max(value, ship.Value)
	}
	return value
}

// fleetThreat 返回舰队的聚合威胁值。
// 舰船价值总和越大、距离基地越近，威胁越高，结果限制在 [0, 1]。
func fleetThreat(cluster fleetCluster, base Base) float64 {
	value := 0.0
	for _, ship := range cluster.Ships {
		value += max(0, ship.Value)
	}
	distanceScore := 1 / (1 + minShipDistance(base.Pos, cluster.Ships)/(FleetClusterRadius*2))
	return clamp01(value/6) * distanceScore
}

// minShipDistance 返回点到舰队中任意舰船的最短距离。
func minShipDistance(point Point, ships []EnemyShip) float64 {
	if len(ships) == 0 {
		return math.Inf(1)
	}
	minDistance := math.Inf(1)
	for _, ship := range ships {
		minDistance = min(minDistance, distance(point, ship.Pos))
	}
	return minDistance
}

// distance 返回两个规划坐标之间的欧氏距离。
func distance(left, right Point) float64 {
	return math.Hypot(left.X-right.X, left.Y-right.Y)
}

// clamp01 把数值限制在 [0, 1]。
func clamp01(value float64) float64 {
	return min(1, max(0, value))
}
