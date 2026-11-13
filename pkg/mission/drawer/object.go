package drawer

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	diveImg "github.com/narasux/jutland/pkg/resources/images/dive"
	planeImg "github.com/narasux/jutland/pkg/resources/images/plane"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	weaponImg "github.com/narasux/jutland/pkg/resources/images/weapon"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// weaponIconBaseSize 武器图标在 1x（zoom=4）下的显示边长，
// 与 resources/images/weapons 的 16px 资源档一致，用于计算整行图标宽度
const weaponIconBaseSize = 16.0

const (
	// submarineSubmergedAlpha 完全下潜后的舰体透明度
	submarineSubmergedAlpha = 0.55
	// submarineSubmergedScale 完全下潜后的舰体缩放，配合淡出表现下沉
	submarineSubmergedScale = 0.86
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

// diveResource 返回潜航状态图标、水下青色图标、过渡金色图标和残余绘制缩放比例。
func diveResource(
	diveState diveImg.State, sceneZoom int,
) (icon, underwater, transition *ebiten.Image, scale float64) {
	target := float64(state.NormalizeZoom(sceneZoom)) / float64(state.DefaultZoom())
	resourceZoom := closestResourceZoom(target, []int{1, 2, 4})
	return diveImg.Get(diveState, resourceZoom),
		diveImg.GetUnderwater(diveState, resourceZoom),
		diveImg.GetTransition(diveState, resourceZoom),
		target / float64(resourceZoom)
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
		clr := color.RGBA{R: 255, A: 80}
		if battleUnit.Player() == ms.Player.CurPlayer {
			clr = color.RGBA{G: 255, A: 80}
		}
		vector.StrokeLine(
			screen,
			float32(corner[0]), float32(corner[1]),
			float32(nextCorner[0]), float32(nextCorner[1]),
			1, clr, false,
		)
	}
}

// drawSubmarineRevealBox 调试秘籍：用红色舰体框标注敌方潜艇，只做表现。
func drawSubmarineRevealBox(screen *ebiten.Image, ms *state.MissionState, ship *objUnit.BattleShip) {
	x, y := ms.CameraPosToScreen(ship.CurPos)
	blockSize := ms.MapBlockDisplaySize()
	corners := rotatedRectangleCorners(
		x, y,
		ship.Length/constants.MapBlockSize*blockSize,
		ship.Width/constants.MapBlockSize*blockSize,
		ship.CurRotation,
	)
	clr := color.RGBA{R: 255, G: 64, B: 64, A: 230}
	for idx, corner := range corners {
		next := corners[(idx+1)%len(corners)]
		vector.StrokeLine(
			screen,
			float32(corner[0]), float32(corner[1]),
			float32(next[0]), float32(next[1]),
			2, clr, false,
		)
	}
}

// submarineDepthVisual 按潜航进度返回舰体透明度与缩放：0=水面，1=完全水下。
func submarineDepthVisual(ship *objUnit.BattleShip) (alpha, scale float64) {
	progress := ship.DiveProgress()
	return 1 - (1-submarineSubmergedAlpha)*progress, 1 - (1-submarineSubmergedScale)*progress
}

// diveBadgeLayout 潜航状态图标、过渡进度与尺寸；
// 绘制和武器图标行居中共用同一份结果，避免两处算法漂移。
type diveBadgeLayout struct {
	visible  bool
	state    diveImg.State
	width    float64
	scale    float64
	progress float64
}

// diveStateIcon 把潜艇深度状态映射到对应图标。
func diveStateIcon(diveState objUnit.DiveState) diveImg.State {
	switch diveState {
	case objUnit.DiveStateDiving:
		return diveImg.Diving
	case objUnit.DiveStateSubmerged:
		return diveImg.Submerged
	case objUnit.DiveStateSurfacing:
		return diveImg.Surfacing
	default:
		return diveImg.Surface
	}
}

// newDiveBadgeLayout 为潜艇生成潜航状态图标；非潜艇不显示。
// 水面状态也保留图标，供选中/常显状态行使用。
func newDiveBadgeLayout(ms *state.MissionState, ship *objUnit.BattleShip) diveBadgeLayout {
	if ship == nil || ship.Type != objUnit.ShipTypeSubmarine {
		return diveBadgeLayout{}
	}
	scale := ms.ZoomScale()
	return diveBadgeLayout{
		visible: true,
		state:   diveStateIcon(ship.DiveState),
		// 与武器图标同尺寸，方便并排放在同一条状态行里
		width:    weaponIconBaseSize * scale,
		scale:    scale,
		progress: ship.DiveProgress(),
	}
}

