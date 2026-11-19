package unit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestSimClockHook 验证无头模拟可以替换战斗路径的墙钟，并在恢复后回到真实时间。
// 场景：航母起飞 / 医疗船治疗原本读 time.Now，跨进程跑同 seed 会因墙钟不同而不一致。
// 预期：设置固定 12345ms 后 NowMillis 返回 12345；调用 restore 后返回值不再等于 12345。
// 覆盖：SetNowMillis(固定值) -> NowMillis 断言 -> restore -> NowMillis 断言，清理全局 hook。
func TestSimClockHook(t *testing.T) {
	restore := SetNowMillis(func() int64 { return 12345 })
	defer restore()
	require.Equal(t, int64(12345), NowMillis())

	restore()
	if NowMillis() == 12345 {
		t.Fatal("restore did not leave the fixed clock")
	}
}

// TestPlaneUIDGeneratorHook 验证 NewPlane 使用可注入的确定性 UID，而不是 uuid。
// 场景：飞机 UID 随机会让 OrderedPlanes/事件顺序跨进程不一致；模拟器需要稳定 ID。
// 预期：安装 seq 生成器后 NewPlane 得到 plane-1；恢复后 NextPlaneUID 不再返回 plane-2。
// 覆盖：PlaneMap 注册 test-plane，SetPlaneUIDGenerator 计数器，调用 NewPlane 两次断言 UID，
// 恢复 hook 后检查 NextPlaneUID 不等于内置序列的下一个值。
func TestPlaneUIDGeneratorHook(t *testing.T) {
	PlaneMap["test-plane-uid"] = &Plane{Name: "test-plane-uid", Type: PlaneTypeFighter}
	t.Cleanup(func() { delete(PlaneMap, "test-plane-uid") })

	seq := 0
	restore := SetPlaneUIDGenerator(func() string {
		seq++
		return fmt.Sprintf("plane-%d", seq)
	})
	defer restore()

	p1 := NewPlane("test-plane-uid", objPos.NewR(0, 0), 0, "ship", faction.HumanAlpha)
	p2 := NewPlane("test-plane-uid", objPos.NewR(0, 0), 0, "ship", faction.HumanAlpha)
	require.Equal(t, "plane-1", p1.Uid)
	require.Equal(t, "plane-2", p2.Uid)

	restore()
	if NextPlaneUID() == "plane-3" {
		t.Fatal("plane UID generator was not restored")
	}
}
