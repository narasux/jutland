package audio

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/common/types"
	"github.com/narasux/jutland/pkg/loader"
)

// 对局里会反复播放的短音效。启动时解码一次，长 BGM 不放进这里。
var shortAudioPaths = []string{
	"/button_hover.wav",
	"/button_click.wav",
	"/loaded.wav",
	"/cheating.wav",
	"/hit/ship_explode.wav",
	"/fire/gun_silent.wav",
	"/fire/gun_small.wav",
	"/fire/gun_medium.wav",
	"/fire/gun_large.wav",
	"/fire/gun_rail.wav",
	"/fire/bomb_spawn.wav",
	"/fire/torpedo_launch.wav",
	"/fire/rocketspawn.wav",
	"/hit/rocketgroundexplo.wav",
}

var shortClips = map[string][]byte{}

func init() {
	log.Println("testing audio resources...")

	for _, path := range shortAudioPaths {
		cacheShortClip(path)
	}

	// 测试资源是否正确加载。长背景音乐仍在每次播放时解码。
	NewGameStartBackground()
	NewGameEndBackground()
	NewMenuBackground()
	NewMissionsBackground()
	NewMissionSuccess()
	NewMissionFailed()
	NewMenuButtonClick()
	NewMenuButtonHover()
	// 港口背景音乐
	NewHarborUS()
	NewHarborJP()
	NewHarborUK()
	NewHarborGEM()
	NewHarborFR()
	NewHarborNeutral()

	log.Println("audio resources tested")
}

// pcmStream 在同一段 PCM 上新建读取位置，避免两个 player 共用一个 reader。
type pcmStream struct {
	*bytes.Reader
}

func (s *pcmStream) Length() int64 {
	return s.Size()
}

func cacheShortClip(path string) {
	stream := mustNewAudio(path)
	if stream.Length() > constants.AudioSampleRate*30 {
		log.Fatalf("audio too long to cache %s: %d", path, stream.Length())
	}
	buf := make([]byte, stream.Length())
	if _, err := io.ReadFull(stream, buf); err != nil {
		log.Fatalf("decode %s: %s", path, err)
	}
	shortClips[path] = buf
}

func mustCachedAudio(path string) types.AudioStream {
	data, ok := shortClips[path]
	if !ok {
		log.Fatalf("short audio %s was not cached", path)
	}
	return &pcmStream{Reader: bytes.NewReader(data)}
}

// 由于同一 Audio 资源不能被多个 player 同时播放，因此每次都给新的实例
func mustNewAudio(audioPath string) types.AudioStream {
	ads, err := loader.LoadAudio(audioPath)
	if err != nil {
		log.Fatalf("missing %s: %s", audioPath, err)
	}
	return ads
}

// NewGameStartBackground 游戏开始 背景音乐
func NewGameStartBackground() types.AudioStream {
	return mustNewAudio("/bgm/start_bgm.mp3")
}

// NewGameEndBackground 游戏结束 背景音乐
func NewGameEndBackground() types.AudioStream {
	return mustNewAudio("/bgm/end_bgm.mp3")
}

// NewMenuBackground 菜单页面 背景音乐
func NewMenuBackground() types.AudioStream {
	return mustNewAudio("/bgm/menu_bgm.mp3")
}

// NewMissionsBackground 任务选择 背景音乐
func NewMissionsBackground() types.AudioStream {
	return mustNewAudio("/bgm/mission_bgm.mp3")
}

// NewMenuButtonHover 鼠标悬停菜单按钮
func NewMenuButtonHover() types.AudioStream {
	return mustCachedAudio("/button_hover.wav")
}

// NewMenuButtonClick 鼠标点击菜单按钮
func NewMenuButtonClick() types.AudioStream {
	return mustCachedAudio("/button_click.wav")
}

// NewMissionLoaded 关卡加载完成
func NewMissionLoaded() types.AudioStream {
	return mustCachedAudio("/loaded.wav")
}

// NewCheating 开始作弊
func NewCheating() types.AudioStream {
	return mustCachedAudio("/cheating.wav")
}

// NewMissionSuccess 任务成功
func NewMissionSuccess() types.AudioStream {
	return mustNewAudio("/bgm/mission_success.mp3")
}

// NewMissionFailed 任务失败
func NewMissionFailed() types.AudioStream {
	return mustNewAudio("/bgm/mission_failed.mp3")
}

// NewShipExplode 战舰爆炸
func NewShipExplode() types.AudioStream {
	return mustCachedAudio("/hit/ship_explode.wav")
}

// NewHarborUS 美国港口
func NewHarborUS() types.AudioStream {
	return mustNewAudio("/bgm/harbor_us.mp3")
}

// NewHarborJP 日本港口
func NewHarborJP() types.AudioStream {
	return mustNewAudio("/bgm/harbor_jp.mp3")
}

// NewHarborUK 英国港口
func NewHarborUK() types.AudioStream {
	return mustNewAudio("/bgm/harbor_uk.mp3")
}

// NewHarborGEM 德国港口
func NewHarborGEM() types.AudioStream {
	return mustNewAudio("/bgm/harbor_gem.mp3")
}

// NewHarborFR 法国港口
func NewHarborFR() types.AudioStream {
	return mustNewAudio("/bgm/harbor_fr.mp3")
}

// NewHarborNeutral 中立港口
func NewHarborNeutral() types.AudioStream {
	return mustNewAudio("/bgm/harbor_neutral.mp3")
}

// 轨道炮
const railGunBulletDiameter = 1024

const (
	// 大口径炮弹阈值
	largeGunBulletDiameterThreshold = 305
	// 中口径炮弹
	mediumGunBulletDiameterThreshold = 152
	// 小口径炮弹
	smallGunBulletDiameterThreshold = 127
)

// NewGunFire 火炮开火
func NewGunFire(bulletDiameter int) types.AudioStream {
	audioType := "silent"
	if bulletDiameter == railGunBulletDiameter {
		audioType = "rail"
	} else if bulletDiameter >= largeGunBulletDiameterThreshold {
		audioType = "large"
	} else if bulletDiameter >= mediumGunBulletDiameterThreshold {
		audioType = "medium"
	} else if bulletDiameter >= smallGunBulletDiameterThreshold {
		audioType = "small"
	}
	return mustCachedAudio(fmt.Sprintf("/fire/gun_%s.wav", audioType))
}

// NewBombSpawn 炸弹投放
func NewBombSpawn() types.AudioStream {
	return mustCachedAudio("/fire/bomb_spawn.wav")
}

// NewTorpedoLaunch 鱼雷发射
func NewTorpedoLaunch() types.AudioStream {
	return mustCachedAudio("/fire/torpedo_launch.wav")
}

// NewRocketSpawn 火箭弹发射
func NewRocketSpawn() types.AudioStream {
	return mustCachedAudio("/fire/rocketspawn.wav")
}

// NewRocketExplode 火箭弹爆炸
func NewRocketExplode() types.AudioStream {
	return mustCachedAudio("/hit/rocketgroundexplo.wav")
}
