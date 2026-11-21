package sim

import "github.com/narasux/jutland/pkg/config"

// ApplyEnvironment 把场景 environment 段写进生产配置，并返回恢复函数。
// speedMultiplier / enablePenetration / enableRamming 平时来自 configs/game_settings.json5；
// 无头 runner 需要让单个 scenario 独立覆盖这几项，跑完必须恢复，避免污染同进程的下一次 run。
func ApplyEnvironment(sc *Scenario) func() {
	restore := func() {}
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	if sc == nil {
		return restore
	}
	oldSpeed, oldPenetration := config.G.SpeedMultiplier, config.G.EnablePenetration
	oldRamming := config.G.EnableRamming
	config.G.SpeedMultiplier = sc.Environment.SpeedMultiplier
	config.G.EnablePenetration = sc.Environment.EnablePenetration
	config.G.EnableRamming = sc.Environment.EnableRamming
	return func() {
		config.G.SpeedMultiplier = oldSpeed
		config.G.EnablePenetration = oldPenetration
		config.G.EnableRamming = oldRamming
	}
}
