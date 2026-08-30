package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

func TestGunApplyAirburst(t *testing.T) {
	gun := &Gun{ProximityRadius: 0.20, BlastRadius: 0.32}

	planeShell := &objBullet.Bullet{TargetObjType: object.TypePlane}
	gun.applyAirburst(planeShell, object.TypePlane)
	assert.InDelta(t, 0.20, planeShell.ProximityRadius, 1e-9)
	assert.InDelta(t, 0.32, planeShell.BlastRadius, 1e-9)
	assert.True(t, planeShell.HasAirburst())

	shipShell := &objBullet.Bullet{}
	gun.applyAirburst(shipShell, object.TypeShip)
	assert.Equal(t, 0.0, shipShell.ProximityRadius)
	assert.Equal(t, 0.0, shipShell.BlastRadius)

	ww2 := &Gun{}
	plain := &objBullet.Bullet{}
	ww2.applyAirburst(plain, object.TypePlane)
	assert.Equal(t, 0.0, plain.BlastRadius)
}
