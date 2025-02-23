package audio

import (
	"fmt"
	"log"
	"slices"

	"github.com/narasux/jutland/pkg/common/types"
	"github.com/narasux/jutland/pkg/loader"
)

func init() {
	log.Println("testing audio resources...")

	// 测试资源是否正确加载
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
	return mustNewAudio("/button_hover.wav")
}

// NewMenuButtonClick 鼠标点击菜单按钮
func NewMenuButtonClick() types.AudioStream {
	return mustNewAudio("/button_click.wav")
}

// NewMissionLoaded 关卡加载完成
func NewMissionLoaded() types.AudioStream {
	return mustNewAudio("/loaded.wav")
}

// NewCheating 开始作弊
func NewCheating() types.AudioStream {
	return mustNewAudio("/cheating.wav")
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
	return mustNewAudio("/hit/ship_explode.wav")
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

// 大口径炮弹
var largeGunBulletDiameter = []int{460, 406, 381, 356, 305}

// 中口径炮弹
var mediumGunBulletDiameter = []int{203, 155, 152}

// 小口径炮弹
var smallGunBulletDiameter = []int{140, 127}

// NewGunFire 火炮开火
func NewGunFire(bulletDiameter int) types.AudioStream {
	audioType := "silent"
	if bulletDiameter == railGunBulletDiameter {
		audioType = "rail"
	} else if slices.Contains(largeGunBulletDiameter, bulletDiameter) {
		audioType = "large"
	} else if slices.Contains(mediumGunBulletDiameter, bulletDiameter) {
		audioType = "medium"
	} else if slices.Contains(smallGunBulletDiameter, bulletDiameter) {
		audioType = "small"
	}
	return mustNewAudio(fmt.Sprintf("/fire/gun_%s.wav", audioType))
}

// NewBombSpawn 炸弹投放
func NewBombSpawn() types.AudioStream {
	return mustNewAudio("/fire/bomb_spawn.wav")
}

// NewTorpedoLaunch 鱼雷发射
func NewTorpedoLaunch() types.AudioStream {
	return mustNewAudio("/fire/torpedo_launch.wav")
}
