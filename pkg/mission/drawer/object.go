package drawer

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	planeImg "github.com/narasux/jutland/pkg/resources/images/plane"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	weaponImg "github.com/narasux/jutland/pkg/resources/images/weapon"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// closestResourceZoom 选择最接近目标显示倍率的资源档位。
// 这样可以优先复用已有缓存图片，再用少量 GeoM 缩放补齐剩余比例
func closestResourceZoom(target float64, candidates []int) int {
	best := candidates[0]
	bestDiff := math.Abs(target - float64(best))
	for _, candidate := range candidates[1:] {
		diff := math.Abs(target - float64(candidate))
		if diff < bestDiff {
			best = candidate
			bestDiff = diff
		}
	}
	return best
}

// shipResource 返回战舰图片和残余绘制缩放比例。
// 普通战舰按最接近档位取图，特殊默认素材只取原图并完全依赖场景缩放
func shipResource(name string, sceneZoom int) (*ebiten.Image, float64) {
	target := float64(state.NormalizeZoom(sceneZoom)) / float64(state.DefaultZoom())
	if name == "duck" || name == "waterdrop" || name == "molamola" {
		return shipImg.GetTop(name, 1), target
	}
	resourceZoom := closestResourceZoom(target, []int{1, 2, 4})
	return shipImg.GetTop(name, resourceZoom), target / float64(resourceZoom)
}

// planeResource 返回飞机图片和残余绘制缩放比例。
// 飞机保留旧实现中比战舰大一档的视觉效果，再匹配最接近的飞机资源档位
func planeResource(name string, sceneZoom int) (*ebiten.Image, float64) {
	// 飞机沿用原来的视觉意图：默认比战舰大一档。
	target := float64(state.NormalizeZoom(sceneZoom)) / float64(state.DefaultZoom()) * 2
	resourceZoom := closestResourceZoom(target, []int{1, 2, 4, 8, 10})
	return planeImg.Get(name, resourceZoom), target / float64(resourceZoom)
}

// weaponResource 返回武器状态图标和残余绘制缩放比例。
// 它把场景 zoom 转成资源档位，避免把新的场景缩放值直接传给旧资源接口
func weaponResource(
	weapon weaponImg.WeaponType, status weaponImg.WeaponStatus, sceneZoom int,
) (*ebiten.Image, float64) {
	target := float64(state.NormalizeZoom(sceneZoom)) / float64(state.DefaultZoom())
	resourceZoom := closestResourceZoom(target, []int{1, 2, 4})
	return weaponImg.Get(weapon, status, resourceZoom), target / float64(resourceZoom)
}

// rotatedRectangleCorners 返回以屏幕坐标为中心、长度沿 Y 轴的旋转矩形四角。
func rotatedRectangleCorners(centerX, centerY, length, width, rotation float64) [4][2]float64 {
	halfLength, halfWidth := length/2, width/2
	corners := [4][2]float64{
		{-halfWidth, -halfLength},
		{halfWidth, -halfLength},
		{halfWidth, halfLength},
		{-halfWidth, halfLength},
	}

	sinA, cosA := math.Sincos(rotation * degToRad)
	for idx, corner := range corners {
		x, y := corner[0], corner[1]
		corners[idx] = [2]float64{
			centerX + x*cosA - y*sinA,
			centerY + x*sinA + y*cosA,
		}
	}
	return corners
}

// drawUnitHitBox 按伤害判定使用的中心、长宽和朝向绘制单位受打击范围。
func drawUnitHitBox(screen *ebiten.Image, ms *state.MissionState, battleUnit objUnit.BattleUnit) {
	movementState := battleUnit.MovementState()
	geometricSize := battleUnit.GeometricSize()
	centerX, centerY := ms.CameraPosToScreen(movementState.CurPos)
	blockSize := ms.MapBlockDisplaySize()
	corners := rotatedRectangleCorners(
		centerX,
		centerY,
		geometricSize.Length/constants.MapBlockSize*blockSize,
		geometricSize.Width/constants.MapBlockSize*blockSize,
		movementState.CurRotation,
	)
	for idx, corner := range corners {
		nextCorner := corners[(idx+1)%len(corners)]
		vector.StrokeLine(
			screen,
			float32(corner[0]),
			float32(corner[1]),
			float32(nextCorner[0]),
			float32(nextCorner[1]),
			2,
			colorx.Red,
			false,
		)
	}
}

