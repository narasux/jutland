package layout

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/resources/font"
)

func TestWrapWordsPreservesWordBoundaries(t *testing.T) {
	source := font.JetbrainsMono
	size := 18.0
	maxWidth := CalcTextWidth("alpha beta", size, source) - 1

	require.Equal(t, []string{"alpha", "beta"}, wrapWords("alpha beta", maxWidth, size, source))
}

func TestWrapRunesUsesMeasuredWidth(t *testing.T) {
	source := font.Kai
	size := 18.0
	maxWidth := CalcTextWidth("中文", size, source) + 1

	require.Equal(t, []string{"中文", "换行"}, wrapRunes("中文换行", maxWidth, size, source))
}
