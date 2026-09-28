package audio

import (
	"bytes"
	"io"
	"testing"
)

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
