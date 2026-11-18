package initialize

import (
	"fmt"
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestOoiTorpedoFanSurvivesNewShipCopy 验证 ooi 经 NewShip 深拷贝后仍保留鱼雷扇面分组与航迹。
// 场景：默认 config.G 下复制 ooi 实例（左右舷各 5 座四联装、共 40 管），向正东 4 格的静止驱逐舰开火；旧实现复制后扇面分组丢失，多发射器会把鱼雷全打向同一点。
// 预期：每座 FanTracks=20、FanStride=5 且组内 FanTrack 唯一，总管数 40；首轮齐射产生 5 个不同角度，并恰有一根落在瞄准线上。
// 覆盖：遍历 Weapon.Torpedoes 校验分组字段与槽位去重，再用 shooter.Fire(target) 收集鱼雷 Rotation 和相对瞄准线偏移。
func TestOoiTorpedoFanSurvivesNewShipCopy(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	generator := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	shooter := objUnit.NewShip(
		generator, "ooi", objPos.NewR(10, 10), 0, faction.HumanAlpha,
	)
	target := &objUnit.BattleShip{
		Uid:          "target",
		Type:         objUnit.ShipTypeDestroyer,
		CurHP:        100,
		TotalHP:      100,
		CurPos:       objPos.NewR(14, 10),
		BelongPlayer: faction.ComputerAlpha,
	}

	totalSlots := 0
	seenTrack := map[string]bool{}
	for _, launcher := range shooter.Weapon.Torpedoes {
		// ooi 左舷 5 座 / 右舷 5 座，每舷 20 根管：同一射界的发射器共用一张扇形，
		// 组内交错分航迹，所以 FanTrack 在组内唯一、组内座数就是 FanStride。
		if launcher.FanTracks != 20 || launcher.FanStride != 5 {
			t.Fatalf("ooi fan group = %d tracks / %d stride, want 20 / 5",
				launcher.FanTracks, launcher.FanStride)
		}
		key := fmt.Sprintf("%v|%v|%d", launcher.LeftFiringArc, launcher.RightFiringArc, launcher.FanTrack)
		if seenTrack[key] {
			t.Fatalf("duplicate ooi fan track %s", key)
		}
		seenTrack[key] = true
		totalSlots += launcher.BulletCount
	}
	if totalSlots != 40 {
		t.Fatalf("ooi torpedo tubes = %d, want 40", totalSlots)
	}

	angles := map[float64]bool{}
	hasAimLine := false
	for _, bullet := range shooter.Fire(target) {
		if bullet.Type != objBullet.TypeTorpedo {
			continue
		}
		angles[bullet.Rotation] = true
		// 扇面是围绕出管位置旋转的，所以相对瞄准线的偏移也要从出管位置量。
		if math.Abs(torpedoOffsetFromAim(bullet, target)) < 1e-6 {
			hasAimLine = true
		}
	}
	if len(angles) != 5 {
		t.Fatalf("ooi starboard fan produced %d unique angles, want 5", len(angles))
	}
	if !hasAimLine {
		t.Fatal("ooi 首发齐射没有落在瞄准线上的那一根")
	}
}

// TestFirstTorpedoVolleyCoversTheAimLine 验证所有带鱼雷模板的首发齐射都覆盖瞄准线且不越过最内侧航迹。
// 场景：遍历 AllShipNames 中有鱼雷的舰船，在艏/右舷/艉/左舷 0°、90°、180°、270° 四个方位向 0.4×射程处的 104m×11m 驱逐舰开火；旧实现按 (槽位-(管数-1)/2) 排扇形，porter 2×4 首发为 −12°/+1.71°、maury 右舷为 −12.03°/+0.8°，正中那根永远空缺。
// 预期：每轮齐射至少一根偏移 <1e-6° 压在瞄准线上，且所有偏移不超过 4.0°×ceil(齐射根数/2)；艏艉鱼雷管方向相反，也必须各自覆盖。
// 覆盖：对每个模板、每个方位调用 NewShip + Fire(target)，按射界筛选后逐根收集鱼雷偏移并断言。
func TestFirstTorpedoVolleyCoversTheAimLine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	checked := 0
	for _, name := range objUnit.AllShipNames {
		ship := objUnit.ShipMap[name]
		if ship == nil || !ship.Weapon.HasTorpedo {
			continue
		}
		// 艏、左右舷、艉各试一次：只有射界覆盖该方位的发射器会开火。
		for _, bearing := range []float64{0, 90, 180, 270} {
			generator := objUnit.NewShipUidGenerator(faction.HumanAlpha)
			shooter := objUnit.NewShip(generator, name, objPos.NewR(0, 0), 0, faction.HumanAlpha)
			distance := 0.4 * shooter.Weapon.Torpedoes[0].Range
			rad := bearing * math.Pi / 180
			target := &objUnit.BattleShip{
				Uid: "target", Type: objUnit.ShipTypeDestroyer,
				CurHP: 100, TotalHP: 100, Length: 104, Width: 11,
				CurPos:       objPos.NewR(math.Sin(rad)*distance, -math.Cos(rad)*distance),
				BelongPlayer: faction.ComputerAlpha,
			}

			var offsets []float64
			for _, bullet := range shooter.Fire(target) {
				if bullet.Type != objBullet.TypeTorpedo {
					continue
				}
				offsets = append(offsets, torpedoOffsetFromAim(bullet, target))
			}
			if len(offsets) == 0 {
				continue
			}
			checked++

			hasCenter := false
			for _, offset := range offsets {
				if math.Abs(offset) < 1e-6 {
					hasCenter = true
				}
			}
			if !hasCenter {
				t.Fatalf("%s 朝 %.0f° 的首发齐射 %v 没有瞄准线上的那一根", name, bearing, offsets)
			}
			// 首发是每座的最内侧航迹：不会超出“基础角 x 齐射根数的一半”。
			limit := 4.0 * math.Ceil(float64(len(offsets))/2)
			for _, offset := range offsets {
				if math.Abs(offset) > limit+1e-6 {
					t.Fatalf("%s 朝 %.0f° 的首发偏移 %.2f 超出最内侧航迹 %.2f", name, bearing, offset, limit)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("没有检查到任何鱼雷齐射")
	}
}

// TestMissionInitShipsStayOnWater 验证所有任务初始舰中心点在水面，舰体压岸只允许出现在珍珠港系泊位。
// 场景：遍历 metadata.AllMissions() 的 InitShips，用 MapCfg.Map.IsLand 检查中心格，并用 HullLandOverlapSamples 采样舰体；与启动期 validateMissionShipPlacement 的硬校验保持一致。
// 预期：每个任务都有地图配置、每艘初始舰都有模板，中心点不得为 ChrCoast/ChrLand；舰体压岸集合只允许 PearlHarbor1941，其余任务出现即失败并打印舰名。
// 覆盖：逐任务逐舰断言模板与中心格，累计 moored 映射后按任务名校验，最后 t.Logf 输出系泊位。
func TestMissionInitShipsStayOnWater(t *testing.T) {
	moored := map[string][]string{}
	for _, mission := range metadata.AllMissions() {
		md := metadata.Get(mission)
		if md.MapCfg == nil {
			t.Fatalf("任务 %q 缺少地图配置", mission)
		}
		for _, initShip := range md.InitShips {
			ship, ok := objUnit.ShipMap[initShip.ShipName]
			if !ok {
				t.Fatalf("任务 %q 初始舰 %q 没有对应模板", mission, initShip.ShipName)
			}
			if md.MapCfg.Map.IsLand(initShip.Pos.MX, initShip.Pos.MY) {
				t.Fatalf(
					"任务 %q 初始舰 %q 中心点落在陆格 (%d,%d)",
					mission, initShip.ShipName, initShip.Pos.MX, initShip.Pos.MY,
				)
			}
			if objUnit.HullLandOverlapSamples(
				md.MapCfg, initShip.Pos, initShip.Rotation, ship.Length,
			) > 0 {
				moored[mission] = append(moored[mission], initShip.ShipName)
			}
		}
	}
	// 珍珠港 1941 的码头泊位是有意贴着岸的，其余任务不应出现舰体压岸
	for mission, ships := range moored {
		if mission != "PearlHarbor1941" {
			t.Fatalf("任务 %q 出现非预期舰体压岸：%v", mission, ships)
		}
	}
	t.Logf("舰体压岸（系泊位）：%v", moored)
}

// torpedoOffsetFromAim 鱼雷相对瞄准线的角度偏移，归一到 (-180, 180]。
// 目标静止时提前量就是目标当前位置，而扇面是围绕出管位置旋转的，所以必须从
// 弹药的出管位置量，不能从舰体中心量（两者对目标的角度差可达几度）。
func torpedoOffsetFromAim(bullet *objBullet.Bullet, target *objUnit.BattleShip) float64 {
	aim := bullet.CurPos.Angle(target.CurPos)
	return math.Mod(bullet.CurPos.Angle(bullet.TargetPos)-aim+540, 360) - 180
}

// TestGuidedTorpedoConfig 验证两款制导鱼雷在配置中显式开启 pursuit 制导，普通鱼雷保持无制导。
// 场景：本期落地的 JP/TB/Kaiten/1944/Type1 与 US/TB/324/1957/Mk44 分别要求 120°/s、150°/s 转向；对照 US/TB/533/1935/Mk15 与 JP/TB/610/1933/Type93Mod1 不得被误加制导。
// 预期：两枚制导鱼雷存在且 IsGuided()、Guidance.Mode=GuidanceModePursuit、TurnRate 精确匹配；两枚普通鱼雷若存在则 IsGuided() 为 false。
// 覆盖：查 objBullet.Map 逐字段断言引导参数，再对普通鱼雷做反例检查。
func TestGuidedTorpedoConfig(t *testing.T) {
	guided := map[string]float64{
		"JP/TB/Kaiten/1944/Type1": 120,
		"US/TB/324/1957/Mk44":     150,
	}
	for name, wantTurnRate := range guided {
		bt := objBullet.Map[name]
		if bt == nil {
			t.Fatalf("bullet %q not found", name)
		}
		if !bt.IsGuided() {
			t.Fatalf("bullet %q is not guided", name)
		}
		if bt.Guidance.Mode != objBullet.GuidanceModePursuit {
			t.Fatalf("bullet %q mode = %q, want pursuit", name, bt.Guidance.Mode)
		}
		if bt.Guidance.TurnRate != wantTurnRate {
			t.Fatalf("bullet %q turnRate = %v, want %v", name, bt.Guidance.TurnRate, wantTurnRate)
		}
	}

	// 普通鱼雷不能被误加制导。
	for _, name := range []string{"US/TB/533/1935/Mk15", "JP/TB/610/1933/Type93Mod1"} {
		if bt := objBullet.Map[name]; bt != nil && bt.IsGuided() {
			t.Fatalf("bullet %q should stay unguided", name)
		}
	}
}

// TestASWWeaponRangesAfterScaling 验证反潜武器射程在初始化 ÷2 后落在拍板格数。
// 场景：2026-11-10 平衡值：舰载深弹架 2 格（配置 4km）、RBU 4 格（8km）、Mk32 发射 Mk44 反潜鱼雷 8 格（16km），对应 gurkha/porter/hammann/jiujiang/long_beach。
// 预期：五艘船的 MaxToSubmergedRange 精确等于 2/2/2/4/8，且 HasAntiSubWeapon=true；失败通常意味着配置公里数或初始化 Range/=2 换算被改错。
// 覆盖：表驱动查 ShipMap 模板，用 1e-9 容差比较射程并检查反潜标记。
func TestASWWeaponRangesAfterScaling(t *testing.T) {
	cases := []struct {
		ship string
		want float64
	}{
		{"gurkha", 2},     // 英 Mk VII 深弹架：配置 4 km
		{"porter", 2},     // 美 Mk 6 深弹架：配置 4 km
		{"hammann", 2},    // 同上
		{"jiujiang", 4},   // RBU 反潜火箭：配置 8 km
		{"long_beach", 8}, // Mk32 反潜鱼雷：配置 16 km
	}
	for _, tc := range cases {
		ship, ok := objUnit.ShipMap[tc.ship]
		if !ok {
			t.Fatalf("ship template %s missing", tc.ship)
		}
		if math.Abs(ship.Weapon.MaxToSubmergedRange-tc.want) > 1e-9 {
			t.Fatalf("%s MaxToSubmergedRange = %v, want %v", tc.ship, ship.Weapon.MaxToSubmergedRange, tc.want)
		}
		if !ship.Weapon.HasAntiSubWeapon {
			t.Fatalf("%s should be flagged as anti-submarine capable", tc.ship)
		}
	}
}

// TestSubmarineSubmergedSpeedMatchesHistory 验证三艘潜艇的水下航速符合历史实测值。
// 场景：2026-11-10 采用 Wikipedia 数据：VIIC（u_96）7.6 kn、I-168 8.2 kn、Nautilus 8.0 kn；配置用“水面航速 × submergedSpeedRatio”表达。
// 预期：MaxSpeed×ShipSpeedScale×SubmergedSpeedRatio 还原出的节数与目标值误差 ≤0.1 kn，避免水下航速倍率被误改。
// 覆盖：表驱动查询 ShipMap，逐艘断言模板存在并按 0.1 容差比较。
func TestSubmarineSubmergedSpeedMatchesHistory(t *testing.T) {
	cases := []struct {
		ship string
		want float64
	}{
		{"u_96", 7.6},
		{"i_168", 8.2},
		{"uss_nautilus", 8.0},
	}
	for _, tc := range cases {
		ship, ok := objUnit.ShipMap[tc.ship]
		if !ok {
			t.Fatalf("ship template %s missing", tc.ship)
		}
		// MaxSpeed 初始化时已除以 ShipSpeedScale，这里还原成配置节数。
		kn := ship.MaxSpeed * objUnit.ShipSpeedScale * ship.SubmergedSpeedRatio
		if math.Abs(kn-tc.want) > 0.1 {
			t.Fatalf("%s submerged speed = %.2f kn, want %.1f kn", tc.ship, kn, tc.want)
		}
	}
}

// TestSubmarineTemplatesDefaultSubmerged 验证所有潜艇模板默认入场即水下。
// 场景：遍历 ShipMap 中 Type=ShipTypeSubmarine 的模板；水面开局必须在配置里显式写 surface。
// 预期：每艘潜艇 DiveState=DiveStateSubmerged，且至少加载到一艘潜艇；失败意味着某艘潜艇未配置 defaultDiveState 而 ResolveDiveConfig 没有补成水下，或配置误写了 surface。
// 覆盖：按类型过滤全量模板计数，逐艘比较 DiveState 并在末尾检查 count>0。
func TestSubmarineTemplatesDefaultSubmerged(t *testing.T) {
	count := 0
	for name, ship := range objUnit.ShipMap {
		if ship.Type != objUnit.ShipTypeSubmarine {
			continue
		}
		count++
		if ship.DiveState != objUnit.DiveStateSubmerged {
			t.Fatalf("submarine %s default dive state = %v, want submerged", name, ship.DiveState)
		}
	}
	if count == 0 {
		t.Fatal("no submarine templates loaded")
	}
}

// TestAntiAircraftGunsAreAACapable 验证 AntiAircraftGuns 槽位里只能放真正能对空的炮。
// 场景：antiAircraft=false 的炮放进防空槽后不累计 MaxToPlaneRange，防空射击的类型判定也会跳过它，等于占着挂点却打不到飞机。
// 预期：遍历所有舰船模板的防空炮位，每门 gun.AntiAircraft 都必须为 true；失败会报出舰名与炮名。
// 覆盖：全量 ShipMap × Weapon.AntiAircraftGuns 双重循环，用 t.Errorf 汇总所有违规配置。
func TestAntiAircraftGunsAreAACapable(t *testing.T) {
	for name, ship := range objUnit.ShipMap {
		for _, gun := range ship.Weapon.AntiAircraftGuns {
			if !gun.AntiAircraft {
				t.Errorf("%s 的防空炮位配了不能对空的 %s", name, gun.Name)
			}
		}
	}
}

// TestAllUnitsHaveHistoricalStatus 验证全部舰船与飞机都显式标注合法的历史属性。
// 场景：图鉴新增史实 / 图纸 / 架空徽标，配置漏写或把 fictional 拼错会让徽标显示未知或默认成史实。
// 预期：ShipMap 与 PlaneMap 均非空，且每一项 Historical 都是三个受支持取值之一；失败时输出对象名与原始值。
// 覆盖：遍历初始化后的全量 ShipMap / PlaneMap，调用 HistoricalStatus.IsValid 并统计条目数。
func TestAllUnitsHaveHistoricalStatus(t *testing.T) {
	if len(objUnit.ShipMap) == 0 || len(objUnit.PlaneMap) == 0 {
		t.Fatal("ship or plane map is empty")
	}
	for name, ship := range objUnit.ShipMap {
		if !ship.Historical.IsValid() {
			t.Errorf("ship %s historical status = %q, want real/paper/fictional", name, ship.Historical)
		}
	}
	for name, plane := range objUnit.PlaneMap {
		if !plane.Historical.IsValid() {
			t.Errorf("plane %s historical status = %q, want real/paper/fictional", name, plane.Historical)
		}
	}
}

// TestEveryPlaneHasYear 验证全部飞机都配置了服役年份，供图鉴基础数据展示。
// 场景：planes.json5 曾漏掉 Fulmar 与 Swordfish 的 year，飞机卡片的年份行只能显示 --。
// 预期：PlaneMap 中每一架飞机的 Year 都大于 0；失败会列出缺失年份的机名。
// 覆盖：遍历初始化后的全量 PlaneMap，逐架断言 Year > 0。
func TestEveryPlaneHasYear(t *testing.T) {
	for name, plane := range objUnit.PlaneMap {
		if plane.Year <= 0 {
			t.Errorf("plane %s year = %d, want > 0", name, plane.Year)
		}
	}
}
