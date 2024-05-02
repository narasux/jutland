package loader

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/pkg/errors"

	"github.com/narasux/jutland/pkg/common/types"
	"github.com/narasux/jutland/pkg/envs"
)

// LoadImage 加载图片资源
func LoadImage(path string) (*ebiten.Image, error) {
	imgData, err := os.ReadFile(envs.ImgResBaseDir + path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

// LoadFont 加载字体资源
func LoadFont(path string) (*text.GoTextFaceSource, error) {
	fontData, err := os.ReadFile(envs.FontResBaseDir + path)
	if err != nil {
		return nil, err
	}
	return text.NewGoTextFaceSource(bytes.NewReader(fontData))
}

// LoadAudio 加载音频资源
func LoadAudio(path string) (types.AudioStream, error) {
	audioData, err := os.ReadFile(envs.AudioResBaseDir + path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".ogg") {
		return vorbis.DecodeWithoutResampling(bytes.NewReader(audioData))
	} else if strings.HasSuffix(path, ".wav") {
		return wav.DecodeWithoutResampling(bytes.NewReader(audioData))
	} else if strings.HasSuffix(path, ".mp3") {
		return mp3.DecodeWithoutResampling(bytes.NewReader(audioData))
	}
	return nil, errors.New("unsupported audio format")
}
