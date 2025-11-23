package object

import "time"

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
	// 最大射程（各类武器射程最大值）
	MaxToShipRange  float64
	MaxToPlaneRange float64
	// 拥有的武器情况
	HasMainGun         bool
	HasSecondaryGun    bool
	HasAntiAircraftGun bool
	HasTorpedo         bool
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
	// 最小释放间隔
	ReleaseInterval int64 `json:"releaseInterval"`
	// 最近释放时间
	LatestReleaseAt int64
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

// PlaneGroup 飞机分组
type PlaneGroup struct {
	// Name 战机名称
	Name string `json:"name"`
	// MaxCount 总数量
	MaxCount int64 `json:"maxCount"`
	// TargetType 目标类型（战斗机制空，轰炸机、鱼雷机对地）
	TargetType ObjectType `json:"targetType"`
	// CurCount 当前数量（起飞 -1，回收 +1）
	CurCount int64 `json:"curCount"`
}

// ShipAircraft 战舰上的飞机，也能算是武器吧 :D
type ShipAircraft struct {
	// TakeOffTime 起飞耗时（单位：秒）
	TakeOffTime float64 `json:"takeOffTime"`
	// Groups 战机分组
	Groups []PlaneGroup `json:"groups"`

	// 是否禁用舰载机
	Disable bool
	// 是否拥有舰载机
	HasPlane bool
	// 最近起飞时间（毫秒时间戳)
	LatestTakeOffAt int64
}

// TakeOff 起飞战机（不区分飞机种类，只看打击对象类型）
func (sa *ShipAircraft) TakeOff(ship *BattleShip, targetObjType ObjectType) *Plane {
	// 判断起飞冷却，冷却中不允许起飞
	if sa.LatestTakeOffAt+int64(sa.TakeOffTime*1e3) > time.Now().UnixMilli() {
		return nil
	}

	for idx, g := range sa.Groups {
		if g.TargetType != targetObjType {
			continue
		}
		if g.CurCount <= 0 {
			continue
		}
		// 非指针需要通过索引修改
		sa.Groups[idx].CurCount--
		sa.LatestTakeOffAt = time.Now().UnixMilli()
		return NewPlane(g.Name, ship.CurPos, ship.CurRotation, ship.Uid, ship.BelongPlayer)
	}
	return nil
}

// Recovery 回收飞机
func (sa *ShipAircraft) Recovery(plane *Plane) {
	// 飞机血量低于 15% 时，没有回收价值
	if plane.CurHP/plane.TotalHP < 0.15 {
		return
	}
	// 逐个组按名称匹配
	for idx, g := range sa.Groups {
		if g.Name != plane.Name {
			continue
		}
		if g.CurCount >= g.MaxCount {
			continue
		}
		// 添加库存数量（非指针需要通过索引修改）
		sa.Groups[idx].CurCount++
		return
	}
}