// 绘制医疗船治疗范围圈（仅在选中己方医疗船时显示）
func (d *Drawer) drawHospitalShipHealRange(screen *ebiten.Image, ms *state.MissionState) {
	for _, ship := range ms.Arena.Ships {
		// 只绘制己方存活的医疗船
		if ship.Type != objUnit.ShipTypeHospital || ship.BelongPlayer != ms.Player.CurPlayer || ship.CurHP <= 0 {
			continue
		}
		// 检查是否被选中
		if !slices.Contains(ms.Interaction.SelectedShips, ship.Uid) {
			continue
		}
		// 只有在屏幕中的才渲染
		if !ms.View.Camera.Contains(ship.CurPos) {
			continue
		}
		x, y := ms.CameraPosToScreen(ship.CurPos)
		cx := float32(x)
		cy := float32(y)
		// 计算半径像素值
		radius := float32(objUnit.HospitalShipEffectRange * ms.MapBlockDisplaySize())
		// 绘制半透明浅绿色实线圆
		vector.StrokeCircle(screen, cx, cy, radius, 1, colorx.LightGreen, false)
	}
}

// 绘制尾流（战舰，鱼雷，炮弹）
func (d *Drawer) drawObjectTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.Arena.Trails {
		// 只有在屏幕中，且不处于延迟/消亡的尾流才渲染
		if !(ms.View.Camera.Contains(trail.Pos) && trail.IsActive()) {
			continue
		}

		trailImg := textureImg.GetTrail(trail.Shape, trail.CurSize, trail.CurLife, trail.Color)
		drawImageCenteredAtMapPos(screen, ms, trailImg, trail.Pos, trail.Rotation, ms.ZoomScale())
	}
}

