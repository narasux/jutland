package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestDetailSpeedUsesSameSpeedMultiplier 锁定调试面板的速度口径。
//
// CurSpeed 是世界系速度（已乘全局速度倍率），比较基准必须同样乘倍率，否则在
// “快”（2.0）倍速下，每一架巡航中的飞机、每一艘船都会被显示成超过速度上限。
// 同时锁定飞行阶段字段：起降阶段（滑跑 / 待场 / 进近）的速度是世界系位移，
// 允许高于设计上限，调试行要能一眼看出超速读数来自哪个阶段。
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

// TestPlaneDetailLinesReportReleaserState 锁定释放器调试行。
//
// 飞机“为什么不投弹”只有几种可能：已投放、投放间隔闸门、目标门槛、射程、
// 射界、鱼雷航迹压陆地。调试行必须把每枚炸弹 / 鱼雷的 Released 状态和逐项
// 判定都打出来，否则只能靠猜。
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

// TestPlaneDetailLinesMergeIdenticalReleasers 锁定释放器合并规则：
// 整舱挂载常是同型号重复，逐枚铺开会把面板撑到半个屏幕高；但状态不同的释放器
// 必须各自成组，否则会看漏“第 3 枚已经投掉了”这类关键差异。
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

// TestPlaneDetailLinesShowReadableBase 锁定所属基地的展示口径。
// BelongShip 是随机 uuid，单看它认不出是航母、载机舰船还是陆地机场；
// 基地被击沉等导致解析不到时必须显式标成 missing，而不是静默退回裸 uuid。
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

// TestPlaneDetailLinesGateTorpedoOnGroundTargets 锁定目标门槛的调试输出：
// 鱼雷无法在陆地使用，挂在唯一释放器上时必须在面板上直接说明原因。
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
