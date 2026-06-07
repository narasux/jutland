package state

import (
	"image"
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

const defaultZoom = 4

// AvailableZooms 是主战场支持的固定缩放档位。
var AvailableZooms = []int{1, 2, 4, 8, 16}

// DefaultZoom 返回任务主战场默认缩放档位
func DefaultZoom() int {
	return defaultZoom
}

// NormalizeZoom 将任意缩放值修正到支持的档位
func NormalizeZoom(zoom int) int {
	for _, availableZoom := range AvailableZooms {
		if zoom == availableZoom {
			return zoom
		}
	}
	return defaultZoom
}

// ZoomScale 返回主战场显示倍率，Zoom=4 表示 1x
func (s *MissionState) ZoomScale() float64 {
	return float64(NormalizeZoom(s.UI.GameOpts.Zoom)) / float64(defaultZoom)
}

// MapBlockDisplaySize 返回当前缩放档位下一个地图格对应的屏幕像素
func (s *MissionState) MapBlockDisplaySize() float64 {
	return float64(constants.MapBlockSize) * s.ZoomScale()
}

// CameraPosToScreen 将地图坐标转换为主战场屏幕坐标
func (s *MissionState) CameraPosToScreen(pos objPos.MapPos) (float64, float64) {
	blockSize := s.MapBlockDisplaySize()
	return (pos.RX - s.View.Camera.Pos.RX) * blockSize,
		(pos.RY - s.View.Camera.Pos.RY) * blockSize
}

// ScreenToCameraPos 将主战场屏幕坐标转换为地图坐标
func (s *MissionState) ScreenToCameraPos(sx, sy float64) objPos.MapPos {
	blockSize := s.MapBlockDisplaySize()
	return objPos.NewR(
		s.View.Camera.Pos.RX+sx/blockSize,
		s.View.Camera.Pos.RY+sy/blockSize,
	)
}

// ScreenPointToCameraPos 将主战场屏幕坐标转换为地图坐标
func (s *MissionState) ScreenPointToCameraPos(pos image.Point) objPos.MapPos {
	return s.ScreenToCameraPos(float64(pos.X), float64(pos.Y))
}

// RefreshCameraSize 按当前缩放档位刷新相机视野范围
func (s *MissionState) RefreshCameraSize() {
	blockSize := s.MapBlockDisplaySize()
	s.View.Camera.Width = int(math.Ceil(float64(s.View.Layout.Width)/blockSize)) + 1
	s.View.Camera.Height = int(math.Ceil(float64(s.View.Layout.Height)/blockSize)) + 1
	s.View.Camera.Pos.EnsureBorder(s.CameraPosBorder())
}

// StepZoomAtScreenPoint 按固定档位缩放，并保持屏幕点下的地图位置尽量不变
func (s *MissionState) StepZoomAtScreenPoint(direction int, sx, sy int) {
	curZoom := NormalizeZoom(s.UI.GameOpts.Zoom)
	curIndex := 0
	for idx, zoom := range AvailableZooms {
		if zoom == curZoom {
			curIndex = idx
			break
		}
	}

	nextIndex := curIndex + direction
	if nextIndex < 0 || nextIndex >= len(AvailableZooms) {
		return
	}

	anchor := s.ScreenToCameraPos(float64(sx), float64(sy))
	s.UI.GameOpts.Zoom = AvailableZooms[nextIndex]
	s.RefreshCameraSize()

	blockSize := s.MapBlockDisplaySize()
	nextPos := s.View.Camera.Pos.Copy()
	nextPos.AssignRxy(
		anchor.RX-float64(sx)/blockSize,
		anchor.RY-float64(sy)/blockSize,
	)
	nextPos.EnsureBorder(s.CameraPosBorder())
	s.View.Camera.Pos = nextPos
}
