//go:build manual

// 手工截图验收：JUTLAND_UNITPANEL_CAPTURE_DIR=... go test -tags manual -run TestCaptureUnitPanelScenarios ./pkg/mission/unitpanel
package unitpanel

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

const (
	renderTestEnv = "JUTLAND_UNITPANEL_RENDER_TEST"
	captureDirEnv = "JUTLAND_UNITPANEL_CAPTURE_DIR"
)

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

// TestMain 验证截图测试入口：仅在显式环境变量下启动 Ebiten 主循环，否则直接跑普通测试。
// 场景：设置了 JUTLAND_UNITPANEL_RENDER_TEST 或 JUTLAND_UNITPANEL_CAPTURE_DIR 时，绘制必须跑在游戏循环线程；
// 未设置时 CI/无窗口环境应正常执行其余测试。
// 预期：两变量为空则 os.Exit(m.Run())；否则 RunGame 持续出帧，测试完成后以测试码退出。
// 覆盖：后台 goroutine 跑 m.Run，经 renderEndCh 通知 Update 返回 Termination，另设 SetRunnableOnUnfocused。
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

// TestCaptureUnitPanelScenarios 验证截图验收流程能按三种场景绘制单位面板并导出 PNG。
// 场景：未设 JUTLAND_UNITPANEL_CAPTURE_DIR 时跳过；设置后借 Ebiten 主循环绘制，避免后台窗口暂停 Draw。
// 预期：未设目录 t.Skip；设置后为 3 个场景各生成 <name>.png，绘制与编码不报错；失败意味着渲染或面板绘制崩溃。
// 覆盖：1280 战列舰、1280 航母、1920 双舰三场景，region 宽 280，runOnRenderFrame 内 Draw 并 png.Encode。
func TestCaptureUnitPanelScenarios(t *testing.T) {
	captureDir := os.Getenv(captureDirEnv)
	if captureDir == "" {
		t.Skip("screenshots are generated only by the explicit capture workflow")
	}
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		name   string
		screen layout.ScreenLayout
		state  *state.MissionState
	}{
		{
			name:   "1280-battleship-weapons",
			screen: layout.ScreenLayout{Width: 1280, Height: 720},
			state:  captureState([]*objUnit.BattleShip{captureBattleship()}),
		},
		{
			name:   "1280-carrier-aircraft",
			screen: layout.ScreenLayout{Width: 1280, Height: 720},
			state:  captureState([]*objUnit.BattleShip{captureCarrier()}),
		},
		{
			name:   "1920-multi",
			screen: layout.ScreenLayout{Width: 1920, Height: 1080},
			state:  captureState([]*objUnit.BattleShip{captureBattleship(), captureCarrier()}),
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			scenario.state.View.Layout = scenario.screen
			region := Rect{
				X: float64(scenario.screen.Width) - 312,
				Y: 40,
				W: 280,
				H: float64(scenario.screen.Height) - 80,
			}
			runOnRenderFrame(func() {
				screen := ebiten.NewImage(scenario.screen.Width, scenario.screen.Height)
				screen.Fill(color.RGBA{R: 16, G: 48, B: 58, A: 255})
				panel := New()
				panel.updateWithPointer(scenario.state, region, 0, pointerInput{})
				panel.Draw(screen, scenario.state, region, 0)
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

func captureState(ships []*objUnit.BattleShip) *state.MissionState {
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
		planes["fighter-active"] = &objUnit.Plane{
			Name:        "F6F-3",
			BelongShip:  ship.Uid,
			FlightPhase: objUnit.PlaneFlightPhaseCruising,
		}
		planes["fighter-return"] = &objUnit.Plane{
			Name:        "F6F-3",
			BelongShip:  ship.Uid,
			FlightPhase: objUnit.PlaneFlightPhaseLandingApproach,
		}
	}
	return &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Interaction: state.MissionInteractionState{
			SelectedShips:  selected,
			FocusedShipUid: focus,
		},
		Arena: state.MissionArenaState{Ships: shipMap, Planes: planes},
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
