package unitpanel

import (
	"testing"

	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestPanelLayoutKeepsHandleAndColumnsOnScreen(t *testing.T) {
	tests := []struct {
		name       string
		screen     layout.ScreenLayout
		rightInset float64
	}{
		{name: "1280x720", screen: layout.ScreenLayout{Width: 1280, Height: 720}},
		{name: "1280x720 with sidebar", screen: layout.ScreenLayout{Width: 1280, Height: 720}, rightInset: 307},
		{name: "1920x1080", screen: layout.ScreenLayout{Width: 1920, Height: 1080}},
		{name: "1920x1080 with sidebar", screen: layout.ScreenLayout{Width: 1920, Height: 1080}, rightInset: 360},
	}
	for _, testCase := range tests {
		for _, expanded := range []bool{false, true} {
			t.Run(testCase.name+map[bool]string{false: "/collapsed", true: "/expanded"}[expanded], func(t *testing.T) {
				ui := calcLayout(testCase.screen, expanded, testCase.rightInset)
				assertRectInside(t, ui.Handle, rect{W: float64(testCase.screen.Width), H: float64(testCase.screen.Height)})
				if !expanded {
					if got := float64(testCase.screen.Height) - (ui.Handle.Y + ui.Handle.H); got != handleBottomMargin {
						t.Fatalf("collapsed handle bottom margin = %v, want %v", got, handleBottomMargin)
					}
					return
				}
				assertRectInside(t, ui.Panel, rect{W: float64(testCase.screen.Width) - testCase.rightInset, H: float64(testCase.screen.Height)})
				if ui.Handle.Y+ui.Handle.H != ui.Panel.Y {
					t.Fatalf("expanded handle bottom = %v, panel top = %v", ui.Handle.Y+ui.Handle.H, ui.Panel.Y)
				}
				if ui.Visual.X+ui.Visual.W > ui.Info.X || ui.Info.X+ui.Info.W > ui.Systems.X {
					t.Fatalf("columns overlap: visual=%+v info=%+v systems=%+v", ui.Visual, ui.Info, ui.Systems)
				}
				assertRectInside(t, ui.Visual, ui.Panel)
				assertRectInside(t, ui.Info, ui.Panel)
				assertRectInside(t, ui.Systems, ui.Panel)
			})
		}
	}
}

func TestFiveWeaponRowsFitSystemsColumn(t *testing.T) {
	p := New()
	p.layout = calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, 307)
	for index := range 5 {
		assertRectInside(t, p.weaponToggleRect(index, 5), p.layout.Systems)
	}
}

func assertRectInside(t *testing.T, inner, outer rect) {
	t.Helper()
	if inner.X < outer.X || inner.Y < outer.Y ||
		inner.X+inner.W > outer.X+outer.W || inner.Y+inner.H > outer.Y+outer.H {
		t.Fatalf("rect %+v is outside %+v", inner, outer)
	}
}
