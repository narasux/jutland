package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"sort"
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

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func luma(r, g, b uint8) float64 {
	return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
}

type colorStat struct {
	c     rgb
	a     uint8
	count int
}

type component struct {
	area                   int
	minX, minY, maxX, maxY int
}

func (c component) w() int { return c.maxX - c.minX + 1 }
func (c component) h() int { return c.maxY - c.minY + 1 }
func (c component) fill() float64 {
	return float64(c.area) / float64(c.w()*c.h())
}

func inBoxes(x, y int, boxes []box) bool {
	if len(boxes) == 0 {
		return true
	}
	for _, b := range boxes {
		if x >= b.minX && x <= b.maxX && y >= b.minY && y <= b.maxY {
			return true
		}
	}
	return false
}

// componentsOf returns 4-connected components of the exact target color.
func componentsOf(img image.Image, target rgb, minArea int, boxes []box) []component {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	visited := make([]bool, w*h)
	index := func(x, y int) int { return y*w + x }
	matched := func(x, y int) bool {
		if !inBoxes(x, y, boxes) {
			return false
		}
		r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
		if a == 0 {
			return false
		}
		return uint8(r>>8) == target.r && uint8(g>>8) == target.g && uint8(b>>8) == target.b
	}

	var out []component
	stack := make([]int, 0, 1024)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if visited[index(x, y)] || !matched(x, y) {
				continue
			}
			visited[index(x, y)] = true
			stack = append(stack[:0], index(x, y))
			comp := component{minX: x, minY: y, maxX: x, maxY: y}
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				cx, cy := cur%w, cur/w
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
				neighbors := [4][2]int{{cx - 1, cy}, {cx + 1, cy}, {cx, cy - 1}, {cx, cy + 1}}
				for _, n := range neighbors {
					nx, ny := n[0], n[1]
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					if visited[index(nx, ny)] || !matched(nx, ny) {
						continue
					}
					visited[index(nx, ny)] = true
					stack = append(stack, index(nx, ny))
				}
			}
			if comp.area >= minArea {
				out = append(out, comp)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].area > out[j].area })
	return out
}

func main() {
	input := flag.String("input", "", "input image path")
	var boxes boxFlags
	flag.Var(&boxes, "box", "limit analysis to box minX,minY,maxX,maxY; repeatable")
	top := flag.Int("top", 30, "number of most frequent colors to print")
	colorFlag := flag.String("color", "", "print connected components of this exact R,G,B")
	components := flag.Bool("components", false, "list connected components of -color")
	minArea := flag.Int("min-area", 4, "minimum component area to report")
	flag.Parse()

	if *input == "" {
		log.Fatal("-input is required")
	}
	img, err := loadImage(*input)
	if err != nil {
		log.Fatal(err)
	}
	bounds := img.Bounds()

	if *colorFlag != "" {
		target, err := parseRGB(*colorFlag)
		if err != nil {
			log.Fatal(err)
		}
		if !*components {
			*components = true
		}
		comps := componentsOf(img, target, *minArea, boxes)
		total := 0
		for _, c := range comps {
			total += c.area
		}
		fmt.Printf(
			"color=%d,%d,%d components=%d total_pixels=%d source=%s\n",
			target.r, target.g, target.b, len(comps), total, *input,
		)
		for i, c := range comps {
			if i >= *top {
				fmt.Printf("... %d more components omitted\n", len(comps)-i)
				break
			}
			fmt.Printf(
				"  area=%6d bbox=(%4d,%4d,%4d,%4d) w=%4d h=%4d fill=%.2f\n",
				c.area, c.minX, c.minY, c.maxX, c.maxY, c.w(), c.h(), c.fill(),
			)
		}
		return
	}

	counts := map[rgb]map[uint8]int{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !inBoxes(x-bounds.Min.X, y-bounds.Min.Y, boxes) {
				continue
			}
			r, g, b, a := img.At(x, y).RGBA()
			c := rgb{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
			if counts[c] == nil {
				counts[c] = map[uint8]int{}
			}
			counts[c][uint8(a>>8)]++
		}
	}
	list := make([]colorStat, 0, len(counts))
	for c, alphas := range counts {
		for a, n := range alphas {
			list = append(list, colorStat{c: c, a: a, count: n})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].count > list[j].count })

	fmt.Printf("size=%dx%d unique_colors=%d source=%s\n", bounds.Dx(), bounds.Dy(), len(list), *input)
	for i, e := range list {
		if i >= *top {
			break
		}
		fmt.Printf(
			"  %3d,%3d,%3d a=%3d count=%8d luma=%6.1f\n",
			e.c.r, e.c.g, e.c.b, e.a, e.count, luma(e.c.r, e.c.g, e.c.b),
		)
	}
}
