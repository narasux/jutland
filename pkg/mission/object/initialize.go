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
		for _, gunMD := range p.Weapon.MainGunsMD {
			p.Weapon.MainGuns = append(p.Weapon.MainGuns, newGun(
				gunMD.Name, gunMD.PosPercent,
				FiringArc{Start: gunMD.LeftFiringArc[0], End: gunMD.LeftFiringArc[1]},
				FiringArc{Start: gunMD.RightFiringArc[0], End: gunMD.RightFiringArc[1]},
			))
		}
		// 当前生命值
		p.CurHP = p.TotalHP
		// 折算速度
		p.MaxSpeed /= 600
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
		for _, gun := range s.Weapon.MainGuns {
			if s.Weapon.MaxRange < gun.Range {
				s.Weapon.MaxRange = gun.Range
			}
		}
		// 虽然副炮射程比主炮远不太可能，不过还是加上吧
		for _, gun := range s.Weapon.SecondaryGuns {
			if s.Weapon.MaxRange < gun.Range {
				s.Weapon.MaxRange = gun.Range
			}
		}
		for _, torpedo := range s.Weapon.Torpedoes {
			if s.Weapon.MaxRange < torpedo.Range {
				s.Weapon.MaxRange = torpedo.Range
			}
		}
		s.CurHP = s.TotalHP
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
