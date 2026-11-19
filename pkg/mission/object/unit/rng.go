package unit

import "math/rand"

// Combat random source. Production delegates to math/rand's global source;
// a headless run can install a seeded *rand.Rand so the same seed produces the
// same fire spread, criticals and AA rolls. The simulation is single-threaded;
// do not swap the source while an update is running.
var (
	randFloat64 = rand.Float64
	randIntn    = rand.Intn
)

// SetRandSource replaces the combat random source and returns a restore
// function. Passing nil restores math/rand's global source.
func SetRandSource(src *rand.Rand) (restore func()) {
	oldFloat, oldIntn := randFloat64, randIntn
	if src == nil {
		randFloat64 = rand.Float64
		randIntn = rand.Intn
	} else {
		randFloat64 = src.Float64
		randIntn = src.Intn
	}
	return func() {
		randFloat64 = oldFloat
		randIntn = oldIntn
	}
}

// RandFloat64 returns a [0,1) value from the active combat source.
func RandFloat64() float64 {
	return randFloat64()
}

// RandIntn returns a [0,n) value from the active combat source.
func RandIntn(n int) int {
	return randIntn(n)
}
