package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
)

type box struct {
	minX, minY, maxX, maxY int
}

type boxFlags []box

func (b *boxFlags) String() string {
	if b == nil {
		return ""
	}
	s := ""
	for i, c := range *b {
		if i > 0 {
			s += ";"
		}
		s += fmt.Sprintf("%d,%d,%d,%d", c.minX, c.minY, c.maxX, c.maxY)
	}
	return s
}

func (b *boxFlags) Set(value string) error {
	var p box
	if _, err := fmt.Sscanf(value, "%d,%d,%d,%d", &p.minX, &p.minY, &p.maxX, &p.maxY); err != nil {
		return fmt.Errorf("invalid box %q, want minX,minY,maxX,maxY: %w", value, err)
	}
	if p.minX > p.maxX || p.minY > p.maxY {
		return fmt.Errorf("invalid box %q: min values must not exceed max values", value)
	}
	*b = append(*b, p)
	return nil
}

type rgb struct{ r, g, b uint8 }

func parseRGB(value string) (rgb, error) {
	var c rgb
	if _, err := fmt.Sscanf(value, "%d,%d,%d", &c.r, &c.g, &c.b); err != nil {
		return rgb{}, fmt.Errorf("invalid RGB %q, want R,G,B: %w", value, err)
	}
	return c, nil
}

type component struct {
	pixels                 []int
	area                   int
	minX, minY, maxX, maxY int
}

func (c component) w() int { return c.maxX - c.minX + 1 }
func (c component) h() int { return c.maxY - c.minY + 1 }
func (c component) fill() float64 {
	return float64(c.area) / float64(c.w()*c.h())
}

type bbox struct{ minX, minY, maxX, maxY int }

func (b bbox) intersects(o box) bool {
	return b.minX <= o.maxX && b.maxX >= o.minX && b.minY <= o.maxY && b.maxY >= o.minY
}

func (b bbox) within(o box) bool {
	return b.minX >= o.minX && b.maxX <= o.maxX && b.minY >= o.minY && b.maxY <= o.maxY
}

// inBoxes reports whether (x,y) is inside any box; an empty list means no restriction.
func inBoxes(x, y int, boxes boxFlags) bool {
	if len(boxes) == 0 {
		return true
	}
	return inAnyBox(x, y, boxes)
}

// inAnyBox reports whether (x,y) is inside any box; an empty list matches nothing.
func inAnyBox(x, y int, boxes boxFlags) bool {
	for _, b := range boxes {
		if x >= b.minX && x <= b.maxX && y >= b.minY && y <= b.maxY {
			return true
		}
	}
	return false
}

func loadPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	img := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), src, b.Min, draw.Src)
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

