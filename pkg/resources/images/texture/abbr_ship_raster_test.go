package texture

import (
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPointInPolyClassifiesInsideAndOutside 验证小图标栅格器的多边形内外判定。
// 场景：舰种缩略图由多边形拼成，fillPoly 用 2×2 子采样判断像素覆盖；内外判定错了整块甲板会缺角。
// 预期：正方形内部点 true、外部点 false；三角形斜边两侧分别 true/false。
// 覆盖：构造正方形与三角形顶点，调用 pointInPoly 对内部/外部/斜边两侧各取点断言。
func TestPointInPolyClassifiesInsideAndOutside(t *testing.T) {
	square := [][2]float32{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	require.True(t, pointInPoly(5, 5, square))
	require.False(t, pointInPoly(15, 5, square))
	require.False(t, pointInPoly(-1, 5, square))

	triangle := [][2]float32{{0, 0}, {10, 0}, {0, 10}}
	require.True(t, pointInPoly(1, 1, triangle))
	require.False(t, pointInPoly(8, 8, triangle))
}

// TestDistToSegHandlesEndpointsAndPerpendicular 验证点到线段的距离在垂足、端点和零长线段上的取值。
// 场景：图标描边用距离决定抗锯齿覆盖率；线段端点外必须量到端点，不能量到无限延长线。
// 预期：垂足 (5,3)→3；左端点外 (-2,0)→2；右端点外 (12,0)→2；斜向端点外 (12,4)→√20；零长段→点距。
// 覆盖：五组点/线段调用 distToSeg，使用 InDelta 断言。
func TestDistToSegHandlesEndpointsAndPerpendicular(t *testing.T) {
	require.InDelta(t, 3, distToSeg(5, 3, 0, 0, 10, 0), 1e-6)
	require.InDelta(t, 2, distToSeg(-2, 0, 0, 0, 10, 0), 1e-6)
	require.InDelta(t, 2, distToSeg(12, 0, 0, 0, 10, 0), 1e-6)
	require.InDelta(t, math.Sqrt(20), distToSeg(12, 4, 0, 0, 10, 0), 1e-6)
	require.InDelta(t, 2, distToSeg(2, 2, 2, 4, 2, 4), 1e-6)
}

// TestCoverageTransitionsAcrossAntiAliasBand 验证覆盖率在半径 ±0.55px 的抗锯齿带内线性过渡。
// 场景：strokeCircle/strokeLine 用 coverage(d, halfWidth) 求像素 alpha；硬切换会让圆形图标出现锯齿。
// 预期：距离 0 → 1；距离等于半径 → 0.5；距离超过半径+0.55 → 0；带内 4.5 → 约 0.9545。
// 覆盖：四组 (dist,radius) 调用 coverage，前三组精确断言、带内用 InDelta。
func TestCoverageTransitionsAcrossAntiAliasBand(t *testing.T) {
	require.Equal(t, float32(1), coverage(0, 5))
	require.InDelta(t, 0.5, coverage(5, 5), 1e-5)
	require.Equal(t, float32(0), coverage(6, 5))
	require.InDelta(t, 0.954545, coverage(4.5, 5), 1e-5)
}

// TestFillPolyClipsAndBlendsInsideRaster 验证多边形填充只写栅格内像素，并在内部产生非零 alpha。
// 场景：舰种图标绘制三角形甲板/舰艏，顶点可能超出小画布；越界像素必须跳过而不是 panic。
// 预期：内部 (4,4) alpha>0；越界三角形不 panic；fillPoly 顶点不足 3 个时直接返回。
// 覆盖：8×8 iconRaster 上填充三角形并检查 image()，再填充越界/两点多边形验证安全。
func TestFillPolyClipsAndBlendsInsideRaster(t *testing.T) {
	raster := newIconRaster(8, 8)
	raster.fillPoly(
		[][2]float32{{1, 1}, {7, 1}, {4, 7}},
		color.RGBA{R: 255, A: 255},
	)
	_, _, _, alpha := raster.image().At(4, 4).RGBA()
	require.Greater(t, alpha, uint32(0), "center of triangle should be filled")

	raster.fillPoly([][2]float32{{-5, -5}, {-1, -1}, {-2, -3}}, color.RGBA{A: 255})
	raster.fillPoly([][2]float32{{0, 0}, {1, 1}}, color.RGBA{A: 255})
}
