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
	buildingImg "github.com/narasux/jutland/pkg/resources/images/building"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, ms *state.MissionState) {
	// 增援点（只有在屏幕中的才渲染）
	for _, rp := range ms.ReinforcePoints {
		if !ms.Camera.Contains(rp.Pos) {
			continue
		}
		img := lo.Ternary(
			rp.BelongPlayer == ms.CurPlayer,
			buildingImg.ReinforcePoint,
			buildingImg.EnemyReinforcePoint,
		)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, img, rp.Rotation)
		opts.GeoM.Translate(
			(rp.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(img.Bounds().Dx()/2),
			(rp.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(img.Bounds().Dy()/2),
		)
		screen.DrawImage(img, opts)

		if process := rp.Progress(); process > 0 {
			d.drawText(
				screen, strconv.Itoa(process),
				(rp.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-10,
				(rp.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-12,
				20,
				font.Hang,
				colorx.White,
			)
		}
	}
}

// 绘制尾流（战舰，鱼雷，炮弹）
func (d *Drawer) drawObjectTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.Trails {
		// 只有在屏幕中，且不处于延迟/消亡的尾流才渲染
		if !(ms.Camera.Contains(trail.Pos) && trail.IsActive()) {
			continue
		}

		trailImg := textureImg.GetTrail(trail.Shape, trail.CurSize, trail.CurLife, trail.Color)
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

		sImg := shipImg.Get(s.Name)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, sImg, s.CurRotation)
		opts.GeoM.Translate(
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(sImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(sImg.Bounds().Dy()/2),
		)
		screen.DrawImage(sImg, opts)

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP，武器状态
		isShipSelected := slices.Contains(ms.SelectedShips, s.Uid)
		if (ms.GameOpts.ForceDisplayState || isShipSelected) && s.BelongPlayer == ms.CurPlayer {
			opts = d.genDefaultDrawImageOptions()

			// 绘制当前生命值
			opts.GeoM.Translate(
				(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-25,
				(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-30,
			)
			hpImg := textureImg.GetHP(s.CurHP, s.TotalHP)
			screen.DrawImage(hpImg, opts)

			if isShipSelected {
				opts.GeoM.Translate(-35, -10)
				screen.DrawImage(textureImg.ShipSelected, opts)
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
					textureImg.MainGunDisabled,
					textureImg.MainGunEnabled,
				)
				screen.DrawImage(gunImg, opts)
				opts.GeoM.Translate(0, 45)
			}

			// 绘制副炮状态
			if s.Weapon.HasSecondaryGun {
				opts.GeoM.Translate(weaponInterstitialSpacing, -45)
				gunImg := lo.Ternary(
					s.Weapon.SecondaryGunDisabled,
					textureImg.SecondaryGunDisabled,
					textureImg.SecondaryGunEnabled,
				)
				screen.DrawImage(gunImg, opts)
				opts.GeoM.Translate(0, 45)
			}

			// 绘制鱼雷发射器状态
			if s.Weapon.HasTorpedo {
				opts.GeoM.Translate(weaponInterstitialSpacing, -45)
				torpedoImg := lo.Ternary(
					s.Weapon.TorpedoDisabled,
					textureImg.TorpedoDisabled,
					textureImg.TorpedoEnabled,
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
			hpImg := textureImg.GetEnemyHP(s.CurHP, s.TotalHP)
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

		sImg := shipImg.Get(s.Name)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, sImg, s.CurRotation)
		opts.GeoM.Translate(
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(sImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(sImg.Bounds().Dy()/2),
		)
		screen.DrawImage(sImg, opts)

		// 绘制爆炸效果
		explodeImg := textureImg.GetExplode(s.CurHP)
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
