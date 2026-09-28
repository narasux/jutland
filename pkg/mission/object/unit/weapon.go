package unit

import "github.com/narasux/jutland/pkg/mission/object"

// FiringArc 火炮射界
type FiringArc struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Contains 是否在射界内
func (f *FiringArc) Contains(angle float64) bool {
	// 排除 [0, 0], [360, 360] 的情况
	if f.Start == f.End {
		return false
	}
	return f.Start <= angle && angle <= f.End
}

// WeaponType 武器类型
type WeaponType string

const (
	// WeaponTypeAll 所有
	WeaponTypeAll WeaponType = "all"
	// WeaponTypeMainGun 主炮
	WeaponTypeMainGun WeaponType = "mainGun"
	// WeaponTypeSecondaryGun 副炮
	WeaponTypeSecondaryGun WeaponType = "secondaryGun"
	// WeaponTypeAntiAircraftGun 防空炮
	WeaponTypeAntiAircraftGun WeaponType = "antiAircraftGun"
	// WeaponTypeTorpedo 鱼雷
	WeaponTypeTorpedo WeaponType = "torpedo"
	// WeaponTypeRocket 火箭炮
	WeaponTypeRocket WeaponType = "rocket"
	// WeaponTypeMissile 导弹
	WeaponTypeMissile WeaponType = "missile"
)

