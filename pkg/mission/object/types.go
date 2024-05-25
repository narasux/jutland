package object

// FiringArc 火炮射界
type FiringArc struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Contains 是否在射界内
func (f *FiringArc) Contains(angle float64) bool {
	return f.Start <= angle && angle <= f.End
}
