package bullet

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
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

func TestNewCopiesTemplateByValue(t *testing.T) {
	const name = "test/shallow-copy"
	original, existed := Map[name]
	t.Cleanup(func() {
		if existed {
			Map[name] = original
			return
		}
		delete(Map, name)
	})

	Map[name] = &Bullet{
		Name:     name,
		Type:     TypeShell,
		Diameter: 127,
		Damage:   42,
		CurPos:   objPos.NewR(99, 99),
	}

	b := New(
		name,
		objPos.NewR(3, 4),
		objPos.NewR(5, 6),
		"shooter",
		object.TypeShip,
		faction.HumanAlpha,
		ShotTypeDirect,
		object.TypeShip,
		1.5,
		20,
	)
	b.Damage = 99

	assert.Equal(t, 42.0, Map[name].Damage)
	assert.Equal(t, name, b.Name)
	assert.Equal(t, TypeShell, b.Type)
	assert.Equal(t, 127, b.Diameter)
	assert.Equal(t, 3.0, b.CurPos.RX)
	assert.Equal(t, 4.0, b.CurPos.RY)
	assert.Equal(t, ShotTypeDirect, b.ShotType)
	assert.Equal(t, 1.5, b.Speed)
	assert.Equal(t, 20, b.Life)
	assert.Equal(t, faction.HumanAlpha, b.BelongPlayer)
}
