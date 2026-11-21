package geometry

import "math"

/*
ebiten 游戏引擎坐标体系：
|		    ^ y
|		 -3 |
|		 -2 |
|		 -1 | 1 2 3     x
------------+------------>
|  -3 -2 -1	| 1
|			| 2
| 			| 3
*/
// CalcAngleBetweenPoints 计算两个点之间的夹角（+90 转换成顺时针角度, +360 确保非负数）
func CalcAngleBetweenPoints(x1, y1, x2, y2 float64) float64 {
	return math.Mod(math.Atan2(y2-y1, x2-x1)*180/math.Pi+90+360, 360)
}

// CalcDistance 计算两个点之间的距离
func CalcDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

/*
示例：length 4, width 1, angle 0
 ——
|  |
|  | 3
|  |
 ——
 1
示例：length 4, width 1, angle 90
 ——————
|      | 1
 ——————
   3
*/
// IsPointInRotatedRect 判断点 (x, y) 是否在由中心点 (cx, cy)，旋转角度 angle，
// 长度 length（Y 轴）和宽度 width（X 轴）的长方形内（不含边界）
func IsPointInRotatedRectangle(x, y, cx, cy, length, width, angle float64) bool {
	radians := angle * math.Pi / 180

	// 围绕中心旋转点 (x, y)
	sinA, cosA := math.Sin(-radians), math.Cos(-radians)

	// 将点 (x, y) 平移到以长方形中点为原点的坐标系中
	translatedX, translatedY := x-cx, y-cy

	// 在原坐标系里面旋转这个点
	rotatedX := translatedX*cosA - translatedY*sinA
	rotatedY := translatedX*sinA + translatedY*cosA

	// 旋转后，检查点是否在未旋转的长方形内
	halfLength, halfWidth := length/2, width/2

	return -halfWidth < rotatedX && rotatedX < halfWidth && -halfLength < rotatedY && rotatedY < halfLength
}

// IsSegmentIntersectRotatedRectangle 判断线段 (x1, y1) 到 (x2, y2) 是否
// 与 由中心点 (cx, cy)，旋转角度 angle，长度 length（Y 轴）和宽度 width（X 轴）的长方形相交（含边界）
func IsSegmentIntersectRotatedRectangle(x1, y1, x2, y2, cx, cy, length, width, angle float64) bool {
	radians := angle * math.Pi / 180

	// 围绕中心旋转点 (x, y)
	sinA, cosA := math.Sin(-radians), math.Cos(-radians)

	// 将点 (x, y) 平移到以长方形中点为原点的坐标系中，并在原坐标系里面旋转这个点
	dx, dy := x1-cx, y1-cy
	x1, y1 = dx*cosA-dy*sinA, dx*sinA+dy*cosA

	dx, dy = x2-cx, y2-cy
	x2, y2 = dx*cosA-dy*sinA, dx*sinA+dy*cosA

	// 计算长方形的四个顶点坐标
	halfLength, halfWidth := length/2, width/2

	minX, minY := -halfWidth, -halfLength
	maxX, maxY := halfWidth, halfLength

	// 端点测试
	if (x1 >= minX && x1 <= maxX && y1 >= minY && y1 <= maxY) ||
		(x2 >= minX && x2 <= maxX && y2 >= minY && y2 <= maxY) {
		return true
	}

	// 快速排斥测试
	if (x1 < minX && x2 < minX) || (x1 > maxX && x2 > maxX) ||
		(y1 < minY && y2 < minY) || (y1 > maxY && y2 > maxY) {
		return false
	}

	// 检查线段是否与矩形四边相交
	edges := [][4]float64{
		// 上边
		{minX, minY, maxX, minY},
		// 下边
		{minX, maxY, maxX, maxY},
		// 左边
		{minX, minY, minX, maxY},
		// 右边
		{maxX, minY, maxX, maxY},
	}

	for _, e := range edges {
		if isSegmentsIntersect(x1, y1, x2, y2, e[0], e[1], e[2], e[3]) {
			return true
		}
	}

	return false
}

// 向量叉乘 (x1, y1) (x2, y2) * (x1, y1) * (x3, y3)
func cross(x1, y1, x2, y2, x3, y3 float64) float64 {
	return (x2-x1)*(y3-y1) - (y2-y1)*(x3-x1)
}

// 带浮点误差容限的符号函数
func signFunc(f float64) int {
	const epsilon = 1e-9 // 浮点误差阈值
	if math.Abs(f) < epsilon {
		return 0
	}
	if f > 0 {
		return 1
	}
	return -1
}

// 判断两线段是否相交（x1, y1）到 (x2, y2) 和 (x3, y3) 到 (x4, y4) 是否相交
func isSegmentsIntersect(x1, y1, x2, y2, x3, y3, x4, y4 float64) bool {
	// 快速排斥实验
	if max(x1, x2) < min(x3, x4) || min(x1, x2) > max(x3, x4) ||
		max(y1, y2) < min(y3, y4) || min(y1, y2) > max(y3, y4) {
		return false
	}

	// 跨立实验（带浮点误差处理）
	c1 := cross(x1, y1, x2, y2, x3, y3)
	c2 := cross(x1, y1, x2, y2, x4, y4)
	c3 := cross(x3, y3, x4, y4, x1, y1)
	c4 := cross(x3, y3, x4, y4, x2, y2)

	// 当两线段端点重合时的特殊处理
	if c1 == 0 && c2 == 0 && c3 == 0 && c4 == 0 {
		// 共线且包围盒相交
		return true
	}

	// 处理浮点误差（1e-9容差）
	sign1 := signFunc(c1) * signFunc(c2)
	sign2 := signFunc(c3) * signFunc(c4)

	return sign1 <= 0 && sign2 <= 0
}

