package initialize

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mohae/deepcopy"
	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	"github.com/narasux/jutland/pkg/mission/object/combatpower"
	ObjRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 规定初始化顺序，避免出现多个 init() 顺序问题
func init() {
	initBulletMap()
	initGunMap()
	initTorpedoLauncherMap()
	initRocketLauncherMap()
	initPlaneRocketLauncherMap()
	initReleaserMap()
	initCarrierDeckMap()
	initPlaneMap()
	initShipMap()
	initReferenceMap()
	initCombatPower()
	validateMissionShipPlacement()
}

// validateMissionShipPlacement 启动期校验各任务初始舰位：中心点落在陆格上直接报错，
// 舰体压岸只告警（系泊位是有意为之，例如珍珠港 1941 的码头泊位）。
// 与 validateAirfieldMetadata 一样，把配置错误挡在启动阶段，而不是等玩家点进去才发现。
func validateMissionShipPlacement() {
	for _, mission := range metadata.AllMissions() {
		md := metadata.Get(mission)
		if md.MapCfg == nil {
			continue
		}
		for _, initShip := range md.InitShips {
			ship, ok := objUnit.ShipMap[initShip.ShipName]
			if !ok {
				log.Fatalf("mission %q init ship %q not found", mission, initShip.ShipName)
			}
			if md.MapCfg.Map.IsLand(initShip.Pos.MX, initShip.Pos.MY) {
				log.Fatalf(
					"mission %q init ship %q pos (%d,%d) is on land in map %s",
					mission, initShip.ShipName, initShip.Pos.MX, initShip.Pos.MY, md.MapCfg.Name,
				)
			}
			if objUnit.HullLandOverlapSamples(md.MapCfg, initShip.Pos, initShip.Rotation, ship.Length) > 0 {
				log.Printf(
					"[WARN] mission %q init ship %q pos (%d,%d) rot %.0f: hull overlaps land (moored?)",
					mission, initShip.ShipName, initShip.Pos.MX, initShip.Pos.MY, initShip.Rotation,
				)
			}
		}
	}
}

func initBulletMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "bullets.json5"))
	if err != nil {
		log.Fatal("failed to open bullets.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var bullets []objBullet.Bullet
	if err = json5.Unmarshal(bytes, &bullets); err != nil {
		log.Fatal("failed to unmarshal bullets.json5: ", err)
	}

	for _, b := range bullets {
		objBullet.Map[b.Name] = &b
	}
	log.Println("bullets data loaded from json5 file")
}

func initGunMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "guns.json5"))
	if err != nil {
		log.Fatal("failed to open guns.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var guns []objUnit.Gun
	if err = json5.Unmarshal(bytes, &guns); err != nil {
		log.Fatal("failed to unmarshal guns.json5: ", err)
	}

	for _, g := range guns {
		g.Range /= 2
		g.BulletSpeed /= 4000
		objUnit.GunMap[g.Name] = &g
	}
	log.Println("guns data loaded from json5 file")
}

func initTorpedoLauncherMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "torpedo_launchers.json5"))
	if err != nil {
		log.Fatal("failed to open torpedo_launchers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var torpedoLaunchers []objUnit.TorpedoLauncher
	if err = json5.Unmarshal(bytes, &torpedoLaunchers); err != nil {
		log.Fatal("failed to unmarshal torpedo_launchers.json5: ", err)
	}

	for _, lc := range torpedoLaunchers {
		lc.Range /= 2
		lc.BulletSpeed /= 600
		objUnit.TorpedoLauncherMap[lc.Name] = &lc
	}
	log.Println("torpedo launchers data loaded from json5 file")
}

func initRocketLauncherMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "rocket_launchers.json5"))
	if err != nil {
		log.Fatal("failed to open rocket_launchers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var rocketLaunchers []objUnit.RocketLauncher
	if err = json5.Unmarshal(bytes, &rocketLaunchers); err != nil {
		log.Fatal("failed to unmarshal rocket_launchers.json5: ", err)
	}

	for _, lc := range rocketLaunchers {
		lc.Range /= 2
		lc.BulletSpeed /= 4000
		objUnit.RocketLauncherMap[lc.Name] = &lc
	}
	log.Println("rocket launchers data loaded from json5 file")
}

func initPlaneRocketLauncherMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "plane_rocket_launchers.json5"))
	if err != nil {
		log.Fatal("failed to open plane_rocket_launchers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var rocketLaunchers []objUnit.PlaneRocketLauncher
	if err = json5.Unmarshal(bytes, &rocketLaunchers); err != nil {
		log.Fatal("failed to unmarshal plane_rocket_launchers.json5: ", err)
	}

	for _, lc := range rocketLaunchers {
		lc.Range /= 2
		lc.BulletSpeed /= 4000
		objUnit.PlaneRocketLauncherMap[lc.Name] = &lc
	}
	log.Println("plane rocket launchers data loaded from json5 file")
}

func initReleaserMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "releasers.json5"))
	if err != nil {
		log.Fatal("failed to open releasers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var releasers []objUnit.Releaser
	if err = json5.Unmarshal(bytes, &releasers); err != nil {
		log.Fatal("failed to unmarshal releasers.json5: ", err)
	}

	for _, r := range releasers {
		r.BulletSpeed /= 600
		objUnit.ReleaserMap[r.Name] = &r
	}
	log.Println("releasers data loaded from json5 file")
}

func initCarrierDeckMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "carrier_decks.json5"))
	if err != nil {
		log.Fatal("failed to open carrier_decks.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var decks []objUnit.CarrierDeck
	if err = json5.Unmarshal(bytes, &decks); err != nil {
		log.Fatal("failed to unmarshal carrier_decks.json5: ", err)
	}

	for _, deck := range decks {
		if deck.Name == "" {
			log.Fatal("carrier deck name is required in carrier_decks.json5")
		}
		if _, exists := objUnit.DeckMap[deck.Name]; exists {
			log.Fatal("duplicated carrier deck name in carrier_decks.json5: ", deck.Name)
		}
		if len(deck.TakeoffPoints) == 0 {
			log.Fatal("carrier deck must define at least one takeoff point: ", deck.Name)
		}
		deck.Validate()
		objUnit.DeckMap[deck.Name] = &deck
	}
	log.Println("carrier decks data loaded from json5 file")
}

func initPlaneMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "planes.json5"))
	if err != nil {
		log.Fatal("failed to open planes.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var planes []objUnit.Plane
	if err = json5.Unmarshal(bytes, &planes); err != nil {
		log.Fatal("failed to unmarshal planes.json5: ", err)
	}

	for _, p := range planes {
		// 机关炮
		for _, gunMD := range p.Weapon.GunsMD {
			p.Weapon.Guns = append(p.Weapon.Guns, objUnit.NewGun(
				gunMD.Name, gunMD.PosPercent,
				objUnit.FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		// 炸弹
		for _, bombMD := range p.Weapon.BombsMD {
			p.Weapon.Bombs = append(p.Weapon.Bombs, objUnit.NewReleaser(
				bombMD.Name, bombMD.PosPercent,
				objUnit.FiringArc{Start: bombMD.LeftFiringArc[0], End: bombMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: bombMD.RightFiringArc[0], End: bombMD.RightFiringArc[1]},
			))
		}
		// 鱼雷
		for _, torpedoMD := range p.Weapon.TorpedoesMD {
			p.Weapon.Torpedoes = append(p.Weapon.Torpedoes, objUnit.NewReleaser(
				torpedoMD.Name, torpedoMD.PosPercent,
				objUnit.FiringArc{Start: torpedoMD.LeftFiringArc[0], End: torpedoMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: torpedoMD.RightFiringArc[0], End: torpedoMD.RightFiringArc[1]},
			))
		}
		// 火箭弹
		for _, rocketMD := range p.Weapon.RocketsMD {
			p.Weapon.Rockets = append(p.Weapon.Rockets, objUnit.NewPlaneRocketLauncher(
				rocketMD.Name, rocketMD.PosPercent,
				objUnit.FiringArc{Start: rocketMD.LeftFiringArc[0], End: rocketMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: rocketMD.RightFiringArc[0], End: rocketMD.RightFiringArc[1]},
			))
		}
		// 计算最大射程
		for _, gun := range p.Weapon.Guns {
			if gun.AntiShip {
				p.Weapon.MaxToShipRange = max(p.Weapon.MaxToShipRange, gun.Range)
			}
			if gun.AntiAircraft {
				p.Weapon.MaxToPlaneRange = max(p.Weapon.MaxToPlaneRange, gun.Range)
			}
		}
		for _, bomb := range p.Weapon.Bombs {
			p.Weapon.MaxToShipRange = max(p.Weapon.MaxToShipRange, bomb.Range)
		}
		for _, torpedo := range p.Weapon.Torpedoes {
			p.Weapon.MaxToShipRange = max(p.Weapon.MaxToShipRange, torpedo.Range)
		}
		for _, rocket := range p.Weapon.Rockets {
			if p.Type == objUnit.PlaneTypeFighter {
				p.Weapon.MaxToPlaneRange = max(p.Weapon.MaxToPlaneRange, rocket.Range)
			} else if p.Type.AttacksShips() {
				p.Weapon.MaxToShipRange = max(p.Weapon.MaxToShipRange, rocket.Range)
			}
		}

		// 当前生命值
		p.CurHP = p.TotalHP
		// 折算移动速度（公里换成节）
		p.MaxSpeed /= objUnit.PlaneSpeedScale
		// 折算总航程
		p.Range /= 8
		// 剩余航程
		p.RemainRange = p.Range
		p.SightRange = objUnit.ResolvePlaneSight(p.Type, p.SightRange)
		// 折算加速度
		p.Acceleration /= 600 * 2
		// 折算转向速度
		p.RotateSpeed /= 2
		// 检查伤害减免值不能超过 1
		p.DamageReduction = min(1, p.DamageReduction)
		objUnit.PlaneMap[p.Name] = &p
		objUnit.AllPlaneNames = append(objUnit.AllPlaneNames, p.Name)
	}
	log.Println("planes data loaded from json5 file")
}

func initShipMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "ships.json5"))
	if err != nil {
		log.Fatal("failed to open ships.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var ships []objUnit.BattleShip
	if err = json5.Unmarshal(bytes, &ships); err != nil {
		log.Fatal("failed to unmarshal ships.json5: ", err)
	}

	for _, s := range ships {
		// 主炮
		for _, gunMD := range s.Weapon.MainGunsMD {
			s.Weapon.MainGuns = append(s.Weapon.MainGuns, objUnit.NewGun(
				gunMD.Name, gunMD.PosPercent,
				objUnit.FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasMainGun = len(s.Weapon.MainGuns) > 0
		// 副炮
		for _, gunMD := range s.Weapon.SecondaryGunsMD {
			s.Weapon.SecondaryGuns = append(s.Weapon.SecondaryGuns, objUnit.NewGun(
				gunMD.Name, gunMD.PosPercent,
				objUnit.FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasSecondaryGun = len(s.Weapon.SecondaryGuns) > 0
		// 防空炮
		for _, gunMD := range s.Weapon.AntiAircraftGunsMD {
			s.Weapon.AntiAircraftGuns = append(s.Weapon.AntiAircraftGuns, objUnit.NewGun(
				gunMD.Name, gunMD.PosPercent,
				objUnit.FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasAntiAircraftGun = len(s.Weapon.AntiAircraftGuns) > 0
		// 鱼雷发射器：按舷别分组共用扇形，每组的扇面中心都是瞄准线，
		// 侧舷齐射不会有一半鱼雷打到反侧。
		s.Weapon.Torpedoes = newTorpedoLaunchers(s.Weapon.TorpedoesMD)
		s.Weapon.HasTorpedo = len(s.Weapon.Torpedoes) > 0
		// 火箭炮
		for _, rocketMD := range s.Weapon.RocketsMD {
			s.Weapon.Rockets = append(s.Weapon.Rockets, objUnit.NewRocketLauncher(
				rocketMD.Name, rocketMD.PosPercent,
				objUnit.FiringArc{Start: rocketMD.LeftFiringArc[0], End: rocketMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: rocketMD.RightFiringArc[0], End: rocketMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasRocket = len(s.Weapon.Rockets) > 0
		// 计算最大射程
		for _, guns := range [][]*objUnit.Gun{
			s.Weapon.MainGuns, s.Weapon.SecondaryGuns, s.Weapon.AntiAircraftGuns,
		} {
			for _, gun := range guns {
				if gun.AntiShip {
					s.Weapon.MaxToShipRange = max(s.Weapon.MaxToShipRange, gun.Range)
				}
				if gun.AntiAircraft {
					s.Weapon.MaxToPlaneRange = max(s.Weapon.MaxToPlaneRange, gun.Range)
				}
			}
		}
		for _, torpedo := range s.Weapon.Torpedoes {
			s.Weapon.MaxToShipRange = max(s.Weapon.MaxToShipRange, torpedo.Range)
		}
		for _, rocket := range s.Weapon.Rockets {
			if rocket.AntiShip {
				s.Weapon.MaxToShipRange = max(s.Weapon.MaxToShipRange, rocket.Range)
			}
			if rocket.AntiAircraft {
				s.Weapon.MaxToPlaneRange = max(s.Weapon.MaxToPlaneRange, rocket.Range)
			}
		}
		mainGunRange := 0.0
		for _, gun := range s.Weapon.MainGuns {
			if gun != nil && gun.AntiShip {
				mainGunRange = max(mainGunRange, gun.Range)
			}
		}
		// 视距在武器折算之后定，主炮射程已经是地图格。
		s.SightRange = objUnit.ResolveShipSight(s.Type, s.SightRange, mainGunRange)
		// 飞机相关状态
		s.Aircraft.HasPlane = len(s.Aircraft.Groups) > 0
		for i := 0; i < len(s.Aircraft.Groups); i++ {
			// 初始化飞机数量
			s.Aircraft.Groups[i].CurCount = s.Aircraft.Groups[i].MaxCount
			// 根据飞机名称，设置飞机目标类型
			s.Aircraft.Groups[i].TargetType = objUnit.GetPlaneTargetObjType(s.Aircraft.Groups[i].Name)
		}
		// 解析起降甲板模板；搭载舰载机的舰船必须显式配置 deck，缺失直接报错
		if s.Aircraft.HasPlane {
			deck, ok := objUnit.DeckMap[s.Aircraft.Deck]
			if !ok {
				log.Fatalf(
					"ship %q has aircraft but deck %q not found in carrier_decks.json5",
					s.Name, s.Aircraft.Deck,
				)
			}
			// 模板只读共享，舰船实例各持有一份拷贝，起飞点冷却等运行期状态互相隔离
			resolved := deepcopy.Copy(*deck).(objUnit.CarrierDeck)
			s.Aircraft.ResolveDeck(&resolved)
		}
		// 初始化当前生命值
		s.CurHP = s.TotalHP
		// 计算吨位（即最大生命值）
		s.Tonnage = s.TotalHP
		// 折算移动速度
		s.MaxSpeed /= objUnit.ShipSpeedScale
		// 折算加速度
		s.Acceleration /= 600 * 2
		// 折算转向速度
		s.RotateSpeed /= 2
		// 检查伤害减免值不能超过 1
		s.HorizontalDamageReduction = min(1, s.HorizontalDamageReduction)
		s.VerticalDamageReduction = min(1, s.VerticalDamageReduction)

		objUnit.ShipMap[s.Name] = &s
		objUnit.AllShipNames = append(objUnit.AllShipNames, s.Name)
	}
	log.Println("ships data loaded from json5 file")
}

// 鱼雷扇形的分组：能覆盖的舷别相同的发射器算一组。
// 中轴（两舷都能打）一组，只打右舷的一组，只打左舷的一组。
// 分组只能粗不能细：同一舷的发射器射界略有宽窄（如 shimakaze 的三座中轴五联装）
// 也必须共用一张扇面，否则每座各排一张扇面，齐射会退化成几根鱼雷重叠在同一角度。
const (
	// torpedoFanSideBoth 两舷都能打的发射器（中轴 / 全向）
	torpedoFanSideBoth = iota
	// torpedoFanSideStarboard 只能打右舷的发射器
	torpedoFanSideStarboard
	// torpedoFanSidePort 只能打左舷的发射器
	torpedoFanSidePort
)

// torpedoFanSideOf 按左右射界判断发射器属于哪一舷。
// FiringArc 用 Start == End 表示该舷没有射界。
func torpedoFanSideOf(md objUnit.WeaponMetadata) int {
	right := md.RightFiringArc[0] != md.RightFiringArc[1]
	left := md.LeftFiringArc[0] != md.LeftFiringArc[1]
	switch {
	case left && right:
		return torpedoFanSideBoth
	case right:
		return torpedoFanSideStarboard
	case left:
		return torpedoFanSidePort
	default:
		// 两舷射界都写没了（配置错误）：按中轴处理。
		return torpedoFanSideBoth
	}
}

// newTorpedoLaunchers 按舷别把鱼雷发射器分组，再为每座分配组内航迹。
// 同一舷的发射器共用一张以瞄准线为中心的扇面：航迹从中路往外排，第 m 座的第 t 根
// 走航迹 t*组内座数 + m，于是每拍齐射（每座出一根）拿到的都是相邻航迹，
// 并且总有一根落在瞄准线上。
func newTorpedoLaunchers(mds []objUnit.WeaponMetadata) []*objUnit.TorpedoLauncher {
	type fanGroup struct {
		// launchers 组内发射器在 mds 中的下标，按配置顺序
		launchers []int
		// tracks 组内鱼雷管总数
		tracks int
	}

	groups := map[int]*fanGroup{}
	var order []*fanGroup
	for idx, md := range mds {
		launcher, ok := objUnit.TorpedoLauncherMap[md.Name]
		if !ok {
			log.Fatalf("torpedo launcher %s no found", md.Name)
		}
		side := torpedoFanSideOf(md)
		group, ok := groups[side]
		if !ok {
			group = &fanGroup{}
			groups[side] = group
			order = append(order, group)
		}
		group.launchers = append(group.launchers, idx)
		group.tracks += launcher.BulletCount
	}

	out := make([]*objUnit.TorpedoLauncher, len(mds))
	for _, group := range order {
		for track, idx := range group.launchers {
			md := mds[idx]
			out[idx] = objUnit.NewTorpedoLauncher(
				md.Name, md.PosPercent,
				objUnit.FiringArc{Start: md.LeftFiringArc[0], End: md.LeftFiringArc[1]},
				objUnit.FiringArc{Start: md.RightFiringArc[0], End: md.RightFiringArc[1]},
				track, len(group.launchers), group.tracks,
			)
		}
	}
	return out
}

func initCombatPower() {
	for _, plane := range objUnit.PlaneMap {
		plane.CombatPower = combatpower.CalculatePlane(plane, objBullet.Map)
	}
	for _, ship := range objUnit.ShipMap {
		ship.CombatPower = combatpower.CalculateShip(ship, objUnit.PlaneMap, objBullet.Map)
	}
	log.Println("combat power calculated for planes and ships")
}

func initReferenceMap() {
	locales := []struct {
		Language i18n.Language
		Filename string
	}{
		{i18n.LanguageZhHans, "references.json5"},
		{i18n.LanguageEnglish, "references.en.json5"},
		{i18n.LanguageRussian, "references.ru.json5"},
		{i18n.LanguageJapanese, "references.ja.json5"},
	}
	loaded := make(map[i18n.Language][]ObjRef.Reference, len(locales))
	for _, locale := range locales {
		references, err := ObjRef.Load(filepath.Join(config.ConfigBaseDir, locale.Filename))
		if err != nil {
			log.Fatalf("failed to load %s: %s", locale.Filename, err)
		}
		loaded[locale.Language] = references
	}
	if err := ObjRef.ValidateLocales(loaded); err != nil {
		log.Fatal("invalid localized references: ", err)
	}
	for lang, references := range loaded {
		for idx := range references {
			ref := &references[idx]
			ObjRef.SetReference(lang, ref.Name, ref)
		}
	}
	log.Println("localized references data loaded from json5 files")
}
