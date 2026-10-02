package state

import (
	"math"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// FactionVision 一个玩家的已探索和当前可见格子。
// 关闭迷雾时不分配。格子按行主序，下标是 my*Width+mx。
type FactionVision struct {
	Width    int
	Height   int
	Explored []byte
	Visible  []byte
	// Light 是 4 倍分辨率的照亮强度，重叠取最大。Shade 是可直接上传的 RGBA。
	Light []byte
	Shade []byte
	// ExploredLight 是同样 4 倍分辨率的「曾经照亮」强度，只增不减。
	// 蒙层用它区分看过和没看过，边界跟照亮圆一样是渲染分辨率上的渐变，
	// 不会像按格读 Explored 那样留下整格的阶梯。
	ExploredLight []byte
	// RendersShade 这一侧的视野是否要生成迷雾蒙层。
	// 只有当前玩家的蒙层会被绘制；电脑侧只需要逻辑格，
	// 跳过 Light/ExploredLight/蒙层可以省掉每拍数毫秒的白算。
	RendersShade bool
	// frameSeq 本侧视野的拍计数，用于蒙层像素的降频合成。
	frameSeq int64
	// Contacts 离开可见后仍记住的敌舰位置。
	Contacts map[string]Contact
}

// NewFactionVision 按地图尺寸分配两张字节图。
// 蒙层默认渲染；电脑侧视野由 allocateVisions 按玩家关掉。
func NewFactionVision(width, height int) *FactionVision {
	n := width * height
	return &FactionVision{
		Width:        width,
		Height:       height,
		Explored:     make([]byte, n),
		Visible:      make([]byte, n),
		RendersShade: true,
	}
}

// ClearVisible 每拍盖圆之前清掉当前可见。已探索不动。
func (v *FactionVision) ClearVisible() {
	if v == nil {
		return
	}
	clear(v.Visible)
}

// CommitExplored 把这一拍可见并进已探索。
func (v *FactionVision) CommitExplored() {
	if v == nil {
		return
	}
	for i, lit := range v.Visible {
		if lit != 0 {
			v.Explored[i] = 1
		}
	}
}

// VisibleAt 格子当前是否可见。越界视为不可见。
func (v *FactionVision) VisibleAt(mx, my int) bool {
	if v == nil || mx < 0 || my < 0 || mx >= v.Width || my >= v.Height {
		return false
	}
	return v.Visible[my*v.Width+mx] != 0
}

// ExploredAt 格子是否曾经可见。
func (v *FactionVision) ExploredAt(mx, my int) bool {
	if v == nil || mx < 0 || my < 0 || mx >= v.Width || my >= v.Height {
		return false
	}
	return v.Explored[my*v.Width+mx] != 0
}

const (
	visionRenderScale = 4
	// shadeComposeInterval 蒙层像素的合成降频（拍）。蒙层是缓慢变化的软边渐变，
	// 30Hz 的刷新肉眼无感，能把每拍全图合成与 WritePixels 的上传开销减半。
	shadeComposeInterval = 2
)

func (v *FactionVision) composeShadeIfDue() {
	if !v.RendersShade {
		return
	}
	// 首拍立即合成，之后按间隔降频。
	v.frameSeq++
	if v.frameSeq != 1 && v.frameSeq%shadeComposeInterval != 0 {
		return
	}
	v.composeShade()
}

func (v *FactionVision) renderSize() (int, int) {
	return v.Width * visionRenderScale, v.Height * visionRenderScale
}

// BeginFrame 清掉这一拍的可见格子和软边亮度。已探索保留。
// 不渲染蒙层的一侧不清 Light（它根本不会被盖章）。
func (v *FactionVision) BeginFrame() {
	if v == nil {
		return
	}
	v.ClearVisible()
	if !v.RendersShade {
		return
	}
	rw, rh := v.renderSize()
	n := rw * rh
	if len(v.Light) != n {
		v.Light = make([]byte, n)
		v.Shade = make([]byte, n*4)
		// 已探索亮度是累积的，只在尺寸变化时重建，之后不再清空。
		v.ExploredLight = make([]byte, n)
	} else {
		clear(v.Light)
	}
}

// FinishFrame 合并已探索，并按降频节奏生成蒙层像素。
func (v *FactionVision) FinishFrame() {
	if v == nil {
		return
	}
	v.CommitExplored()
	v.composeShadeIfDue()
}

// StampLight 往渲染图上盖一张预先算好的圆形衰减，重叠取最大。
// 不渲染蒙层的一侧直接跳过，省掉每拍数毫秒的圆盘合并。
func (v *FactionVision) StampLight(rx, ry, radius float64) {
	if v == nil || !v.RendersShade || radius <= 0 || len(v.Light) == 0 {
		return
	}
	outer := int(math.Ceil(radius * visionRenderScale))
	if outer < 1 {
		outer = 1
	}
	stamp := softVisionCircle(outer)
	side := 2*outer + 1
	rw, rh := v.renderSize()
	originX := int(math.Round(rx*visionRenderScale)) - outer
	originY := int(math.Round(ry*visionRenderScale)) - outer
	for y := 0; y < side; y++ {
		py := originY + y
		if py < 0 || py >= rh {
			continue
		}
		row := py * rw
		stampRow := y * side
		for x := 0; x < side; x++ {
			px := originX + x
			if px < 0 || px >= rw {
				continue
			}
			if a := stamp[stampRow+x]; a > v.Light[row+px] {
				v.Light[row+px] = a
				// Light 每一拍清空，ExploredLight 只增不减，所以这里顺手取最大就够。
				if a > v.ExploredLight[row+px] {
					v.ExploredLight[row+px] = a
				}
			}
		}
	}
}

// 蒙层的两个基准暗度：看过但这一拍看不见，和从没看过。
const (
	shadeExplored = 140
	shadeUnknown  = 220
)

func (v *FactionVision) composeShade() {
	for i := range v.Light {
		// 已探索程度按渲染分辨率渐变，未知海面的边界因此不会整格跳变；
		// 再用这一拍的照亮强度把看得见的部分挖空。两者都平滑，接缝处就不会
		// 出现一圈比周围更黑的硬边。
		base := shadeUnknown - (shadeUnknown-shadeExplored)*int(v.ExploredLight[i])/255
		alpha := base * (255 - int(v.Light[i])) / 255
		o := i * 4
		v.Shade[o] = 0
		v.Shade[o+1] = 0
		v.Shade[o+2] = 0
		v.Shade[o+3] = byte(alpha)
	}
}

var softVisionCircles = map[int][]byte{}

func softVisionCircle(outer int) []byte {
	if stamp, ok := softVisionCircles[outer]; ok {
		return stamp
	}
	inner := outer - visionRenderScale
	if inner < 0 {
		inner = 0
	}
	side := 2*outer + 1
	stamp := make([]byte, side*side)
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			dist := math.Hypot(float64(x-outer), float64(y-outer))
			alpha := 0.0
			if dist <= float64(inner) {
				alpha = 255
			} else if inner < outer && dist < float64(outer) {
				t := (dist - float64(inner)) / float64(outer-inner)
				t = t * t * (3 - 2*t)
				alpha = 255 * (1 - t)
			}
			stamp[y*side+x] = byte(alpha)
		}
	}
	softVisionCircles[outer] = stamp
	return stamp
}

