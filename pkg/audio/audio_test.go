package audio

import (
	"bytes"
	"math"
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

// TestAudioVolumesClamp 验证 BGM 与战场音效音量分别夹到 [0,1]，NaN 视为静音。
// 场景：设置页两个滑杆或手写配置可能传入 -0.5 / 1.5 / NaN，而 audio.Player.SetVolume 只接受 [0,1]。
// 预期：两通道负数与 NaN 得到 0、超过 1 得到 1，且设置 BGM 不改变音效值；失败说明非法音量进入播放器或两通道串线。
// 覆盖：保存并恢复两个全局音量，依次 SetBGMVolume(-0.5/1.5/NaN)、SetEffectsVolume(-0.5/0.25/1.5/NaN) 后逐次断言 getter。
func TestAudioVolumesClamp(t *testing.T) {
	previousBGM, previousEffects := BGMVolume(), EffectsVolume()
	t.Cleanup(func() {
		SetBGMVolume(previousBGM)
		SetEffectsVolume(previousEffects)
	})

	SetEffectsVolume(0.25)
	SetBGMVolume(-0.5)
	if got := BGMVolume(); got != 0 {
		t.Fatalf("negative BGM volume = %v, want 0", got)
	}
	if got := EffectsVolume(); got != 0.25 {
		t.Fatalf("BGM change must not touch effects volume, got %v", got)
	}
	SetBGMVolume(1.5)
	if got := BGMVolume(); got != 1 {
		t.Fatalf("oversized BGM volume = %v, want 1", got)
	}
	SetBGMVolume(math.NaN())
	if got := BGMVolume(); got != 0 {
		t.Fatalf("NaN BGM volume = %v, want 0", got)
	}
	SetEffectsVolume(-0.5)
	if got := EffectsVolume(); got != 0 {
		t.Fatalf("negative effects volume = %v, want 0", got)
	}
	SetEffectsVolume(1.5)
	if got := EffectsVolume(); got != 1 {
		t.Fatalf("oversized effects volume = %v, want 1", got)
	}
	SetEffectsVolume(math.NaN())
	if got := EffectsVolume(); got != 0 {
		t.Fatalf("NaN effects volume = %v, want 0", got)
	}
}
