package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestDetailSpeedUsesSameSpeedMultiplier 验证调试面板的速度口径随全局速度倍率一起缩放，并打印飞行阶段。
// 场景：CurSpeed 是世界系速度（已乘全局速度倍率），比较基准若仍用未乘倍率的 MaxSpeed，
// 在“快”（SpeedMultiplier=2.0）倍速下，每一架巡航中的飞机、每一艘船都会被显示成超过速度上限。
// 同时起降阶段（滑跑 / 待场 / 进近）的速度是世界系位移，允许高于设计上限，
// 调试行要能一眼看出超速读数来自哪个阶段。
// 预期：SpeedMultiplier=2 时，A6M2（MaxSpeed=533/PlaneSpeedScale）巡航行含
// "Speed: <CurSpeed>/<MaxSpeed×2>" 与 "Phase: cruising"；切到 landing_staging、CurSpeed=0.09 后
// 阶段标成 landing_staging 且基准仍为 MaxSpeed×2；akagi（31kn）的 Detail 同样按 ×2 基准。
// 覆盖：手动设置 config.G.SpeedMultiplier=2，构造巡航飞机、待场飞机与航母，断言 Detail 字符串内容。
func TestDetailSpeedUsesSameSpeedMultiplier(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 2}

	plane := &Plane{
		Name:        "A6M2",
		Uid:         "p",
		MaxSpeed:    533.0 / PlaneSpeedScale,
		CurSpeed:    533.0 / PlaneSpeedScale * 2,
		CurHP:       26,
		TotalHP:     26,
		FlightPhase: PlaneFlightPhaseCruising,
		CurPos:      objPos.NewR(10, 20),
	}
	got := plane.Detail()
	if want := fmt.Sprintf("Speed: %.2f/%.2f", plane.CurSpeed, plane.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("巡航速度口径不一致: %s, 期望包含 %s", got, want)
	}
	if !strings.Contains(got, "Phase: cruising") {
		t.Fatalf("调试行缺少飞行阶段: %s", got)
	}

	// 待场进近的速度是世界系位移，允许高于设计上限；阶段字段必须能区分出来。
	plane.FlightPhase = PlaneFlightPhaseLandingStaging
	plane.CurSpeed = 0.09
	got = plane.Detail()
	if !strings.Contains(got, "Phase: landing_staging") {
		t.Fatalf("待场阶段未标注: %s", got)
	}
	if want := fmt.Sprintf("Speed: %.2f/%.2f", 0.09, plane.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("待场速度口径不一致: %s, 期望包含 %s", got, want)
	}

	ship := &BattleShip{
		Name:     "akagi",
		Uid:      "s",
		MaxSpeed: 31.0 / ShipSpeedScale,
		CurSpeed: 31.0 / ShipSpeedScale * 2,
		CurHP:    100,
		TotalHP:  100,
		CurPos:   objPos.NewR(5, 6),
	}
	got = ship.Detail()
	if want := fmt.Sprintf("Speed: %.2f/%.2f", ship.CurSpeed, ship.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("舰船速度口径不一致: %s, 期望包含 %s", got, want)
	}
}

