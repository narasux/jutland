package bullet

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/object"
)

func TestHasAirburst(t *testing.T) {
	bt := &Bullet{BlastRadius: 0.32, TargetObjType: object.TypePlane}
	assert.True(t, bt.HasAirburst())

	bt.TargetObjType = object.TypeShip
	assert.False(t, bt.HasAirburst())

	bt.TargetObjType = object.TypePlane
	bt.BlastRadius = 0
	assert.False(t, bt.HasAirburst())
}
