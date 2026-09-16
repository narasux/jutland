package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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

func inAny(x, y int, boxes boxFlags) bool {
	for _, b := range boxes {
		if x >= b.minX && x <= b.maxX && y >= b.minY && y <= b.maxY {
			return true
		}
	}
	return false
}

func main() {
	beforePath := flag.String("before", "", "original image")
	afterPath := flag.String("after", "", "candidate image")
	var allowed boxFlags
	var protected boxFlags
	var protectedColors multiRGB
	flag.Var(&allowed, "allowed-box", "box minX,minY,maxX,maxY where changes are allowed; repeatable; empty means anywhere")
	flag.Var(&protected, "protected-box", "box minX,minY,maxX,maxY that must not change; repeatable")
	flag.Var(&protectedColors, "protected-color", "exact R,G,B in the before image that must not change; repeatable")
	maxAlphaChanges := flag.Int("max-alpha-changes", 0, "maximum number of pixels whose alpha may change")
	maxProtectedChanges := flag.Int("max-protected-changes", 0, "maximum number of changed pixels inside protected boxes or with a protected color")
	flag.Parse()

	if *beforePath == "" || *afterPath == "" {
		log.Fatal("-before and -after are required")
	}
	before, err := loadPNG(*beforePath)
	if err != nil {
		log.Fatal(err)
	}
	after, err := loadPNG(*afterPath)
	if err != nil {
		log.Fatal(err)
	}
	if before.Bounds() != after.Bounds() {
		log.Fatalf("size mismatch: before=%v after=%v", before.Bounds(), after.Bounds())
	}

	bounds := before.Bounds()
	changed, alphaChanged, protectedChanges, outside := 0, 0, 0, 0
	changeBox := box{minX: bounds.Dx(), minY: bounds.Dy(), maxX: -1, maxY: -1}
	haveChange := false

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			b := before.NRGBAAt(x, y)
			a := after.NRGBAAt(x, y)
			if b == a {
				continue
			}
			changed++
			if !haveChange {
				haveChange = true
				changeBox = box{x, y, x, y}
			}
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
			if b.A != a.A {
				alphaChanged++
			}
			if len(allowed) > 0 && !inAny(x, y, allowed) {
				outside++
			}
			if inAny(x, y, protected) {
				protectedChanges++
				continue
			}
			for _, pc := range protectedColors {
				if b.R == pc.r && b.G == pc.g && b.B == pc.b {
					protectedChanges++
					break
				}
			}
		}
	}

	status := "ok"
	failures := []string{}
	if outside > 0 {
		failures = append(failures, fmt.Sprintf("%d changed pixels outside allowed boxes", outside))
	}
	if alphaChanged > *maxAlphaChanges {
		failures = append(failures, fmt.Sprintf("%d alpha changes > %d", alphaChanged, *maxAlphaChanges))
	}
	if protectedChanges > *maxProtectedChanges {
		failures = append(failures, fmt.Sprintf("%d protected changes > %d", protectedChanges, *maxProtectedChanges))
	}
	if len(failures) > 0 {
		status = "FAILED"
	}

	fmt.Printf(
		"size=%dx%d changed_pixels=%d alpha_changes=%d protected_changes=%d outside_allowed=%d status=%s\n",
		bounds.Dx(), bounds.Dy(), changed, alphaChanged, protectedChanges, outside, status,
	)
	if haveChange {
		fmt.Printf(
			"changed_bbox=(%d,%d,%d,%d)\n",
			changeBox.minX, changeBox.minY, changeBox.maxX, changeBox.maxY,
		)
	} else {
		fmt.Println("changed_bbox=none")
	}
	for _, f := range failures {
		fmt.Println("  violation: " + f)
	}
	if len(failures) > 0 {
		os.Exit(1)
	}
}

type multiRGB []rgb

func (m *multiRGB) String() string {
	if m == nil {
		return ""
	}
	s := ""
	for i, c := range *m {
		if i > 0 {
			s += ";"
		}
		s += fmt.Sprintf("%d,%d,%d", c.r, c.g, c.b)
	}
	return s
}

func (m *multiRGB) Set(value string) error {
	c, err := parseRGB(value)
	if err != nil {
		return err
	}
	*m = append(*m, c)
	return nil
}
