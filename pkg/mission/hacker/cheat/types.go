package cheat

import "github.com/narasux/jutland/pkg/mission/state"

// Cheat 秘籍
type Cheat interface {
	// 秘籍内容
	String() string
	// 秘籍描述
	Desc() string
	// 检查命令是否匹配
	Match(cmd string) bool
	// 执行命令，返回日志
	Exec(s *state.MissionState) string
}