// WeaponMetadata 武器元数据
type WeaponMetadata struct {
	Name string `json:"name"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64 `json:"posPercent"`
	// 左射界
	LeftFiringArc [2]float64 `json:"leftFiringArc"`
	// 右射界
	RightFiringArc [2]float64 `json:"rightFiringArc"`
}

// ShipWeapon 战舰武器系统
type ShipWeapon struct {
	// 主炮元数据
	MainGunsMD []WeaponMetadata `json:"mainGuns"`
	// 副炮元数据
	SecondaryGunsMD []WeaponMetadata `json:"secondaryGuns"`
	// 防空炮元数据
	AntiAircraftGunsMD []WeaponMetadata `json:"antiAircraftGuns"`
	// 鱼雷元数据
	TorpedoesMD []WeaponMetadata `json:"torpedoes"`
	// 火箭炮元数据
	RocketsMD []WeaponMetadata `json:"rockets"`
	// 释放器元数据
	ReleasersMD []WeaponMetadata `json:"releasers"`
	// 主炮
	MainGuns []*Gun
	// 副炮
	SecondaryGuns []*Gun
	// 防空炮
	AntiAircraftGuns []*Gun
	// 鱼雷
	Torpedoes []*TorpedoLauncher
	// 火箭炮
	Rockets []*RocketLauncher
	// 最大射程（各类武器射程最大值）
	MaxToShipRange  float64
	MaxToPlaneRange float64
	// 拥有的武器情况
	HasMainGun         bool
	HasSecondaryGun    bool
	HasAntiAircraftGun bool
	HasTorpedo         bool
	HasRocket          bool
	// 武器禁用情况
	MainGunDisabled         bool
	SecondaryGunDisabled    bool
	AntiAircraftGunDisabled bool
	TorpedoDisabled         bool
	RocketDisabled          bool
}

// MainGunReloaded 主炮是否已装填
func (w *ShipWeapon) MainGunReloaded() bool {
	for _, g := range w.MainGuns {
		if g.Reloaded() {
			return true
		}
	}
	return false
}

// SecondaryGunReloaded 副炮是否已装填
func (w *ShipWeapon) SecondaryGunReloaded() bool {
	for _, g := range w.SecondaryGuns {
		if g.Reloaded() {
			return true
		}
	}
	return false
}

// TorpedoLauncherReloaded 鱼雷是否已装填
func (w *ShipWeapon) TorpedoLauncherReloaded() bool {
	for _, t := range w.Torpedoes {
		if t.Reloaded() {
			return true
		}
	}
	return false
}

// RocketLauncherReloaded 火箭炮是否已装填并可发射下一组
func (w *ShipWeapon) RocketLauncherReloaded() bool {
	for _, r := range w.Rockets {
		if r.Reloaded() {
			return true
		}
	}
	return false
}

// AnyReloaded 是否还有一门没被禁用、并且已经能开火的武器。
func (w *ShipWeapon) AnyReloaded() bool {
	return gunsReady(w.MainGunDisabled, w.MainGuns) ||
		gunsReady(w.SecondaryGunDisabled, w.SecondaryGuns) ||
		gunsReady(w.AntiAircraftGunDisabled, w.AntiAircraftGuns) ||
		torpedoesReady(w.TorpedoDisabled, w.Torpedoes) ||
		rocketsReady(w.RocketDisabled, w.Rockets)
}

func gunsReady(groupDisabled bool, guns []*Gun) bool {
	if groupDisabled {
		return false
	}
	for _, gun := range guns {
		if gun != nil && !gun.Disable && gun.Reloaded() {
			return true
		}
	}
	return false
}

func torpedoesReady(groupDisabled bool, launchers []*TorpedoLauncher) bool {
	if groupDisabled {
		return false
	}
	for _, launcher := range launchers {
		if launcher != nil && !launcher.Disable && launcher.Reloaded() {
			return true
		}
	}
	return false
}

func rocketsReady(groupDisabled bool, launchers []*RocketLauncher) bool {
	if groupDisabled {
		return false
	}
	for _, launcher := range launchers {
		if launcher != nil && !launcher.Disable && launcher.Reloaded() {
			return true
		}
	}
	return false
}

// PlaneWeapon 战机武器系统
type PlaneWeapon struct {
	// 机炮元数据
	GunsMD []WeaponMetadata `json:"guns"`
	// 炸弹元数据
	BombsMD []WeaponMetadata `json:"bombs"`
	// 鱼雷元数据
	TorpedoesMD []WeaponMetadata `json:"torpedoes"`
	// 火箭弹元数据
	RocketsMD []WeaponMetadata `json:"rockets"`
	// 最小释放间隔（秒）
	ReleaseInterval float64 `json:"releaseInterval"`
	// 最近释放时间
	LatestReleaseTick int64
	LatestReleaseAt   int64
	// 固定机炮
	Guns []*Gun
	// 炸弹
	Bombs []*Releaser
	// 鱼雷
	Torpedoes []*Releaser
	// 火箭弹
	Rockets []*PlaneRocketLauncher
	// 最大射程（各类武器射程最大值）
	MaxToShipRange  float64
	MaxToPlaneRange float64
}

// AnyReloaded 是否还有机炮、未投放的炸弹或鱼雷、或能发射的火箭。
func (w *PlaneWeapon) AnyReloaded() bool {
	for _, gun := range w.Guns {
		if gun != nil && !gun.Disable && gun.Reloaded() {
			return true
		}
	}
	if w.hasUnreleasedOrdnance() {
		return true
	}
	for _, rocket := range w.Rockets {
		if rocket != nil && rocket.Reloaded() {
			return true
		}
	}
	return false
}

// AntiAircraftReady 是否还有能对空开火的机炮或火箭。
func (w *PlaneWeapon) AntiAircraftReady() bool {
	for _, gun := range w.Guns {
		if gun != nil && !gun.Disable && gun.AntiAircraft && gun.Reloaded() {
			return true
		}
	}
	for _, rocket := range w.Rockets {
		if rocket != nil && rocket.AntiAircraft && rocket.Reloaded() {
			return true
		}
	}
	return false
}

func (w *PlaneWeapon) hasUnreleasedOrdnance() bool {
	for _, bomb := range w.Bombs {
		if bomb != nil && !bomb.Released {
			return true
		}
	}
	for _, torpedo := range w.Torpedoes {
		if torpedo != nil && !torpedo.Released {
			return true
		}
	}
	return false
}

// PlaneGroup 飞机分组
type PlaneGroup struct {
	// Name 战机名称
	Name string `json:"name"`
	// MaxCount 总数量
	MaxCount int64 `json:"maxCount"`
	// TargetType 目标类型（战斗机制空，轰炸机、鱼雷机对地）
	TargetType object.Type `json:"targetType"`
	// CurCount 当前数量（起飞 -1，回收 +1）
	CurCount int64 `json:"curCount"`
}
