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

// TestShortVoicePoolCapsAtEight 验证短音效并发池最多保留 maxConcurrentShortSounds=8 个播放器。
// 场景：齐射时一帧内触发大量炮声；池满后新声音应被丢弃，而不是无限创建 player 拖垮音频上下文。
// 预期：连续播放 8 个 0.1 秒静音后 players 恰为 8；第 9 次调用后仍为 8。
// 覆盖：memStream 包装 AudioSampleRate/10 个 4 字节静音帧，循环 PlayAudioToEnd；
// Cleanup 关闭所有 player 并清空 voices.players。
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
