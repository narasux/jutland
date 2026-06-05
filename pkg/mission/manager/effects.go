package manager

import (
	"math"
	"math/rand"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/common/constants"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	"github.com/narasux/jutland/pkg/mission/object/trail"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 更新尾流状态（战舰，鱼雷，炮弹）
func (m *MissionManager) updateObjectTrails() {
	for i := 0; i < len(m.state.Arena.Trails); i++ {
		m.state.Arena.Trails[i].Update()
	}
	// 生命周期结束的，不再需要
	trails := m.state.Arena.Trails[:0]
	for _, t := range m.state.Arena.Trails {
		if t.IsAlive() {
			trails = append(trails, t)
		}
	}
	m.state.Arena.Trails = trails
	for _, ship := range m.state.Arena.Ships {
		if trails := ship.GenTrails(); trails != nil {
			m.state.Arena.Trails = append(m.state.Arena.Trails, trails...)
		}
	}
	for _, bt := range m.state.Arena.ForwardingBullets {
		// 炸弹目前没有尾流
		if bt.Type == objBullet.TypeBomb {
			continue
		}
		if trails := bt.GenTrails(); trails != nil {
			m.state.Arena.Trails = append(m.state.Arena.Trails, trails...)
		}
	}
	// 消亡中的飞机生成火焰 + 黑烟尾流（拉烟效果）
	for _, plane := range m.state.Arena.DestroyedPlanes {
		if plane.CurSpeed <= 0 {
			continue
		}
		// 计算飞机尾部位置（相对飞机朝向的后方偏移）
		tailPos := plane.CurPos.Copy()
		sinVal := math.Sin(plane.CurRotation * math.Pi / 180)
		cosVal := math.Cos(plane.CurRotation * math.Pi / 180)
		tailOffset := plane.Length / constants.MapBlockSize * 0.3
		tailPos.SubRx(sinVal * tailOffset)
		tailPos.AddRy(cosVal * tailOffset)

		// 火焰尾流（橙红色，较小，扩散快，生命短）
		m.state.Arena.Trails = append(m.state.Arena.Trails, trail.New(
			tailPos, textureImg.TrailShapeCircle,
			3.0, 0.8, // 初始尺寸 3，扩散速度 0.8
			80, 3.0, // 生命值 80，衰减速度 3.0
			0, 0,
			colorx.Orange,
		))
		// 黑烟尾流（深灰色，较大，扩散慢，生命长）
		m.state.Arena.Trails = append(m.state.Arena.Trails, trail.New(
			tailPos, textureImg.TrailShapeCircle,
			2.0, 0.5, // 初始尺寸 2，扩散速度 0.5
			120, 2.0, // 生命值 120，衰减速度 2.0
			2, 0, // 延迟 2 帧出现（略慢于火焰）
			colorx.DarkSilver,
		))
	}
}

// 更新局内战舰
func (m *MissionManager) updateMissionShips() {
	audioPlayQuota := 2
	// 如果战舰 HP 为 0，则需要走消亡流程
	for uid, ship := range m.state.Arena.Ships {
		if ship.CurHP <= 0 {
			// 这里做了取巧，复用 CurHP 用于后续渲染爆炸效果
			ship.CurHP = textureImg.MaxShipExplodeState

			if audioPlayQuota > 0 && m.state.View.Camera.Contains(ship.CurPos) {
				audio.PlayAudioToEnd(audioRes.NewShipExplode())
				audioPlayQuota--
			}

			m.state.Arena.DestroyedShips = append(m.state.Arena.DestroyedShips, ship)
			delete(m.state.Arena.Ships, uid)
		}
	}

	// 消亡中的战舰会逐渐掉血到 0
	for _, ship := range m.state.Arena.DestroyedShips {
		ship.CurHP -= 0.5
		// 支持逐渐减速的效果，而不是直接就变成 0
		ship.CurSpeed = max(0, ship.CurSpeed-ship.MaxSpeed/30)
	}

	// 移除已经完全消亡的战舰
	destroyedShips := m.state.Arena.DestroyedShips[:0]
	for _, ship := range m.state.Arena.DestroyedShips {
		if ship.CurHP > 0 {
			destroyedShips = append(destroyedShips, ship)
		}
	}
	m.state.Arena.DestroyedShips = destroyedShips
}

// 更新局内战机
func (m *MissionManager) updateMissionPlanes() {
	// 如果战机 HP 为 0，则需要走消亡流程
	for uid, plane := range m.state.Arena.Planes {
		if plane.CurHP <= 0 {
			// 这里做了取巧，复用 CurHP 用于后续渲染爆炸效果
			plane.CurHP = textureImg.MaxPlaneExplodeState

			// 随机决定坠落偏转方向和幅度，存储在 RemainRange 中（借用该字段）
			// 范围 [-0.5, 0.5]，正值右偏，负值左偏
			plane.RemainRange = rand.Float64() - 0.5

			m.state.Arena.DestroyedPlanes = append(m.state.Arena.DestroyedPlanes, plane)
			delete(m.state.Arena.Planes, uid)
		}
	}

	mapCfg := m.state.Core.MissionMD.MapCfg
	for _, plane := range m.state.Arena.DestroyedPlanes {
		// 消亡中的战机会逐渐掉血到 0
		plane.CurHP -= 1
		// 模拟被击落效果：逐渐减速（比战舰更缓）并保持惯性前进
		plane.CurSpeed = max(0, plane.CurSpeed-plane.MaxSpeed/60)
		// 模拟失控螺旋：每帧添加微小旋转偏转（借用 RemainRange 存储偏转方向）
		plane.CurRotation += plane.RemainRange
		if plane.CurSpeed > 0 {
			nextPos := plane.CurPos.Copy()
			nextPos.AddRx(math.Sin(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
			nextPos.SubRy(math.Cos(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
			nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
			plane.CurPos = nextPos
		}
	}

	// 移除已经完全消亡的战舰
	destroyedPlanes := m.state.Arena.DestroyedPlanes[:0]
	for _, plane := range m.state.Arena.DestroyedPlanes {
		if plane.CurHP > 0 {
			destroyedPlanes = append(destroyedPlanes, plane)
		}
	}
	m.state.Arena.DestroyedPlanes = destroyedPlanes
}
