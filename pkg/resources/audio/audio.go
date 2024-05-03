package audio

import (
	"log"

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
	NewMissionStartBackground()
	NewMissionSuccess()
	NewMissionFailed()
	NewMenuButtonClick()
	NewMenuButtonHover()

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
	return mustNewAudio("/start_bgm.wav")
}

// NewGameEndBackground 游戏结束 背景音乐
func NewGameEndBackground() types.AudioStream {
	return mustNewAudio("/end_bgm.wav")
}

// NewMenuBackground 菜单页面 背景音乐
func NewMenuBackground() types.AudioStream {
	return mustNewAudio("/menu_bgm.wav")
}

// NewMissionsBackground 任务选择 背景音乐 TODO 更换音频
func NewMissionsBackground() types.AudioStream {
	return mustNewAudio("/menu_bgm.wav")
}

// NewMissionStartBackground 任务开始 背景音乐 TODO 更换音频
func NewMissionStartBackground() types.AudioStream {
	return mustNewAudio("/menu_bgm.wav")
}

// NewMenuButtonHover 鼠标悬停菜单按钮
func NewMenuButtonHover() types.AudioStream {
	return mustNewAudio("/button_hover.wav")
}

// NewMenuButtonClick 鼠标点击菜单按钮
func NewMenuButtonClick() types.AudioStream {
	return mustNewAudio("/button_click.wav")
}

// NewMissionSuccess 任务成功
func NewMissionSuccess() types.AudioStream {
	return mustNewAudio("/mission_success.wav")
}

// NewMissionFailed 任务失败
func NewMissionFailed() types.AudioStream {
	return mustNewAudio("/mission_failed.wav")
}