// drawDiveBadge 绘制潜航状态图标：
// 下潜时金色底图被青色从下往上覆盖，上浮时金色底图被绿色从上往下覆盖；
// 水下稳定态为青色，水面稳定态为绿色。
func drawDiveBadge(screen *ebiten.Image, layout diveBadgeLayout, x, y float64, sceneZoom int) {
	if !layout.visible {
		return
	}
	icon, underwater, transition, iconScale := diveResource(layout.state, sceneZoom)
	switch layout.state {
	case diveImg.Submerged:
		drawImageAtScale(screen, underwater, x, y, iconScale)
	case diveImg.Diving:
		// 入水：青色从底部往上覆盖整个金色图标
		drawImageAtScale(screen, transition, x, y, iconScale)
		drawDiveOverlayBottom(screen, underwater, layout.progress, x, y, iconScale)
	case diveImg.Surfacing:
		// 出水：绿色从顶部往下覆盖整个金色图标
		drawImageAtScale(screen, transition, x, y, iconScale)
		drawDiveOverlayTop(screen, icon, 1-layout.progress, x, y, iconScale)
	default:
		drawImageAtScale(screen, icon, x, y, iconScale)
	}
}

// drawDiveOverlayBottom 把覆盖层底部按进度高度贴到图标底部，形成从下往上的填充。
func drawDiveOverlayBottom(
	screen, overlay *ebiten.Image, progress, x, y, scale float64,
) {
	if overlay == nil {
		return
	}
	progress = min(1, max(0, progress))
	bounds := overlay.Bounds()
	height := bounds.Dy()
	fillRows := int(math.Ceil(progress * float64(height)))
	if fillRows <= 0 {
		return
	}
	fillRows = min(fillRows, height)

	sub := overlay.SubImage(image.Rect(
		bounds.Min.X, bounds.Min.Y+height-fillRows, bounds.Max.X, bounds.Max.Y,
	)).(*ebiten.Image)
	drawImageAtScale(screen, sub, x, y+float64(height-fillRows)*scale, scale)
}

// drawDiveOverlayTop 把覆盖层顶部按进度高度贴到图标顶部，形成从上往下的填充。
func drawDiveOverlayTop(
	screen, overlay *ebiten.Image, progress, x, y, scale float64,
) {
	if overlay == nil {
		return
	}
	progress = min(1, max(0, progress))
	bounds := overlay.Bounds()
	height := bounds.Dy()
	fillRows := int(math.Ceil(progress * float64(height)))
	if fillRows <= 0 {
		return
	}
	fillRows = min(fillRows, height)

	sub := overlay.SubImage(image.Rect(
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+fillRows,
	)).(*ebiten.Image)
	drawImageAtScale(screen, sub, x, y, scale)
}

// drawSonarRanges 绘制选中己方舰船的声呐/水听器半径（仅表现，不参与判定）。
func (d *Drawer) drawSonarRanges(screen *ebiten.Image, ms *state.MissionState) {
	selected := selectedShipSet(ms.Interaction.SelectedShips)
	// 单选时才标注名称与格数；多选时逐舰标注会互相遮挡，也增加每帧文字测量开销
	showLabel := len(selected) == 1
	for _, ship := range ms.Arena.OrderedShips() {
		if ship.CurHP <= 0 || !ms.View.Camera.Contains(ship.CurPos) {
			continue
		}
		if _, ok := selected[ship.Uid]; !ok {
			continue
		}
		radius := ship.SonarRange()
		if radius <= 0 {
			continue
		}
		x, y := ms.CameraPosToScreen(ship.CurPos)
		radiusPx := radius * ms.MapBlockDisplaySize()
		vector.StrokeCircle(
			screen, float32(x), float32(y),
			float32(radiusPx), 1, colorx.LightGreen, false,
		)
		// 在舰船下方标注「声呐 + 格数」，避免被误认为攻击或治疗范围。
		// 声呐圈在默认缩放下常常大于屏幕，贴在圈边会跑到屏幕外，所以跟随舰船。
		if showLabel {
			halfLen := ship.Length / constants.MapBlockSize * ms.MapBlockDisplaySize() / 2
			labelY := y + max(halfLen, 18*ms.ZoomScale()) + 4*ms.ZoomScale()
			d.drawSonarLabel(screen, ms, x, labelY, radius)
		}
	}
}