// UnexploredCentroid 返回还没探索过的海面重心，给舰队一个「往哪推」的方向。
// 全图探明时返回 false。
func (v *FactionVision) UnexploredCentroid() (objPos.MapPos, bool) {
	if v == nil {
		return objPos.MapPos{}, false
	}
	sumX, sumY, count := 0, 0, 0
	for y := 0; y < v.Height; y++ {
		for x := 0; x < v.Width; x++ {
			if v.ExploredAt(x, y) {
				continue
			}
			sumX += x
			sumY += y
			count++
		}
	}
	if count == 0 {
		return objPos.MapPos{}, false
	}
	return objPos.New(sumX/count, sumY/count), true
}

// BestUncover 选一个前线格：新揭开的格子多，又离起点近。
func (v *FactionVision) BestUncover(from objPos.MapPos, sight float64) (objPos.MapPos, bool) {
	if v == nil || sight <= 0 {
		return objPos.MapPos{}, false
	}
	bestScore := 0.0
	var best objPos.MapPos
	found := false
	for y := 0; y < v.Height; y++ {
		for x := 0; x < v.Width; x++ {
			if !v.ExploredAt(x, y) || !hasDarkNeighbor(v, x, y) {
				continue
			}
			// 分数是这一圈还能新揭开的格子数，再除以到起点的距离，近的前线优先。
			fresh := v.countUnexploredNear(x, y, sight)
			if fresh == 0 {
				continue
			}
			dist := math.Hypot(float64(x)-from.RX, float64(y)-from.RY)
			if dist < 1 {
				dist = 1
			}
			score := float64(fresh) / dist
			if !found || score > bestScore {
				bestScore = score
				best = objPos.New(x, y)
				found = true
			}
		}
	}
	return best, found
}

