package drawer

import (
	"math"
	"slices"
)

const (
	sonarTwoPi    = 2 * math.Pi
	sonarAngleEps = 1e-9
)

// sonarDisk 是屏幕坐标系下的一个声呐覆盖圆。
type sonarDisk struct {
	x, y, radius float64
}

// sonarArc 是并集边界上的一段圆弧，角度单位为弧度，按屏幕顺时针方向从
// startAngle 画到 endAngle。
type sonarArc struct {
	x, y, radius         float64
	startAngle, endAngle float64
}

// angleRange 是圆周上被其他圆覆盖的角度区间。
type angleRange struct {
	start, end float64
}

// sonarCoverageArcs 返回一组圆的并集外轮廓。
// 只保留未被其他圆覆盖的圆周片段，重叠圆的内部弧不会重复绘制。
func sonarCoverageArcs(disks []sonarDisk) []sonarArc {
	if len(disks) == 0 {
		return nil
	}
	covered := make([][]angleRange, len(disks))
	hidden := make([]bool, len(disks))

	for i := range disks {
		for j := range disks {
			if i == j || hidden[i] {
				continue
			}
			di, dj := disks[i], disks[j]
			dx, dy := dj.x-di.x, dj.y-di.y
			distance := math.Hypot(dx, dy)
			ri, rj := di.radius, dj.radius

			if distance >= ri+rj {
				continue
			}
			if distance <= math.Abs(ri-rj) {
				if distance == 0 && ri == rj {
					// 完全重合的圆只保留一个
					if j > i {
						hidden[i] = true
					}
					continue
				}
				if ri <= rj {
					hidden[i] = true
				}
				continue
			}

			// 两圆相交：i 的圆周落在 j 内的角度区间以 i→j 方向为中心。
			alpha := math.Atan2(dy, dx)
			cosBeta := (distance*distance + ri*ri - rj*rj) / (2 * distance * ri)
			cosBeta = math.Max(-1, math.Min(1, cosBeta))
			beta := math.Acos(cosBeta)
			if beta >= math.Pi-sonarAngleEps {
				hidden[i] = true
				continue
			}
			addAngleRange(&covered[i], alpha-beta, alpha+beta)
		}
	}

	arcs := make([]sonarArc, 0, len(disks))
	for i, disk := range disks {
		if hidden[i] {
			continue
		}
		arcs = append(arcs, diskUncoveredArcs(disk, covered[i])...)
	}
	return arcs
}

// addAngleRange 把一个可能跨过 0/2π 的覆盖区间拆成至多两段。
func addAngleRange(ranges *[]angleRange, start, end float64) {
	start = normalizeSonarAngle(start)
	end = normalizeSonarAngle(end)
	if start < end {
		*ranges = append(*ranges, angleRange{start: start, end: end})
		return
	}
	if start > end {
		*ranges = append(*ranges,
			angleRange{start: start, end: sonarTwoPi},
			angleRange{start: 0, end: end},
		)
	}
}

func normalizeSonarAngle(angle float64) float64 {
	angle = math.Mod(angle, sonarTwoPi)
	if angle < 0 {
		angle += sonarTwoPi
	}
	return angle
}

// diskUncoveredArcs 合并覆盖区间，返回该圆仍暴露在并集边界上的弧段。
func diskUncoveredArcs(disk sonarDisk, ranges []angleRange) []sonarArc {
	if len(ranges) == 0 {
		return []sonarArc{{
			x: disk.x, y: disk.y, radius: disk.radius,
			startAngle: 0, endAngle: sonarTwoPi,
		}}
	}
	slices.SortFunc(ranges, func(a, b angleRange) int {
		switch {
		case a.start < b.start:
			return -1
		case a.start > b.start:
			return 1
		default:
			return 0
		}
	})
	merged := ranges[:0]
	for _, r := range ranges {
		if len(merged) == 0 || r.start > merged[len(merged)-1].end+sonarAngleEps {
			merged = append(merged, r)
			continue
		}
		if r.end > merged[len(merged)-1].end {
			merged[len(merged)-1].end = r.end
		}
	}

	arcs := make([]sonarArc, 0, len(merged)+1)
	start := 0.0
	for _, r := range merged {
		if r.start > start+sonarAngleEps {
			arcs = append(arcs, sonarArc{
				x: disk.x, y: disk.y, radius: disk.radius,
				startAngle: start, endAngle: r.start,
			})
		}
		if r.end > start {
			start = r.end
		}
	}
	if start < sonarTwoPi-sonarAngleEps {
		arcs = append(arcs, sonarArc{
			x: disk.x, y: disk.y, radius: disk.radius,
			startAngle: start, endAngle: sonarTwoPi,
		})
	}
	return arcs
}
