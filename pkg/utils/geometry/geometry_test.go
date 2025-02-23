package geometry_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/utils/geometry"
)

/*
|			^
|		 -3 |
|		 -2 |
|		 -1 | 1 2 3
------------+------------>
|  -3 -2 -1	| 1
|			| 2
| 			| 3
*/
func TestCalcAngleBetweenPoints(t *testing.T) {
	assert.Equal(t, float64(0), geometry.CalcAngleBetweenPoints(0, 0, 0, -1))
	assert.Equal(t, float64(45), geometry.CalcAngleBetweenPoints(0, 0, 1, -1))
	assert.Equal(t, float64(90), geometry.CalcAngleBetweenPoints(0, 0, 1, 0))
	assert.Equal(t, float64(135), geometry.CalcAngleBetweenPoints(0, 0, 1, 1))
	assert.Equal(t, float64(180), geometry.CalcAngleBetweenPoints(0, 0, 0, 1))
	assert.Equal(t, float64(225), geometry.CalcAngleBetweenPoints(0, 0, -1, 1))
	assert.Equal(t, float64(270), geometry.CalcAngleBetweenPoints(0, 0, -1, 0))
	assert.Equal(t, float64(315), geometry.CalcAngleBetweenPoints(0, 0, -1, -1))
}

func TestCalcDistance(t *testing.T) {
	assert.Equal(t, float64(0), geometry.CalcDistance(0, 0, 0, 0))
	assert.Equal(t, float64(1), geometry.CalcDistance(0, 0, 1, 0))
	assert.Equal(t, float64(1), geometry.CalcDistance(0, 0, 0, 1))

	// 浮点数运算允许有一些误差
	epsilon := 0.01

	sqrtRootTwo := math.Sqrt(2)
	assert.InEpsilon(t, sqrtRootTwo, geometry.CalcDistance(0, 0, 1, 1), epsilon)
	assert.InEpsilon(t, sqrtRootTwo, geometry.CalcDistance(0, 0, 1, -1), epsilon)
	assert.InEpsilon(t, sqrtRootTwo, geometry.CalcDistance(0, 0, -1, 1), epsilon)
	assert.InEpsilon(t, sqrtRootTwo, geometry.CalcDistance(0, 0, -1, -1), epsilon)

	twoSqrtRootTwo := 2 * sqrtRootTwo
	assert.Equal(t, float64(2), geometry.CalcDistance(0, 0, 2, 0))
	assert.Equal(t, float64(2), geometry.CalcDistance(0, 0, 0, 2))
	assert.InEpsilon(t, twoSqrtRootTwo, geometry.CalcDistance(0, 0, 2, 2), epsilon)
	assert.InEpsilon(t, twoSqrtRootTwo, geometry.CalcDistance(0, 0, 2, -2), epsilon)
	assert.InEpsilon(t, twoSqrtRootTwo, geometry.CalcDistance(0, 0, -2, 2), epsilon)
	assert.InEpsilon(t, twoSqrtRootTwo, geometry.CalcDistance(0, 0, -2, -2), epsilon)

	threeSqrtRootThree := 3 * sqrtRootTwo
	assert.Equal(t, float64(3), geometry.CalcDistance(0, 0, 3, 0))
	assert.Equal(t, float64(3), geometry.CalcDistance(0, 0, 0, 3))
	assert.InEpsilon(t, threeSqrtRootThree, geometry.CalcDistance(0, 0, 3, 3), epsilon)
	assert.InEpsilon(t, threeSqrtRootThree, geometry.CalcDistance(0, 0, 3, -3), epsilon)
	assert.InEpsilon(t, threeSqrtRootThree, geometry.CalcDistance(0, 0, -3, 3), epsilon)
	assert.InEpsilon(t, threeSqrtRootThree, geometry.CalcDistance(0, 0, -3, -3), epsilon)
}

func TestIsPointInRotatedRect(t *testing.T) {
	cx, cy := 0.0, 0.0 // 中心点坐标
	assert.True(t, geometry.IsPointInRotatedRectangle(-1.0, 4.9, cx, cy, 10.0, 4.0, 0.0))

	// 边缘不算
	assert.False(t, geometry.IsPointInRotatedRectangle(2.0, 4.9, cx, cy, 10.0, 4.0, 0.0))
	assert.False(t, geometry.IsPointInRotatedRectangle(1.0, -5.0, cx, cy, 10.0, 4.0, 0.0))

	// 旋转一定角度
	assert.True(t, geometry.IsPointInRotatedRectangle(1.0, -1.0, cx, cy, 10.0, 4.0, 45.0))
	assert.True(t, geometry.IsPointInRotatedRectangle(3.0, -3.0, cx, cy, 10.0, 4.0, 45.0))
	assert.False(t, geometry.IsPointInRotatedRectangle(5.0, -1.0, cx, cy, 10.0, 4.0, 90.0))
	assert.True(t, geometry.IsPointInRotatedRectangle(2.0, 2.0, cx, cy, 10.0, 4.0, 135.0))
}

