package abbrmap

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCompositePreservesRectangularMapAspectRatio(t *testing.T) {
	img := NewComposite("pearl_harbor", 128, 160, 100)
	require.Equal(t, image.Pt(80, 100), img.Bounds().Size())
}
