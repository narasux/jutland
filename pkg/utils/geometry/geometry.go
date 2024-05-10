package geometry

import "math"

// CalcAngleBetweenPoints 计算两个点之间的夹角（+90 转换成顺时针角度）
func CalcAngleBetweenPoints(x1, y1, x2, y2 float64) float64 {
	return math.Mod(math.Atan2(y2-y1, x2-x1)*180/math.Pi+90, 360)
}

// CalcDistance 计算两个点之间的距离
func CalcDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}
