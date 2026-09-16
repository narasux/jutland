package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
)

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

func chunk(kind string, data []byte) []byte {
	out := make([]byte, 0, len(data)+12)
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(data)))
	out = append(out, length[:]...)
	out = append(out, kind...)
	out = append(out, data...)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(data)
	var sum [4]byte
	binary.BigEndian.PutUint32(sum[:], crc.Sum32())
	out = append(out, sum[:]...)
	return out
}

// writeRGBAAlways encodes img as an 8-bit RGBA PNG (color type 6) even when every
// pixel is opaque, which image/png would otherwise downgrade to color type 2.
func writeRGBAAlways(path string, img *image.NRGBA) error {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	raw := make([]byte, 0, h*(1+w*4))
	for y := 0; y < h; y++ {
		raw = append(raw, 0) // filter type 0 (None)
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		raw = append(raw, row...)
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	out := make([]byte, 0, compressed.Len()+128)
	out = append(out, 0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a)
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type RGBA
	ihdr[10] = 0
	ihdr[11] = 0
	ihdr[12] = 0
	out = append(out, chunk("IHDR", ihdr)...)
	out = append(out, chunk("IDAT", compressed.Bytes())...)
	out = append(out, chunk("IEND", nil)...)

	return os.WriteFile(path, out, 0o644)
}

func main() {
	input := flag.String("input", "", "input image path")
	output := flag.String("output", "", "output PNG path (RGBA, color type 6)")
	flag.Parse()

	if *input == "" || *output == "" {
		log.Fatal("-input and -output are required")
	}
	img, err := loadPNG(*input)
	if err != nil {
		log.Fatal(err)
	}

	opaque := true
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 255 {
			opaque = false
			break
		}
	}

	if err := writeRGBAAlways(*output, img); err != nil {
		log.Fatal(err)
	}

	check, err := os.Open(*output)
	if err != nil {
		log.Fatal(err)
	}
	defer check.Close()
	decoded, err := png.Decode(check)
	if err != nil {
		log.Fatal(err)
	}
	if got := decoded.Bounds(); got.Dx() != img.Bounds().Dx() || got.Dy() != img.Bounds().Dy() {
		log.Fatalf("round-trip size mismatch: %v != %v", got, img.Bounds())
	}

	fmt.Printf(
		"size=%dx%d input_fully_opaque=%v output=%s color_type=6(RGBA)\n",
		img.Bounds().Dx(), img.Bounds().Dy(), opaque, *output,
	)
}
