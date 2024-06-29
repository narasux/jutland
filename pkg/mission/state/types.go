package state

type MissionStatus string

const (
	// MissionRunning 任务进行中
	MissionRunning MissionStatus = "running"
	// MissionSuccess 任务成功
	MissionSuccess MissionStatus = "success"
	// MissionFailed 任务失败
	MissionFailed MissionStatus = "failed"
	// MissionPaused 任务暂停
	MissionPaused MissionStatus = "paused"
	// MissionError 任务错误
	MissionError MissionStatus = "error"
)

type MapDisplayMode int

const (
	// MapDisplayModeNone 不显示
	MapDisplayModeNone MapDisplayMode = iota
	// MapDisplayModeFull 全屏现实
	MapDisplayModeFull
)
