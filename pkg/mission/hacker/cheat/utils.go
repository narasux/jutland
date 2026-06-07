package cheat

import (
	"regexp"
	"strings"
)

// 空白字符
var spaceRegexp = regexp.MustCompile(`\s+`)

// 忽略空白字符 & 大小写比较字符串
func isCommandEqual(s1, s2 string) bool {
	s1 = spaceRegexp.ReplaceAllString(s1, "")
	s2 = spaceRegexp.ReplaceAllString(s2, "")
	return strings.EqualFold(s1, s2)
}

// normalizeCommandToken 归一化动态秘籍参数，只保留终端可输入的 ASCII 字母和数字。
func normalizeCommandToken(s string) string {
	var b strings.Builder
	for _, r := range s {
		if '0' <= r && r <= '9' {
			b.WriteRune(r)
		} else if 'a' <= r && r <= 'z' {
			b.WriteRune(r)
		} else if 'A' <= r && r <= 'Z' {
			b.WriteRune(r + 'a' - 'A')
		}
	}
	return b.String()
}