// drawSonarLabel 在指定位置绘制声呐名称与半径；文字贴到屏幕下沿时上移，避免看不见。
func (d *Drawer) drawSonarLabel(
	screen *ebiten.Image, ms *state.MissionState, centerX, labelY, radius float64,
) {
	scale := ms.ZoomScale()
	fontSize := 11 * scale
	face := font.LocalizedUI(font.Kai)
	label := fmt.Sprintf("%s %.0f", i18n.Text(i18n.MsgUnitPanelSonar), radius)
	textFace := text.GoTextFace{Source: face, Size: fontSize}
	textW, textH := text.Measure(label, &textFace, 0)
	x := centerX - textW/2
	y := labelY
	if limit := float64(ms.View.Layout.Height) - textH - 2*scale; limit > 0 {
		y = min(y, limit)
	}
	clr := color.RGBA{R: 160, G: 235, B: 160, A: 230}
	d.drawText(screen, label, x+scale, y+scale, fontSize, face, color.RGBA{A: 180})
	d.drawText(screen, label, x, y, fontSize, face, clr)
}

// 绘制支援舰治疗范围圈（仅在选中己方医疗船或维修船时显示）
func (d *Drawer) drawHospitalShipHealRange(screen *ebiten.Image, ms *state.MissionState) {
	selected := selectedShipSet(ms.Interaction.SelectedShips)
	for _, ship := range ms.Arena.Ships {
		// 只绘制己方存活的医疗船或维修船
		if (ship.Type != objUnit.ShipTypeHospital && ship.Type != objUnit.ShipTypeRepair) ||
			ship.BelongPlayer != ms.Player.CurPlayer || ship.CurHP <= 0 {
			continue
		}
		// 检查是否被选中
		if _, ok := selected[ship.Uid]; !ok {
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

// trailHiddenByFog 判断这条尾流要不要被当前玩家的迷雾藏起来。
// 尾流不带归属（旧数据或测试造的对象）时照旧渲染。
func trailHiddenByFog(ms *state.MissionState, trail *objTrail.Trail) bool {
	if trail.BelongPlayer == "" {
		return false
	}
	// 潜艇下潜后连旧尾流也不能泄露位置；能看见本舰时才按迷雾规则处理。
	if owner, ok := ms.Arena.Ships[trail.OwnerUid]; ok {
		if owner.Submerged() || !ms.CanSeeShip(ms.Player.CurPlayer, owner) {
			return true
		}
	}
	return ms.ConcealsEnemy(trail.BelongPlayer, trail.Pos.MX, trail.Pos.MY)
}

// 绘制尾流（战舰，鱼雷，炮弹）
func (d *Drawer) drawObjectTrails(screen *ebiten.Image, ms *state.MissionState) {
	for _, trail := range ms.Arena.Trails {
		// 只有在屏幕中，且不处于延迟/消亡的尾流才渲染
		if !(ms.View.Camera.Contains(trail.Pos) && trail.IsActive()) {
			continue
		}
		// 敌舰尾流同样受迷雾遮挡，否则顺着尾流就能看出敌方舰队在哪
		if trailHiddenByFog(ms, trail) {
			continue
		}

		trailImg := textureImg.TrailImage(trail.Shape)
		scale := textureImg.TrailDrawScale(trail.Shape, trail.CurSize) * ms.ZoomScale()
		x, y := ms.CameraPosToScreen(trail.Pos)
		red, green, blue, alpha := textureImg.TrailColorScale(trail.Color, trail.CurLife)
		drawImageCenteredColored(screen, trailImg, x, y, trail.Rotation, scale, red, green, blue, alpha)
	}
}

// drawExplosions 绘制舰船之下的局部爆炸效果（水花 / 尘土 / 火箭空爆）
func (d *Drawer) drawExplosions(screen *ebiten.Image, ms *state.MissionState) {
	d.drawExplosionLayer(screen, ms, false)
}

// drawExplosionsOverShips 绘制压在舰船之上的命中火球：命中点就在舰体上，
// 若与舰船同层会被舰体完全遮住。
func (d *Drawer) drawExplosionsOverShips(screen *ebiten.Image, ms *state.MissionState) {
	d.drawExplosionLayer(screen, ms, true)
}

// drawExplosionLayer 按种类把特效分到舰船下 / 舰船上两层绘制。
func (d *Drawer) drawExplosionLayer(screen *ebiten.Image, ms *state.MissionState, overShips bool) {
	for _, explosion := range ms.Arena.Explosions {
		if explosion.Kind.OverShips() != overShips {
			continue
		}
		if !ms.View.Camera.Contains(explosion.Pos) ||
			ms.ConcealsEnemy(explosion.BelongPlayer, explosion.Pos.MX, explosion.Pos.MY) {
			continue
		}
		x, y := ms.CameraPosToScreen(explosion.Pos)
		if explosion.Kind == objExplosion.KindRocketAirburst {
			frame := textureImg.MaxPlaneExplodeState - 1 - explosion.Frame()
			explodeImg := textureImg.GetPlaneExplode(float64(max(frame, 0)))
			scale := explosion.Size * constants.MapBlockSize / planeExplodeImageSize * ms.ZoomScale()
			drawImageCentered(screen, explodeImg, x, y, explosion.Rotation, scale)
			continue
		}
		drawEffectParticles(screen, explosion, x, y, ms.ZoomScale())
	}
}

// planeExplodeImageSize 飞机爆炸序列帧压到 1/4 后的边长。
const planeExplodeImageSize = 64

// 命中特效粒子：位置 = 中心 + 极坐标(angle, distance × 扩散进度)，尺寸与透明度随进度演化。
// 与尾流同一套做法（白色柔圆 + ColorScale 缩放），只是把"每 tick 发射"换成了
// "一次爆开的一组粒子"。
type effectParticle struct {
	angle float64
	// distance 相对特效直径的比例（扩散到位时）
	distance float64
	// size 相对特效直径的比例
	size float64
	// sizeGrow 尺寸随进度增减的比例（负值表示收缩）
	sizeGrow float64
	// alpha 峰值透明度
	alpha float64
	// alphaPow 衰减指数
	alphaPow float64
	// rise 为 true 时先涨后落（烟），否则从 0 时刻就按衰减曲线走（水花 / 尘土 / 火星）
	rise bool
	// delay 进度小于该值时不显示（用于"先闪后烟"）
	delay float64
	// lean 沿弹道反方向的偏移（相对特效直径的比例）：烟往弹道后方拖，轮廓不再居中对称
	lean float64
	r    float32
	g    float32
	b    float32
}

// maxEffectParticles 单次命中特效的粒子上限。
const maxEffectParticles = 24

// effectRand 由种子与索引派生的确定性伪随机（0..1）。
// 刻意不用全局 rand：命中特效不能挪动战斗随机序列，否则散布与暴击结果会变。
func effectRand(seed uint32, index, salt uint32) float64 {
	value := seed*1664525 + index*1013904223 + salt*2654435761
	value ^= value >> 15
	value *= 2246822519
	value ^= value >> 13
	return float64(value%100000) / 100000
}

// effectParticles 按特效种类生成粒子（写入调用方提供的缓冲区，避免每帧分配）。
func effectParticles(e *objExplosion.Explosion, buf []effectParticle) []effectParticle {
	seed := e.Seed
	n := 0
	add := func(p effectParticle) {
		if n < len(buf) {
			buf[n] = p
			n++
		}
	}
	switch e.Kind {
	case objExplosion.KindSplash, objExplosion.KindTorpedoPlume:
		plume := e.Kind == objExplosion.KindTorpedoPlume
		foam := 9
		spread, alphaScale := 0.45, 1.0
		if plume {
			foam, spread, alphaScale = 12, 0.6, 1.25
		}
		// 中心泡沫：水花的心，压暗压小
		add(effectParticle{size: 0.34, sizeGrow: 0.4, alpha: 0.4 * alphaScale, alphaPow: 1.2, r: 0.88, g: 0.94, b: 1})
		// 周围的泡沫团：贴着中心散开，不再往外甩"雪花"
		for i := 0; i < foam; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 1) * 2 * math.Pi,
				distance: 0.12 + spread*effectRand(seed, uint32(i), 2),
				size:     0.2 + 0.22*effectRand(seed, uint32(i), 3),
				sizeGrow: 0.25 + 0.4*effectRand(seed, uint32(i), 4),
				alpha:    (0.18 + 0.2*effectRand(seed, uint32(i), 5)) * alphaScale,
				alphaPow: 1.1,
				r:        0.8, g: 0.9, b: 1,
			})
		}
	case objExplosion.KindDust:
		for i := 0; i < 10; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 21) * 2 * math.Pi,
				distance: 0.1 + 0.5*effectRand(seed, uint32(i), 22),
				size:     0.28 + 0.42*effectRand(seed, uint32(i), 23),
				sizeGrow: 0.5 + 0.6*effectRand(seed, uint32(i), 24),
				alpha:    0.2 + 0.28*effectRand(seed, uint32(i), 25),
				alphaPow: 0.9,
				rise:     true,
				r:        0.82, g: 0.72, b: 0.56,
			})
		}
		for i := 0; i < 5; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 31) * 2 * math.Pi,
				distance: 0.3 + 0.7*effectRand(seed, uint32(i), 32),
				size:     0.09 + 0.1*effectRand(seed, uint32(i), 33),
				sizeGrow: -0.02,
				alpha:    0.5 + 0.4*effectRand(seed, uint32(i), 34),
				alphaPow: 1.6,
				r:        0.55, g: 0.46, b: 0.33,
			})
		}
	default: // KindFlash：命中爆炸 = 短促亮核 + 破片火星 + 拖尾硝烟
		// 硝烟是主体：颜色压暗（比海面还暗），轮廓靠随机距离拉开，不能是同心圆
		for i := 0; i < 9; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 41) * 2 * math.Pi,
				distance: 0.2 + 0.9*effectRand(seed, uint32(i), 42),
				size:     0.28 + 0.5*effectRand(seed, uint32(i), 43),
				sizeGrow: 0.7 + 1.0*effectRand(seed, uint32(i), 44),
				alpha:    0.3 + 0.3*effectRand(seed, uint32(i), 45),
				alphaPow: 0.7,
				rise:     true,
				lean:     0.1 + 0.35*effectRand(seed, uint32(i), 46),
				r:        0.2, g: 0.2, b: 0.21,
			})
		}
		// 亮核：一闪而过，不留光晕
		for i := 0; i < 2; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 51) * 2 * math.Pi,
				distance: 0.08 * effectRand(seed, uint32(i), 52),
				size:     0.45 + 0.3*effectRand(seed, uint32(i), 53),
				sizeGrow: 0.5,
				alpha:    0.95,
				alphaPow: 3.2,
				r:        1, g: 0.84, b: 0.56,
			})
		}
		// 白热中心：只有最初两帧
		add(effectParticle{size: 0.22, sizeGrow: 0.3, alpha: 0.9, alphaPow: 3.6, r: 1, g: 0.96, b: 0.86})
		// 破片火星：贴着亮核飞一小段就消失（飞太远会变成一圈小点，像撒了一把火星）
		for i := 0; i < 8; i++ {
			add(effectParticle{
				angle:    effectRand(seed, uint32(i), 61) * 2 * math.Pi,
				distance: 0.25 + 0.5*effectRand(seed, uint32(i), 62),
				size:     0.07 + 0.08*effectRand(seed, uint32(i), 63),
				sizeGrow: -0.03,
				alpha:    0.65 + 0.35*effectRand(seed, uint32(i), 64),
				alphaPow: 3.2,
				r:        1, g: 0.72, b: 0.32,
			})
		}
	}
	return buf[:n]
}

