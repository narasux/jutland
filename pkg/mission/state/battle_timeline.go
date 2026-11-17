package state

import "github.com/narasux/jutland/pkg/mission/faction"

// battleTimelineSampleTicks 趋势采样间隔：1 秒（60 TPS）。
// 采样点用于结算页的累计建造 / 损失曲线，间隔太大曲线会呈阶梯状，
// 间隔太小则长时间战斗会积累过多点，因此统一按秒采样。
const battleTimelineSampleTicks = 60

// battleTimelineMaxSamples 趋势采样上限。
// 超过上限后对半抽稀，保留时间轴首尾；抽稀一次分辨率降一半，
// 后续新点仍按原间隔追加，长时间战斗不会让内存随游戏时长线性增长。
const battleTimelineMaxSamples = 720

// BattleTimelineSide 是某个阵营在单个采样点上的累计战果快照。
// 字段与 BattleSideStats 中的累计计数保持一致，绘制层只读。
type BattleTimelineSide struct {
	ShipsBuilt  int
	ShipsLost   int
	PlanesBuilt int
	PlanesLost  int
	// 当前在役数量 / 吨位，用于绘制会随战损下降的兵力曲线。
	ShipsRemaining   int
	PlanesRemaining  int
	SurvivingTonnage float64
	SunkTonnage      float64
	DamageDealt      float64
}

// BattleTimelineSample 是一次趋势采样：游戏拍 + 双方累计战果。
type BattleTimelineSample struct {
	Tick  int64
	Sides map[faction.Player]BattleTimelineSide
}

// TimelineSide 返回指定阵营在该采样点的累计战果；未记录时返回零值。
func (s BattleTimelineSample) TimelineSide(player faction.Player) BattleTimelineSide {
	return s.Sides[player]
}

// SampleBattleTimeline 在模拟推进时每拍调用，只在跨过采样间隔时记录一个点。
// 同一拍重复调用会被忽略，因此调用方不需要自己维护采样计时。
func (s *BattleStats) SampleBattleTimeline(now int64) {
	s.sampleTimeline(now, false)
}

// sampleTimeline 记录采样点。force 为 true 时忽略采样间隔，
// 用于开局（固定在 0）与结算（补一个终止点）；同一拍已存在时只刷新快照，
// 保证时间轴上的 X 坐标严格递增。
func (s *BattleStats) sampleTimeline(now int64, force bool) {
	if now < 0 {
		now = 0
	}
	if n := len(s.Timeline); n > 0 {
		last := s.Timeline[n-1]
		// 采样时钟不倒退：即使调用方传入旧拍，也按最后一个采样点处理。
		if now < last.Tick {
			now = last.Tick
		}
		if now == last.Tick {
			if force {
				s.Timeline[n-1] = s.timelineSnapshot(now)
			}
			return
		}
		if !force && now-last.Tick < battleTimelineSampleTicks {
			return
		}
	}
	if len(s.Timeline) >= battleTimelineMaxSamples {
		s.decimateTimeline()
	}
	s.Timeline = append(s.Timeline, s.timelineSnapshot(now))
}

// timelineSnapshot 把当前双方累计战果复制成一个采样点。
// 复制而不是持有 *BattleSideStats 指针，避免后续记账改动已经过去的历史点。
func (s *BattleStats) timelineSnapshot(now int64) BattleTimelineSample {
	sample := BattleTimelineSample{
		Tick:  now,
		Sides: make(map[faction.Player]BattleTimelineSide, len(s.Sides)),
	}
	for player, side := range s.Sides {
		if side == nil {
			continue
		}
		sample.Sides[player] = BattleTimelineSide{
			ShipsBuilt:       side.ShipsBuilt,
			ShipsLost:        side.ShipsLost,
			PlanesBuilt:      side.PlanesBuilt,
			PlanesLost:       side.PlanesLost,
			ShipsRemaining:   side.RemainingShips(),
			PlanesRemaining:  side.RemainingPlanes(),
			SurvivingTonnage: side.SurvivingTonnage(),
			SunkTonnage:      side.SunkTonnage,
			DamageDealt:      side.DamageDealt,
		}
	}
	return sample
}

// decimateTimeline 在采样点达到上限时抽掉一半旧点，并保留最后一个点。
// 只影响绘制分辨率，不改变累计数值：曲线仍经过所有保留点。
func (s *BattleStats) decimateTimeline() {
	n := len(s.Timeline)
	if n < 2 {
		return
	}
	kept := make([]BattleTimelineSample, 0, n/2+1)
	for idx := 0; idx < n; idx += 2 {
		kept = append(kept, s.Timeline[idx])
	}
	if last := s.Timeline[n-1]; kept[len(kept)-1].Tick != last.Tick {
		kept = append(kept, last)
	}
	s.Timeline = kept
}
