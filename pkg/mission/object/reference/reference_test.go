package reference

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
)

func TestGetReferenceUsesCurrentLanguageAndFallsBackToChinese(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	previousReferences := referencesByLanguage
	t.Cleanup(func() {
		i18n.SetLanguage(string(previousLanguage))
		referencesByLanguage = previousReferences
	})

	referencesByLanguage = map[i18n.Language]map[string]*Reference{}
	zhRef := &Reference{Name: "ship", DisplayName: "舰船"}
	enRef := &Reference{Name: "ship", DisplayName: "Ship"}
	zhOnly := &Reference{Name: "fallback", DisplayName: "回退"}
	SetReference(i18n.LanguageZhHans, zhRef.Name, zhRef)
	SetReference(i18n.LanguageZhHans, zhOnly.Name, zhOnly)
	SetReference(i18n.LanguageEnglish, enRef.Name, enRef)

	i18n.SetLanguage(string(i18n.LanguageEnglish))
	require.Same(t, enRef, GetReference("ship"))
	require.Same(t, zhOnly, GetReference("fallback"))

	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Same(t, zhRef, GetReference("ship"))
}

func TestLocalizedReferenceFilesStayInSync(t *testing.T) {
	chinese, err := Load(filepath.Join(config.ConfigBaseDir, "references.json5"))
	require.NoError(t, err)
	english, err := Load(filepath.Join(config.ConfigBaseDir, "references.en.json5"))
	require.NoError(t, err)

	require.Len(t, chinese, 116)
	require.NoError(t, ValidateLocales(chinese, english))

	englishByName := make(map[string]Reference, len(english))
	for _, ref := range english {
		englishByName[ref.Name] = ref
	}
	for name, expectedDisplayName := range map[string]string{
		"yubari":    "Yūbari",
		"yorktown":  "Yorktown",
		"bismarck":  "Bismarck",
		"peace_ark": "Peace Ark",
		"A6M_0":     "A6M Zero",
	} {
		ref, ok := englishByName[name]
		require.True(t, ok)
		require.Equal(t, expectedDisplayName, ref.DisplayName)
		require.NotEmpty(t, ref.Links)
		require.True(t, strings.HasPrefix(ref.Links[len(ref.Links)-1].URL, "https://en.wikipedia.org/wiki/"))
	}
	require.NotEmpty(t, englishByName["yubari"].Description)
}

func TestValidateLocalesRejectsMissingTranslation(t *testing.T) {
	chinese, err := Load(filepath.Join(config.ConfigBaseDir, "references.json5"))
	require.NoError(t, err)
	english, err := Load(filepath.Join(config.ConfigBaseDir, "references.en.json5"))
	require.NoError(t, err)

	require.Error(t, ValidateLocales(chinese, english[:len(english)-1]))
}
