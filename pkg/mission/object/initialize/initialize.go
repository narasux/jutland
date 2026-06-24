package initialize

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
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
	initPlaneMap()
	initShipMap()
	initReferenceMap()
	initCombatPower()
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
			} else if p.Type == objUnit.PlaneTypeDiveBomber || p.Type == objUnit.PlaneTypeTorpedoBomber {
				p.Weapon.MaxToShipRange = max(p.Weapon.MaxToShipRange, rocket.Range)
			}
		}

		// 当前生命值
		p.CurHP = p.TotalHP
		// 折算速度（公里换成节）
		p.MaxSpeed /= 3000 * 1.8
		// 折算总航程
		p.Range /= 8 * 1.8
		// 剩余航程
		p.RemainRange = p.Range
		p.Acceleration /= 600
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
		// 鱼雷发射器
		for _, torpedoMD := range s.Weapon.TorpedoesMD {
			s.Weapon.Torpedoes = append(s.Weapon.Torpedoes, objUnit.NewTorpedoLauncher(
				torpedoMD.Name, torpedoMD.PosPercent,
				objUnit.FiringArc{Start: torpedoMD.LeftFiringArc[0], End: torpedoMD.LeftFiringArc[1]},
				objUnit.FiringArc{Start: torpedoMD.RightFiringArc[0], End: torpedoMD.RightFiringArc[1]},
			))
		}
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
		// 飞机相关状态
		s.Aircraft.HasPlane = len(s.Aircraft.Groups) > 0
		for i := 0; i < len(s.Aircraft.Groups); i++ {
			// 初始化飞机数量
			s.Aircraft.Groups[i].CurCount = s.Aircraft.Groups[i].MaxCount
			// 根据飞机名称，设置飞机目标类型
			s.Aircraft.Groups[i].TargetType = objUnit.GetPlaneTargetObjType(s.Aircraft.Groups[i].Name)
		}
		// 初始化当前生命值
		s.CurHP = s.TotalHP
		// 计算吨位（即最大生命值）
		s.Tonnage = s.TotalHP
		// 折算速度
		s.MaxSpeed /= 600
		s.Acceleration /= 600
		// 检查伤害减免值不能超过 1
		s.HorizontalDamageReduction = min(1, s.HorizontalDamageReduction)
		s.VerticalDamageReduction = min(1, s.VerticalDamageReduction)

		objUnit.ShipMap[s.Name] = &s
		objUnit.AllShipNames = append(objUnit.AllShipNames, s.Name)
	}
	log.Println("ships data loaded from json5 file")
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
	}
	loaded := make(map[i18n.Language][]ObjRef.Reference, len(locales))
	for _, locale := range locales {
		references, err := ObjRef.Load(filepath.Join(config.ConfigBaseDir, locale.Filename))
		if err != nil {
			log.Fatalf("failed to load %s: %s", locale.Filename, err)
		}
		loaded[locale.Language] = references
	}
	if err := ObjRef.ValidateLocales(
		loaded[i18n.LanguageZhHans], loaded[i18n.LanguageEnglish],
	); err != nil {
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
