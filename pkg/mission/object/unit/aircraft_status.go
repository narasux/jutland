package unit

// AircraftGroupStatus 是航空表格中一种机型的实时数量。
type AircraftGroupStatus struct {
	Name      string
	Standby   int64
	InCombat  int64
	Returning int64
	Lost      int64
}

// AircraftStatus 汇总一艘战舰各航空组与全舰的实时数量。
type AircraftStatus struct {
	Groups AircraftGroupStatuses
	Total  AircraftGroupStatus
}

// AircraftGroupStatuses 是按舰船配置顺序排列的航空组状态。
type AircraftGroupStatuses []AircraftGroupStatus

// Status 根据当前仍存在于战场中的飞机计算航空联队状态。
func (sa *ShipAircraft) Status(shipUid string, planes map[string]*Plane) AircraftStatus {
	status := AircraftStatus{
		Groups: make(AircraftGroupStatuses, 0, len(sa.Groups)),
		Total:  AircraftGroupStatus{Name: "total"},
	}
	for _, group := range sa.Groups {
		row := AircraftGroupStatus{Name: group.Name, Standby: group.CurCount}
		for _, plane := range planes {
			if plane.BelongShip != shipUid || plane.Name != group.Name {
				continue
			}
			switch plane.FlightPhase {
			case PlaneFlightPhaseLandingStaging,
				PlaneFlightPhaseLandingApproach,
				PlaneFlightPhaseLandingDeck:
				row.Returning++
			default:
				// 空阶段是旧存档或测试对象中的巡航状态，也应计入作战。
				row.InCombat++
			}
		}
		row.Lost = max(0, group.MaxCount-row.Standby-row.InCombat-row.Returning)
		status.Groups = append(status.Groups, row)
		status.Total.Standby += row.Standby
		status.Total.InCombat += row.InCombat
		status.Total.Returning += row.Returning
		status.Total.Lost += row.Lost
	}
	return status
}

// Alive 返回仍在舰上或空中的飞机总数。
func (s AircraftGroupStatus) Alive() int64 {
	return s.Standby + s.InCombat + s.Returning
}

// Initial 返回航空组初始总数。
func (s AircraftGroupStatus) Initial() int64 {
	return s.Alive() + s.Lost
}
