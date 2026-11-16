package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// TestGunApplyAirburst 验证近炸引信只给对空炮弹写入 proximity/blast 参数。
// 场景：高射炮对飞机开火时应生成空爆弹；对舰平射炮弹不应带空爆参数；无配置的炮不应产生爆炸。
// 预期：对空弹 ProximityRadius/BlastRadius 等于炮配置且 HasAirburst；对舰弹两者为 0；
// 空配置炮对空也不爆炸。
// 覆盖：构造三组 Gun/Bullet，调用 applyAirburst 后逐字段断言。
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
