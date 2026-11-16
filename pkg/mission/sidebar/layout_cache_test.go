package sidebar

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
)

// TestEnsureLayoutReusesUnchangedInputs 验证布局缓存以屏幕尺寸/展开状态/地图宽高比为键，键不变时复用。
// 场景：每帧 Update/Draw 都调 ensureLayout；若每帧重算 calcLayout，窗口尺寸不变时会产生无谓分配。
// 预期：第二次返回同一缓存（手工写入的 -1 被保留），首次宽度非 -1；失败意味着缓存命中的键比较失效。
// 覆盖：mapAspect=1.5、1280×720，两次 ensureLayout，中间把 panel.layout.Panel.W 改成 -1 作探针。
func TestEnsureLayoutReusesUnchangedInputs(t *testing.T) {
	panel := &Panel{mapAspect: 1.5}
	ms := &state.MissionState{}
	ms.View.Layout.Width = 1280
	ms.View.Layout.Height = 720

	first := panel.ensureLayout(ms)
	panel.layout.Panel.W = -1
	second := panel.ensureLayout(ms)
	if second.Panel.W != -1 || first.Panel.W == -1 {
		t.Fatalf("second layout = %+v, want the cached panel", second.Panel)
	}
}