func TestIsSegmentIntersectRotatedRectangle(t *testing.T) {
	cx, cy := 0.0, 0.0 // 中心点坐标

	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, -2, 5, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, 0, 5, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, 2, 5, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, 0, 3, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, 3, 3, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, 2, -5, cx, cy, 10, 4, 0))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, -3, 5, cx, cy, 10, 4, 0))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 3, -2, 7, cx, cy, 10, 4, 0))

	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(1, -1, 10, 10, cx, cy, 10, 4, 45))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(1, -1, 10, 10, cx, cy, 10, 4, 45))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(3, -3, 8, 8, cx, cy, 10, 4, 45))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, -1, 5, -5, cx, cy, 10, 4, 90))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, -1, 5, 4, cx, cy, 10, 4, 90))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, 2, 5, 5, cx, cy, 10, 4, 135))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, 2, 10, 5, cx, cy, 10, 4, 135))

	// 线段完全在矩形内部
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-1, -1, 1, 1, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(0, -2, 0, 2, cx, cy, 10, 4, 0))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-1.5, 0, 1.5, 0, cx, cy, 10, 4, 0))

	// 线段与矩形边界重合
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, -5, -2, 5, cx, cy, 10, 4, 0)) // 左边界重合
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, -5, 2, 5, cx, cy, 10, 4, 0))   // 右边界重合
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, 5, 2, 5, cx, cy, 10, 4, 0))   // 上边界重合
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, -5, 2, -5, cx, cy, 10, 4, 0)) // 下边界重合

	// 线段端点在矩形顶点上
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, 5, 2, -5, cx, cy, 10, 4, 0))  // 对角线
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, -5, 2, 5, cx, cy, 10, 4, 0))  // 对角线
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(-2, 5, -2, -5, cx, cy, 10, 4, 0)) // 垂直边
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, 5, 2, -5, cx, cy, 10, 4, 0))   // 垂直边

	// 不同旋转角度的测试
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(1, -1, 10, 10, cx, cy, 10, 4, 45))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(3, -3, 8, 8, cx, cy, 10, 4, 45))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, -1, 5, -5, cx, cy, 10, 4, 90))
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, -1, 5, 4, cx, cy, 10, 4, 90))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, 2, 5, 5, cx, cy, 10, 4, 135))
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(2, 2, 10, 5, cx, cy, 10, 4, 135))

	// 线段与矩形无交点
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-6, -6, -4, -4, cx, cy, 10, 4, 0)) // 左下方
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(4, 6, 6, 8, cx, cy, 10, 4, 0))     // 右上方
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-3, 6, -2, 8, cx, cy, 10, 4, 0))   // 左上方
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(4, -6, 6, -8, cx, cy, 10, 4, 0))   // 右下方
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-8, 0, -6, 0, cx, cy, 10, 4, 0))   // 左侧
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, 0, 8, 0, cx, cy, 10, 4, 0))     // 右侧
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(0, 6, 0, 8, cx, cy, 10, 4, 0))     // 上方
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(0, -8, 0, -6, cx, cy, 10, 4, 0))   // 下方

	// 旋转后的无交点测试
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(-6, -6, -4, -4, cx, cy, 10, 4, 45)) // 旋转45度
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(6, 0, 8, 0, cx, cy, 10, 4, 90))     // 旋转90度
	assert.False(t, geometry.IsSegmentIntersectRotatedRectangle(0, 6, 0, 8, cx, cy, 10, 4, 135))

	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(30, 71, 30, 69, 30, 70, 2.9140625, 2.9140625, 0))
}

func TestCalcWeaponFireAngle(t *testing.T) {
	// 追击（斜边）
	angle, _, _ := geometry.CalcWeaponFireAngle(0, 0, 1, 10, -10, 1, 45)
	assert.Equal(t, float64(45), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 10, 10, 1, 135)
	assert.Equal(t, float64(135), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, -10, 10, 1, 225)
	assert.Equal(t, float64(225), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, -10, -10, 1, 315)
	assert.Equal(t, float64(315), angle)

	// 追击（轴向）
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 0, -10, 1, 0)
	assert.Equal(t, float64(0), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 10, 0, 1, 90)
	assert.Equal(t, float64(90), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 0, 10, 1, 180)
	assert.Equal(t, float64(180), angle)

	// 迎头
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 0, -10, 1, 180)
	assert.Equal(t, float64(0), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 10, 0, 1, 270)
	assert.Equal(t, float64(90), angle)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 0, 10, 1, 0)
	assert.Equal(t, float64(180), angle)

	// 三角形
	epsilon := 0.01
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 1, 10, 10, 1, 45)
	assert.InEpsilon(t, float64(50.71), angle, epsilon)
	angle, _, _ = geometry.CalcWeaponFireAngle(0, 0, 100, 100, 10, 20, 45)
	assert.InEpsilon(t, float64(86.81), angle, epsilon)
}
