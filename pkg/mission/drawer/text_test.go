package drawer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestCursorDebugLinesListEveryOverlappingUnitInStableOrder 验证光标下的重叠对象全部列出且按 Uid 稳定排序。
// 场景：航母 akagi（Uid b-ship、200×40）与战机 A6M2（Uid a-plane、20×20）的 CurPos 同为 (10,10)，
// map 遍历顺序随机会让同一位置的多行信息逐帧换序跳动；(60,60) 为无对象空海域。
// 预期：重叠处同时含 "Plane A6M2(a-plane)" 与 "Ship akagi(b-ship)"，且飞机行排在舰船行之前；
// 空海域返回 nil。
// 覆盖：cursorDebugLines 汇总 Ships/Planes 命中、按 Unit.ID() 排序以及无命中返回 nil。
func TestCursorDebugLinesListEveryOverlappingUnitInStableOrder(t *testing.T) {
	ship := &objUnit.BattleShip{
		Name: "akagi", Uid: "b-ship", CurHP: 100, TotalHP: 100,
		Length: 200, Width: 40, CurPos: objPos.NewR(10, 10),
	}
	plane := &objUnit.Plane{
		Name: "A6M2", Uid: "a-plane", Length: 20, Width: 20, CurPos: objPos.NewR(10, 10),
	}
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}

	lines := cursorDebugLines(ms, objPos.NewR(10, 10))
	require.NotEmpty(t, lines, "重叠对象应产生调试行")
	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "Plane A6M2(a-plane)")
	assert.Contains(t, joined, "Ship akagi(b-ship)")
	assert.Less(t, strings.Index(joined, "Plane A6M2"), strings.Index(joined, "Ship akagi"),
		"重叠对象应按 Uid 排序，飞机 a-plane 在舰船 b-ship 之前")

	assert.Nil(t, cursorDebugLines(ms, objPos.NewR(60, 60)), "空海域不应产生调试行")
}

// TestDebugPanelClearsVersionAndFPSLine 验证调试面板顶端让开每帧最后绘制的 "VER x FPS y" 行。
// 场景：pkg/game 在 (0,0) 用 Ebiten 点阵字体（字格 6×16，即高 16px）画版本/FPS，且叠在调试面板之上；
// 面板起点若低于 16px 会被压住第一行。
// 预期：debugOriginY（当前 20.0）≥ fpsLineHeight（16.0），并留出 4px 余量。
// 覆盖：直接断言调试面板起点常量与 FPS 行高的关系，不依赖 Ebiten 窗口。
func TestDebugPanelClearsVersionAndFPSLine(t *testing.T) {
	const fpsLineHeight = 16.0
	assert.GreaterOrEqual(t, debugOriginY, fpsLineHeight)
}

// TestPlaneBaseLabelResolvesBaseKind 验证飞行面板把 BelongShip 随机 uuid 解析成可读基地类型。
// 场景：基地分别为航母 akagi(c001)（ShipTypeAircraftCarrier）、巡洋舰 tone(s002)、机场 af01@(103,156)，
// 以及已不存在于 Ships/Airfields 的 uuid "sunk"；unit 包不能反向依赖 building，解析放在 drawer。
// 预期：依次得到 "carrier akagi(c001)"、"ship tone(s002)"、"airfield@(103,156)(af01)"；缺失基地返回空串。
// 覆盖：planeBaseLabel 经 FindAircraftBase 命中 BattleShip/Airfield 两条分支及缺失兜底。
func TestPlaneBaseLabelResolvesBaseKind(t *testing.T) {
	carrier := &objUnit.BattleShip{Uid: "c001", Name: "akagi", Type: objUnit.ShipTypeAircraftCarrier}
	cruiser := &objUnit.BattleShip{Uid: "s002", Name: "tone", Type: objUnit.ShipTypeCruiser}
	airfield := &objBuilding.Airfield{Uid: "af01", Pos: objPos.New(103, 156)}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships:     map[string]*objUnit.BattleShip{carrier.Uid: carrier, cruiser.Uid: cruiser},
		Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
	}}

	assert.Equal(t, "carrier akagi(c001)", planeBaseLabel(ms, &objUnit.Plane{BelongShip: "c001"}))
	assert.Equal(t, "ship tone(s002)", planeBaseLabel(ms, &objUnit.Plane{BelongShip: "s002"}))
	assert.Equal(t, "airfield@(103,156)(af01)", planeBaseLabel(ms, &objUnit.Plane{BelongShip: "af01"}))
	assert.Empty(t, planeBaseLabel(ms, &objUnit.Plane{BelongShip: "sunk"}), "基地已不存在时应返回空串")
}

// TestWrapDebugLinesFitsWidthAndKeepsContent 验证调试文本折行不超宽、不丢内容、保留空行且能硬拆超长单词。
// 场景：以 len(s) 为等宽假字体、maxWidth=40 折行含 US/BB/907/LB/AN-M66 释放器身份行与空行的输入；
// 另有单个词 "[1] "+95 个 "X"、maxWidth=20 的硬拆用例。
// 预期：所有行 ≤40；连续空白归一后与原文逐字相同；空行保留；硬拆后每行 ≤20 且去空白比较不丢字符。
// 覆盖：wrapDebugLines 的按单词换行、悬挂缩进、空行透传，以及 debugCutIndex 的 rune 级切分。
func TestWrapDebugLinesFitsWidthAndKeepsContent(t *testing.T) {
	// 等宽假字体：一个字符算 1 像素，便于断言。
	measure := func(s string) float64 { return float64(len(s)) }
	lines := []string{
		"  Bombs 0/8 released:",
		"    [1] US/BB/907/LB/AN-M66 (US/BB/2000/907, bomb): ready, range 4.50, speed 0.05",
		"        dist 50.95 (lead point), inRange false, inArc true, canDrop false",
		"",
		"  Guns: 5 (ready 5), MaxToShipRange: 4.50, MaxToPlaneRange: 1.25",
	}
	const maxWidth = 40

	wrapped := wrapDebugLines(lines, maxWidth, measure)
	for _, line := range wrapped {
		if len(line) > maxWidth {
			t.Fatalf("折行后仍超宽（%d > %d）：%q", len(line), maxWidth, line)
		}
	}
	// 折行只允许改变空白：把连续空白压成一个空格后内容必须逐字相同。
	normalize := func(in []string) string {
		return strings.Join(strings.Fields(strings.Join(in, " ")), " ")
	}
	if got, want := normalize(wrapped), normalize(lines); got != want {
		t.Fatalf("折行改变了内容:\n got: %s\nwant: %s", got, want)
	}
	// 空行必须保留，否则对象之间的分隔会消失。
	assert.Contains(t, wrapped, "", "空行应原样保留")

	// 单个词比整行还宽时按字符硬拆，且不丢字符。
	// 硬拆处会多出一个换行（渲染上就是一个空格），所以这里忽略全部空白比较。
	long := []string{"[1] " + strings.Repeat("X", 95)}
	hardWrapped := wrapDebugLines(long, 20, measure)
	strip := func(in []string) string {
		return strings.Join(strings.Fields(strings.Join(in, " ")), "")
	}
	assert.Equal(t, strip(long), strip(hardWrapped), "硬拆不应丢字符")
	for _, line := range hardWrapped {
		assert.LessOrEqual(t, len(line), 20, "硬拆后仍超宽: %q", line)
	}
}
