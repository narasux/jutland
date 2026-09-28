package sidebar

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
)

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
