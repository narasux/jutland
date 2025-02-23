package object

type WeaponType string

const (
	// 所有
	WeaponTypeAll WeaponType = "all"
	// 主炮
	WeaponTypeMainGun WeaponType = "mainGun"
	// 副炮
	WeaponTypeSecondaryGun WeaponType = "secondaryGun"
	// 防空炮
	WeaponTypeAntiAircraftGun WeaponType = "antiAircraftGun"
	// 鱼雷
	WeaponTypeTorpedo WeaponType = "torpedo"
	// 导弹
	WeaponTypeMissile WeaponType = "missile"
)

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
	// 释放器
	Releasers []*Releaser
	// 最大射程（各类武器射程最大值）
	MaxToShipRange  float64
	MaxToPlaneRange float64
	// 拥有的武器情况
	HasMainGun         bool
	HasSecondaryGun    bool
	HasAntiAircraftGun bool
	HasTorpedo         bool
	HasReleaser        bool
	// 武器禁用情况
	MainGunDisabled         bool
	SecondaryGunDisabled    bool
	AntiAircraftGunDisabled bool
	TorpedoDisabled         bool
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

// PlaneWeapon 战机武器系统
type PlaneWeapon struct {
	// 机炮元数据
	GunsMD []WeaponMetadata `json:"guns"`
	// 炸弹元数据
	BombsMD []WeaponMetadata `json:"bombs"`
	// 鱼雷元数据
	TorpedoesMD []WeaponMetadata `json:"torpedoes"`
	// 固定机炮
	Guns []*Gun
	// 炸弹
	Bombs []*Releaser
	// 鱼雷
	Torpedoes []*Releaser
	// 最大射程（各类武器射程最大值）
	MaxToShipRange  float64
	MaxToPlaneRange float64
}