// drawExplosions 绘制火箭弹等局部爆炸效果
func (d *Drawer) drawExplosions(screen *ebiten.Image, ms *state.MissionState) {
	for _, explosion := range ms.Arena.Explosions {
		if !ms.View.Camera.Contains(explosion.Pos) {
			continue
		}
		explodeImg := textureImg.GetPlaneExplode(explosion.FrameHP())
		explodeX, explodeY := ms.CameraPosToScreen(explosion.Pos)
		drawImageCentered(screen, explodeImg, explodeX, explodeY, explosion.Rotation, ms.ZoomScale())
	}
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	// 战舰排序，确保渲染顺序是一致的（否则重叠战舰会出现问题）
	ships := lo.Values(ms.Arena.Ships)
	slices.SortFunc(ships, func(a, b *objUnit.BattleShip) int {
		return strings.Compare(a.Uid, b.Uid)
	})

	for _, s := range ships {
		// 只有在屏幕中的才渲染
		if !ms.View.Camera.Contains(s.CurPos) {
			continue
		}

		sImg, sImgScale := shipResource(s.CurrentTopImageName(), ms.UI.GameOpts.Zoom)
		shipX, shipY := ms.CameraPosToScreen(s.CurPos)
		drawImageCentered(screen, sImg, shipX, shipY, s.CurRotation, sImgScale)
		if ms.UI.DebugFlags.ShowHitBoxes {
			drawUnitHitBox(screen, ms, s)
		}

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP，武器状态
		isShipSelected := slices.Contains(ms.Interaction.SelectedShips, s.Uid)
		if (ms.UI.GameOpts.ForceDisplayState || isShipSelected) && s.BelongPlayer == ms.Player.CurPlayer {
			sceneScale := ms.ZoomScale()
			// 绘制当前生命值
			hpImg := textureImg.GetHP(s.CurHP, s.TotalHP)
			drawImageAtScale(screen, hpImg, shipX-25*sceneScale, shipY-30*sceneScale, sceneScale)

			if isShipSelected {
				drawImageAtScale(screen, textureImg.ShipSelected, shipX-60*sceneScale, shipY-40*sceneScale, sceneScale)
			}

			// 渲染武器状态时候的 X 方向间隙大小
			var weaponInterstitialSpacing float64
			weaponCnt := len(lo.Filter(
				[]bool{
					s.Weapon.HasMainGun,
					s.Weapon.HasSecondaryGun,
					s.Weapon.HasAntiAircraftGun,
					s.Weapon.HasTorpedo,
					s.Weapon.HasRocket,
				},
				func(b bool, _ int) bool {
					return b
				},
			))
			switch weaponCnt {
			case 5:
				weaponInterstitialSpacing = 12.0
			case 4:
				weaponInterstitialSpacing = 15.0
			case 3:
				weaponInterstitialSpacing = 20.0
			default:
				weaponInterstitialSpacing = 35.0
			}
			weaponX := shipX - 45*sceneScale

			// 绘制主炮状态
			if s.Weapon.HasMainGun {
				status := weaponImg.WeaponStatusReloading
				if s.Weapon.MainGunDisabled {
					status = weaponImg.WeaponStatusDisabled
				} else if s.Weapon.MainGunReloaded() {
					status = weaponImg.WeaponStatusLoaded
				}

				weaponIcon, weaponScale := weaponResource(weaponImg.WeaponTypeMainGun, status, ms.UI.GameOpts.Zoom)
				drawImageAtScale(screen, weaponIcon, weaponX+20*sceneScale, shipY-60*sceneScale, weaponScale)
			}

			// 绘制副炮状态
			if s.Weapon.HasSecondaryGun {
				status := weaponImg.WeaponStatusReloading
				if s.Weapon.SecondaryGunDisabled {
					status = weaponImg.WeaponStatusDisabled
				} else if s.Weapon.SecondaryGunReloaded() {
					status = weaponImg.WeaponStatusLoaded
				}

				weaponIcon, weaponScale := weaponResource(
					weaponImg.WeaponTypeSecondaryGun,
					status,
					ms.UI.GameOpts.Zoom,
				)
				weaponX += weaponInterstitialSpacing * sceneScale
				drawImageAtScale(screen, weaponIcon, weaponX+20*sceneScale, shipY-60*sceneScale, weaponScale)
			}

			// 绘制防空炮状态（注：由于防空炮装填速度很快，所以不需要绘制装填中的状态，即只有红绿两种）
			if s.Weapon.HasAntiAircraftGun {
				status := weaponImg.WeaponStatusLoaded
				if s.Weapon.AntiAircraftGunDisabled {
					status = weaponImg.WeaponStatusDisabled
				}

				weaponIcon, weaponScale := weaponResource(
					weaponImg.WeaponTypeAntiAircraftGun,
					status,
					ms.UI.GameOpts.Zoom,
				)
				weaponX += weaponInterstitialSpacing * sceneScale
				drawImageAtScale(screen, weaponIcon, weaponX+20*sceneScale, shipY-60*sceneScale, weaponScale)
			}

			// 绘制鱼雷发射器状态
			if s.Weapon.HasTorpedo {
				status := weaponImg.WeaponStatusReloading
				if s.Weapon.TorpedoDisabled {
					status = weaponImg.WeaponStatusDisabled
				} else if s.Weapon.TorpedoLauncherReloaded() {
					status = weaponImg.WeaponStatusLoaded
				}

				weaponIcon, weaponScale := weaponResource(weaponImg.WeaponTypeTorpedo, status, ms.UI.GameOpts.Zoom)
				weaponX += weaponInterstitialSpacing * sceneScale
				drawImageAtScale(screen, weaponIcon, weaponX+20*sceneScale, shipY-60*sceneScale, weaponScale)
			}

			// 绘制火箭炮发射器状态
			if s.Weapon.HasRocket {
				status := weaponImg.WeaponStatusReloading
				if s.Weapon.RocketDisabled {
					status = weaponImg.WeaponStatusDisabled
				} else if s.Weapon.RocketLauncherReloaded() {
					status = weaponImg.WeaponStatusLoaded
				}

				weaponIcon, weaponScale := weaponResource(weaponImg.WeaponTypeRocket, status, ms.UI.GameOpts.Zoom)
				weaponX += weaponInterstitialSpacing * sceneScale
				drawImageAtScale(screen, weaponIcon, weaponX+20*sceneScale, shipY-60*sceneScale, weaponScale)
			}

			// 如果被编组，需要标记出来
			if s.GroupID != object.GroupIDNone {
				textStr, fontSize := strconv.Itoa(int(s.GroupID)), float64(30)*sceneScale
				posX := shipX - 55*sceneScale
				posY := shipY - 85*sceneScale
				d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
			}
		}

		// 如果全局启用状态展示，则敌方战舰也要绘制 HP 值
		if ms.UI.GameOpts.ForceDisplayState && s.BelongPlayer != ms.Player.CurPlayer {
			sceneScale := ms.ZoomScale()
			hpImg := textureImg.GetEnemyHP(s.CurHP, s.TotalHP)
			drawImageAtScale(screen, hpImg, shipX-25*sceneScale, shipY-30*sceneScale, sceneScale)
		}

		// TODO 绘制战损情况，开火情况
	}
}

