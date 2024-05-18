package audio

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/narasux/jutland/pkg/common/types"
)

// 音频采样率
const sampleRate = 48000

var Context = audio.NewContext(sampleRate)

// PlayAudioToEnd 音频播放（一次性使用，可并发，播放到完成，只能用于短音频）
func PlayAudioToEnd(ads types.AudioStream) {
	if ads.Length() > sampleRate*30 {
		log.Fatalf("audio too long for PlayAudioToEnd: %d", ads.Length())
	}
	go func() {
		p, _ := Context.NewPlayer(ads)
		p.Play()
	}()
}

// Player 音频播放
type Player struct {
	ctx    *audio.Context
	player *audio.Player
}

// NewPlayer ...
func NewPlayer(ctx *audio.Context) *Player {
	return &Player{ctx: ctx}
}

// Play 音频播放（如果有正在播放的则跳过）
func (p *Player) Play(ads types.AudioStream) {
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
func (p *Player) Pause() {
	p.player.Pause()
}

// Close 关闭音频播放
func (p *Player) Close() {
	if p.player == nil {
		return
	}
	if err := p.player.Close(); err != nil {
		log.Fatalf("failed to close audio: %s", err)
	}
	p.player = nil
}