// FarthestExplored 选离自己最远、且不在其他侦察机视距里的已探索格。
func (v *FactionVision) FarthestExplored(from objPos.MapPos, blocked []objPos.MapPos) (objPos.MapPos, bool) {
	if v == nil {
		return objPos.MapPos{}, false
	}
	bestDist := -1.0
	var best objPos.MapPos
	found := false
	for y := 0; y < v.Height; y++ {
		for x := 0; x < v.Width; x++ {
			if !v.ExploredAt(x, y) || coveredByScout(x, y, blocked) {
				continue
			}
			dist := math.Hypot(float64(x)-from.RX, float64(y)-from.RY)
			if dist > bestDist {
				bestDist = dist
				best = objPos.New(x, y)
				found = true
			}
		}
	}
	return best, found
}

func hasDarkNeighbor(v *FactionVision, x, y int) bool {
	for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= v.Width || ny >= v.Height {
			continue
		}
		if !v.ExploredAt(nx, ny) {
			return true
		}
	}
	return false
}

func (v *FactionVision) countUnexploredNear(x, y int, sight float64) int {
	radius := int(math.Ceil(sight))
	count := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || ny < 0 || nx >= v.Width || ny >= v.Height {
				continue
			}
			if !v.ExploredAt(nx, ny) {
				count++
			}
		}
	}
	return count
}

func coveredByScout(x, y int, blocked []objPos.MapPos) bool {
	for _, pos := range blocked {
		dx := float64(x) - pos.RX
		dy := float64(y) - pos.RY
		if dx*dx+dy*dy <= 36*36 {
			return true
		}
	}
	return false
}

// Contact 一艘敌舰离开可见之后暂时记住的位置。
type Contact struct {
	RX, RY     float64
	ExpireTick int64
}

// SeenBy 该玩家现在能不能看见这个格子。没开迷雾时整张图都算看见。
func (s *MissionState) SeenBy(player faction.Player, mx, my int) bool {
	if !s.UsesFog(player) {
		return true
	}
	vision := s.Player.Visions[player]
	if vision == nil {
		return true
	}
	return vision.VisibleAt(mx, my)
}

// ConcealsEnemy 当前玩家的迷雾是否挡住这个格子上的敌方单位。己方不挡。
func (s *MissionState) ConcealsEnemy(owner faction.Player, mx, my int) bool {
	if owner == s.Player.CurPlayer {
		return false
	}
	return !s.SeenBy(s.Player.CurPlayer, mx, my)
}

// Stamp 把圆心 (rx, ry)、半径 radius 内的格子标为可见。
// 用格子中心是否落在半径内，避免只看格子角点时圆被咬缺。
func (v *FactionVision) Stamp(rx, ry, radius float64) {
	if v == nil || radius <= 0 {
		return
	}
	// 只扫圆的外接矩形，再看格子中心是否落在半径里，避免把圆外的角点也点亮。
	radiusSq := radius * radius
	minX := int(math.Floor(rx - radius))
	maxX := int(math.Floor(rx + radius))
	minY := int(math.Floor(ry - radius))
	maxY := int(math.Floor(ry + radius))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= v.Width {
		maxX = v.Width - 1
	}
	if maxY >= v.Height {
		maxY = v.Height - 1
	}
	for my := minY; my <= maxY; my++ {
		cy := float64(my) + 0.5
		dy := cy - ry
		row := my * v.Width
		for mx := minX; mx <= maxX; mx++ {
			cx := float64(mx) + 0.5
			dx := cx - rx
			if dx*dx+dy*dy <= radiusSq {
				v.Visible[row+mx] = 1
			}
		}
	}
}
