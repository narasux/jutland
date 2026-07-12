package unitpanel

import (
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

const renderTestEnv = "JUTLAND_UNITPANEL_RENDER_TEST"
const captureDirEnv = "JUTLAND_UNITPANEL_CAPTURE_DIR"

var (
	renderDrawCh = make(chan func())
	renderEndCh  chan struct{}
)

type renderTestGame struct{}

func (*renderTestGame) Update() error {
	select {
	case <-renderEndCh:
		return ebiten.Termination
	case draw := <-renderDrawCh:
		// 在游戏循环线程执行真实绘制，避免后台窗口暂停 Draw 时阻塞截图流程。
		draw()
		return nil
	default:
		return nil
	}
}

func (*renderTestGame) Draw(*ebiten.Image) {}

func (*renderTestGame) Layout(_, _ int) (int, int) { return 1280, 720 }

func TestMain(m *testing.M) {
	if os.Getenv(renderTestEnv) == "" && os.Getenv(captureDirEnv) == "" {
		os.Exit(m.Run())
	}
	codeCh := make(chan int)
	renderEndCh = make(chan struct{})
	go func() {
		codeCh <- m.Run()
		close(renderEndCh)
	}()
	// 截图验收可能由后台终端触发；即使测试窗口没有焦点，也必须持续提交绘制帧。
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(&renderTestGame{}); err != nil {
		panic(err)
	}
	os.Exit(<-codeCh)
}

func TestCaptureUnitPanelScenarios(t *testing.T) {
	captureDir := os.Getenv(captureDirEnv)
	if captureDir == "" {
		t.Skip("screenshots are generated only by the explicit capture workflow")
	}
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		name       string
		screen     layout.ScreenLayout
		rightInset float64
		state      *state.MissionState
	}{
		{name: "1280-collapsed", screen: layout.ScreenLayout{Width: 1280, Height: 720}, state: captureState(nil, false)},
		{name: "1280-battleship-weapons", screen: layout.ScreenLayout{Width: 1280, Height: 720}, state: captureState([]*objUnit.BattleShip{captureBattleship()}, true)},
		{name: "1280-carrier-aircraft", screen: layout.ScreenLayout{Width: 1280, Height: 720}, state: captureState([]*objUnit.BattleShip{captureCarrier()}, true)},
		{name: "1920-multi-with-right-inset", screen: layout.ScreenLayout{Width: 1920, Height: 1080}, rightInset: 360, state: captureState([]*objUnit.BattleShip{captureBattleship(), captureCarrier()}, true)},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			scenario.state.View.Layout = scenario.screen
			runOnRenderFrame(func() {
				screen := ebiten.NewImage(scenario.screen.Width, scenario.screen.Height)
				screen.Fill(color.RGBA{R: 16, G: 48, B: 58, A: 255})
				panel := New()
				panel.updateWithPointer(scenario.state, scenario.rightInset, pointerInput{})
				panel.Draw(screen, scenario.state, scenario.rightInset)
				path := filepath.Join(captureDir, scenario.name+".png")
				file, err := os.Create(path)
				if err != nil {
					t.Error(err)
					return
				}
				defer file.Close()
				if err = png.Encode(file, screen); err != nil {
					t.Error(err)
				}
			})
		})
	}
}

func captureState(ships []*objUnit.BattleShip, expanded bool) *state.MissionState {
	shipMap := make(map[string]*objUnit.BattleShip, len(ships)+1)
	selected := make([]string, 0, len(ships))
	for _, ship := range ships {
		shipMap[ship.Uid] = ship
		selected = append(selected, ship.Uid)
	}
	if len(ships) > 0 {
		target := &objUnit.BattleShip{Uid: "target", Name: "hood", CurHP: 1, CurPos: objPos.NewR(45, 36)}
		shipMap[target.Uid] = target
		ships[0].AttackTarget = target.Uid
	}
	focus := ""
	if len(ships) > 0 {
		focus = ships[0].Uid
	}
	planes := map[string]*objUnit.Plane{}
	for _, ship := range ships {
		if !ship.Aircraft.HasPlane {
			continue
		}
		planes["fighter-active"] = &objUnit.Plane{Name: "F6F-3", BelongShip: ship.Uid, FlightPhase: objUnit.PlaneFlightPhaseCruising}
		planes["fighter-return"] = &objUnit.Plane{Name: "F6F-3", BelongShip: ship.Uid, FlightPhase: objUnit.PlaneFlightPhaseLandingApproach}
	}
	return &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Interaction: state.MissionInteractionState{
			SelectedShips:  selected,
			FocusedShipUid: focus,
		},
		Arena: state.MissionArenaState{Ships: shipMap, Planes: planes},
		UI:    state.MissionUIState{UnitPanelExpanded: expanded},
	}
}

