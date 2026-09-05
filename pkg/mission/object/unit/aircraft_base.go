package unit

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// AircraftBase 飞机起降基地（航母 / 陆地机场）需要提供的几何与机库契约。
// 起飞滑跑与着舰进近的几何计算只依赖这些量：位置、朝向、长宽、速度和机库配置。
// 航母是会移动转向的基地；陆地机场是静态基地（速度恒为 0，转向速率恒为 0）。
// 既有几何函数（carrierRelativePos2D 等）沿用了 "carrier" 命名，实际同时服务两类基地。
type AircraftBase interface {
	// BaseUid 基地唯一标识（舰船 Uid 或机场 Uid），作为飞机的 BelongShip
	BaseUid() string
	// BasePos 基地中心位置
	BasePos() objPos.MapPos
	// BaseRotation 基地朝向（度）
	BaseRotation() float64
	// BaseLength 基地长度（资源像素，换算地图块时除以 MapBlockSize）
	BaseLength() float64
	// BaseWidth 基地宽度（资源像素）
	BaseWidth() float64
	// BaseSpeed 基地当前速度（地图坐标每模拟帧），静态基地恒为 0
	BaseSpeed() float64
	// BaseBelongPlayer 基地所属阵营
	BaseBelongPlayer() faction.Player
	// BaseAircraft 机库
	BaseAircraft() *ShipAircraft
}

var _ AircraftBase = (*BattleShip)(nil)

// BaseUid 基地唯一标识。
func (s *BattleShip) BaseUid() string { return s.Uid }

// BasePos 基地中心位置。
func (s *BattleShip) BasePos() objPos.MapPos { return s.CurPos }

// BaseRotation 基地朝向。
func (s *BattleShip) BaseRotation() float64 { return s.CurRotation }

// BaseLength 基地长度。
func (s *BattleShip) BaseLength() float64 { return s.Length }

// BaseWidth 基地宽度。
func (s *BattleShip) BaseWidth() float64 { return s.Width }

// BaseSpeed 基地当前速度。
func (s *BattleShip) BaseSpeed() float64 { return s.CurSpeed }

// BaseBelongPlayer 基地所属阵营。
func (s *BattleShip) BaseBelongPlayer() faction.Player { return s.BelongPlayer }

// BaseAircraft 机库。
func (s *BattleShip) BaseAircraft() *ShipAircraft { return &s.Aircraft }

// isBaseMissing 判断基地是否缺失（飞机所属航母已被击沉等）。
// 注意：nil 的 *BattleShip 装入接口后接口值非 nil，必须按具体类型判空。
func isBaseMissing(base AircraftBase) bool {
	if base == nil {
		return true
	}
	if ship, ok := base.(*BattleShip); ok {
		return ship == nil
	}
	return false
}
