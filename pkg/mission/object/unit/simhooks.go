package unit

import (
	"time"

	"github.com/google/uuid"

	"github.com/narasux/jutland/pkg/common/constants"
)

// nowMillis is the combat-path wall clock. Production uses real time; a
// headless runner can replace it with a fixed/advancing simulation clock so
// carrier takeoff and healing do not depend on wall time.
var nowMillis = func() int64 { return time.Now().UnixMilli() }

// planeUIDGenerator produces aircraft IDs. Production keeps UUIDs; a simulation
// can install a deterministic sequence so plane ordering is reproducible.
var planeUIDGenerator = func() string { return uuid.New().String() }

// SetNowMillis replaces the wall clock and returns a restore function.
// Passing nil restores real time. Simulation is single-threaded, so this is not
// synchronized; do not call it while a game update is running.
func SetNowMillis(fn func() int64) (restore func()) {
	old := nowMillis
	if fn == nil {
		nowMillis = func() int64 { return time.Now().UnixMilli() }
	} else {
		nowMillis = fn
	}
	return func() { nowMillis = old }
}

// NowMillis returns the current combat-path wall clock in milliseconds.
func NowMillis() int64 {
	return nowMillis()
}

// SimClockMillis 返回按模拟拍推进的毫秒时钟，供无头 runner 注入 SetNowMillis。
// 固定墙钟会让航母起飞冷却、损管回血等计时永远停在第一拍；这里把 60 拍折算成一秒，
// 既保持同 seed 可复现，又让计时按模拟时间而不是真实耗时前进。
func SimClockMillis(baseMS int64) func() int64 {
	return func() int64 {
		return baseMS + SimTick()*1000/int64(constants.MaxTPS)
	}
}

// SetPlaneUIDGenerator replaces the aircraft ID source and returns a restore
// function. Passing nil restores UUIDs. See SetNowMillis for the threading note.
func SetPlaneUIDGenerator(fn func() string) (restore func()) {
	old := planeUIDGenerator
	if fn == nil {
		planeUIDGenerator = func() string { return uuid.New().String() }
	} else {
		planeUIDGenerator = fn
	}
	return func() { planeUIDGenerator = old }
}

// NextPlaneUID returns one aircraft ID from the active generator.
func NextPlaneUID() string {
	return planeUIDGenerator()
}
