package font

import (
	"testing"

	gamei18n "github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
)

func TestForLanguage(t *testing.T) {
	require.Same(t, Kai, ForLanguage(gamei18n.LanguageZhHans, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(gamei18n.LanguageEnglish, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(gamei18n.LanguageEnglish, Hang))
}
