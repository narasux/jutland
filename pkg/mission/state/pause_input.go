package state

// PauseInput 表示暂停面板中的一次退出/恢复操作输入。
type PauseInput int

const (
	// PauseInputNone 表示没有触发暂停面板操作。
	PauseInputNone PauseInput = iota
	// PauseInputResume 表示恢复任务或取消放弃确认。
	PauseInputResume
	// PauseInputQuit 表示请求放弃任务或确认放弃任务。
	PauseInputQuit
)

// ApplyPauseInput 根据当前暂停确认状态计算下一帧任务状态。
func ApplyPauseInput(status MissionStatus, confirmQuit bool, input PauseInput) (MissionStatus, bool) {
	if status != MissionPaused {
		return status, false
	}

	switch input {
	case PauseInputResume:
		if confirmQuit {
			return MissionPaused, false
		}
		return MissionRunning, false
	case PauseInputQuit:
		if confirmQuit {
			return MissionFailed, true
		}
		return MissionPaused, true
	default:
		return status, confirmQuit
	}
}
