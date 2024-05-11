package drawer

import (
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
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

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	for _, ship := range ms.Ships {
		ebutil.DebugPrint(screen,
			fmt.Sprintf("\n\nship.MX: %d, ship.MY: %d, ship.RX: %f, ship.RY: %f\nspeed: %f, rotation: %f",
				ship.CurPos.MX, ship.CurPos.MY,
				ship.CurPos.RX, ship.CurPos.RY,
				ship.CurSpeed, ship.CurRotation,
			))
		// 只有在屏幕中的才渲染
		if ship.CurPos.MX < ms.Camera.Pos.MX ||
			ship.CurPos.MX > ms.Camera.Pos.MX+ms.Camera.Width ||
			ship.CurPos.MY < ms.Camera.Pos.MY ||
			ship.CurPos.MY > ms.Camera.Pos.MY+ms.Camera.Height {
			continue
		}

		isShipSelected := slices.Contains(ms.SelectedShips, ship.Uid)

		// 如果战舰被选中，则需要绘制选中框
		if isShipSelected {
			// 绘制选中框
			selectBoxImg := texture.SelectBoxWhiteImg
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-float64(selectBoxImg.Bounds().Dx()/2),
				(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-float64(selectBoxImg.Bounds().Dy()/2),
			)
			screen.DrawImage(selectBoxImg, opts)
		}

		shipImg := obj.GetShipImg(ship.Name)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, shipImg, ship.CurRotation)
		opts.GeoM.Translate(
			(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-float64(shipImg.Bounds().Dx()/2),
			(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 如果战舰被选中，则需要绘制 HP，武器状态 TODO 如果全局启用状态展示，也要绘制
		isDisplayShipState := false
		if isShipSelected || isDisplayShipState {
			opts = d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize-25,
				(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize-30,
			)

			// 绘制当前生命值
			hpImg := texture.GetHpImg(ship.CurHP, ship.TotalHP)
			screen.DrawImage(hpImg, opts)

			// 绘制武器状态
			gunImg := lo.Ternary(ship.Weapon.GunDisabled, texture.GunDisableImg, texture.GunEnableImg)
			opts.GeoM.Translate(-10, -30)
			screen.DrawImage(gunImg, opts)

			torpedoImg := lo.Ternary(ship.Weapon.TorpedoDisabled, texture.TorpedoDisableImg, texture.TorpedoEnableImg)
			opts.GeoM.Translate(40, 0)
			screen.DrawImage(torpedoImg, opts)
		}

		// TODO 绘制战损情况，开火情况
		// TODO 绘制尾流（速度不同，尺寸不同，尾流不同？）
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
}
