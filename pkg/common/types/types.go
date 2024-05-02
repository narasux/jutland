package types

import "io"

// AudioStream 音频流
type AudioStream interface {
	io.ReadSeeker
	Length() int64
}
