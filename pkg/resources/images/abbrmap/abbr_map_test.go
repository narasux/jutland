package abbrmap

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewCompositePreservesRectangularMapAspectRatio 验证矩形地图缩略图按地图宽高比算宽度。
// 场景：pearl_harbor 网格 128×160、目标高 100；若直接按高建方图，缩略图会被横向拉伸。
// 预期：NewComposite 返回 100*128/160=80 宽、100 高，即 image.Pt(80,100)；
// 失败表示缩略图比例与地图不一致。
// 覆盖：调用 NewComposite("pearl_harbor",128,160,100) 并对 Bounds().Size() 全等断言。
func TestNewCompositePreservesRectangularMapAspectRatio(t *testing.T) {
	img := NewComposite("pearl_harbor", 128, 160, 100)
	require.Equal(t, image.Pt(80, 100), img.Bounds().Size())
}
