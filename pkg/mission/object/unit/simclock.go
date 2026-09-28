package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
)

var simTick int64

// SetSimTick 由任务主循环在每拍开始时写入当前游戏拍。
func SetSimTick(tick int64) {
	simTick = tick
}

// SimTick 返回当前游戏拍。
func SimTick() int64 {
	return simTick
}

// ReloadTicks 把装填秒数换成游戏拍。速度倍率 1 时 1 秒是 60 拍。
func ReloadTicks(seconds float64) int64 {
	if seconds <= 0 {
		return 0
	}
	mult := 1.0
	if config.G != nil && config.G.SpeedMultiplier > 0 {
		mult = config.G.SpeedMultiplier
	}
	ticks := int64(math.Round(seconds * float64(constants.MaxTPS) / mult))
	if ticks < 1 {
		return 1
	}
	return ticks
}

func ticksReady(start int64, seconds float64) bool {
	if start <= 0 {
		return true
	}
	return SimTick()-start >= ReloadTicks(seconds)
}
