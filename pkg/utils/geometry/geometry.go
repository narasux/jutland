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
// IsPointInRotatedRect 判断点 (x, y) 是否在由中心点 (cx, cy)，旋转角度 angle，长度 length 和宽度 width 的长方形内
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
