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

// 光标位置可能同时压住多个对象（例如战机正停在航母上方）。调试面板必须把
// 命中的对象全部列出来，并且按 Uid 稳定排序：map 遍历顺序随机，不排序会让
// 同一个位置的多行信息逐帧换序跳动。
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

// pkg/game 每帧最后在 (0,0) 用 Ebiten 点阵字体画 "VER x FPS y"，而且画在调试
// 面板之上。面板起点必须低于那一行，否则第一行会被压住（字格 6x16，即高 16px）。
func TestDebugPanelClearsVersionAndFPSLine(t *testing.T) {
	const fpsLineHeight = 16.0
	assert.GreaterOrEqual(t, debugOriginY, fpsLineHeight)
}

// 飞机面板要能看出所属基地是航母、其他载机舰船还是陆地机场：
// BelongShip 只是个随机 uuid，光看它认不出类型，也认不出是哪一艘。
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

// 释放器配置名很长（如 US/BB/907/LB/AN-M66），单行会横穿整个屏幕。
// 折行必须保证：所有行都在宽度内、单词不丢、超长单词能被硬拆。
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
