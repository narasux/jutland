//go:build darwin

package magnify

/*
#cgo LDFLAGS: -framework Cocoa

#include "magnify_darwin.h"
*/
import "C"

// Init 向 macOS 注册 NSEventMaskMagnify 监听，pinch 手势会积累放大倍率。
func Init() {
	C.jutland_magnify_init()
}

// Poll 返回自上次调用以来积累的放大倍率并清零。
func Poll() float64 {
	return float64(C.jutland_magnify_poll())
}
