package position

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewAndNewRKeepMapAndRealCoordinatesInSync 验证 MapPos 的两套坐标在构造时保持同步。
// 场景：单位初始化既可能拿到整数格坐标（New），也可能拿到浮点格坐标（NewR）；
// 后续逻辑同时依赖 MX/MY 做格子判定、RX/RY 做精确移动和屏幕换算。
// 预期：New 的 MX/MY 与 RX/RY 相等；NewR 的 MX/MY 是 RX/RY 的向下取整。
// 覆盖：分别构造整数与带小数坐标，逐字段断言四个分量。
func TestNewAndNewRKeepMapAndRealCoordinatesInSync(t *testing.T) {
	integerPos := New(3, 4)
	require.Equal(t, 3, integerPos.MX)
	require.Equal(t, 4, integerPos.MY)
	require.InDelta(t, 3, integerPos.RX, 1e-9)
	require.InDelta(t, 4, integerPos.RY, 1e-9)

	realPos := NewR(3.9, 4.1)
	require.Equal(t, 3, realPos.MX)
	require.Equal(t, 4, realPos.MY)
	require.InDelta(t, 3.9, realPos.RX, 1e-9)
	require.InDelta(t, 4.1, realPos.RY, 1e-9)
}

// TestAssignMethodsOverwriteBothCoordinatePairs 验证重新赋值时不会留下旧坐标。
// 场景：单位被传送、寻路点吸附或测试夹具复用同一个 MapPos 时，会先改 MX/MY
// 或先改 RX/RY；若另一套坐标不更新，格子判定和绘制位置就会错位。
// 预期：AssignMxy 后 RX/RY 等于整数格坐标；AssignRxy 后 MX/MY 等于浮点坐标的 floor。
// 覆盖：先 AssignMxy 再 AssignRxy，逐字段断言两套坐标。
func TestAssignMethodsOverwriteBothCoordinatePairs(t *testing.T) {
	pos := New(1, 2)

	pos.AssignMxy(7, 8)
	require.Equal(t, 7, pos.MX)
	require.Equal(t, 8, pos.MY)
	require.InDelta(t, 7, pos.RX, 1e-9)
	require.InDelta(t, 8, pos.RY, 1e-9)

	pos.AssignRxy(3.8, 4.2)
	require.Equal(t, 3, pos.MX)
	require.Equal(t, 4, pos.MY)
	require.InDelta(t, 3.8, pos.RX, 1e-9)
	require.InDelta(t, 4.2, pos.RY, 1e-9)
}

// TestAddAndSubKeepMapCoordinatesInStep 验证浮点增减后 MX/MY 跟随 RX/RY。
// 场景：舰船/飞机每帧按速度累加位置；格坐标用于碰撞、索敌和地图索引，
// 如果只在初始化时同步，移动后的单位会一直被当成在出生格。
// 预期：AddRx/AddRy 后 MX/MY 为对应浮点值的 floor；AddMx/AddMy 同时平移 RX/RY。
// 覆盖：连续做一次浮点增减和一次整数增减，断言四个分量。
func TestAddAndSubKeepMapCoordinatesInStep(t *testing.T) {
	pos := NewR(3.8, 4.2)

	pos.AddRx(0.5)
	require.InDelta(t, 4.3, pos.RX, 1e-9)
	require.Equal(t, 4, pos.MX)

	pos.AddRy(0.9)
	require.InDelta(t, 5.1, pos.RY, 1e-9)
	require.Equal(t, 5, pos.MY)

	pos.AddMx(2)
	require.Equal(t, 6, pos.MX)
	require.InDelta(t, 6.3, pos.RX, 1e-9)

	pos.SubMy(1)
	require.Equal(t, 4, pos.MY)
	require.InDelta(t, 4.1, pos.RY, 1e-9)

	pos.AddMy(2)
	require.Equal(t, 6, pos.MY)
	require.InDelta(t, 6.1, pos.RY, 1e-9)
}

