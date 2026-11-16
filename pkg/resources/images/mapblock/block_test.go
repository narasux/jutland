package mapblock

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestWaterVariantsMatchMD5 验证预计算的海面花纹字节与逐格 MD5 算法完全一致。
// 场景：地图 4×3，海面按 byte%7、深海按 byte%3 取贴图；缓存与现算不同会让同一格换图。
// 预期：每个 (x,y) 的字节等于 waterVariantByte 的 MD5 首字节，对 7/3 套贴图的取模也一致；
// 越界 (-1,0) 回退现算；失败表示海面花纹会随预热时机跳变。
// 覆盖：initWaterVariants(4,3) 后双重循环比对字节与两种取模索引，再单独检查负坐标分支。
func TestWaterVariantsMatchMD5(t *testing.T) {
	cache := &sceneBlockCache{}
	cache.initWaterVariants(4, 3)

	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			got := cache.waterVariant(x, y)
			want := waterVariantByte(x, y)
			if got != want {
				t.Fatalf("variant (%d,%d) = %d, want %d", x, y, got, want)
			}
			if int(got)%seaBlockCount != int(want)%seaBlockCount {
				t.Fatalf("sea index (%d,%d) = %d, want %d", x, y, int(got)%seaBlockCount, int(want)%seaBlockCount)
			}
			if int(got)%deepSeaBlockCount != int(want)%deepSeaBlockCount {
				t.Fatalf(
					"deep sea index (%d,%d) = %d, want %d",
					x,
					y,
					int(got)%deepSeaBlockCount,
					int(want)%deepSeaBlockCount,
				)
			}
		}
	}

	if cache.waterVariant(-1, 0) != waterVariantByte(-1, 0) {
		t.Fatal("out-of-range variant should use the same MD5 byte")
	}
}

// TestCellKeySeparatesNegativeCoordinates 验证 cellKey/queueKey 不会把负坐标折叠成正坐标。
// 场景：视野外扩会出现 (-1,0) 这类负格；若键只做简单拼接或截断，负格会覆盖正格缓存。
// 预期：cellKey(-1,0)!=cellKey(1,0)、queueKey(-1,0,4)!=queueKey(1,0,4)；
// 缓存分别放入 1×1 图后 Get(-1,0)/Get(1,0) 各取各的，Get(2,3) 也能命中已存块。
// 覆盖：直接比较键，再用 data map 手工塞入 left/right 两张图并调用 Get 验证不串格。
func TestCellKeySeparatesNegativeCoordinates(t *testing.T) {
	if cellKey(-1, 0) == cellKey(1, 0) {
		t.Fatal("(-1,0) and (1,0) share a cell key")
	}
	if queueKey(-1, 0, 4) == queueKey(1, 0, 4) {
		t.Fatal("(-1,0) and (1,0) share a queue key")
	}

	cache := &sceneBlockCache{data: map[uint64]*ebiten.Image{}}
	left := ebiten.NewImage(1, 1)
	right := ebiten.NewImage(1, 1)
	cache.data[cellKey(-1, 0)] = left
	cache.data[cellKey(1, 0)] = right
	cache.data[cellKey(2, 3)] = right

	if cache.Get(-1, 0) != left || cache.Get(1, 0) != right {
		t.Fatal("negative and positive coordinates collided")
	}
	if cache.Get(2, 3) != right {
		t.Fatal("stored block was not found")
	}
}

// TestMissingZoomCacheStaysMissingWithoutBudget 验证零预算时预热不生成缩放块，缺口仍被报告缺失。
// 场景：帧预算被其他系统耗尽，StepPrewarm 拿到 budget=0；既不能抢跑建图，也不能丢任务。
// 预期：SchedulePrewarmAround(...,zooms=[4],margin=0) 入队 1 个任务；StepPrewarm(0,0) 返回 0；
// HasMissingAround(...,4,0) 仍为 true；失败表示零预算会偷偷建块或提前丢任务。
// 覆盖：PutBaseBlock(0,0,8×8) 造底图，排队后检查 PrewarmQueueLen、StepPrewarm 返回值与缺口探测。
func TestMissingZoomCacheStaysMissingWithoutBudget(t *testing.T) {
	cache := &sceneBlockCache{data: map[uint64]*ebiten.Image{}}
	cache.PutBaseBlock(0, 0, ebiten.NewImage(8, 8))
	cache.SchedulePrewarmAround(0, 0, 0, 0, []int{4}, 0)
	if cache.PrewarmQueueLen() != 1 {
		t.Fatalf("queue = %d, want 1", cache.PrewarmQueueLen())
	}
	if cache.StepPrewarm(0, 0) != 0 {
		t.Fatal("zero budget should not build a zoom block")
	}
	if !cache.HasMissingAround(0, 0, 0, 0, 4, 0) {
		t.Fatal("uncached land should stay missing")
	}
}
