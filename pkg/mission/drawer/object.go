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
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, ms *state.MissionState) {
}

// 绘制尾流（战舰，鱼雷，炮弹）
func (d *Drawer) drawObjectTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.Trails {
		// 只有在屏幕中，且不处于延迟/消亡的尾流才渲染
		if !(ms.Camera.Contains(trail.Pos) && trail.IsActive()) {
			continue
		}

		trailImg := texture.GetTrailImg(trail.Shape, trail.CurSize, trail.CurLife, trail.Color)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, trailImg, trail.Rotation)
		opts.GeoM.Translate(
			(trail.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(trailImg.Bounds().Dx()/2),
			(trail.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(trailImg.Bounds().Dy()/2),
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
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(shipImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP，武器状态
		isShipSelected := slices.Contains(ms.SelectedShips, s.Uid)
		if (ms.GameOpts.ForceDisplayState || isShipSelected) && s.BelongPlayer == ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()

			// 绘制当前生命值
			opts.GeoM.Translate(
				(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-25,
				(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-30,
			)
			hpImg := texture.GetHpImg(s.CurHP, s.TotalHP)
			screen.DrawImage(hpImg, opts)

			if isShipSelected {
				opts.GeoM.Translate(-35, -10)
				screen.DrawImage(texture.ShipSelectedImg, opts)
				opts.GeoM.Translate(35, 10)
			}
			opts.GeoM.Translate(-20, 0)

			// 渲染武器状态时候的 X 方向间隙大小
			weaponInterstitialSpacing := 30.0
			if lo.EveryBy(
				[]bool{s.Weapon.HasMainGun, s.Weapon.HasSecondaryGun, s.Weapon.HasTorpedo},
				func(b bool) bool {
					return b
				},
			) {
				weaponInterstitialSpacing = 20.0
			}

			// 绘制主炮状态
			if s.Weapon.HasMainGun {
				opts.GeoM.Translate(20, -45)
				gunImg := lo.Ternary(
					s.Weapon.MainGunDisabled,
					texture.MainGunDisabledImg,
					texture.MainGunEnabledImg,
				)
				screen.DrawImage(gunImg, opts)
				opts.GeoM.Translate(0, 45)
			}

			// 绘制副炮状态
			if s.Weapon.HasSecondaryGun {
				opts.GeoM.Translate(weaponInterstitialSpacing, -45)
				gunImg := lo.Ternary(
					s.Weapon.SecondaryGunDisabled,
					texture.SecondaryGunDisabledImg,
					texture.SecondaryGunEnabledImg,
				)
				screen.DrawImage(gunImg, opts)
				opts.GeoM.Translate(0, 45)
			}

			// 绘制鱼雷发射器状态
			if s.Weapon.HasTorpedo {
				opts.GeoM.Translate(weaponInterstitialSpacing, -45)
				torpedoImg := lo.Ternary(
					s.Weapon.TorpedoDisabled,
					texture.TorpedoDisabledImg,
					texture.TorpedoEnabledImg,
				)
				screen.DrawImage(torpedoImg, opts)
				opts.GeoM.Translate(0, 45)
			}

			// 如果被编组，需要标记出来
			if s.GroupID != obj.GroupIDNone {
				textStr, fontSize := strconv.Itoa(int(s.GroupID)), float64(30)
				posX := (s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize - 55
				posY := (s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize - 85
				d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
			}
		}

		// 如果全局启用状态展示，则敌方战舰也要绘制 HP 值
		if ms.GameOpts.ForceDisplayState && s.BelongPlayer != ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-25,
				(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-30,
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
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(shipImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(shipImg.Bounds().Dy()/2),
		)
		screen.DrawImage(shipImg, opts)

		// 绘制爆炸效果
		explodeImg := texture.GetExplodeImg(s.CurHP)
		opts = d.genDefaultDrawImageOptions()
		opts.GeoM.Translate(
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(explodeImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(explodeImg.Bounds().Dy()/2)-30,
		)
		screen.DrawImage(explodeImg, opts)
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
	for _, b := range ms.ForwardingBullets {
		img := obj.GetBulletImg(b.Type, b.Diameter)

		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, img, b.Rotation)
		opts.GeoM.Translate(
			(b.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(img.Bounds().Dx()/2),
			(b.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(img.Bounds().Dy()/2),
		)
		screen.DrawImage(img, opts)
	}
}
