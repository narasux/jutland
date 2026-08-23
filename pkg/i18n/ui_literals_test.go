package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func TestUIHasNoHardcodedChineseStrings(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	dirs := []string{
		"pkg/game",
		"pkg/collection",
		"pkg/mission/sidebar",
		"pkg/mission/unitpanel",
		"pkg/mission/drawer",
		"pkg/mission/object/unit",
	}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".go" || filepath.Ext(entry.Name()) != ".go" {
				return walkErr
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, unquoteErr := strconv.Unquote(literal.Value)
				require.NoError(t, unquoteErr)
				for _, r := range value {
					require.Falsef(t, unicode.Is(unicode.Han, r), "hardcoded Chinese UI string %q in %s", value, path)
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
}
