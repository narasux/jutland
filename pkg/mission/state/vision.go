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
	// Shade 是 4 倍分辨率、可直接上传的 RGBA 蒙层像素。
	// 它由格级的 Visible / Explored 并集一次性合成，不按视野源逐个盖圆。
	Shade []byte
	// ShadeStamp 每次重新合成蒙层时递增。绘制层记录已上传的版本号，
	// 同一份像素在主视图和小地图之间不会重复上传。
	ShadeStamp int64
	// RendersShade 这一侧的视野是否要生成迷雾蒙层。
	// 只有当前玩家的蒙层会被绘制；电脑侧只需要逻辑格，
	// 跳过蒙层合成可以省掉每拍的全图遍历与像素上传。
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

// BeginFrame 清掉这一拍的可见格子。已探索和蒙层缓冲保留。
// 不渲染蒙层的一侧不分配蒙层缓冲。
func (v *FactionVision) BeginFrame() {
	if v == nil {
		return
	}
	v.ClearVisible()
	if !v.RendersShade {
		return
	}
	rw, rh := v.renderSize()
	if len(v.Shade) != rw*rh*4 {
		v.Shade = make([]byte, rw*rh*4)
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

// 蒙层的两个基准暗度：看过但这一拍看不见，和从没看过。
const (
	shadeExplored = 140
	shadeUnknown  = 220
)

// shadeLUT 预计算「可见与已探索的 2×2 格组合 × 各渲染子像素」的最终暗度。
// 蒙层的每个渲染像素只取决于它所在格的四个角（本格与右、下、右下邻居），
// 角点状态是 0/1，因此整张蒙层可以退化成查表：不再需要给每个视野源
// 盖一张软边圆盘，开销与视野源数量无关。
// 下标依次是可见组合、已探索组合、子像素序号（行主序）。
var shadeLUT [16][16][16]byte

// 表尺寸写死了 visionRenderScale = 4 的 16 个子像素；改分辨率时这里会编译失败。
const _ = uint(16 - visionRenderScale*visionRenderScale)

func init() {
	for visible := 0; visible < 16; visible++ {
		for explored := 0; explored < 16; explored++ {
			for qy := 0; qy < visionRenderScale; qy++ {
				for qx := 0; qx < visionRenderScale; qx++ {
					tx := (float64(qx) + 0.5) / visionRenderScale
					ty := (float64(qy) + 0.5) / visionRenderScale
					// 已探索程度决定底色，这一拍的照亮把看得见的部分挖空；
					// 两者都在同一格内平滑过渡，接缝处不会出现更黑的硬边。
					seen := smoothStep01(bilinearCorners(explored, tx, ty))
					lit := smoothStep01(bilinearCorners(visible, tx, ty))
					base := float64(shadeUnknown) - float64(shadeUnknown-shadeExplored)*seen
					shadeLUT[visible][explored][qy*visionRenderScale+qx] = byte(base * (1 - lit))
				}
			}
		}
	}
}

// bilinearCorners 在 2×2 的 0/1 角点上做双线性插值。位序与 quadIndex 一致。
func bilinearCorners(bits int, tx, ty float64) float64 {
	top := float64(bits&1)*(1-tx) + float64((bits>>1)&1)*tx
	bottom := float64((bits>>2)&1)*(1-tx) + float64((bits>>3)&1)*tx
	return top*(1-ty) + bottom*ty
}

func smoothStep01(t float64) float64 {
	return t * t * (3 - 2*t)
}

// quadIndex 把格 (cx, cy) 与右、下、右下三个邻居的 0/1 状态打包成 4 位下标。
// 越界按边界格补齐，免得地图外缘多出一圈半亮的假边界。
func quadIndex(field []byte, width, height, cx, cy int) int {
	right, below := cx+1, cy+1
	if right >= width {
		right = width - 1
	}
	if below >= height {
		below = height - 1
	}
	index := 0
	if field[cy*width+cx] != 0 {
		index |= 1
	}
	if field[cy*width+right] != 0 {
		index |= 2
	}
	if field[below*width+cx] != 0 {
		index |= 4
	}
	if field[below*width+right] != 0 {
		index |= 8
	}
	return index
}

// composeShade 按格合成整张蒙层。可见与已探索都只取格级并集，
// 软边在同一格内统一生成，因此不再随视野源数量线性变慢。
func (v *FactionVision) composeShade() {
	rw, rh := v.renderSize()
	if len(v.Shade) != rw*rh*4 {
		v.Shade = make([]byte, rw*rh*4)
	}
	for cy := 0; cy < v.Height; cy++ {
		for cx := 0; cx < v.Width; cx++ {
			visible := quadIndex(v.Visible, v.Width, v.Height, cx, cy)
			explored := quadIndex(v.Explored, v.Width, v.Height, cx, cy)
			alphas := &shadeLUT[visible][explored]
			for qy := 0; qy < visionRenderScale; qy++ {
				offset := ((cy*visionRenderScale+qy)*rw + cx*visionRenderScale) * 4
				for qx := 0; qx < visionRenderScale; qx++ {
					v.Shade[offset], v.Shade[offset+1], v.Shade[offset+2] = 0, 0, 0
					v.Shade[offset+3] = alphas[qy*visionRenderScale+qx]
					offset += 4
				}
			}
		}
	}
	v.ShadeStamp++
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
