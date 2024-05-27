package drawer

import (
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/colorx"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/resources/images/bullet"
	"github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/resources/images/texture"
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

		trailImg := texture.GetTrailImg(trail.Size, trail.Life)
		opts := d.genDefaultDrawImageOptions()
		opts.GeoM.Translate(
			(trail.Pos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-trail.Size,
			(trail.Pos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-trail.Size,
		)
		screen.DrawImage(trailImg, opts)
	}
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	// 战舰排序，确保渲染顺序是一致的（否则重叠战舰会出现问题）
	ships := lo.Values(ms.Ships)
	slices.SortFunc(ships, func(a, b *obj.BattleShip) int {
		return strings.Compare(a.Uid, b.Uid)
	})

	for _, s := range ships {
		// 只有在屏幕中的才渲染
		if !ms.Camera.Contains(s.CurPos) {
			continue
		}

		shipImg := ship.GetImg(s.Name)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, shipImg, s.CurRotation)
		opts.GeoM.Translate(
			(s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-float64(shipImg.Bounds().Dx()/2),
			(s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP，武器状态
		isShipSelected := slices.Contains(ms.SelectedShips, s.Uid)
		if (ms.GameOpts.ForceDisplayState || isShipSelected) && s.BelongPlayer == ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()

			// 绘制当前生命值
			opts.GeoM.Translate(
				(s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize,
				(s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-80,
			)
			hpImg := texture.GetHpImg(s.CurHP, s.TotalHP)
			screen.DrawImage(hpImg, opts)

			opts.GeoM.Translate(20, 0)
			// 绘制武器状态
			gunImg := lo.Ternary(s.Weapon.GunDisabled, texture.GunDisableImg, texture.GunEnableImg)
			screen.DrawImage(gunImg, opts)

			torpedoImg := lo.Ternary(s.Weapon.TorpedoDisabled, texture.TorpedoDisableImg, texture.TorpedoEnableImg)
			opts.GeoM.Translate(0, 25)
			screen.DrawImage(torpedoImg, opts)

			if isShipSelected {
				opts.GeoM.Translate(-55, 5)
				screen.DrawImage(texture.ShipSelectedImg, opts)
			}

			// 如果被编组，需要标记出来
			if s.GroupID != obj.GroupIDNone {
				textStr, fontSize := strconv.Itoa(int(s.GroupID)), float64(30)
				posX := (s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize - 30
				posY := (s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize - 85
				d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
			}
		}

		// 如果全局启用状态展示，则敌方战舰也要绘制 HP 值
		if ms.GameOpts.ForceDisplayState && s.BelongPlayer != ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-25,
				(s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-30,
			)
			hpImg := texture.GetEnemyHpImg(s.CurHP, s.TotalHP)
			screen.DrawImage(hpImg, opts)
		}

		// TODO 绘制战损情况，开火情况
	}
}

// 绘制消亡中的战舰
func (d *Drawer) drawDestroyedShips(screen *ebiten.Image, ms *state.MissionState) {
	for _, s := range ms.DestroyedShips {
		// 只有在屏幕中的才渲染
		if !ms.Camera.Contains(s.CurPos) {
			continue
		}

		shipImg := ship.GetImg(s.Name)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, shipImg, s.CurRotation)
		opts.GeoM.Translate(
			(s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-float64(shipImg.Bounds().Dx()/2),
			(s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 绘制爆炸效果
		explodeImg := texture.GetExplodeImg(s.CurHP)
		opts = d.genDefaultDrawImageOptions()
		opts.GeoM.Translate(
			(s.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-float64(explodeImg.Bounds().Dx()/2),
			(s.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-float64(explodeImg.Bounds().Dy()/2)-30,
		)
		screen.DrawImage(explodeImg, opts)
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
	for _, b := range ms.ForwardingBullets {
		img := bullet.GetImg(b.Name)

		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, img, b.Rotation)
		opts.GeoM.Translate(
			(b.CurPos.RX-float64(ms.Camera.Pos.MX))*constants.MapBlockSize-float64(img.Bounds().Dx()/2),
			(b.CurPos.RY-float64(ms.Camera.Pos.MY))*constants.MapBlockSize-float64(img.Bounds().Dy()/2),
		)
		screen.DrawImage(img, opts)
	}
}
