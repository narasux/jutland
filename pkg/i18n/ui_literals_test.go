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

// TestUIHasNoHardcodedChineseStrings 验证 UI 包的非测试源码中没有硬编码中文字符串。
// 场景：界面文案迁移到 i18n 后，若 pkg/game、sidebar、unitpanel 等仍写字面量「任务选择」，
// 切到 en/ru/ja/zh-Hant 时会残留中文。
// 预期：6 个 UI 目录的字符串字面量都不含汉字；失败会打印违规字符串与文件路径，
// 说明该文案绕过了消息目录，切语言时不会翻译。
// 覆盖：从仓库根递归解析 pkg/game、collection、sidebar、unitpanel、drawer、object/unit 的 .go，
// 跳过 _test.go，用 go/ast 对每个字符串字面量 Unquote 后查 unicode.Han。
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
