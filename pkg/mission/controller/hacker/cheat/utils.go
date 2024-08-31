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
