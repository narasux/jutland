package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
)

// WeaponReloadStatus 汇总一种舰载武器在指定时刻的装填与启停状态。
// Progress 表示下一座武器可发射前的进度，取值范围为 [0, 1]。
type WeaponReloadStatus struct {
	Equipped        int
	Ready           int
	Progress        float64
	RemainingMillis int64
	Disabled        bool
}

type reloadGate struct {
	startAt   int64
	duration  int64
	tickClock bool
}

func (g reloadGate) readyAt() int64 {
	return g.startAt + g.duration
}

func (g reloadGate) remainingAt(now int64) int64 {
	return max(0, g.readyAt()-now)
}

func (g reloadGate) remainingMillis(now int64) int64 {
	left := g.remainingAt(now)
	if g.tickClock {
		return left * 1000 / constants.MaxTPS
	}
	return left
}

func (g reloadGate) progressAt(now int64) float64 {
	if g.duration <= 0 || now >= g.readyAt() {
		return 1
	}
	return min(1, max(0, float64(now-g.startAt)/float64(g.duration)))
}

func laterReloadGate(a, b reloadGate) reloadGate {
	if b.readyAt() > a.readyAt() {
		return b
	}
	return a
}

func gunReloadGate(g *Gun) reloadGate {
	if g.ReloadStartTick > 0 {
		return reloadGate{startAt: g.ReloadStartTick, duration: ReloadTicks(g.ReloadTime), tickClock: true}
	}
	if g.ReloadStartAt > 0 {
		return reloadGate{startAt: g.ReloadStartAt, duration: int64(g.ReloadTime * 1e3)}
	}
	return reloadGate{}
}

func torpedoReloadGate(launcher *TorpedoLauncher) reloadGate {
	if launcher.ReloadStartTick > 0 || launcher.LatestFireTick > 0 {
		reload := reloadGate{tickClock: true}
		if launcher.ReloadStartTick > 0 {
			reload = reloadGate{
				startAt:   launcher.ReloadStartTick,
				duration:  ReloadTicks(launcher.ReloadTime),
				tickClock: true,
			}
		}
		interval := reloadGate{
			startAt:   launcher.LatestFireTick,
			duration:  ReloadTicks(launcher.ShotInterval),
			tickClock: true,
		}
		return laterReloadGate(reload, interval)
	}
	reload := reloadGate{
		startAt:  launcher.ReloadStartAt,
		duration: int64(launcher.ReloadTime * 1e3),
	}
	interval := reloadGate{
		startAt:  launcher.LatestFireAt,
		duration: int64(launcher.ShotInterval * 1e3),
	}
	return laterReloadGate(reload, interval)
}

func rocketReloadGate(launcher *RocketLauncher) reloadGate {
	if launcher.ReloadStartTick > 0 || launcher.LatestFireTick > 0 {
		reload := reloadGate{tickClock: true}
		if launcher.ReloadStartTick > 0 {
			reload = reloadGate{
				startAt:   launcher.ReloadStartTick,
				duration:  ReloadTicks(launcher.ReloadTime),
				tickClock: true,
			}
		}
		if launcher.ShotCountBeforeReload <= 0 {
			return reload
		}
		intervalSeconds := launcher.ShotInterval
		if launcher.ShotCountBeforeReload%launcher.groupSize() == 0 {
			intervalSeconds = launcher.GroupInterval
		}
		interval := reloadGate{
			startAt:   launcher.LatestFireTick,
			duration:  ReloadTicks(intervalSeconds),
			tickClock: true,
		}
		return laterReloadGate(reload, interval)
	}
	reload := reloadGate{
		startAt:  launcher.ReloadStartAt,
		duration: int64(launcher.ReloadTime * 1e3),
	}
	if launcher.ShotCountBeforeReload <= 0 {
		return reload
	}

	intervalSeconds := launcher.ShotInterval
	if launcher.ShotCountBeforeReload%launcher.groupSize() == 0 {
		intervalSeconds = launcher.GroupInterval
	}
	interval := reloadGate{
		startAt:  launcher.LatestFireAt,
		duration: int64(intervalSeconds * 1e3),
	}
	return laterReloadGate(reload, interval)
}

func aggregateReloadStatus(disabled bool, gates []reloadGate, now int64) WeaponReloadStatus {
	status := WeaponReloadStatus{
		Equipped: len(gates),
		Progress: 1,
		Disabled: disabled,
	}
	if len(gates) == 0 {
		return status
	}

	nextRemaining := int64(math.MaxInt64)
	for _, gate := range gates {
		remaining := gate.remainingMillis(now)
		if remaining == 0 {
			status.Ready++
			continue
		}
		if remaining < nextRemaining {
			nextRemaining = remaining
			status.RemainingMillis = remaining
			status.Progress = gate.progressAt(now)
		}
	}
	return status
}

// ReloadStatus 返回指定武器类型在 nowMillis 时刻的装填汇总。
// WeaponTypeAll 仅用于总开关，不对应装填进度，因此返回零装备状态。
func (w *ShipWeapon) ReloadStatus(weaponType WeaponType, nowMillis int64) WeaponReloadStatus {
	switch weaponType {
	case WeaponTypeMainGun:
		gates := make([]reloadGate, 0, len(w.MainGuns))
		for _, gun := range w.MainGuns {
			gates = append(gates, gunReloadGate(gun))
		}
		return aggregateReloadStatus(w.MainGunDisabled, gates, nowMillis)
	case WeaponTypeSecondaryGun:
		gates := make([]reloadGate, 0, len(w.SecondaryGuns))
		for _, gun := range w.SecondaryGuns {
			gates = append(gates, gunReloadGate(gun))
		}
		return aggregateReloadStatus(w.SecondaryGunDisabled, gates, nowMillis)
	case WeaponTypeAntiAircraftGun:
		gates := make([]reloadGate, 0, len(w.AntiAircraftGuns))
		for _, gun := range w.AntiAircraftGuns {
			gates = append(gates, gunReloadGate(gun))
		}
		return aggregateReloadStatus(w.AntiAircraftGunDisabled, gates, nowMillis)
	case WeaponTypeTorpedo:
		gates := make([]reloadGate, 0, len(w.Torpedoes))
		for _, launcher := range w.Torpedoes {
			gates = append(gates, torpedoReloadGate(launcher))
		}
		return aggregateReloadStatus(w.TorpedoDisabled, gates, nowMillis)
	case WeaponTypeRocket:
		gates := make([]reloadGate, 0, len(w.Rockets))
		for _, launcher := range w.Rockets {
			gates = append(gates, rocketReloadGate(launcher))
		}
		return aggregateReloadStatus(w.RocketDisabled, gates, nowMillis)
	default:
		return WeaponReloadStatus{}
	}
}
