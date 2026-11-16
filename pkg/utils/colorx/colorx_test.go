package colorx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetColorByNameNormalizesInputAndFallsBackToNil 验证颜色名大小写/空格归一，未知名称返回 nil。
// 场景：配置/终端允许玩家输入 "Sky Blue"、":set color Gold" 等带空格大小写的名称；
// 调用方用 nil 判断“颜色不存在”，不能返回黑色冒充。
// 预期：" sky blue "、"SKYBLUE" 都命中 SkyBlue；"Gold" 命中 Gold；"not-a-color" 返回 nil。
// 覆盖：四组名称调用 GetColorByName，已知颜色比较 RGBA 值，未知名称 require.Nil。
func TestGetColorByNameNormalizesInputAndFallsBackToNil(t *testing.T) {
	for _, name := range []string{" sky blue ", "SKYBLUE", "SkyBlue"} {
		got := GetColorByName(name)
		require.NotNil(t, got)
		require.Equal(t, SkyBlue, got, name)
	}

	got := GetColorByName(" Gold ")
	require.NotNil(t, got)
	require.Equal(t, Gold, got)

	require.Nil(t, GetColorByName("not-a-color"))
}
