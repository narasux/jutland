package audio

import (
	"bytes"
	"io"
	"testing"
)

// TestNewGunFireReusesPCM 验证同口径炮声复用 init 阶段解码好的 PCM，而不是每次重新解码。
// 场景：305 命中 largeGunBulletDiameterThreshold，映射到 /fire/gun_large.wav，齐射频繁取用。
// 预期：NewGunFire(305) 返回的流共享 shortClips 的底层数组，两次读出的字节都等于缓存；
// 失败表示播放路径重复解码 WAV，会拉高开火帧耗时。
// 覆盖：先取 shortClips["/fire/gun_large.wav"] 确认非空，再两次 NewGunFire(305)，
// 比较首元素地址并 ReadAll 比对内容。
func TestNewGunFireReusesPCM(t *testing.T) {
	const path = "/fire/gun_large.wav"
	cached := shortClips[path]
	if len(cached) == 0 {
		t.Fatal("gun_large was not cached")
	}

	first := NewGunFire(305)
	second := NewGunFire(305)
	if &shortClips[path][0] != &cached[0] {
		t.Fatal("NewGunFire decoded the clip again")
	}

	firstBytes, err := io.ReadAll(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := io.ReadAll(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, cached) || !bytes.Equal(secondBytes, cached) {
		t.Fatal("played bytes differ from the cached PCM")
	}
}

// TestCachedReadersDoNotSharePosition 验证每次播放都新建独立读取位置，不共享缓存切片的读游标。
// 场景：同一帧两门炮播同一段 gun_large PCM；若返回同一 reader，第二声会从第一声末尾开始而静音。
// 预期：读完第一个流后，第二个流前 8 字节仍从开头读起，n=8 且等于 drained[:8]。
// 覆盖：两次 NewGunFire(305)，io.ReadAll 排空第一个（要求至少 8 字节），再对第二个 Read 8 字节比较。
func TestCachedReadersDoNotSharePosition(t *testing.T) {
	first := NewGunFire(305)
	second := NewGunFire(305)
	drained, err := io.ReadAll(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(drained) < 8 {
		t.Fatal("clip is too short")
	}

	head := make([]byte, 8)
	n, err := second.Read(head)
	if err != nil {
		t.Fatal(err)
	}
	if n != 8 || !bytes.Equal(head, drained[:8]) {
		t.Fatal("second reader did not start at the beginning")
	}
}