// 迭代法求解弹药前进时间
func solveBulletForwardTime(curX, curY, bulletSpeed, enemyX, enemyY, enemySpeed, enemyRotation float64) float64 {
	time := 0.0
	for i := 0; i < 100; i++ {
		// 100 次迭代
		rad := enemyRotation * math.Pi / 180
		targetX := enemyX + enemySpeed*time*math.Sin(rad)
		targetY := enemyY - enemySpeed*time*math.Cos(rad)
		distance := math.Hypot(targetX-curX, targetY-curY)
		actualTime := distance / bulletSpeed
		if math.Abs(actualTime-time) < 0.0001 {
			break
		}
		time = actualTime
	}
	return time
}

// CalcWeaponFireAngle 计算武器发射角度
func CalcWeaponFireAngle(
	curX, curY, bulletSpeed, enemyX, enemyY, enemySpeed, enemyRotation float64,
) (angle float64, targetX float64, targetY float64) {
	time := solveBulletForwardTime(curX, curY, bulletSpeed, enemyX, enemyY, enemySpeed, enemyRotation)

	rad := enemyRotation * math.Pi / 180
	targetX = enemyX + enemySpeed*time*math.Sin(rad)
	targetY = enemyY - enemySpeed*time*math.Cos(rad)
	angleRad := math.Atan2(targetY-curY, targetX-curX)

	return math.Mod(angleRad*180/math.Pi+90+360, 360), targetX, targetY
}

// RectContact 两个旋转矩形的接触信息。
type RectContact struct {
	// OK 是否接触
	OK bool
	// Depth 最小平移深度（地图格）
	Depth float64
	// NormalX / NormalY 从 B 指向 A 的单位法线
	NormalX, NormalY float64
	// PointX / PointY 近似接触点（地图格）
	PointX, PointY float64
}

// RotatedRectContact 用分离轴判断两个旋转矩形是否接触，并返回近似接触点。
// 长度 / 宽度使用地图格口径，角度为度（0 朝上，顺时针）。
func RotatedRectContact(
	ax, ay, aLen, aWid, aRot float64,
	bx, by, bLen, bWid, bRot float64,
) RectContact {
	a := rotatedRect{cx: ax, cy: ay, hl: aLen / 2, hw: aWid / 2, rot: aRot}
	b := rotatedRect{cx: bx, cy: by, hl: bLen / 2, hw: bWid / 2, rot: bRot}

	axes := [4][2]float64{
		rectForward(a), rectRight(a),
		rectForward(b), rectRight(b),
	}
	minOverlap := math.MaxFloat64
	best := axes[0]
	for _, axis := range axes {
		minA, maxA := projectRect(a, axis)
		minB, maxB := projectRect(b, axis)
		overlap := min(maxA, maxB) - max(minA, minB)
		if overlap <= 0 {
			return RectContact{}
		}
		if overlap < minOverlap {
			minOverlap, best = overlap, axis
		}
	}

	// 法线固定从 B 指向 A，后续撞击判定直接使用。
	if (ax-bx)*best[0]+(ay-by)*best[1] < 0 {
		best[0], best[1] = -best[0], -best[1]
	}
	pax, pay := closestPointOnRect(a, bx, by)
	pbx, pby := closestPointOnRect(b, ax, ay)
	return RectContact{
		OK: true, Depth: minOverlap,
		NormalX: best[0], NormalY: best[1],
		PointX: (pax + pbx) / 2, PointY: (pay + pby) / 2,
	}
}

type rotatedRect struct {
	cx, cy, hl, hw, rot float64
}

func rectForward(r rotatedRect) [2]float64 {
	s, c := math.Sin(r.rot*math.Pi/180), math.Cos(r.rot*math.Pi/180)
	return [2]float64{s, -c}
}

func rectRight(r rotatedRect) [2]float64 {
	s, c := math.Sin(r.rot*math.Pi/180), math.Cos(r.rot*math.Pi/180)
	return [2]float64{c, s}
}

func projectRect(r rotatedRect, axis [2]float64) (float64, float64) {
	f, right := rectForward(r), rectRight(r)
	extent := math.Abs(f[0]*axis[0]+f[1]*axis[1])*r.hl +
		math.Abs(right[0]*axis[0]+right[1]*axis[1])*r.hw
	center := r.cx*axis[0] + r.cy*axis[1]
	return center - extent, center + extent
}

// closestPointOnRect 返回矩形上离给定世界坐标最近的点。
func closestPointOnRect(r rotatedRect, x, y float64) (float64, float64) {
	f, right := rectForward(r), rectRight(r)
	dx, dy := x-r.cx, y-r.cy
	localF := min(max(dx*f[0]+dy*f[1], -r.hl), r.hl)
	localR := min(max(dx*right[0]+dy*right[1], -r.hw), r.hw)
	return r.cx + f[0]*localF + right[0]*localR, r.cy + f[1]*localF + right[1]*localR
}
