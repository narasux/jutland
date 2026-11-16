package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/utils/layout"
)

// TestCalcPauseUILayoutCentersPanelAndButtons 验证暂停面板居中、两按钮同高且以面板中线对称。
// 场景：1280×720 窗口打开暂停面板；面板 420×220，两个 112×38 按钮间距 34，需整体居中。
// 预期：Panel=(430,250,420,220)；按钮 Y=376；Primary 在左、Danger 在右，二者整体中心与面板中心重合。
// 覆盖：CalcPauseUILayout 后断言 Panel 坐标、按钮 Y、左右顺序、间距与整体中心。
func TestCalcPauseUILayoutCentersPanelAndButtons(t *testing.T) {
	ui := CalcPauseUILayout(layout.ScreenLayout{Width: 1280, Height: 720})

	require.Equal(t, 430.0, ui.Panel.X)
	require.Equal(t, 250.0, ui.Panel.Y)
	require.Equal(t, 420.0, ui.Panel.W)
	require.Equal(t, 220.0, ui.Panel.H)

	require.Equal(t, ui.PrimaryButton.Y, ui.DangerButton.Y)
	require.Less(t, ui.PrimaryButton.X, ui.DangerButton.X)
	require.InDelta(t, 34, ui.DangerButton.X-(ui.PrimaryButton.X+ui.PrimaryButton.W), 1e-9)
	pairCenter := (ui.PrimaryButton.X + ui.DangerButton.X + ui.DangerButton.W) / 2
	require.InDelta(t, ui.Panel.X+ui.Panel.W/2, pairCenter, 1e-9)
}

// TestPauseUIRectContainsUsesClosedEdges 验证暂停按钮命中测试包含左/上边界并排除右/下边界外一点。
// 场景：鼠标点在按钮边缘时，UI 应把边界算作命中，避免“差一像素点不到”；点击外部则必须放行。
// 预期：左上角、右下角、内部点、边界内一点为 true；右侧/下方越界一点为 false。
// 覆盖：Rect(10,20,100,50) 上调用 Contains，逐点断言。
func TestPauseUIRectContainsUsesClosedEdges(t *testing.T) {
	rect := PauseUIRect{X: 10, Y: 20, W: 100, H: 50}
	for _, tc := range []struct {
		name string
		x, y int
		want bool
	}{
		{name: "top left", x: 10, y: 20, want: true},
		{name: "bottom right", x: 110, y: 70, want: true},
		{name: "inside", x: 60, y: 45, want: true},
		{name: "just inside", x: 109, y: 69, want: true},
		{name: "right outside", x: 111, y: 45, want: false},
		{name: "bottom outside", x: 60, y: 71, want: false},
		{name: "left outside", x: 9, y: 45, want: false},
	} {
		require.Equal(t, tc.want, rect.Contains(tc.x, tc.y), tc.name)
	}
}

// TestApplyPauseInputStateTable 验证暂停面板 Esc/Q/确认放弃的状态迁移表。
// 场景：玩家在 Paused 态按 Esc（恢复）、按 Q（请求放弃）、再按 Q（确认失败）；非 Paused 态输入应被忽略。
// 预期：Running+任意输入→Running/false；Paused+None→保持 confirm；Paused+Resume+confirm=true→Paused/false；
// Paused+Resume+false→Running/false；Paused+Quit+false→Paused/true；Paused+Quit+true→Failed/true。
// 覆盖：表驱动七组 (status,confirm,input)，断言返回的下一状态与 confirm 标志。
func TestApplyPauseInputStateTable(t *testing.T) {
	cases := []struct {
		name    string
		status  MissionStatus
		confirm bool
		input   PauseInput
		want    MissionStatus
		wantCfm bool
	}{
		{name: "running ignores resume", status: MissionRunning, input: PauseInputResume, want: MissionRunning},
		{name: "running ignores quit", status: MissionRunning, input: PauseInputQuit, want: MissionRunning},
		{
			name:    "paused none keeps confirm",
			status:  MissionPaused,
			confirm: true,
			input:   PauseInputNone,
			want:    MissionPaused,
			wantCfm: true,
		},
		{
			name:    "paused resume cancels quit",
			status:  MissionPaused,
			confirm: true,
			input:   PauseInputResume,
			want:    MissionPaused,
		},
		{name: "paused resume returns running", status: MissionPaused, input: PauseInputResume, want: MissionRunning},
		{
			name:    "paused quit asks confirm",
			status:  MissionPaused,
			input:   PauseInputQuit,
			want:    MissionPaused,
			wantCfm: true,
		},
		{
			name:    "paused quit confirmed fails",
			status:  MissionPaused,
			confirm: true,
			input:   PauseInputQuit,
			want:    MissionFailed,
			wantCfm: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotCfm := ApplyPauseInput(tc.status, tc.confirm, tc.input)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantCfm, gotCfm)
		})
	}
}
