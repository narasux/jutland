package geometry

import "math"

// CalcAngleBetweenPoints 计算两个点之间的夹角
func CalcAngleBetweenPoints(x1, y1, x2, y2 float64) float64 {
	return math.Atan2(y2-y1, x2-x1) * 180 / math.Pi
}
