package unit

// ShipAnimation 描述舰船俯视图逐帧动画。
type ShipAnimation struct {
	TopFrames    []string `json:"topFrames"`
	IdleTopFrame string   `json:"idleTopFrame"`
	FrameTicks   float64  `json:"frameTicks"`
}

// AdvanceAnimation 按模拟速度推进舰船动画。
func (s *BattleShip) AdvanceAnimation(multiplier float64) {
	if len(s.Animation.TopFrames) == 0 || s.CurSpeed <= 0 {
		s.AnimationAge = 0
		s.LastTrailAnimationStep = -1
		return
	}
	s.AnimationAge += max(0, multiplier)
}

// CurrentTopImageName 返回场景中当前应绘制的俯视资源名。
func (s *BattleShip) CurrentTopImageName() string {
	if len(s.Animation.TopFrames) == 0 {
		return s.Name
	}
	if s.CurSpeed <= 0 && s.Animation.IdleTopFrame != "" {
		return s.Animation.IdleTopFrame
	}
	frameTicks := s.Animation.FrameTicks
	if frameTicks <= 0 {
		frameTicks = 1
	}
	idx := int(s.AnimationAge/frameTicks) % len(s.Animation.TopFrames)
	return s.Animation.TopFrames[idx]
}
