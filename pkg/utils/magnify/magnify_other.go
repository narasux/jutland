//go:build !darwin

package magnify

// Init 非 macOS 平台的空实现。
func Init() {}

// Poll 非 macOS 平台始终返回 0。
func Poll() float64 { return 0 }
