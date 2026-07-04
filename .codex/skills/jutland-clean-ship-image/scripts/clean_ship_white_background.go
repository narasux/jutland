package main

import (
	"container/list"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type point struct {
	x int
	y int
}

type component struct {
	area  int
	seedX int
	seedY int
	minX  int
	minY  int
	maxX  int
	maxY  int
}

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

func nearWhite(c color.NRGBA, min uint8, maxSpread uint8) bool {
	maxC := c.R
	if c.G > maxC {
		maxC = c.G
	}
	if c.B > maxC {
		maxC = c.B
	}

	minC := c.R
	if c.G < minC {
		minC = c.G
	}
	if c.B < minC {
		minC = c.B
	}

	return c.A > 0 && c.R >= min && c.G >= min && c.B >= min && maxC-minC <= maxSpread
}

func nrgbaAt(img *image.NRGBA, x, y int) color.NRGBA {
	return img.NRGBAAt(x, y)
}

func floodEdgeBackground(img *image.NRGBA, min uint8, maxSpread uint8) int {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	visited := make([]bool, w*h)
	queue := list.New()

	push := func(x, y int) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		idx := y*w + x
		if visited[idx] {
			return
		}
		visited[idx] = true
		c := nrgbaAt(img, x, y)
		if c.A == 0 || nearWhite(c, min, maxSpread) {
			queue.PushBack(point{x: x, y: y})
		}
	}

	for x := 0; x < w; x++ {
		push(x, 0)
		push(x, h-1)
	}
	for y := 0; y < h; y++ {
		push(0, y)
		push(w-1, y)
	}

	removed := 0
	for queue.Len() > 0 {
		elem := queue.Front()
		queue.Remove(elem)
		p := elem.Value.(point)
		c := nrgbaAt(img, p.x, p.y)
		if c.A != 0 {
			img.SetNRGBA(p.x, p.y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0})
			removed++
		}
		push(p.x+1, p.y)
		push(p.x-1, p.y)
		push(p.x, p.y+1)
		push(p.x, p.y-1)
	}

	return removed
}

func findComponents(img *image.NRGBA, min uint8, maxSpread uint8) []component {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	visited := make([]bool, w*h)
	components := make([]component, 0)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if visited[idx] {
				continue
			}
			visited[idx] = true
			if !nearWhite(nrgbaAt(img, x, y), min, maxSpread) {
				continue
			}

			current := component{seedX: x, seedY: y, minX: x, minY: y, maxX: x, maxY: y}
			queue := list.New()
			queue.PushBack(point{x: x, y: y})

			for queue.Len() > 0 {
				elem := queue.Front()
				queue.Remove(elem)
				p := elem.Value.(point)
				current.area++
				if p.x < current.minX {
					current.minX = p.x
				}
				if p.y < current.minY {
					current.minY = p.y
				}
				if p.x > current.maxX {
					current.maxX = p.x
				}
				if p.y > current.maxY {
					current.maxY = p.y
				}

				neighbors := [4]point{
					{x: p.x + 1, y: p.y},
					{x: p.x - 1, y: p.y},
					{x: p.x, y: p.y + 1},
					{x: p.x, y: p.y - 1},
				}
				for _, next := range neighbors {
					if next.x < 0 || next.y < 0 || next.x >= w || next.y >= h {
						continue
					}
					nextIdx := next.y*w + next.x
					if visited[nextIdx] {
						continue
					}
					visited[nextIdx] = true
					if nearWhite(nrgbaAt(img, next.x, next.y), min, maxSpread) {
						queue.PushBack(next)
					}
				}
			}

			components = append(components, current)
		}
	}

	sort.Slice(components, func(i, j int) bool {
		return components[i].area > components[j].area
	})
	return components
}

func componentInAnyBox(comp component, boxes []box) bool {
	if len(boxes) == 0 {
		return true
	}
	for _, current := range boxes {
		if comp.minX >= current.minX && comp.maxX <= current.maxX && comp.minY >= current.minY && comp.maxY <= current.maxY {
			return true
		}
	}
	return false
}

func removeLargeComponents(img *image.NRGBA, components []component, minArea int, boxes []box, min uint8, maxSpread uint8) (int, int) {
	if minArea <= 0 {
		return 0, 0
	}

	removedComponents := 0
	removedPixels := 0
	for _, comp := range components {
		if comp.area < minArea {
			continue
		}
		if !componentInAnyBox(comp, boxes) {
			continue
		}
		if !nearWhite(nrgbaAt(img, comp.seedX, comp.seedY), min, maxSpread) {
			continue
		}
		removedComponents++
		queue := list.New()
		queue.PushBack(point{x: comp.seedX, y: comp.seedY})

		for queue.Len() > 0 {
			elem := queue.Front()
			queue.Remove(elem)
			p := elem.Value.(point)
			c := nrgbaAt(img, p.x, p.y)
			if !nearWhite(c, min, maxSpread) {
				continue
			}
			img.SetNRGBA(p.x, p.y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0})
			removedPixels++

			neighbors := [4]point{
				{x: p.x + 1, y: p.y},
				{x: p.x - 1, y: p.y},
				{x: p.x, y: p.y + 1},
				{x: p.x, y: p.y - 1},
			}
			for _, next := range neighbors {
				if next.x < comp.minX || next.y < comp.minY || next.x > comp.maxX || next.y > comp.maxY {
					continue
				}
				if nearWhite(nrgbaAt(img, next.x, next.y), min, maxSpread) {
					queue.PushBack(next)
				}
			}
		}
	}
	return removedComponents, removedPixels
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

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func defaultOutput(input string) string {
	ext := filepath.Ext(input)
	stem := strings.TrimSuffix(input, ext)
	return stem + ".cleaned.png"
}

