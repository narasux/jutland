package unit

import (
	"time"

	"github.com/google/uuid"
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
