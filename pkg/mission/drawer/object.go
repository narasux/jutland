package drawer

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, ms *state.MissionState) {
}

// 绘制战舰尾流
func (d *Drawer) drawShipTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.ShipTrails {
		// 只有在屏幕中的才渲染
		if !ms.Camera.Contains(trail.Pos) {
			continue
		}
		// 尾流太近 / 太远则不渲染
		if trail.Life > 50 || trail.Life < 0 {
			continue
		}

		cx := (trail.Pos.RX - float64(ms.Camera.Pos.MX)) * mapblock.BlockSize
		cy := (trail.Pos.RY - float64(ms.Camera.Pos.MY)) * mapblock.BlockSize
		clr := color.NRGBA{255, 255, 255, uint8(trail.Life)}
		vector.DrawFilledCircle(screen, float32(cx), float32(cy), float32(trail.Size), clr, false)
	}
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	// 战舰排序，确保渲染顺序是一致的（否则重叠战舰会出现问题）
	ships := lo.Values(ms.Ships)
	slices.SortFunc(ships, func(a, b *obj.BattleShip) int {
		return strings.Compare(a.Uid, b.Uid)
	})

	for _, ship := range ships {
		ebutil.DebugPrint(screen,
			fmt.Sprintf("\n\nship.MX: %d, ship.MY: %d, ship.RX: %f, ship.RY: %f\nspeed: %f, rotation: %f",
				ship.CurPos.MX, ship.CurPos.MY,
				ship.CurPos.RX, ship.CurPos.RY,
				ship.CurSpeed, ship.CurRotation,
			))
		// 只有在屏幕中的才渲染
		if !ms.Camera.Contains(ship.CurPos) {
			continue
		}

		shipImg := obj.GetShipImg(ship.Name)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, shipImg, ship.CurRotation)
		opts.GeoM.Translate(
			(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-float64(shipImg.Bounds().Dx()/2),
			(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP，武器状态
		isShipSelected := slices.Contains(ms.SelectedShips, ship.Uid)
		if (ms.GameOpts.ForceDisplayState || isShipSelected) && ship.BelongPlayer == ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-25,
				(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-85,
			)
			// 绘制武器状态
			gunImg := lo.Ternary(ship.Weapon.GunDisabled, texture.GunDisableImg, texture.GunEnableImg)
			screen.DrawImage(gunImg, opts)

			torpedoImg := lo.Ternary(ship.Weapon.TorpedoDisabled, texture.TorpedoDisableImg, texture.TorpedoEnableImg)
			opts.GeoM.Translate(0, 25)
			screen.DrawImage(torpedoImg, opts)

			// 绘制当前生命值
			opts.GeoM.Translate(40, -22)
			hpImg := texture.GetHpImg(ship.CurHP, ship.TotalHP)
			screen.DrawImage(hpImg, opts)
		}

		// 如果全局启用状态展示，则敌方战舰也要绘制 HP 值
		if ms.GameOpts.ForceDisplayState && ship.BelongPlayer != ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-25,
				(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-30,
			)
			hpImg := texture.GetEnemyHpImg(ship.CurHP, ship.TotalHP)
			screen.DrawImage(hpImg, opts)
		}

		// TODO 绘制战损情况，开火情况
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
}
