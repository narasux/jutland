package mapblock

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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
				t.Fatalf("deep sea index (%d,%d) = %d, want %d", x, y, int(got)%deepSeaBlockCount, int(want)%deepSeaBlockCount)
			}
		}
	}

	if cache.waterVariant(-1, 0) != waterVariantByte(-1, 0) {
		t.Fatal("out-of-range variant should use the same MD5 byte")
	}
}

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
