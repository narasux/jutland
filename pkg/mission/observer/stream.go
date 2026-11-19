package observer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StreamEvent is the JSONL envelope written by StreamRecorder.
type StreamEvent struct {
	Type  string `json:"type"`
	Event any    `json:"event"`
}

// StreamRecorder writes observer events as JSONL. It is intended for debug
// recordings only; the game keeps it nil unless explicitly enabled.
type StreamRecorder struct {
	mu     sync.Mutex
	file   *os.File
	writer *bufio.Writer
	counts map[string]int
	err    error
}

// NewStreamRecorder writes to an existing writer; Close is a no-op for the writer.
func NewStreamRecorder(w io.Writer) *StreamRecorder {
	return &StreamRecorder{writer: bufio.NewWriter(w), counts: map[string]int{}}
}

// NewFileRecorder creates dir/record-<timestamp>.jsonl.
func NewFileRecorder(dir string) (*StreamRecorder, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("record-%s.jsonl", time.Now().Format("20060102-150405")))
	file, err := os.Create(path)
	if err != nil {
		return nil, "", err
	}
	rec := NewStreamRecorder(file)
	rec.file = file
	return rec, path, nil
}

func (r *StreamRecorder) write(kind string, event any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return
	}
	payload, err := json.Marshal(StreamEvent{Type: kind, Event: event})
	if err != nil {
		r.err = err
		return
	}
	if _, err := r.writer.Write(append(payload, '\n')); err != nil {
		r.err = err
		return
	}
	r.counts[kind]++
}

// OnShot records a shot event.
func (r *StreamRecorder) OnShot(event Shot) { r.write("shot", event) }

// OnHit records a hit event.
func (r *StreamRecorder) OnHit(event Hit) { r.write("hit", event) }

// OnSink records a sink event.
func (r *StreamRecorder) OnSink(event Sink) { r.write("sink", event) }

// OnLoss records an aircraft loss event.
func (r *StreamRecorder) OnLoss(event Loss) { r.write("loss", event) }

// Counts returns event counts by type.
func (r *StreamRecorder) Counts() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]int, len(r.counts))
	for k, v := range r.counts {
		out[k] = v
	}
	return out
}

// Err returns the first write/marshal error.
func (r *StreamRecorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// Close flushes and closes the underlying file.
func (r *StreamRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.writer != nil {
		if err := r.writer.Flush(); err != nil && r.err == nil {
			r.err = err
		}
	}
	if r.file != nil {
		if err := r.file.Close(); err != nil && r.err == nil {
			r.err = err
		}
	}
	return r.err
}

var _ Observer = (*StreamRecorder)(nil)
