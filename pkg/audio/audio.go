package audio

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/common/types"
)

var Context = audio.NewContext(constants.AudioSampleRate)

const maxConcurrentShortSounds = 8

// 背景音乐与战场音效分别控制，1.0 为原始音量。
var (
	bgmVolume     = 1.0
	effectsVolume = 1.0
	livePlayers   []*Player
)

func clampVolume(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}
	return min(value, 1)
}

// BGMVolume 返回当前背景音乐音量。
func BGMVolume() float64 { return bgmVolume }

// EffectsVolume 返回当前战场音效音量。
func EffectsVolume() float64 { return effectsVolume }

// SetBGMVolume 设置背景音乐音量并同步到已经创建的播放器，便于设置页实时试听。
func SetBGMVolume(volume float64) {
	bgmVolume = clampVolume(volume)
	for _, player := range livePlayers {
		player.applyVolume()
	}
}

// SetEffectsVolume 设置战场音效音量，对之后播放的火炮 / 鱼雷 / 爆炸等生效。
func SetEffectsVolume(volume float64) {
	effectsVolume = clampVolume(volume)
}

// shortVoicePool 管理同时播放的短音效，播完就关掉，满了就丢掉新的一声。
type shortVoicePool struct {
	players []*audio.Player
}

var voices shortVoicePool

func (p *shortVoicePool) Play(ads types.AudioStream, volume float64) {
	alive := p.players[:0]
	for _, player := range p.players {
		if player.IsPlaying() {
			alive = append(alive, player)
			continue
		}
		if err := player.Close(); err != nil {
			log.Fatal("failed to close audio: ", err)
		}
	}
	p.players = alive
	if len(p.players) >= maxConcurrentShortSounds {
		return
	}

	player, err := Context.NewPlayer(ads)
	if err != nil {
		log.Fatal("failed to play audio: ", err)
	}
	player.SetVolume(clampVolume(volume * effectsVolume))
	player.Play()
	p.players = append(p.players, player)
}

// PlayAudioToEnd 音频播放（一次性使用，可并发，播放到完成，只能用于短音频）
func PlayAudioToEnd(ads types.AudioStream) {
	PlayAudioToEndWithVolume(ads, 0.3)
}

// PlayAudioToEndWithVolume 按指定音量播放短音频，适合源文件响度不一致的局部修正。
func PlayAudioToEndWithVolume(ads types.AudioStream, volume float64) {
	if ads.Length() > constants.AudioSampleRate*30 {
		log.Fatalf("audio too long for PlayAudioToEnd: %d", ads.Length())
	}
	voices.Play(ads, volume)
}

// Player 音频播放
type Player struct {
	ctx        *audio.Context
	player     *audio.Player
	baseVolume float64
}

// NewPlayer ...
func NewPlayer(ctx *audio.Context) *Player {
	return &Player{ctx: ctx, baseVolume: 0.3}
}

// register 把正在出声的播放器挂到背景音乐音量列表，关闭时会自动摘除。
func (p *Player) register() {
	for _, player := range livePlayers {
		if player == p {
			return
		}
	}
	livePlayers = append(livePlayers, p)
}

// applyVolume 把背景音乐音量叠加到播放器的基准音量上。
func (p *Player) applyVolume() {
	if p.player == nil {
		return
	}
	p.player.SetVolume(clampVolume(p.baseVolume * bgmVolume))
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
		log.Fatal("failed to play audio: ", err)
	}
	p.register()
	p.applyVolume()
	p.player.Play()
}

// PlayLazy 仅在当前没有音频播放时创建新的音频流。
func (p *Player) PlayLazy(factory func() types.AudioStream) {
	if p.player != nil && p.player.IsPlaying() {
		return
	}
	p.Play(factory())
}

// Pause 暂停音频播放
func (p *Player) Pause() {
	p.player.Pause()
}

// Close 关闭音频播放
func (p *Player) Close() {
	for idx, player := range livePlayers {
		if player == p {
			livePlayers = append(livePlayers[:idx], livePlayers[idx+1:]...)
			break
		}
	}
	if p.player == nil {
		return
	}
	if err := p.player.Close(); err != nil {
		log.Fatal("failed to close audio: ", err)
	}
	p.player = nil
}
