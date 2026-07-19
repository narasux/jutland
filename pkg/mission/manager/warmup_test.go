package manager

import (
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/stretchr/testify/require"

	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestWarmupMapBlocksReportsCurrentViewReadiness(t *testing.T) {
	m := New("Midway1942", &ebitenui.UI{})
	m.state.View.Camera.Pos = objPos.New(100, 137)
	m.state.View.Camera.Width = 30
	m.state.View.Camera.Height = 20

	require.False(t, m.WarmupMapBlocks())

	ready := false
	for range 100 {
		if m.WarmupMapBlocks() {
			ready = true
			break
		}
	}
	require.True(t, ready)
	require.True(t, m.WarmupMapBlocks())
}
