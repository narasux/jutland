package texture

import (
	"image"
	"image/color"
	"math"
)

// iconRaster 是小尺寸舰种图标的 CPU 栅格器，便于预览导出且不依赖游戏窗口。
type iconRaster struct {
	w, h int
	pix  []float32
	clip [][2]float32
}

func newIconRaster(w, h int) *iconRaster {
	return &iconRaster{w: w, h: h, pix: make([]float32, w*h*4)}
}

func (r *iconRaster) image() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, r.w, r.h))
	for i := 0; i < r.w*r.h; i++ {
		off := i * 4
		dst.Pix[off] = u8(r.pix[off])
		dst.Pix[off+1] = u8(r.pix[off+1])
		dst.Pix[off+2] = u8(r.pix[off+2])
		dst.Pix[off+3] = u8(r.pix[off+3])
	}
	return dst
}

func (r *iconRaster) fillPoly(pts [][2]float32, clr color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minX, minY, maxX, maxY := polyBounds(pts)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.w || y >= r.h {
				continue
			}
			var cov float32
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					px := float32(x) + (float32(sx)+0.5)/2
					py := float32(y) + (float32(sy)+0.5)/2
					if pointInPoly(px, py, pts) {
						cov += 0.25
					}
				}
			}
			r.blend(x, y, clr, cov)
		}
	}
}

func (r *iconRaster) strokePoly(pts [][2]float32, width float32, clr color.RGBA) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts); i++ {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		r.strokeLine(a[0], a[1], b[0], b[1], width, clr)
		r.fillCircle(a[0], a[1], width/2, clr)
	}
}

func (r *iconRaster) strokeRect(x, y, w, h, width float32, clr color.RGBA) {
	r.strokePoly([][2]float32{
		{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h},
	}, width, clr)
}

func (r *iconRaster) strokeLine(x1, y1, x2, y2, width float32, clr color.RGBA) {
	half := width / 2
	pad := half + 1
	minX := int(math.Floor(float64(min32(x1, x2) - pad)))
	maxX := int(math.Ceil(float64(max32(x1, x2) + pad)))
	minY := int(math.Floor(float64(min32(y1, y2) - pad)))
	maxY := int(math.Ceil(float64(max32(y1, y2) + pad)))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.w || y >= r.h {
				continue
			}
			px, py := float32(x)+0.5, float32(y)+0.5
			if r.clip != nil && !pointInPoly(px, py, r.clip) {
				continue
			}
			d := distToSeg(px, py, x1, y1, x2, y2)
			r.blend(x, y, clr, coverage(d, half))
		}
	}
}

func (r *iconRaster) fillCircle(cx, cy, rad float32, clr color.RGBA) {
	pad := rad + 1
	minX := int(math.Floor(float64(cx - pad)))
	maxX := int(math.Ceil(float64(cx + pad)))
	minY := int(math.Floor(float64(cy - pad)))
	maxY := int(math.Ceil(float64(cy + pad)))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.w || y >= r.h {
				continue
			}
			px, py := float32(x)+0.5, float32(y)+0.5
			if r.clip != nil && !pointInPoly(px, py, r.clip) {
				continue
			}
			dx, dy := px-cx, py-cy
			r.blend(x, y, clr, coverage(float32(math.Hypot(float64(dx), float64(dy))), rad))
		}
	}
}

func (r *iconRaster) strokeCircle(cx, cy, rad, width float32, clr color.RGBA) {
	half := width / 2
	pad := rad + half + 1
	minX := int(math.Floor(float64(cx - pad)))
	maxX := int(math.Ceil(float64(cx + pad)))
	minY := int(math.Floor(float64(cy - pad)))
	maxY := int(math.Ceil(float64(cy + pad)))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.w || y >= r.h {
				continue
			}
			px, py := float32(x)+0.5, float32(y)+0.5
			if r.clip != nil && !pointInPoly(px, py, r.clip) {
				continue
			}
			dx, dy := px-cx, py-cy
			d := float32(math.Abs(math.Hypot(float64(dx), float64(dy)) - float64(rad)))
			r.blend(x, y, clr, coverage(d, half))
		}
	}
}

func (r *iconRaster) strokeArc(cx, cy, rad, width, a0, a1 float32, clr color.RGBA) {
	steps := int(math.Max(12, float64(rad)*3))
	var prevX, prevY float32
	for i := 0; i <= steps; i++ {
		t := a0 + (a1-a0)*float32(i)/float32(steps)
		x := cx + rad*float32(math.Cos(float64(t)))
		y := cy + rad*float32(math.Sin(float64(t)))
		if i > 0 {
			r.strokeLine(prevX, prevY, x, y, width, clr)
		}
		prevX, prevY = x, y
	}
}

func (r *iconRaster) blend(x, y int, clr color.RGBA, cov float32) {
	if cov <= 0 {
		return
	}
	if cov > 1 {
		cov = 1
	}
	off := (y*r.w + x) * 4
	sa := float32(clr.A) / 255 * cov
	if sa <= 0 {
		return
	}
	inv := 1 - sa
	r.pix[off] = float32(clr.R)*sa + r.pix[off]*inv
	r.pix[off+1] = float32(clr.G)*sa + r.pix[off+1]*inv
	r.pix[off+2] = float32(clr.B)*sa + r.pix[off+2]*inv
	r.pix[off+3] = (sa + r.pix[off+3]/255*inv) * 255
}

func coverage(dist, radius float32) float32 {
	aa := float32(0.55)
	if dist <= radius-aa {
		return 1
	}
	if dist >= radius+aa {
		return 0
	}
	return (radius + aa - dist) / (2 * aa)
}

func pointInPoly(x, y float32, pts [][2]float32) bool {
	inside := false
	j := len(pts) - 1
	for i := 0; i < len(pts); i++ {
		xi, yi := pts[i][0], pts[i][1]
		xj, yj := pts[j][0], pts[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

func distToPoly(x, y float32, pts [][2]float32) float32 {
	best := float32(1e9)
	for i := 0; i < len(pts); i++ {
		a, b := pts[i], pts[(i+1)%len(pts)]
		if d := distToSeg(x, y, a[0], a[1], b[0], b[1]); d < best {
			best = d
		}
	}
	return best
}

func distToSeg(px, py, x1, y1, x2, y2 float32) float32 {
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return float32(math.Hypot(float64(px-x1), float64(py-y1)))
	}
	t := ((px-x1)*dx + (py-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return float32(math.Hypot(float64(px-(x1+t*dx)), float64(py-(y1+t*dy))))
}

func polyBounds(pts [][2]float32) (int, int, int, int) {
	minX, minY := pts[0][0], pts[0][1]
	maxX, maxY := minX, minY
	for _, p := range pts[1:] {
		minX = min32(minX, p[0])
		minY = min32(minY, p[1])
		maxX = max32(maxX, p[0])
		maxY = max32(maxY, p[1])
	}
	return int(math.Floor(float64(minX))), int(math.Floor(float64(minY))),
		int(math.Ceil(float64(maxX))), int(math.Ceil(float64(maxY)))
}

func toRGBA(clr color.Color) color.RGBA {
	if c, ok := clr.(color.RGBA); ok {
		return c
	}
	r, g, b, a := clr.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func u8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