// 绘制消亡中的战舰
func (d *Drawer) drawDestroyedShips(screen *ebiten.Image, ms *state.MissionState) {
	for _, s := range ms.Arena.DestroyedShips {
		// 只有在屏幕中的才渲染
		if !ms.View.Camera.Contains(s.CurPos) {
			continue
		}

		sImg, sImgScale := shipResource(s.CurrentTopImageName(), ms.UI.GameOpts.Zoom)
		shipX, shipY := ms.CameraPosToScreen(s.CurPos)
		drawImageCentered(screen, sImg, shipX, shipY, s.CurRotation, sImgScale)

		// 绘制爆炸效果
		explodeImg := textureImg.GetShipExplode(s.CurHP)
		drawImageCentered(screen, explodeImg, shipX, shipY-30*ms.ZoomScale(), s.CurRotation, ms.ZoomScale())
	}
}

// 绘制飞机
func (d *Drawer) drawFlyingPlanes(screen *ebiten.Image, ms *state.MissionState) {
	// 飞机排序，确保渲染顺序是一致的（否则重叠会出现问题）
	planes := lo.Values(ms.Arena.Planes)
	slices.SortFunc(planes, func(a, b *objUnit.Plane) int {
		return strings.Compare(a.Uid, b.Uid)
	})

	for _, p := range planes {
		// 只有在屏幕中的才渲染
		if !ms.View.Camera.Contains(p.CurPos) {
			continue
		}

		pImg, pImgScale := planeResource(p.Name, ms.UI.GameOpts.Zoom)
		pImgScale *= p.VisualScaleMultiplier()
		planeX, planeY := ms.CameraPosToScreen(p.CurPos)
		drawImageCentered(screen, pImg, planeX, planeY, p.CurRotation, pImgScale)
		if ms.UI.DebugFlags.ShowHitBoxes {
			drawUnitHitBox(screen, ms, p)
		}

		// DEBUG: 如果启用了调试显示飞机 HP，则在飞机上部显示生命值
		if ms.UI.DebugFlags.ShowPlaneHP {
			// 格式：当前 HP / 总 HP（例如：85.0/100.0）
			hpText := fmt.Sprintf("%.1f/%.1f", p.CurHP, p.TotalHP)
			fontSize := float64(14) * ms.ZoomScale()
			// 计算文本位置（飞机图片上方居中）
			textWidth := float64(len(hpText)) * fontSize * 0.6 // 估算文本宽度
			posX := planeX - textWidth/2
			posY := planeY - float64(pImg.Bounds().Dy())*pImgScale/2 - 20*ms.ZoomScale()
			// 根据阵营选择颜色：友军绿色，敌军红色
			textColor := colorx.Green
			if p.BelongPlayer != ms.Player.CurPlayer {
				textColor = colorx.Red
			}
			d.drawText(screen, hpText, posX, posY, fontSize, font.Hang, textColor)
		}
	}
}

// 绘制消亡中的战机
func (d *Drawer) drawDestroyedPlanes(screen *ebiten.Image, ms *state.MissionState) {
	for _, p := range ms.Arena.DestroyedPlanes {
		// 只有在屏幕中的才渲染
		if !ms.View.Camera.Contains(p.CurPos) {
			continue
		}

		pImg, pImgScale := planeResource(p.Name, ms.UI.GameOpts.Zoom)
		planeX, planeY := ms.CameraPosToScreen(p.CurPos)
		drawImageCentered(screen, pImg, planeX, planeY, p.CurRotation, pImgScale)

		// 绘制爆炸效果，还会额外添加火焰+黑烟尾流来表现坠落拉烟效果
		explodeImg := textureImg.GetPlaneExplode(p.CurHP)
		drawImageCentered(screen, explodeImg, planeX, planeY, p.CurRotation, ms.ZoomScale())
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
	for _, b := range ms.Arena.ForwardingBullets {
		img := objBullet.GetImg(b.Type, b.Diameter)

		rotation := b.Rotation
		offsetX, offsetY := 0.0, 0.0
		// 激光弹丸：图片以中心旋转绘制，弹丸位置对应图片中心，
		// 导致一半光束向前（射击方向）、一半光束向后（反方向光线）。
		// 修复：将中心向射击方向前移 h/2，使图片尾部对齐弹丸，
		// 光束仅向射击正方向延伸。
		if b.Type == objBullet.TypeLaser {
			halfH := float64(img.Bounds().Dy()) * ms.ZoomScale() / 2.0
			offsetX = halfH * math.Sin(rotation*math.Pi/180)
			offsetY = -halfH * math.Cos(rotation*math.Pi/180)
		}
		x, y := ms.CameraPosToScreen(b.CurPos)
		drawImageCentered(screen, img, x+offsetX, y+offsetY, rotation, ms.ZoomScale())
	}
}
