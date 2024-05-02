package game

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/narasux/jutland/pkg/common/types"
)

// AudioPlayer 音频播放
type AudioPlayer struct {
	ctx    *audio.Context
	player *audio.Player
}

// NewAudioPlayer ...
func NewAudioPlayer(ctx *audio.Context) *AudioPlayer {
	return &AudioPlayer{ctx: ctx}
}

// PlayAudioToEnd 音频播放（一次性使用，可并发，播放到完成，只能用于短音频）
func PlayAudioToEnd(ctx *audio.Context, ads types.AudioStream) {
	if ads.Length() > SampleRate*5 {
		log.Fatalf("audio too long for PlayAudioToEnd: %d", ads.Length())
	}
	p, _ := ctx.NewPlayer(ads)
	p.Play()
}

// Play 音频播放（如果有正在播放的则跳过）
func (p *AudioPlayer) Play(ads types.AudioStream) {
	// 确保先手动 Close 之后再播放下一个
	if p.player != nil && p.player.IsPlaying() {
		return
	}
	var err error
	p.player, err = p.ctx.NewPlayer(ads)
	if err != nil {
		log.Fatalf("failed to play audio: %s", err)
	}
	p.player.Play()
}

// Pause 暂停音频播放
func (p *AudioPlayer) Pause() {
	p.player.Pause()
}

// Close 关闭音频播放
func (p *AudioPlayer) Close() {
	if p.player == nil {
		return
	}
	if err := p.player.Close(); err != nil {
		log.Fatalf("failed to close audio: %s", err)
	}
	p.player = nil
}
