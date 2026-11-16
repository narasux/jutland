package state

import "testing"

// TestDebugFlagsIsActiveWithAnyFlag 验证任一调试开关开启时调试模式即生效，零值保持关闭。
// 场景：开发者通过终端/秘籍逐项打开 ShowHitBoxes、ShowPlaneHP、DamageColorByTeam 等开关；
// 只要有一个开启，调试面板、命中框或跑道线就必须进入绘制/更新分支。
// 预期：五个标志分别置 true 时 IsActive 都为 true；全 false 的零值为 false。
// 覆盖：表驱动构造五个单标志与零值，逐项断言，避免只测到 OR 的第一个分支。
func TestDebugFlagsIsActiveWithAnyFlag(t *testing.T) {
	cases := []struct {
		name  string
		flags DebugFlags
		want  bool
	}{
		{name: "zero value", flags: DebugFlags{}, want: false},
		{name: "damage color by team", flags: DebugFlags{DamageColorByTeam: true}, want: true},
		{name: "cursor object info", flags: DebugFlags{ShowCursorPosObjInfo: true}, want: true},
		{name: "plane hp", flags: DebugFlags{ShowPlaneHP: true}, want: true},
		{name: "hit boxes", flags: DebugFlags{ShowHitBoxes: true}, want: true},
		{name: "airfield runway", flags: DebugFlags{ShowAirfieldRunway: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.flags.IsActive(); got != tc.want {
				t.Fatalf("IsActive(%+v) = %v, want %v", tc.flags, got, tc.want)
			}
		})
	}
}
