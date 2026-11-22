package mapcfg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
)

// TestLocalizedDisplayNameUsesTraditionalChinese 验证地图展示名跟随 zh-Hant，并在缺失时回退简体中文。
// 场景：设置页切到繁体中文后查看关卡副标题；地图配置同时提供简繁英俄日五种名称。
// 预期：zh-Hant 返回「達爾文港」，zh-Hans 返回「达尔文港」，en 返回「Darwin Harbour」；
// 清空繁体名称后 zh-Hant 回退「达尔文港」；失败表示 mapcfg 未接入回退链或语言切换未生效。
// 覆盖：构造含五种名称的 MapCfg，依次 SetLanguage 三次断言 LocalizedDisplayName，再清空 DisplayNameZhHant 验证回退，Cleanup 复位原语言。
func TestLocalizedDisplayNameUsesTraditionalChinese(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	cfg := &MapCfg{
		DisplayName:       "达尔文港",
		DisplayNameZhHant: "達爾文港",
		DisplayNameEn:     "Darwin Harbour",
		DisplayNameRu:     "Порт-Дарвин",
		DisplayNameJa:     "ダーウィン港",
	}

	i18n.SetLanguage(string(i18n.LanguageZhHant))
	require.Equal(t, "達爾文港", cfg.LocalizedDisplayName())
	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Equal(t, "达尔文港", cfg.LocalizedDisplayName())
	i18n.SetLanguage(string(i18n.LanguageEnglish))
	require.Equal(t, "Darwin Harbour", cfg.LocalizedDisplayName())

	cfg.DisplayNameZhHant = ""
	i18n.SetLanguage(string(i18n.LanguageZhHant))
	require.Equal(t, "达尔文港", cfg.LocalizedDisplayName())
}

// TestMapDataGetSupportsRectangularMaps 验证 MapData.Get 支持非正方形、行宽不一致的地图。
// 场景：宽图 abcd/efgh（2×4）与高图 ab/cd/ef（3×2）并存；读取必须按行索引而非假设方阵。
// 预期：(3,1)='h'、(1,2)='f'；x=4（宽图）、x=2（高图）、y=3 越界都返回 ' '；
// 失败表示矩形地图会取到错误地块或漏掉越界保护。
// 覆盖：表驱动 5 个用例（宽/高图各含界内与 x 越界，另加 y 越界），逐项 require.Equal。
func TestMapDataGetSupportsRectangularMaps(t *testing.T) {
	tests := []struct {
		name string
		data MapData
		x    int
		y    int
		want rune
	}{
		{name: "wide map in bounds", data: MapData{"abcd", "efgh"}, x: 3, y: 1, want: 'h'},
		{name: "wide map x out of bounds", data: MapData{"abcd", "efgh"}, x: 4, y: 1, want: ' '},
		{name: "tall map in bounds", data: MapData{"ab", "cd", "ef"}, x: 1, y: 2, want: 'f'},
		{name: "tall map x out of bounds", data: MapData{"ab", "cd", "ef"}, x: 2, y: 1, want: ' '},
		{name: "y out of bounds", data: MapData{"ab", "cd", "ef"}, x: 0, y: 3, want: ' '},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.data.Get(tt.x, tt.y))
		})
	}
}
