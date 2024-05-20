package object

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/envs"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/images/bullet"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// 火炮 / 鱼雷弹药
type Bullet struct {
	// 弹药名称
	Name string `json:"name"`
	// 伤害数值
	Damage float64 `json:"damage"`
	// 生命（前进太多要消亡）
	Life int `json:"life"`

	// 唯一标识
	Uid string
	// 当前位置
	CurPos MapPos
	// 目标位置
	TargetPos MapPos
	// 旋转角度
	Rotation float64
	// 速度
	Speed float64

	// 所属战舰
	BelongShip string
	// 所属阵营（玩家）
	BelongPlayer faction.Player

	// 是否命中战舰/水面/陆地
	HitShip  bool
	HitWater bool
	HitLand  bool
}

// Forward 弹药前进
func (b *Bullet) Forward() {
	// 修改位置
	b.CurPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	b.CurPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)
	// 修改生命
	b.Life--
}

var bulletMap = map[string]*Bullet{}

// NewBullets 新建弹药
func NewBullets(
	name string,
	curPos, targetPos MapPos,
	speed float64,
	shipUid string,
	player faction.Player,
) *Bullet {
	b := deepcopy.Copy(*bulletMap[name]).(Bullet)
	b.Uid = uuid.New().String()
	b.CurPos = curPos
	b.TargetPos = targetPos
	b.Rotation = geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	b.Speed = speed
	b.BelongShip = shipUid
	b.BelongPlayer = player
	return &b
}

// GetBulletImg 获取弹药图片
func GetBulletImg(name string) *ebiten.Image {
	// FIXME 应该加载正确的图片，该方法移动到 resources/bullet
	return bullet.DefaultBulletImg
}

func init() {
	file, err := os.Open(filepath.Join(envs.ConfigBaseDir, "bullets.json"))
	if err != nil {
		log.Fatalf("failed to open bullets.json: %s", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var bullets []Bullet
	if err = json.Unmarshal(bytes, &bullets); err != nil {
		log.Fatalf("failed to unmarshal bullets.json: %s", err)
	}

	for _, b := range bullets {
		bulletMap[b.Name] = &b
	}
}
