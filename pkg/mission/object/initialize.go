package object

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
)

// 规定初始化顺序，避免出现多个 init() 顺序问题
func init() {
	initBulletMap()
	initGunMap()
	initTorpedoLauncherMap()
	initReleaserMap()
	initPlaneMap()
	initShipMap()
	initReferenceMap()
}

func initBulletMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "bullets.json5"))
	if err != nil {
		log.Fatal("failed to open bullets.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var bullets []Bullet
	if err = json5.Unmarshal(bytes, &bullets); err != nil {
		log.Fatal("failed to unmarshal bullets.json5: ", err)
	}

	for _, b := range bullets {
		bulletMap[b.Name] = &b
	}
}

func initGunMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "guns.json5"))
	if err != nil {
		log.Fatal("failed to open guns.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var guns []Gun
	if err = json5.Unmarshal(bytes, &guns); err != nil {
		log.Fatal("failed to unmarshal guns.json5: ", err)
	}

	for _, g := range guns {
		g.Range /= 2
		g.BulletSpeed /= 4000
		gunMap[g.Name] = &g
	}
}

func initTorpedoLauncherMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "torpedo_launchers.json5"))
	if err != nil {
		log.Fatal("failed to open torpedo_launchers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var torpedoLaunchers []TorpedoLauncher
	if err = json5.Unmarshal(bytes, &torpedoLaunchers); err != nil {
		log.Fatal("failed to unmarshal torpedo_launchers.json5: ", err)
	}

	for _, lc := range torpedoLaunchers {
		lc.Range /= 2
		lc.BulletSpeed /= 600
		torpedoLauncherMap[lc.Name] = &lc
	}
}

func initReleaserMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "releasers.json5"))
	if err != nil {
		log.Fatal("failed to open releasers.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var releasers []Releaser
	if err = json5.Unmarshal(bytes, &releasers); err != nil {
		log.Fatal("failed to unmarshal releasers.json5: ", err)
	}

	for _, r := range releasers {
		r.BulletSpeed /= 600
		releaserMap[r.Name] = &r
	}
}

func initPlaneMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "planes.json5"))
	if err != nil {
		log.Fatal("failed to open planes.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var planes []Plane
	if err = json5.Unmarshal(bytes, &planes); err != nil {
		log.Fatal("failed to unmarshal planes.json5: ", err)
	}

	for _, p := range planes {
		// 机关炮
		for _, gunMD := range p.Weapon.GunsMD {
			p.Weapon.Guns = append(p.Weapon.Guns, newGun(
				gunMD.Name, gunMD.PosPercent,
				FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		// 炸弹
		for _, bombMD := range p.Weapon.BombsMD {
			p.Weapon.Bombs = append(p.Weapon.Bombs, newReleaser(
				bombMD.Name, bombMD.PosPercent,
				FiringArc{Start: bombMD.LeftFiringArc[0], End: bombMD.LeftFiringArc[1]},
				FiringArc{Start: bombMD.RightFiringArc[0], End: bombMD.RightFiringArc[1]},
			))
		}
		// 鱼雷
		for _, torpedoMD := range p.Weapon.TorpedoesMD {
			p.Weapon.Torpedoes = append(p.Weapon.Torpedoes, newReleaser(
				torpedoMD.Name, torpedoMD.PosPercent,
				FiringArc{Start: torpedoMD.LeftFiringArc[0], End: torpedoMD.LeftFiringArc[1]},
				FiringArc{Start: torpedoMD.RightFiringArc[0], End: torpedoMD.RightFiringArc[1]},
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

		// 当前生命值
		p.CurHP = p.TotalHP
		// 折算速度（公里换成节）
		p.MaxSpeed /= 4500 * 1.8
		// 折算总航程
		p.Range /= 8 * 1.8
		// 剩余航程
		p.RemainRange = p.Range
		p.Acceleration /= 600
		// 检查伤害减免值不能超过 1
		p.DamageReduction = min(1, p.DamageReduction)
		planeMap[p.Name] = &p
	}
}

func initShipMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "ships.json5"))
	if err != nil {
		log.Fatal("failed to open ships.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var ships []BattleShip
	if err = json5.Unmarshal(bytes, &ships); err != nil {
		log.Fatal("failed to unmarshal ships.json5: ", err)
	}

	for _, s := range ships {
		// 主炮
		for _, gunMD := range s.Weapon.MainGunsMD {
			s.Weapon.MainGuns = append(s.Weapon.MainGuns, newGun(
				gunMD.Name, gunMD.PosPercent,
				FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasMainGun = len(s.Weapon.MainGuns) > 0
		// 副炮
		for _, gunMD := range s.Weapon.SecondaryGunsMD {
			s.Weapon.SecondaryGuns = append(s.Weapon.SecondaryGuns, newGun(
				gunMD.Name, gunMD.PosPercent,
				FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasSecondaryGun = len(s.Weapon.SecondaryGuns) > 0
		// 防空炮
		for _, gunMD := range s.Weapon.AntiAircraftGunsMD {
			s.Weapon.AntiAircraftGuns = append(s.Weapon.AntiAircraftGuns, newGun(
				gunMD.Name, gunMD.PosPercent,
				FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasAntiAircraftGun = len(s.Weapon.AntiAircraftGuns) > 0
		// 鱼雷发射器
		for _, torpedoMD := range s.Weapon.TorpedoesMD {
			s.Weapon.Torpedoes = append(s.Weapon.Torpedoes, newTorpedoLauncher(
				torpedoMD.Name, torpedoMD.PosPercent,
				FiringArc{Start: torpedoMD.LeftFiringArc[0], End: torpedoMD.LeftFiringArc[1]},
				FiringArc{Start: torpedoMD.RightFiringArc[0], End: torpedoMD.RightFiringArc[1]},
			))
		}
		s.Weapon.HasTorpedo = len(s.Weapon.Torpedoes) > 0
		// 计算最大射程
		for _, guns := range [][]*Gun{
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
		// 飞机相关状态
		s.Aircraft.HasPlane = len(s.Aircraft.Groups) > 0
		// 根据飞机名称，设置飞机目标类型
		for i := 0; i < len(s.Aircraft.Groups); i++ {
			s.Aircraft.Groups[i].TargetType = getPlaneTargetObjType(s.Aircraft.Groups[i].Name)
		}
		// 初始化当前生命值
		s.CurHP = s.TotalHP
		// 计算吨位（即最大生命值）
		s.Tonnage = s.TotalHP
		// 添加默认描述
		s.Description = append(
			s.Description, fmt.Sprintf(
				"HP：%.0f，速度：%.0f 节，费用：$%d / %ds",
				s.TotalHP, s.MaxSpeed, s.FundsCost, s.TimeCost,
			),
		)
		// 折算速度
		s.MaxSpeed /= 600
		s.Acceleration /= 600
		// 检查伤害减免值不能超过 1
		s.HorizontalDamageReduction = min(1, s.HorizontalDamageReduction)
		s.VerticalDamageReduction = min(1, s.VerticalDamageReduction)

		shipMap[s.Name] = &s
		allShipNames = append(allShipNames, s.Name)
	}
}

func initReferenceMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "references.json5"))
	if err != nil {
		log.Fatal("failed to open references.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var references []Reference
	err = json5.Unmarshal(bytes, &references)
	if err != nil {
		log.Fatal("failed to unmarshal references.json5: ", err)
	}

	for _, ref := range references {
		referencesMap[ref.Name] = &ref
	}
}
