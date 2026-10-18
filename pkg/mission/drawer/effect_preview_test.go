package drawer

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// TestDumpHitEffectPreview 用真实的粒子配方做离屏合成并输出 PNG，便于人工检查观感。
// 默认跳过；需要时 JUTLAND_EFFECT_PREVIEW=1 运行，产物在 /tmp/effect_*.png。
func TestDumpHitEffectPreview(t *testing.T) {
	if os.Getenv("JUTLAND_EFFECT_PREVIEW") == "" {
		t.Skip("set JUTLAND_EFFECT_PREVIEW=1 to dump preview PNGs")
	}
	cell := 128.0
	cases := []struct {
		name string
		kind objExplosion.Kind
		size float64
		life int
	}{
		{"flash_medium", objExplosion.KindFlash, 0.34, 26},
		{"flash_heavy", objExplosion.KindFlash, 0.5, 32},
		{"splash_medium", objExplosion.KindSplash, 0.45, 60},
		{"splash_heavy", objExplosion.KindSplash, 0.8, 60},
		{"torpedo_plume", objExplosion.KindTorpedoPlume, 1.2, 60},
		{"dust_heavy", objExplosion.KindDust, 0.75, 60},
	}
	const steps = 8
	for _, testCase := range cases {
		width, height := int(cell*steps), int(cell*1.6)
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		for i := range img.Pix {
			img.Pix[i] = 0
			if i%4 == 3 {
				img.Pix[i] = 255
			}
		}
		fillSea(img)
		for i := 0; i < steps; i++ {
			// 同一个种子：这 8 格是同一次爆炸的时间演化，而不是 8 次不同的爆炸
			effect := objExplosion.NewHitEffect(
				testCase.kind, objPos.NewR(0, 0), 0, testCase.size, testCase.life,
				uint32(20261018), faction.HumanAlpha,
			)
			effect.Age = testCase.life * i / steps
			compositeEffect(img, effect, cell*float64(i)+cell/2, cell*0.8, 1)
		}
		dumpPNG(t, fmt.Sprintf("/tmp/effect_%s.png", testCase.name), img)
	}
}

// compositeEffect 用与 drawEffectParticles 相同的粒子配方与混合方式做软件合成，
// 这样无需启动 Ebiten 也能检查真实配方。
func compositeEffect(img *image.NRGBA, e *objExplosion.Explosion, x, y, zoomScale float64) {
	t := float64(e.Age) / float64(e.Life)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	sizePx := e.Size * 128 * zoomScale
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
		dist := p.distance * sizePx * 0.5 * (0.35 + 0.65*t)
		diameter := p.size * sizePx * (1 + p.sizeGrow*t)
		if diameter <= 0 {
			continue
		}
		px, py := x+math.Cos(p.angle)*dist, y+math.Sin(p.angle)*dist
		if p.lean > 0 {
			rad := e.Rotation * math.Pi / 180
			back := p.lean * sizePx * math.Min(1, t/0.5)
			px -= math.Sin(rad) * back
			py += math.Cos(rad) * back
		}
		compositeCircle(
			img,
			px, py, diameter/2,
			float64(p.r), float64(p.g), float64(p.b), alpha,
		)
	}
}

// compositeCircle 按预乘 source-over 把一个柔圆画进 NRGBA 画布。
func compositeCircle(img *image.NRGBA, cx, cy, radius, r, g, b, alpha float64) {
	if radius <= 0 {
		return
	}
	x0, x1 := int(cx-radius)-1, int(cx+radius)+2
	y0, y1 := int(cy-radius)-1, int(cy+radius)+2
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			if px < 0 || py < 0 || px >= img.Rect.Dx() || py >= img.Rect.Dy() {
				continue
			}
			a := textureImg.EffectCircleAlpha(math.Hypot(float64(px)-cx, float64(py)-cy)/radius) * alpha
			if a <= 0 {
				continue
			}
			offset := img.PixOffset(px, py)
			for i, src := range []float64{r * a, g * a, b * a} {
				dst := float64(img.Pix[offset+i]) / 255
				img.Pix[offset+i] = uint8(math.Round(math.Min(1, src+dst*(1-a)) * 255))
			}
		}
	}
}

// fillSea 用海面底色填充，并在每格右侧画出驱逐舰尺寸参照（约 0.32 x 3.25 地图格）。
func fillSea(img *image.NRGBA) {
	cell := 128
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			offset := img.PixOffset(x, y)
			img.Pix[offset] = 22
			img.Pix[offset+1] = 38
			img.Pix[offset+2] = 58
		}
	}
	// 尺寸参照：舰体轮廓
	for band := 0; band < 4; band++ {
		cx := float64(band)*float64(cell)*2 + float64(cell)*1.5
		for y := 20; y < 20+int(3.25*float64(cell)); y++ {
			for x := int(cx) - 20; x < int(cx)+20; x++ {
				if x < 0 || y < 0 || x >= img.Rect.Dx() || y >= img.Rect.Dy() {
					continue
				}
				offset := img.PixOffset(x, y)
				img.Pix[offset] = 150
				img.Pix[offset+1] = 152
				img.Pix[offset+2] = 155
			}
		}
	}
}

func dumpPNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()
	if err = png.Encode(file, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	t.Logf("wrote %s", path)
}
