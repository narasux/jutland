package ebutil

// CalcTextWidth 计算文本宽度
func CalcTextWidth(text string, fontSize float64) float64 {
	// 字体原因，宽度大致是 0.35 的文字高度
	return fontSize * float64(len(text)) / 20 * 7
}