func captureBattleship() *objUnit.BattleShip {
	now := time.Now().UnixMilli()
	return &objUnit.BattleShip{
		Uid: "bismarck-1", Name: "bismarck", Type: objUnit.ShipTypeBattleShip,
		TotalHP: 10_000, CurHP: 8_200, MaxSpeed: 0.05, CurSpeed: 0.035,
		CurRotation: 65, CurPos: objPos.NewR(30, 30), GroupID: object.GroupID2,
		Weapon: objUnit.ShipWeapon{
			MainGuns: []*objUnit.Gun{
				{ReloadStartAt: now - 1_000, ReloadTime: 3},
				{ReloadStartAt: 0, ReloadTime: 1},
			},
			SecondaryGuns:    []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
			AntiAircraftGuns: []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
		},
	}
}

func captureCarrier() *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: "essex-1", Name: "essex", Type: objUnit.ShipTypeAircraftCarrier,
		TotalHP: 9_000, CurHP: 8_100, MaxSpeed: 0.055, CurSpeed: 0.04,
		CurRotation: 128, CurPos: objPos.NewR(34, 33),
		Aircraft: objUnit.ShipAircraft{HasPlane: true, Groups: []objUnit.PlaneGroup{
			{Name: "F6F-3", MaxCount: 24, CurCount: 18},
			{Name: "TBF-1", MaxCount: 12, CurCount: 8},
			{Name: "SB2C-4", MaxCount: 18, CurCount: 12},
		}},
		Weapon: objUnit.ShipWeapon{AntiAircraftGuns: []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}}},
	}
}

func runOnRenderFrame(draw func()) {
	done := make(chan struct{})
	renderDrawCh <- func() {
		draw()
		close(done)
	}
	<-done
}

func TestCollapsedHandleRendersInsideItsHitRegion(t *testing.T) {
	if os.Getenv(renderTestEnv) == "" {
		t.Skip("real Ebiten rendering is enabled by the explicit render-test workflow")
	}
	missionState := panelInteractionState()
	runOnRenderFrame(func() {
		panel := New()
		screen := ebiten.NewImage(missionState.View.Layout.Width, missionState.View.Layout.Height)
		panel.Draw(screen, missionState, 0)

		handle := calcLayout(missionState.View.Layout, false, 0).Handle
		center := color.RGBAModel.Convert(screen.At(int(handle.X+handle.W/2), int(handle.Y+handle.H/2))).(color.RGBA)
		if center.A == 0 {
			t.Error("collapsed handle center is fully transparent")
		}
		outside := color.RGBAModel.Convert(screen.At(int(handle.X)-2, int(handle.Y+handle.H/2))).(color.RGBA)
		if outside.A != 0 {
			t.Errorf("pixels outside handle unexpectedly changed: %+v", outside)
		}
	})
}

func TestHandleForegroundHasVisibleContrast(t *testing.T) {
	if ratio := contrastRatio(handleFill, colorx.Gold); ratio < 3 {
		t.Fatalf("handle arrow contrast ratio = %.2f, want at least 3.0", ratio)
	}
	if ratio := contrastRatio(handleFill, color.White); ratio < 3 {
		t.Fatalf("handle text contrast ratio = %.2f, want at least 3.0", ratio)
	}
}

func contrastRatio(a, b color.Color) float64 {
	left, right := relativeLuminance(a), relativeLuminance(b)
	if left < right {
		left, right = right, left
	}
	return (left + 0.05) / (right + 0.05)
}

func relativeLuminance(value color.Color) float64 {
	r, g, b, _ := value.RGBA()
	linear := func(channel uint32) float64 {
		v := float64(channel) / 65535
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}
