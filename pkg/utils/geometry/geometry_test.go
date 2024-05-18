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
