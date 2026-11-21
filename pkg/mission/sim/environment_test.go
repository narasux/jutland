package sim

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
)

// TestApplyEnvironmentOverridesAndRestores 验证 scenario environment 会写进全局配置并在跑完后恢复。
// 场景：全局 config.G 是 speedMultiplier=1、enablePenetration=false、enableRamming=false；
// 场景要求 2.0、true、true。
// 无头 runner 在 run 开始时调用 ApplyEnvironment，战斗路径全程读 config.G；跑完必须还原，
// 否则同进程的下一次 run 会继承上一个场景的开关，结果不可复现。
// 预期：apply 后 G.SpeedMultiplier=2、G.EnablePenetration=true、G.EnableRamming=true；
// restore 后回到 1/false/false。
// 覆盖：保存旧 G，构造 Scenario{Environment:{SpeedMultiplier:2, EnablePenetration:true,
// EnableRamming:true}}，调用 ApplyEnvironment 断言三字段，再 restore 断言还原；Cleanup 恢复原 G。
func TestApplyEnvironmentOverridesAndRestores(t *testing.T) {
	oldG := config.G
	t.Cleanup(func() { config.G = oldG })
	config.G = &config.GameSettings{SpeedMultiplier: 1, Language: "zh-Hans"}

	sc := &Scenario{Environment: Environment{
		SpeedMultiplier: 2, EnablePenetration: true, EnableRamming: true,
	}}
	restore := ApplyEnvironment(sc)
	require.Equal(t, 2.0, config.G.SpeedMultiplier)
	require.True(t, config.G.EnablePenetration)
	require.True(t, config.G.EnableRamming)

	restore()
	require.Equal(t, 1.0, config.G.SpeedMultiplier)
	require.False(t, config.G.EnablePenetration)
	require.False(t, config.G.EnableRamming)
}

// TestApplyEnvironmentCreatesDefaultSettings 验证全局配置尚未加载时也不会 panic。
// 场景：F1/F3 CLI 直接启动、config.G 为 nil 时应用场景环境，战斗路径却假定配置非 nil。
// 预期：ApplyEnvironment 补一份默认配置，写入场景值；restore 后配置仍非 nil 并还原默认值。
// 覆盖：把 config.G 置 nil，调用 ApplyEnvironment(enablePenetration=true, enableRamming=true)
// 断言非 nil/true，restore 后为 false。
func TestApplyEnvironmentCreatesDefaultSettings(t *testing.T) {
	oldG := config.G
	t.Cleanup(func() { config.G = oldG })
	config.G = nil

	sc := &Scenario{Environment: Environment{
		SpeedMultiplier: 1, EnablePenetration: true, EnableRamming: true,
	}}
	restore := ApplyEnvironment(sc)
	require.NotNil(t, config.G)
	require.True(t, config.G.EnablePenetration)
	require.True(t, config.G.EnableRamming)
	restore()
	require.NotNil(t, config.G)
	require.False(t, config.G.EnablePenetration)
	require.False(t, config.G.EnableRamming)
}
