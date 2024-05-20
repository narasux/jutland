package object

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/envs"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type Gun struct {
	// 火炮名称
	Name string `json:"name"`
	// 炮弹类型
	BulletName string `json:"bulletName"`
	// 单次抛射炮弹数量
	BulletCount int `json:"bulletCount"`
	// 装填时间（单位: s)
	ReloadTime int64 `json:"reloadTime"`
	// 射程
	Range float64 `json:"range"`
	// 炮弹散布
	BulletSpread int `json:"bulletSpread"`
	// 炮弹速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64

	// 当前火炮是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次射击时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (g *Gun) CanFire(curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if g.Disable {
		return false
	}
	// 在重新装填，不可发射
	if g.LastFireTime+g.ReloadTime > time.Now().Unix() {
		return false
	}
	// 不在射程内，不可发射
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	if distance > g.Range {
		return false
	}
	return true
}

// Fire 发射
func (g *Gun) Fire(ship, enemy *BattleShip) []*Bullet {
	shotBullets := []*Bullet{}

	curPos, targetPos := ship.CurPos.Copy(), enemy.CurPos.Copy()
	// 炮塔距离战舰中心的距离
	gunOffset := g.PosPercent * ship.Length / constants.MapBlockSize
	curPos.AddRx(math.Sin(ship.CurRotation*math.Pi/180) * gunOffset)
	curPos.SubRy(math.Cos(ship.CurRotation*math.Pi/180) * gunOffset)
	// FIXME 其实还要考虑提前量（依赖敌舰速度，角度）

	if !g.CanFire(curPos, targetPos) {
		return shotBullets
	}
	g.LastFireTime = time.Now().Unix()

	// 散布应该随着距离减小而减小
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	rangePercent := distance / g.Range
	// 炮弹散布的半径
	radius := float64(g.BulletSpread) / constants.MapBlockSize * rangePercent

	for i := 0; i < g.BulletCount; i++ {
		pos := targetPos.Copy()
		// rand.Intn(3) - 1 算方向，rand.Float64() 算距离
		pos.AddRx(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		pos.AddRy(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		shotBullets = append(shotBullets, NewBullets(
			g.BulletName, curPos, pos, g.BulletSpeed, ship.Uid, ship.BelongPlayer,
		))
	}

	return shotBullets
}

var gunMap = map[string]*Gun{}

func newGun(name string, posPercent float64) *Gun {
	g := deepcopy.Copy(*gunMap[name]).(Gun)
	g.PosPercent = posPercent
	return &g
}

func init() {
	file, err := os.Open(filepath.Join(envs.ConfigBaseDir, "guns.json"))
	if err != nil {
		log.Fatalf("failed to open guns.json: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var guns []Gun
	if err = json.Unmarshal(bytes, &guns); err != nil {
		log.Fatalf("failed to unmarshal guns.json: %s", err)
	}

	for _, g := range guns {
		gunMap[g.Name] = &g
	}
}