func main() {
	input := flag.String("input", "", "input image path")
	output := flag.String("output", "", "output PNG path")
	source := flag.String("source-color", "", "exact source R,G,B to replace")
	target := flag.String("target-color", "", "target R,G,B")
	mode := flag.String("mode", "pixels", "pixels: match exact RGB anywhere in region; components: recolor whole connected components")
	selection := flag.String("selection", "within", "components mode: keep components 'within' or 'intersect' the include boxes")

	var includeBoxes boxFlags
	var excludeBoxes boxFlags
	flag.Var(&includeBoxes, "include-box", "limit changes to box minX,minY,maxX,maxY; repeatable")
	flag.Var(&excludeBoxes, "exclude-box", "exclude box minX,minY,maxX,maxY; repeatable")
	minArea := flag.Int("min-area", 1, "components mode: minimum component area")
	minWidth := flag.Int("min-width", 1, "components mode: minimum component bounding-box width")
	minHeight := flag.Int("min-height", 1, "components mode: minimum component bounding-box height")
	maxFill := flag.Float64("max-fill", 1.01, "components mode: maximum component fill ratio (area / bbox area)")
	verbose := flag.Bool("verbose", false, "print every recolored component")
	flag.Parse()

	if *input == "" || *output == "" {
		log.Fatal("-input and -output are required")
	}
	src, err := parseRGB(*source)
	if err != nil {
		log.Fatal(err)
	}
	dst, err := parseRGB(*target)
	if err != nil {
		log.Fatal(err)
	}

	img, err := loadPNG(*input)
	if err != nil {
		log.Fatal(err)
	}
	bounds := img.Bounds()

	changed := 0
	changeBox := bbox{minX: bounds.Dx(), minY: bounds.Dy(), maxX: -1, maxY: -1}
	mark := func(x, y int) {
		if x < changeBox.minX {
			changeBox.minX = x
		}
		if x > changeBox.maxX {
			changeBox.maxX = x
		}
		if y < changeBox.minY {
			changeBox.minY = y
		}
		if y > changeBox.maxY {
			changeBox.maxY = y
		}
	}

	switch *mode {
	case "pixels":
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				if !inBoxes(x, y, includeBoxes) || inAnyBox(x, y, excludeBoxes) {
					continue
				}
				c := img.NRGBAAt(x, y)
				if c.A == 0 || c.R != src.r || c.G != src.g || c.B != src.b {
					continue
				}
				img.SetNRGBA(x, y, color.NRGBA{R: dst.r, G: dst.g, B: dst.b, A: c.A})
				changed++
				mark(x, y)
			}
		}
	case "components":
		comps := componentsOfExact(img, src, *minArea)
		selected := 0
		for i := range comps {
			c := &comps[i]
			box := bbox{c.minX, c.minY, c.maxX, c.maxY}
			keep := false
			for _, ib := range includeBoxes {
				if *selection == "intersect" {
					if box.intersects(ib) {
						keep = true
					}
				} else if box.within(ib) {
					keep = true
				}
			}
			if !keep {
				continue
			}
			excluded := false
			for _, eb := range excludeBoxes {
				if box.intersects(eb) {
					excluded = true
				}
			}
			if excluded {
				continue
			}
			if c.area < *minArea || c.w() < *minWidth || c.h() < *minHeight || c.fill() > *maxFill {
				continue
			}
			selected++
			for _, idx := range c.pixels {
				x, y := idx%bounds.Dx(), idx/bounds.Dx()
				a := img.NRGBAAt(x, y).A
				img.SetNRGBA(x, y, color.NRGBA{R: dst.r, G: dst.g, B: dst.b, A: a})
				changed++
				mark(x, y)
			}
			if *verbose {
				fmt.Printf(
					"  recolored component area=%d bbox=(%d,%d,%d,%d) w=%d h=%d fill=%.2f\n",
					c.area, c.minX, c.minY, c.maxX, c.maxY, c.w(), c.h(), c.fill(),
				)
			}
		}
		fmt.Printf("components_matched=%d\n", selected)
	default:
		log.Fatalf("unknown -mode %q, want pixels or components", *mode)
	}

	if err := savePNG(*output, img); err != nil {
		log.Fatal(err)
	}
	if changed == 0 {
		fmt.Printf(
			"mode=%s source=%d,%d,%d target=%d,%d,%d changed_pixels=0 output=%s\n",
			*mode, src.r, src.g, src.b, dst.r, dst.g, dst.b, *output,
		)
		return
	}
	fmt.Printf(
		"mode=%s source=%d,%d,%d target=%d,%d,%d changed_pixels=%d changed_bbox=(%d,%d,%d,%d) output=%s\n",
		*mode, src.r, src.g, src.b, dst.r, dst.g, dst.b, changed,
		changeBox.minX, changeBox.minY, changeBox.maxX, changeBox.maxY, *output,
	)
}

// componentsOfExact returns 4-connected components whose pixels all equal target.
func componentsOfExact(img *image.NRGBA, target rgb, minArea int) []component {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	visited := make([]bool, w*h)
	matched := func(x, y int) bool {
		c := img.NRGBAAt(x, y)
		return c.A != 0 && c.R == target.r && c.G == target.g && c.B == target.b
	}
	var out []component
	stack := make([]int, 0, 1024)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			start := y*w + x
			if visited[start] || !matched(x, y) {
				continue
			}
			visited[start] = true
			stack = append(stack[:0], start)
			comp := component{minX: x, minY: y, maxX: x, maxY: y}
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				cx, cy := cur%w, cur/w
				comp.pixels = append(comp.pixels, cur)
				comp.area++
				if cx < comp.minX {
					comp.minX = cx
				}
				if cx > comp.maxX {
					comp.maxX = cx
				}
				if cy < comp.minY {
					comp.minY = cy
				}
				if cy > comp.maxY {
					comp.maxY = cy
				}
				for _, n := range [4][2]int{{cx - 1, cy}, {cx + 1, cy}, {cx, cy - 1}, {cx, cy + 1}} {
					nx, ny := n[0], n[1]
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					idx := ny*w + nx
					if visited[idx] || !matched(nx, ny) {
						continue
					}
					visited[idx] = true
					stack = append(stack, idx)
				}
			}
			if comp.area >= minArea {
				out = append(out, comp)
			}
		}
	}
	return out
}
