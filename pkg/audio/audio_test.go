package audio

import (
	"bytes"
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
)

type memStream struct {
	*bytes.Reader
}

func (s *memStream) Length() int64 {
	return s.Size()
}

func TestShortVoicePoolCapsAtEight(t *testing.T) {
	t.Cleanup(func() {
		for _, player := range voices.players {
			_ = player.Close()
		}
		voices.players = nil
	})
	voices.players = nil

	// 十分之一秒的静音，调用返回时仍然算正在播放。
	silent := bytes.Repeat([]byte{0, 0, 0, 0}, constants.AudioSampleRate/10)
	play := func() {
		PlayAudioToEnd(&memStream{Reader: bytes.NewReader(silent)})
	}
	for range maxConcurrentShortSounds {
		play()
	}
	if len(voices.players) != maxConcurrentShortSounds {
		t.Fatalf("players = %d, want %d", len(voices.players), maxConcurrentShortSounds)
	}

	play()
	if len(voices.players) != maxConcurrentShortSounds {
		t.Fatalf("players = %d, want the cap to hold", len(voices.players))
	}
}