// TestSubMethodsClampAtOrigin 验证向左/向上移动不会越过地图原点。
// 场景：舰船在 (0,0) 附近执行撤退或反向移动，速度累加可能把坐标减成负数；
// 地图索引、绘制和距离计算都不接受负格坐标。
// 预期：SubRx/SubRy/SubMx/SubMy 在越过 0 时钳制到 0，并同步另一套坐标。
// 覆盖：从非零位置减去超过自身坐标的值，断言 RX/RY 与 MX/MY 都为 0。
func TestSubMethodsClampAtOrigin(t *testing.T) {
	pos := NewR(2.5, 3.5)

	pos.SubRx(10)
	require.Zero(t, pos.RX)
	require.Zero(t, pos.MX)

	pos.SubRy(10)
	require.Zero(t, pos.RY)
	require.Zero(t, pos.MY)

	pos = New(2, 3)
	pos.SubMx(10)
	pos.SubMy(10)
	require.Zero(t, pos.MX)
	require.Zero(t, pos.MY)
	require.Zero(t, pos.RX)
	require.Zero(t, pos.RY)
}

// TestEnsureBorderClampsAndOnBorderDetectsEdges 验证边界夹取与边界判定使用同一套规则。
// 场景：相机移动、单位撤退或地图边缘的寻路点可能越界；OnBorder 用来判断
// 是否已到地图边缘并停止继续移动，二者若不一致会出现“夹到边缘但判定未到”的死循环。
// 预期：EnsureBorder 把 RX/RY 夹到 [0, border]，并同步 MX/MY；
// OnBorder 在 <=0 或 >=border 时为 true，内部点为 false。
// 覆盖：构造越界点做夹取，再分别检查原点、内部点、边界点和刚好越过边界的点。
func TestEnsureBorderClampsAndOnBorderDetectsEdges(t *testing.T) {
	pos := NewR(-2.3, 30.7)
	pos.EnsureBorder(10, 20)
	require.Zero(t, pos.RX)
	require.InDelta(t, 20, pos.RY, 1e-9)
	require.Zero(t, pos.MX)
	require.Equal(t, 20, pos.MY)

	for _, tc := range []struct {
		name string
		pos  MapPos
		want bool
	}{
		{name: "origin", pos: NewR(0, 5), want: true},
		{name: "right border", pos: NewR(10, 5), want: true},
		{name: "bottom border", pos: NewR(5, 20), want: true},
		{name: "inside", pos: NewR(5, 5), want: false},
		{name: "just inside", pos: NewR(9.999, 5), want: false},
		{name: "just outside", pos: NewR(10.001, 5), want: true},
	} {
		require.Equal(t, tc.want, tc.pos.OnBorder(10, 20), tc.name)
	}
}

// TestDistanceAngleAndNearUseRealCoordinates 验证距离/角度/邻域判断读的是 RX/RY。
// 场景：同格内但浮点位置不同的两艘舰（例如编队微调、飞机盘旋）需要精确距离；
// 若误用 MX/MY，3-4-5 直角三角形会被算成格点距离，命中/侦察判定会偏差。
// 预期：Distance 返回欧氏距离；Angle 使用北=0、东=90、南=180、西=270 的约定；
// Near 按闭区间比较距离。
// 覆盖：用 (0,0)-(3,4) 验证 5 距离，用四个正方向验证角度，再检查 Near 的边界值。
func TestDistanceAngleAndNearUseRealCoordinates(t *testing.T) {
	origin := NewR(0, 0)
	point := NewR(3, 4)

	require.InDelta(t, 5, origin.Distance(point), 1e-9)
	require.True(t, origin.Near(point, 5))
	require.False(t, origin.Near(point, 4.999))

	require.InDelta(t, 90, origin.Angle(NewR(1, 0)), 1e-9)
	require.InDelta(t, 180, origin.Angle(NewR(0, 1)), 1e-9)
	require.InDelta(t, 270, origin.Angle(NewR(-1, 0)), 1e-9)
	require.InDelta(t, 0, origin.Angle(NewR(0, -1)), 1e-9)
}

// TestCopyReturnsIndependentPosition 验证 Copy 不共享可变状态。
// 场景：相机定位、编队集结点和单位预测位置经常复制 MapPos 后再修改副本；
// 若 Copy 返回指针或共享字段，改副本会悄悄移动原对象。
// 预期：修改副本的坐标不影响原对象。
// 覆盖：复制后对副本 AssignMxy，断言原对象四个分量保持不变。
func TestCopyReturnsIndependentPosition(t *testing.T) {
	original := NewR(1.5, 2.5)
	clone := original.Copy()

	clone.AssignMxy(9, 9)

	require.Equal(t, 1, original.MX)
	require.Equal(t, 2, original.MY)
	require.InDelta(t, 1.5, original.RX, 1e-9)
	require.InDelta(t, 2.5, original.RY, 1e-9)
}