// drawEffectParticles 按当前进度绘制一次命中特效的所有粒子。
func drawEffectParticles(screen *ebiten.Image, e *objExplosion.Explosion, x, y, zoomScale float64) {
	t := float64(e.Age) / float64(e.Life)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	sizePx := e.Size * constants.MapBlockSize * zoomScale
	if sizePx <= 0 {
		return
	}
	circle := textureImg.EffectCircle()
	imgScale := 1 / float64(textureImg.EffectCircleDiameter)

	var buf [maxEffectParticles]effectParticle
	for _, p := range effectParticles(e, buf[:]) {
		if t < p.delay {
			continue
		}
		env := 1.0
		if p.alphaPow > 0 {
			env = math.Pow(1-t, p.alphaPow)
		}
		if p.rise {
			env *= math.Min(1, t/0.15)
		}
		alpha := p.alpha * env
		if alpha <= 0.004 {
			continue
		}
		// 扩散：从中心向外铺开，进度越大越远越大
		dist := p.distance * sizePx * 0.5 * (0.35 + 0.65*t)
		particleSize := p.size * sizePx * (1 + p.sizeGrow*t)
		if particleSize <= 0 {
			continue
		}
		px := x + math.Cos(p.angle)*dist
		py := y + math.Sin(p.angle)*dist
		if p.lean > 0 {
			// 弹道方向是 (sin, -cos)；烟往反方向拖
			rad := e.Rotation * math.Pi / 180
			back := p.lean * sizePx * math.Min(1, t/0.5)
			px -= math.Sin(rad) * back
			py += math.Cos(rad) * back
		}
		// ColorScale 需要预乘 alpha，和尾流一致
		drawImageCenteredColored(
			screen, circle, px, py, 0, particleSize*imgScale,
			p.r*float32(alpha), p.g*float32(alpha), p.b*float32(alpha), float32(alpha),
		)
	}
}

