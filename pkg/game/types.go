package game

type GameMode int

const (
	// 游戏开始
	GameModeStart = iota
	// 菜单选择
	GameModeMenuSelect
	// 任务选择
	GameModeMissionSelect
	// 任务加载
	GameModeMissionLoading
	// 任务开始
	GameModeMissionStart
	// 任务进行中
	GameModeMissionRunning
	// 任务结束 - 成功
	GameModeMissionSuccess
	// 任务结束 - 失败
	GameModeMissionFailed
	// 游戏图鉴
	GameModeCollection
	// 游戏设置
	GameModeGameSetting
	// 游戏结束
	GameModeEnd
)

const (
	// 音频采样率
	SampleRate = 48000
)
