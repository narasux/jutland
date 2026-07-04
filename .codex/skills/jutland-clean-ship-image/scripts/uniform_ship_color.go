package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type box struct {
	minX int
	minY int
	maxX int
	maxY int
}

type boxFlags []box

func (b *boxFlags) String() string {
	if b == nil {
		return ""
	}
	parts := make([]string, 0, len(*b))
	for _, current := range *b {
		parts = append(parts, fmt.Sprintf("%d,%d,%d,%d", current.minX, current.minY, current.maxX, current.maxY))
	}
	return strings.Join(parts, ";")
}

func (b *boxFlags) Set(value string) error {
	var parsed box
	if _, err := fmt.Sscanf(value, "%d,%d,%d,%d", &parsed.minX, &parsed.minY, &parsed.maxX, &parsed.maxY); err != nil {
		return fmt.Errorf("invalid box %q, want minX,minY,maxX,maxY: %w", value, err)
	}
	if parsed.minX > parsed.maxX || parsed.minY > parsed.maxY {
		return fmt.Errorf("invalid box %q: min values must not exceed max values", value)
	}
	*b = append(*b, parsed)
	return nil
}

type rgb struct {
	r float64
	g float64
	b float64
}

func parseRGB(value string) (rgb, error) {
	var parsed rgb
	if _, err := fmt.Sscanf(value, "%f,%f,%f", &parsed.r, &parsed.g, &parsed.b); err != nil {
		return rgb{}, fmt.Errorf("invalid RGB %q, want R,G,B: %w", value, err)
	}
	if parsed.r < 0 || parsed.r > 255 || parsed.g < 0 || parsed.g > 255 || parsed.b < 0 || parsed.b > 255 {
		return rgb{}, fmt.Errorf("RGB values must be in 0..255")
	}
	return parsed, nil
}

func luma(c rgb) float64 {
	return 0.2126*c.r + 0.7152*c.g + 0.0722*c.b
}

func spread(c rgb) float64 {
	maxC := math.Max(c.r, math.Max(c.g, c.b))
	minC := math.Min(c.r, math.Min(c.g, c.b))
	return maxC - minC
}

func inBox(x, y int, b box) bool {
	return x >= b.minX && y >= b.minY && x <= b.maxX && y <= b.maxY
}

func inAnyBox(x, y int, boxes []box) bool {
	if len(boxes) == 0 {
		return true
	}
	for _, current := range boxes {
		if inBox(x, y, current) {
			return true
		}
	}
	return false
}

func loadPNG(path string) (*image.NRGBA, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	src, err := png.Decode(in)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	img := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(img, img.Bounds(), src, bounds.Min, draw.Src)
	return img, nil
}

func savePNG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func writePreviews(output string, img *image.NRGBA) error {
	bounds := img.Bounds()
	for suffix, background := range map[string]color.NRGBA{
		"black":   {R: 0, G: 0, B: 0, A: 255},
		"magenta": {R: 255, G: 0, B: 255, A: 255},
	} {
		canvas := image.NewNRGBA(bounds)
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
		preview := strings.TrimSuffix(output, filepath.Ext(output)) + ".preview-" + suffix + ".png"
		if err := savePNG(preview, canvas); err != nil {
			return err
		}
	}
	return nil
}

func shouldRecolor(c color.NRGBA, minLuma, maxLuma, maxSpread float64, preserveRed bool) bool {
	if c.A == 0 {
		return false
	}
	p := rgb{r: float64(c.R), g: float64(c.G), b: float64(c.B)}
	currentLuma := luma(p)
	if currentLuma < minLuma || currentLuma > maxLuma {
		return false
	}
	if spread(p) > maxSpread {
		return false
	}
	if preserveRed && float64(c.R) > 80 && float64(c.R)-float64(c.G) > 16 && float64(c.R)-float64(c.B) > 10 {
		return false
	}
	return true
}

func main() {
	input := flag.String("input", "", "input transparent PNG path")
	output := flag.String("output", "", "output PNG path")
	targetColor := flag.String("target-color", "56,68,93", "target RGB color; default is sampled US navy blue")
	minLuma := flag.Float64("min-luma", 25, "minimum luma to recolor; preserves black line art below this")
	maxLuma := flag.Float64("max-luma", 255, "maximum luma to recolor")
	maxSpread := flag.Float64("max-spread", 105, "maximum RGB channel spread to recolor")
	shadeStrength := flag.Float64("shade-strength", 0.0005, "local shading strength relative to input luma")
	minFactor := flag.Float64("min-factor", 0.96, "minimum multiplier applied to target color")
	maxFactor := flag.Float64("max-factor", 1.04, "maximum multiplier applied to target color")
	preserveRed := flag.Bool("preserve-red", true, "preserve saturated red pixels such as underwater hulls and flags")
	previews := flag.Bool("previews", false, "write black and magenta background previews next to output")
	var recolorBoxes boxFlags
	var skipBoxes boxFlags
	flag.Var(&recolorBoxes, "recolor-box", "limit recoloring to box minX,minY,maxX,maxY; repeatable")
	flag.Var(&skipBoxes, "skip-box", "exclude box minX,minY,maxX,maxY; repeatable")
	flag.Parse()

	if *input == "" || *output == "" {
		log.Fatal("-input and -output are required")
	}
	target, err := parseRGB(*targetColor)
	if err != nil {
		log.Fatal(err)
	}

	img, err := loadPNG(*input)
	if err != nil {
		log.Fatal(err)
	}

	changed := 0
	bounds := img.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if !inAnyBox(x, y, recolorBoxes) || inAnyBox(x, y, skipBoxes) {
				continue
			}
			c := img.NRGBAAt(x, y)
			if !shouldRecolor(c, *minLuma, *maxLuma, *maxSpread, *preserveRed) {
				continue
			}

			currentLuma := luma(rgb{r: float64(c.R), g: float64(c.G), b: float64(c.B)})
			factor := 1 + (currentLuma-126)*(*shadeStrength)
			if factor < *minFactor {
				factor = *minFactor
			}
			if factor > *maxFactor {
				factor = *maxFactor
			}

			r := uint8(math.Round(math.Max(0, math.Min(255, target.r*factor))))
			g := uint8(math.Round(math.Max(0, math.Min(255, target.g*factor))))
			b := uint8(math.Round(math.Max(0, math.Min(255, target.b*factor))))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: c.A})
			changed++
		}
	}

	if err := savePNG(*output, img); err != nil {
		log.Fatal(err)
	}
	if *previews {
		if err := writePreviews(*output, img); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("size=%dx%d target_color=%s changed_pixels=%d output=%s\n", bounds.Dx(), bounds.Dy(), *targetColor, changed, *output)
}
