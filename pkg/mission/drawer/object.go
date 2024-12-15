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
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// 绘制尾流（战舰，鱼雷，炮弹）
func (d *Drawer) drawObjectTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.Trails {
		// 只有在屏幕中，且不处于延迟/消亡的尾流才渲染
		if !(ms.Camera.Contains(trail.Pos) && trail.IsActive()) {
			continue
		}

		trailImg := textureImg.GetTrail(trail.Shape, trail.CurSize, trail.CurLife, trail.Color)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, trailImg, trail.Rotation)
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

		sImg := shipImg.GetTop(s.Name, ms.GameOpts.Zoom)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, sImg, s.CurRotation)
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
			var weaponInterstitialSpacing float64
			weaponCnt := len(lo.Filter(
				[]bool{
					s.Weapon.HasMainGun,
					s.Weapon.HasSecondaryGun,
					s.Weapon.HasAntiAircraftGun,
					s.Weapon.HasTorpedo,
				},
				func(b bool, _ int) bool {
					return b
				},
			))
			switch weaponCnt {
			case 4:
				weaponInterstitialSpacing = 15.0
			case 3:
				weaponInterstitialSpacing = 20.0
			default:
				weaponInterstitialSpacing = 35.0
			}

			// 绘制主炮状态
			if s.Weapon.HasMainGun {
				clr := colorx.Yellow
				if s.Weapon.MainGunDisabled {
					clr = colorx.Red
				} else if s.Weapon.MainGunReloaded() {
					clr = colorx.Green
				}
				opts.GeoM.Translate(20, -25)
				img := textureImg.GetText("M", font.Hang, 16, clr)
				screen.DrawImage(img, opts)
				opts.GeoM.Translate(0, 25)
			}

			// 绘制副炮状态
			if s.Weapon.HasSecondaryGun {
				clr := colorx.Yellow
				if s.Weapon.SecondaryGunDisabled {
					clr = colorx.Red
				} else if s.Weapon.SecondaryGunReloaded() {
					clr = colorx.Green
				}
				opts.GeoM.Translate(weaponInterstitialSpacing, -25)
				img := textureImg.GetText("S", font.Hang, 16, clr)
				screen.DrawImage(img, opts)
				opts.GeoM.Translate(0, 25)
			}

			// 绘制防空炮状态（注：由于防空炮装填速度很快，所以不需要绘制装填中的状态，即只有红绿两种）
			if s.Weapon.HasAntiAircraftGun {
				clr := lo.Ternary(s.Weapon.AntiAircraftGunDisabled, colorx.Red, colorx.Green)
				opts.GeoM.Translate(weaponInterstitialSpacing, -25)
				img := textureImg.GetText("A", font.Hang, 16, clr)
				screen.DrawImage(img, opts)
				opts.GeoM.Translate(0, 25)
			}

			// 绘制鱼雷发射器状态
			if s.Weapon.HasTorpedo {
				clr := colorx.Yellow
				if s.Weapon.TorpedoDisabled {
					clr = colorx.Red
				} else if s.Weapon.TorpedoLauncherReloaded() {
					clr = colorx.Green
				}
				opts.GeoM.Translate(weaponInterstitialSpacing, -25)
				img := textureImg.GetText("T", font.Hang, 16, clr)
				screen.DrawImage(img, opts)
				opts.GeoM.Translate(0, 25)
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

		sImg := shipImg.GetTop(s.Name, ms.GameOpts.Zoom)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, sImg, s.CurRotation)
		opts.GeoM.Translate(
			(s.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(sImg.Bounds().Dx()/2),
			(s.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(sImg.Bounds().Dy()/2),
		)
		screen.DrawImage(sImg, opts)

		// 绘制爆炸效果
		explodeImg := textureImg.GetShipExplode(s.CurHP)
		opts = d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, explodeImg, s.CurRotation)
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
		ebutil.SetOptsCenterRotation(opts, img, b.Rotation)
		opts.GeoM.Translate(
			(b.CurPos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(img.Bounds().Dx()/2),
			(b.CurPos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(img.Bounds().Dy()/2),
		)
		screen.DrawImage(img, opts)
	}
}