// TestPlaneDetailLinesReportReleaserState 验证释放器调试行逐项输出投放判定，能解释“为什么不投弹”。
// 场景：B5N2 挂 1 枚 250kg 炸弹（射程 2、弹速 0.4、射界 0–180/180–360）和 1 枚已投放的 91 式鱼雷
// （射程 1、弹速 0.2）；敌舰在正南 1 格时炸弹应可投、鱼雷已投；目标移到 10 格外时炸弹应超射程。
// 飞机“为什么不投弹”只有几种可能：已投放、投放间隔闸门、目标门槛、射程、射界、鱼雷航迹压陆地，
// 调试行必须把每枚炸弹 / 鱼雷的 Released 状态和逐项判定都打出来，否则只能靠猜。
// 预期：输出 "Bombs 0/1 released:"、炸弹行 "[1] 250kg (bomb250, bomb): ready"、
// "inRange true, inArc true, canDrop true"、鱼雷行 "[1] type91 (torp91, torpedo): released"、
// "ReleaseGate: ready true"；已投放鱼雷 canDrop false；敌舰移到 10 格外后 inRange false。
// 覆盖：构造炸弹 + 鱼雷挂载，分别以 1 格与 10 格敌舰调用 DetailLines(DetailContext{Enemy}) 并拼接断言。
func TestPlaneDetailLinesReportReleaserState(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 1}

	plane := &Plane{
		Name:   "B5N2",
		Uid:    "p",
		Type:   PlaneTypeTorpedoBomber,
		CurPos: objPos.NewR(10, 10),
		Weapon: PlaneWeapon{
			Bombs: []*Releaser{{
				Name: "250kg", BulletName: "bomb250", BulletType: objBullet.TypeBomb,
				Range: 2, BulletSpeed: 0.4,
				LeftFiringArc: FiringArc{Start: 0, End: 180}, RightFiringArc: FiringArc{Start: 180, End: 360},
			}},
			Torpedoes: []*Releaser{{
				Name: "type91", BulletName: "torp91", BulletType: objBullet.TypeTorpedo,
				Range: 1, BulletSpeed: 0.2, Released: true,
			}},
		},
	}
	// 目标在 1 格外，炸弹射程内、鱼雷射程外（鱼雷只投放至 80% 射程）。
	enemy := &BattleShip{Uid: "e", CurPos: objPos.NewR(10, 11)}

	joined := strings.Join(plane.DetailLines(DetailContext{Enemy: enemy}), "\n")
	for _, want := range []string{
		"Bombs 0/1 released:",
		"[1] 250kg (bomb250, bomb): ready",
		"dist ",
		"inRange true, inArc true, canDrop true",
		"Torpedoes 1/1 released:",
		"[1] type91 (torp91, torpedo): released",
		"ReleaseGate: ready true",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("释放器调试行缺少 %q:\n%s", want, joined)
		}
	}
	// 已投放的鱼雷不能再算作可投放，否则面板会显示成“能投却一直不投”。
	if !strings.Contains(joined, "canDrop false") {
		t.Fatalf("已投放的释放器仍被标记为可投放:\n%s", joined)
	}

	// 目标移出炸弹射程：几何判定要能解释“没投弹”。
	enemy.CurPos = objPos.NewR(10, 20)
	joined = strings.Join(plane.DetailLines(DetailContext{Enemy: enemy}), "\n")
	if !strings.Contains(joined, "inRange false") {
		t.Fatalf("超射程目标未标记 inRange false:\n%s", joined)
	}
}

// TestPlaneDetailLinesMergeIdenticalReleasers 验证渲染完全相同的释放器合并成一组，状态不同则各自成组。
// 场景：B-29A 挂 3 枚同名 250kg 炸弹，前两枚未投放、第三枚已投放；整舱挂载常是同型号重复，
// 逐枚铺开会把面板撑到半个屏幕高，但合并不能吞掉“第 3 枚已经投掉了”这类状态差异。
// 预期：输出 "[1,2] 250kg (bomb250, bomb): ready" 与 "[3] 250kg (bomb250, bomb): released"。
// 覆盖：无 Enemy 上下文调用 DetailLines，按行前缀断言两组合并结果。
func TestPlaneDetailLinesMergeIdenticalReleasers(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 1}

	bomb := func(released bool) *Releaser {
		return &Releaser{
			Name: "250kg", BulletName: "bomb250", BulletType: objBullet.TypeBomb,
			Range: 2, BulletSpeed: 0.4, Released: released,
		}
	}
	plane := &Plane{
		Name: "B-29A", Uid: "p", Type: PlaneTypeLevelBomber, CurPos: objPos.NewR(10, 10),
		Weapon: PlaneWeapon{Bombs: []*Releaser{bomb(false), bomb(false), bomb(true)}},
	}

	joined := strings.Join(plane.DetailLines(DetailContext{}), "\n")
	if !strings.Contains(joined, "[1,2] 250kg (bomb250, bomb): ready") {
		t.Fatalf("同状态释放器未合并:\n%s", joined)
	}
	if !strings.Contains(joined, "[3] 250kg (bomb250, bomb): released") {
		t.Fatalf("已投放释放器未单独成组:\n%s", joined)
	}
}

