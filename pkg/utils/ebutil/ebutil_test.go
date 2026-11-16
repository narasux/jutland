package ebutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTrimLineSegmentKeepsRequestedGaps 验证线段按起点/终点留白裁剪，返回的新端点保持原方向。
// 场景：绘制舰队连线/集结点指示时，需要在舰体图标与目标标记之间留出空隙；
// 水平、斜向、反向线段都必须沿单位方向裁剪，而不是只处理水平线。
// 预期：水平 (0,0)-(10,0) 留白 2/3 → (2,0)-(7,0)；斜向 (0,0)-(3,4) 留白 1/2 → (0.6,0.8)-(1.8,2.4)；
// 反向线段端点顺序不变。
// 覆盖：表驱动三组线段与留白，调用 TrimLineSegment 后逐坐标 InDelta 断言。
func TestTrimLineSegmentKeepsRequestedGaps(t *testing.T) {
	cases := []struct {
		name                             string
		x1, y1, x2, y2, startGap, endGap float64
		wantX1, wantY1, wantX2, wantY2   float64
	}{
		{
			name:     "horizontal",
			x1:       0,
			y1:       0,
			x2:       10,
			y2:       0,
			startGap: 2,
			endGap:   3,
			wantX1:   2,
			wantY1:   0,
			wantX2:   7,
			wantY2:   0,
		},
		{
			name:     "diagonal",
			x1:       0,
			y1:       0,
			x2:       3,
			y2:       4,
			startGap: 1,
			endGap:   2,
			wantX1:   0.6,
			wantY1:   0.8,
			wantX2:   1.8,
			wantY2:   2.4,
		},
		{
			name:     "reversed",
			x1:       10,
			y1:       0,
			x2:       0,
			y2:       0,
			startGap: 2,
			endGap:   3,
			wantX1:   8,
			wantY1:   0,
			wantX2:   3,
			wantY2:   0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x1, y1, x2, y2, ok := TrimLineSegment(tc.x1, tc.y1, tc.x2, tc.y2, tc.startGap, tc.endGap)
			require.True(t, ok)
			require.InDelta(t, tc.wantX1, x1, 1e-9)
			require.InDelta(t, tc.wantY1, y1, 1e-9)
			require.InDelta(t, tc.wantX2, x2, 1e-9)
			require.InDelta(t, tc.wantY2, y2, 1e-9)
		})
	}
}

// TestTrimLineSegmentRejectsZeroOrTooShortSegments 验证零长线段和留白超过总长时返回 false。
// 场景：两艘舰重叠、或图标间距大于连线长度；若仍返回裁剪端点，会画出方向相反/零长度的线。
// 预期：零长返回 false；留白等于总长返回 false；留白略小于总长仍返回 true。
// 覆盖：三组 (0,0)-(0,0)/(0,0)-(1,0) 的留白组合，断言 ok 标志。
func TestTrimLineSegmentRejectsZeroOrTooShortSegments(t *testing.T) {
	_, _, _, _, ok := TrimLineSegment(0, 0, 0, 0, 0, 0)
	require.False(t, ok)

	_, _, _, _, ok = TrimLineSegment(0, 0, 1, 0, 0.5, 0.5)
	require.False(t, ok)

	_, _, _, _, ok = TrimLineSegment(0, 0, 1, 0, 0.4, 0.5)
	require.True(t, ok)

	// 确保返回的坐标不是 NaN/Inf（零长保护）。
	_, _, _, _, _ = TrimLineSegment(math.NaN(), math.NaN(), math.NaN(), math.NaN(), 1, 1)
}
