package object

import (
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
	initShipMap()
}

func initBulletMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "bullets.json5"))
	if err != nil {
		log.Fatalf("failed to open bullets.json5: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var bullets []Bullet
	if err = json5.Unmarshal(bytes, &bullets); err != nil {
		log.Fatalf("failed to unmarshal bullets.json5: %s", err)
	}

	for _, b := range bullets {
		bulletMap[b.Name] = &b
	}
}

func initGunMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "guns.json5"))
	if err != nil {
		log.Fatalf("failed to open guns.json5: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var guns []Gun
	if err = json5.Unmarshal(bytes, &guns); err != nil {
		log.Fatalf("failed to unmarshal guns.json5: %s", err)
	}

	for _, g := range guns {
		gunMap[g.Name] = &g
	}
}

func initTorpedoLauncherMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "torpedo_launchers.json5"))
	if err != nil {
		log.Fatalf("failed to open torpedo_launchers.json5: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var torpedoLaunchers []TorpedoLauncher
	if err = json5.Unmarshal(bytes, &torpedoLaunchers); err != nil {
		log.Fatalf("failed to unmarshal torpedo_launchers.json5: %s", err)
	}

	for _, lc := range torpedoLaunchers {
		torpedoLauncherMap[lc.Name] = &lc
	}
}

func initShipMap() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "ships.json5"))
	if err != nil {
		log.Fatalf("failed to open ships.json5: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var ships []BattleShip
	if err = json5.Unmarshal(bytes, &ships); err != nil {
		log.Fatalf("failed to unmarshal ships.json5: %s", err)
	}

	for _, s := range ships {
		for _, gunMD := range s.Weapon.GunsMD {
			s.Weapon.Guns = append(s.Weapon.Guns, newGun(
				gunMD.Name, gunMD.PosPercent, gunMD.LeftFiringArc, gunMD.RightFiringArc,
			))
		}
		for _, torpedoMD := range s.Weapon.TorpedoesMD {
			s.Weapon.Torpedoes = append(s.Weapon.Torpedoes, newTorpedoLauncher(
				torpedoMD.Name, torpedoMD.PosPercent, torpedoMD.LeftFiringArc, torpedoMD.RightFiringArc,
			))
		}
		for _, gun := range s.Weapon.Guns {
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
		shipMap[s.Name] = &s
	}
}
