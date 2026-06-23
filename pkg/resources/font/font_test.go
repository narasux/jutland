package font

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForLanguage(t *testing.T) {
	require.Same(t, Kai, ForLanguage(i18n.LanguageZhHans, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Hang))
}
