package manager

import (
	"math"
	"testing"

	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 点击命中判定：跑道又细又长，命中区域应为沿跑道轴的胶囊体，
// 端头外 1 格、横向半宽外加余量。
func TestHitAirfieldRunway(t *testing.T) {
	// 与珍珠港福特岛机场一致：中心 (55,63)，朝向 45°，跑道 17 x 0.8 格
	af := objBuilding.NewAirfield(
		objPos.New(55, 63), 45, 17, 0.8, "HA", 2.5, nil,
	)

	degrees := math.Pi / 180
	offset := func(along, lateral float64) objPos.MapPos {
		// 跑道方向 (sin45, -cos45)，垂直方向 (cos45, sin45)
		return objPos.NewR(
			55+math.Sin(45*degrees)*along+math.Cos(45*degrees)*lateral,
			63-math.Cos(45*degrees)*along+math.Sin(45*degrees)*lateral,
		)
	}

	cases := []struct {
		name string
		pos  objPos.MapPos
		hit  bool
	}{
		{"跑道中心", offset(0, 0), true},
		{"跑道前端", offset(8, 0), true},
		{"跑道后端", offset(-8, 0), true},
		{"跑道中段旁 1.1 格", offset(2, 1.1), true},
		{"跑道旁 1.5 格", offset(2, 1.5), false},
		{"端头外 1.5 格", offset(9.5, 0), true},
		{"端头外 3 格", offset(11, 0), false},
	}
	for _, c := range cases {
		if got := hitAirfieldRunway(af, c.pos); got != c.hit {
			t.Errorf("%s: hit = %v, want %v (pos=%s)", c.name, got, c.hit, c.pos.String())
		}
	}
}