// planeIconStatus 决定舰载机图标颜色：
// 绿=甲板有备机可以起飞，黄=甲板已空但飞机仍在空中/返航，灰=起飞被关闭或已无机可用。
func planeIconStatus(aircraft *objUnit.ShipAircraft, airborne int) weaponImg.WeaponStatus {
	if aircraft.Disable {
		return weaponImg.WeaponStatusDisabled
	}
	if aircraft.StandbyCount() > 0 {
		return weaponImg.WeaponStatusLoaded
	}
	if airborne > 0 {
		return weaponImg.WeaponStatusReloading
	}
	return weaponImg.WeaponStatusDisabled
}

// airbornePlaneCounts 统计每艘战舰当前仍在空中的飞机数量。
// 飞机回收入库后会从 Arena.Planes 移除，因此集合里的飞机都属于「已离舰」状态。
func airbornePlaneCounts(planes map[string]*objUnit.Plane) map[string]int {
	counts := make(map[string]int, len(planes))
	for _, plane := range planes {
		if plane == nil {
			continue
		}
		counts[plane.BelongShip]++
	}
	return counts
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	selected := selectedShipSet(ms.Interaction.SelectedShips)
	// 各舰仍在空中的飞机数（含出击、作战与返航）一次性统计，避免逐舰遍历飞机表
	airbornePlaneCnt := airbornePlaneCounts(ms.Arena.Planes)
	for _, s := range ms.Arena.OrderedShips() {
		if !ms.View.Camera.Contains(s.CurPos) {
			continue
		}
		visible := ms.CanSeeShip(ms.Player.CurPlayer, s)
		if !visible {
			// 调试秘籍：未探测到的敌方潜艇只画红框，不画舰体
			if ms.RevealsSubmarine(ms.Player.CurPlayer, s) {
				drawSubmarineRevealBox(screen, ms, s)
			}
			continue
		}

		sImg, sImgScale := shipResource(s.CurrentTopImageName(), ms.UI.GameOpts.Zoom)
		shipX, shipY := ms.CameraPosToScreen(s.CurPos)
		if ms.Player.RevealSubmarines && s.BelongPlayer != ms.Player.CurPlayer &&
			s.Type == objUnit.ShipTypeSubmarine {
			drawSubmarineRevealBox(screen, ms, s)
		}
		if s.BelongPlayer == ms.Player.CurPlayer && s.Type == objUnit.ShipTypeSubmarine {
			// 己方潜艇：随潜航进度淡出并轻微缩小，避免状态切换过于突兀
			alpha, depthScale := submarineDepthVisual(s)
			drawImageCenteredColored(
				screen, sImg, shipX, shipY, s.CurRotation, sImgScale*depthScale,
				1, 1, 1, float32(alpha),
			)
		} else {
			drawImageCentered(screen, sImg, shipX, shipY, s.CurRotation, sImgScale)
			// 被本方声呐点亮、但仍在水下的敌潜艇：加一圈接触提示
			if s.BelongPlayer != ms.Player.CurPlayer && s.Submerged() {
				vector.StrokeCircle(
					screen, float32(shipX), float32(shipY),
					float32(9*ms.ZoomScale()), 1.4, color.RGBA{R: 255, G: 96, B: 96, A: 220}, false,
				)
			}
		}
		if ms.UI.DebugFlags.ShowHitBoxes {
			drawUnitHitBox(screen, ms, s)
		}

		// 如果战舰被选中 或 全局启用状态展示，则需要绘制 HP、武器与潜航状态
		isShipSelected := false
		if _, ok := selected[s.Uid]; ok {
			isShipSelected = true
		}
		var diveBadge diveBadgeLayout
		if s.BelongPlayer == ms.Player.CurPlayer && s.Type == objUnit.ShipTypeSubmarine {
			diveBadge = newDiveBadgeLayout(ms, s)
		}
		showStatusRow := (ms.UI.GameOpts.ForceDisplayState || isShipSelected) &&
			s.BelongPlayer == ms.Player.CurPlayer
		if showStatusRow {
			sceneScale := ms.ZoomScale()
			// 绘制当前生命值
			hpImg := textureImg.GetHP(s.CurHP, s.TotalHP)
			drawImageAtScale(screen, hpImg, shipX-25*sceneScale, shipY-30*sceneScale, sceneScale)

			if isShipSelected {
				drawImageAtScale(screen, textureImg.ShipSelected, shipX-60*sceneScale, shipY-40*sceneScale, sceneScale)
			}

			// 渲染武器状态时候的 X 方向间隙大小
			var weaponInterstitialSpacing float64
			hasPlane := s.Aircraft.HasPlane
			weaponCnt := len(lo.Filter(
				[]bool{
					s.Weapon.HasMainGun,
					s.Weapon.HasSecondaryGun,
					s.Weapon.HasAntiAircraftGun,
					s.Weapon.HasTorpedo,
					s.Weapon.HasRocket,
					hasPlane,
				},
				func(b bool, _ int) bool {
					return b
				},
			))
			switch weaponCnt {
			case 6, 5:
				weaponInterstitialSpacing = 12.0
			case 4:
				weaponInterstitialSpacing = 15.0
			case 3:
				weaponInterstitialSpacing = 20.0
			default:
				weaponInterstitialSpacing = 35.0
			}
			// 武器图标与潜航状态图标共用一行，按原宽度整体居中。
			// 潜航图标占用原主炮位置，武器整体右移一格，给左上角编队号让出空间。
			iconStep := weaponInterstitialSpacing * sceneScale
			iconWidth := 0.0
			if weaponCnt > 0 {
				iconWidth = float64(weaponCnt-1)*iconStep + weaponIconBaseSize*sceneScale
			}
			rowWidth := iconWidth
			diveSlot := 0.0
			if diveBadge.visible {
				diveSlot = diveBadge.width + 6*sceneScale
				rowWidth += diveSlot
			}
			weaponX := shipX - rowWidth/2
			if diveBadge.visible {
				// 空出原潜航图标槽位，使图标落到主炮位置，武器再整体右移一格
				weaponX += diveSlot
				drawDiveBadge(screen, diveBadge, weaponX, shipY-60*sceneScale, ms.UI.GameOpts.Zoom)
				weaponX += diveSlot
			}

			// drawWeaponIcon 在游标处绘制一张图标，然后统一右移一个间距，
			// 保证武器图标在潜航图标之后按固定间距依次排列。
			drawWeaponIcon := func(weaponIcon *ebiten.Image, weaponScale float64) {
				drawImageAtScale(screen, weaponIcon, weaponX, shipY-60*sceneScale, weaponScale)
				weaponX += iconStep
			}

			// 绘制主炮状态
			if s.Weapon.HasMainGun {
				status := weaponImg.WeaponStatusReloading
				if s.Weapon.MainGunDisabled {
					status = weaponImg.WeaponStatusDisabled
				} else if s.Weapon.MainGunReloaded() {
					status = weaponImg.WeaponStatusLoaded
				}

				weaponIcon, weaponScale := weaponResource(weaponImg.WeaponTypeMainGun, status, ms.UI.GameOpts.Zoom)
				drawWeaponIcon(weaponIcon, weaponScale)
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
				drawWeaponIcon(weaponIcon, weaponScale)
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
				drawWeaponIcon(weaponIcon, weaponScale)
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
				drawWeaponIcon(weaponIcon, weaponScale)
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
				drawWeaponIcon(weaponIcon, weaponScale)
			}

			// 绘制舰载机状态：绿=甲板有备机，黄=甲板已空但飞机仍在空中/返航，
			// 灰=起飞被关闭或飞机已全部损失。不展示数量，精确数量交给航空面板。
			if hasPlane {
				status := planeIconStatus(&s.Aircraft, airbornePlaneCnt[s.Uid])
				weaponIcon, weaponScale := weaponResource(weaponImg.WeaponTypePlane, status, ms.UI.GameOpts.Zoom)
				drawWeaponIcon(weaponIcon, weaponScale)
			}

			// 如果被编组，需要标记出来
			if s.GroupID != object.GroupIDNone {
				textStr, fontSize := strconv.Itoa(int(s.GroupID)), float64(30)*sceneScale
				posX := shipX - 55*sceneScale
				posY := shipY - 85*sceneScale
				d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
			}
		}
		// 常显状态关闭时，仍给下潜/上浮中的己方潜艇保留状态图标，
		// 避免下潜后找不到自己的艇
		if !showStatusRow && diveBadge.visible && s.DiveState != objUnit.DiveStateSurface {
			drawDiveBadge(
				screen, diveBadge, shipX-diveBadge.width/2, shipY-44*ms.ZoomScale(),
				ms.UI.GameOpts.Zoom,
			)
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
		if !ms.View.Camera.Contains(s.CurPos) || !ms.CanSeeShip(ms.Player.CurPlayer, s) {
			continue
		}

		sImg, sImgScale := shipResource(s.CurrentTopImageName(), ms.UI.GameOpts.Zoom)
		shipX, shipY := ms.CameraPosToScreen(s.CurPos)
		if ms.Player.RevealSubmarines && s.BelongPlayer != ms.Player.CurPlayer &&
			s.Type == objUnit.ShipTypeSubmarine {
			drawSubmarineRevealBox(screen, ms, s)
		}
		if s.BelongPlayer == ms.Player.CurPlayer && s.Type == objUnit.ShipTypeSubmarine {
			// 己方潜艇：沿用潜航进度淡出与缩放
			alpha, depthScale := submarineDepthVisual(s)
			drawImageCenteredColored(
				screen, sImg, shipX, shipY, s.CurRotation, sImgScale*depthScale,
				1, 1, 1, float32(alpha),
			)
		} else {
			drawImageCentered(screen, sImg, shipX, shipY, s.CurRotation, sImgScale)
			// 被本方声呐点亮、但仍在水下的敌潜艇：加一圈接触提示
			if s.BelongPlayer != ms.Player.CurPlayer && s.Submerged() {
				vector.StrokeCircle(
					screen, float32(shipX), float32(shipY),
					float32(9*ms.ZoomScale()), 1.4, color.RGBA{R: 255, G: 96, B: 96, A: 220}, false,
				)
			}
		}

		// 绘制爆炸效果
		explodeImg := textureImg.GetShipExplode(s.CurHP)
		drawImageCentered(screen, explodeImg, shipX, shipY-30*ms.ZoomScale(), s.CurRotation, ms.ZoomScale())
	}
}

func selectedShipSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// 绘制飞机
func (d *Drawer) drawFlyingPlanes(screen *ebiten.Image, ms *state.MissionState) {
	for _, p := range ms.Arena.OrderedPlanes() {
		if !ms.View.Camera.Contains(p.CurPos) || ms.ConcealsEnemy(p.BelongPlayer, p.CurPos.MX, p.CurPos.MY) {
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
		if !ms.View.Camera.Contains(p.CurPos) || ms.ConcealsEnemy(p.BelongPlayer, p.CurPos.MX, p.CurPos.MY) {
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

const (
	// 炮弹贴图小于一格，外扩一格避免贴边被切掉。
	shotBulletViewMargin = 1
	// 激光贴图高 1000 像素，约 7.82 个地图格，沿射击方向整段伸出。
	laserViewMargin = 8
	// 伤害数字和地图标识都很小。
	markViewMargin = 1
)

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
	for _, b := range ms.Arena.ForwardingBullets {
		margin := shotBulletViewMargin
		if b.Type == objBullet.TypeLaser {
			margin = laserViewMargin
		}
		if !ms.View.Camera.ContainsMargin(b.CurPos, margin) ||
			ms.ConcealsEnemy(b.BelongPlayer, b.CurPos.MX, b.CurPos.MY) {
			continue
		}

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