// TestPlaneDetailLinesShowReadableBase 验证所属基地的展示口径：可读标识 → missing+uuid → none。
// 场景：BelongShip 是随机 uuid，单看它认不出是航母、载机舰船还是陆地机场；
// 基地被击沉等导致解析不到时必须显式标成 missing，而不是静默退回裸 uuid；
// BelongShip 本身为空（初始就没有基地）时展示 none。
// 预期：BaseLabel="carrier akagi(c001)" 时输出 "Base: carrier akagi(c001)"；
// 无 label 但有 uuid 时输出 "Base: missing cb9f52ce-23f7-4b00-970e-51227ec8d4ef"；
// BelongShip 清空后输出 "Base: none"。
// 覆盖：三次调用 DetailLines，分别传入不同 DetailContext.BaseLabel 与 BelongShip。
func TestPlaneDetailLinesShowReadableBase(t *testing.T) {
	plane := &Plane{
		Name: "B5N2", Uid: "p",
		BelongShip: "cb9f52ce-23f7-4b00-970e-51227ec8d4ef",
	}

	joined := strings.Join(
		plane.DetailLines(DetailContext{BaseLabel: "carrier akagi(c001)"}), "\n",
	)
	if !strings.Contains(joined, "Base: carrier akagi(c001)") {
		t.Fatalf("所属基地未展示可读标识:\n%s", joined)
	}

	joined = strings.Join(plane.DetailLines(DetailContext{}), "\n")
	if !strings.Contains(joined, "Base: missing cb9f52ce-23f7-4b00-970e-51227ec8d4ef") {
		t.Fatalf("基地缺失时未标记 missing:\n%s", joined)
	}

	plane.BelongShip = ""
	joined = strings.Join(plane.DetailLines(DetailContext{}), "\n")
	if !strings.Contains(joined, "Base: none") {
		t.Fatalf("无基地飞机未展示 none:\n%s", joined)
	}
}

// TestPlaneDetailLinesGateTorpedoOnGroundTargets 验证目标门槛的调试输出：鱼雷挂在唯一释放器上时，
// 面板必须直接说明为什么不能投放。
// 场景：B5N2 只挂 1 枚 91 式鱼雷（射程 1、弹速 0.2），目标是处于 PlaneFlightPhaseTakingOff
// 的地面飞机；鱼雷无法在陆地使用，targetAcceptable 对地面飞机只放行炸弹。
// 预期：输出 "targetOK false (not droppable target)"，而不是继续打印 dist/inRange/inArc 判定。
// 覆盖：以 DetailContext{Enemy: grounded} 调用 DetailLines，断言包含 "targetOK false"。
func TestPlaneDetailLinesGateTorpedoOnGroundTargets(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 1}

	plane := &Plane{
		Name:   "B5N2",
		Uid:    "p",
		Type:   PlaneTypeTorpedoBomber,
		CurPos: objPos.NewR(10, 10),
		Weapon: PlaneWeapon{
			Torpedoes: []*Releaser{{
				Name: "type91", BulletName: "torp91", BulletType: objBullet.TypeTorpedo,
				Range: 1, BulletSpeed: 0.2,
			}},
		},
	}
	grounded := &Plane{Name: "A6M2", Uid: "g", CurPos: objPos.NewR(10, 10.5), FlightPhase: PlaneFlightPhaseTakingOff}

	joined := strings.Join(plane.DetailLines(DetailContext{Enemy: grounded}), "\n")
	if !strings.Contains(joined, "targetOK false") {
		t.Fatalf("鱼雷对地面目标未标记 targetOK false:\n%s", joined)
	}
}