func backupPath(output string) string {
	ext := filepath.Ext(output)
	stem := strings.TrimSuffix(output, ext)
	return stem + ".before-aggressive-cleanup.png"
}

func writePreviews(output string, img *image.NRGBA) error {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	checker := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(210)
			if ((x/24)+(y/24))%2 == 0 {
				v = 245
			}
			checker.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	draw.Draw(checker, checker.Bounds(), img, image.Point{}, draw.Over)
	if err := savePNG(strings.TrimSuffix(output, filepath.Ext(output))+".preview-checker.png", checker); err != nil {
		return err
	}

	dark := image.NewNRGBA(bounds)
	draw.Draw(dark, dark.Bounds(), &image.Uniform{C: color.NRGBA{R: 28, G: 18, B: 42, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(dark, dark.Bounds(), img, image.Point{}, draw.Over)
	return savePNG(strings.TrimSuffix(output, filepath.Ext(output))+".preview-dark.png", dark)
}

func printComponents(components []component, limit int) {
	total := 0
	for _, comp := range components {
		total += comp.area
	}
	fmt.Printf("residual_nearwhite_components=%d residual_nearwhite_pixels=%d\n", len(components), total)

	if limit > len(components) {
		limit = len(components)
	}
	for i := 0; i < limit; i++ {
		comp := components[i]
		fmt.Printf(
			"%02d area=%d bbox=(%d,%d)-(%d,%d) size=%dx%d\n",
			i+1,
			comp.area,
			comp.minX,
			comp.minY,
			comp.maxX,
			comp.maxY,
			comp.maxX-comp.minX+1,
			comp.maxY-comp.minY+1,
		)
	}
}

func main() {
	input := flag.String("input", "", "input PNG path")
	output := flag.String("output", "", "output PNG path; defaults to <stem>.cleaned.png")
	edgeMin := flag.Int("edge-min", 245, "minimum RGB channel value for edge-connected background")
	componentMin := flag.Int("component-min", 245, "minimum RGB channel value for residual near-white components")
	maxSpread := flag.Int("max-spread", 12, "maximum RGB channel spread for low-saturation near-white pixels")
	removeEnclosedMinArea := flag.Int("remove-enclosed-min-area", 0, "remove residual near-white components with at least this many pixels; 0 disables")
	componentLimit := flag.Int("component-limit", 40, "number of largest residual components to print")
	analyzeOnly := flag.Bool("analyze-only", false, "print residual near-white components after edge cleanup without writing files")
	previews := flag.Bool("previews", false, "write checker and dark-background previews when output is written")
	backupBeforeAggressive := flag.Bool("backup-before-aggressive", false, "copy an existing output before enclosed-component cleanup")
	var removeBoxes boxFlags
	flag.Var(&removeBoxes, "remove-box", "limit enclosed component removal to a box minX,minY,maxX,maxY; repeatable")
	flag.Parse()

	if *input == "" {
		log.Fatal("-input is required")
	}
	if *edgeMin < 0 || *edgeMin > 255 || *componentMin < 0 || *componentMin > 255 || *maxSpread < 0 || *maxSpread > 255 {
		log.Fatal("threshold values must be in 0..255")
	}

	img, err := loadPNG(*input)
	if err != nil {
		log.Fatal(err)
	}

	removedEdge := floodEdgeBackground(img, uint8(*edgeMin), uint8(*maxSpread))
	components := findComponents(img, uint8(*componentMin), uint8(*maxSpread))
	fmt.Printf("size=%dx%d removed_edge_connected_pixels=%d\n", img.Bounds().Dx(), img.Bounds().Dy(), removedEdge)
	printComponents(components, *componentLimit)

	if *analyzeOnly {
		return
	}

	outPath := *output
	if outPath == "" {
		outPath = defaultOutput(*input)
	}

	removedComponents, removedPixels := 0, 0
	if *removeEnclosedMinArea > 0 {
		if *backupBeforeAggressive {
			if _, err := os.Stat(outPath); err == nil {
				backup := backupPath(outPath)
				if err := copyFile(outPath, backup); err != nil {
					log.Fatal(err)
				}
				fmt.Printf("backup_before_aggressive_cleanup=%s\n", backup)
			}
		}
		removedComponents, removedPixels = removeLargeComponents(img, components, *removeEnclosedMinArea, removeBoxes, uint8(*componentMin), uint8(*maxSpread))
	}

	if err := savePNG(outPath, img); err != nil {
		log.Fatal(err)
	}
	if *previews {
		if err := writePreviews(outPath, img); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("output=%s removed_enclosed_components=%d removed_enclosed_pixels=%d\n", outPath, removedComponents, removedPixels)
}
