package mark

import (
	"image/color"
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestDamageFlagTextKeepsSmallDamageVisible 验证伤害飘字取整规则不会把小额伤害显示为 0。
// 场景：分别调用 DamageFlagText(12.9) 与 DamageFlagText(0.4)，再用前者创建文字标记。
// 预期：12.9 截断得 "12"，0.4 保留两位小数得 "0.40"；NewText 的 Text="12" 且 Img=nil。
// 覆盖：两次字符串断言加 NewText 字段断言，确认走直接绘制文字而非图片标记。
func TestDamageFlagTextKeepsSmallDamageVisible(t *testing.T) {
	if got := DamageFlagText(12.9); got != "12" {
		t.Fatalf("damage text = %q, want 12", got)
	}
	if got := DamageFlagText(0.4); got != "0.40" {
		t.Fatalf("damage text = %q, want 0.40", got)
	}
	mark := NewText(objPos.New(1, 2), DamageFlagText(12.9), 16, color.White, 20)
	if mark.Text != "12" || mark.Img != nil {
		t.Fatalf("mark = %+v, want integer text and no cached image", mark)
	}
}
