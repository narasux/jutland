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
// TestCalcAngleBetweenPoints 验证角度以正北为 0°、顺时针增大，覆盖八个方向。
// 场景：炮塔、鱼雷管、飞机航向都用这套角度约定；若 0° 起点或旋转方向错了，所有射界判定会整体偏转。
// 预期：(0,-1)=0°、(1,-1)=45°、(1,0)=90°、…、(-1,-1)=315°。
// 覆盖：以原点为起点，逐个方向调用 geometry.CalcAngleBetweenPoints 并断言精确角度。
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

// TestCalcDistance 验证两点距离按欧氏距离计算：轴向为整数，斜向为 √2 的整数倍。
// 场景：单位间距计算（position.Distance）都走它；坐标含正负方向，斜向必须用勾股而非曼哈顿距离。
// 预期：(0,0)→(1,0)/(0,1) 为 1；对角 (1,±1)/(2,±2)/(3,±3) 为 √2、2√2、3√2；
// 浮点用 epsilon=0.01 的 InEpsilon 断言；失败表示射程/命中判定出现斜向偏差。
// 覆盖：固定原点，取 dx,dy∈{±1,±2,±3} 的轴向与四象限对角点共 19 个断言。
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

// TestIsPointInRotatedRectangle 验证点在旋转矩形内部判定，且边界一律算外部。
// 场景：矩形 length=10（Y 轴）、width=4（X 轴）、中心 (0,0)，用于光标选中、锁定与落点命中。
// 预期：angle=0 时 (-1,4.9) 为真，(2.0,4.9) 与 (1.0,-5.0) 压在 x=±2、y=±5 边界为假；
// 45° 时 (1,-1)、(3,-3) 为真，90° 时 (5,-1) 为假，135° 时 (2,2) 为真；
// 失败表示旋转后命中范围偏移，或边界被算进内部导致擦边命中。
// 覆盖：固定中心，用 7 个点覆盖内部、边界与 0/45/90/135 四种旋转角，逐条 assert.True/False。
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

// TestIsSegmentIntersectRotatedRectangle 验证线段与旋转矩形相交（含边界）的判定。
// 场景：弹道与视线遮挡依赖线段-矩形相交；矩形 length=10（Y 轴）、width=4（X 轴），
// 中心 (0,0)，角度取 0/45/90/135，线段覆盖穿越、相切、内含与远离。
// 预期：穿过矩形、端点在内或边上、与四边重合、过顶点、完全在矩形内均为 true；
// 整段在矩形外（左侧、右上、旋转后的外侧）为 false；0.4×0.1 小矩形的竖线穿越也为 true；
// 失败表示弹道会穿墙，或命中判定漏掉擦边/包含情形。
// 覆盖：45 条断言分五组：0° 穿越/相切、旋转角、内部、边界重合/顶点、无交点；
// 另补 0.4×0.1 与 2.914 见方的极小矩形。
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
	assert.True(t, geometry.IsSegmentIntersectRotatedRectangle(10, 11, 10, 9, 10, 10, 0.4, 0.1, 0))

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

// TestCalcWeaponFireAngle 验证炮口提前量：按目标速度/朝向求拦截点，再换算 0..360 射击角。
// 场景：炮弹速度 1（或 100），目标速度 1（或 20）沿 0/45/90/135/180/225/270/315 度移动。
// 预期：追击斜向/轴向时角度等于目标航向 45/135/225/315/0/90/180；迎头时仍为 0/90/180；
// 三角形拦截为 50.71° 与 86.81°（epsilon=0.01）；失败表示移动目标会被打偏。
// 覆盖：固定炮位 (0,0)，覆盖追击斜向 4 例、轴向 3 例、迎头 3 例、三角形 2 例，只断言返回 angle。
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
