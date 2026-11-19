package observer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

// TestStreamRecorderWritesJSONL 验证调试录制输出逐行 JSONL，类型/事件可解码。
// 场景：Shot/Hit/Sink/Loss 各一条写入 bytes.Buffer，Close 刷新。
// 预期：4 行、type 分别为 shot/hit/sink/loss，Counts 各 1，Err 为空。
// 覆盖：NewStreamRecorder + 四类 On* + Close 的序列化路径。
func TestStreamRecorderWritesJSONL(t *testing.T) {
	var buf bytes.Buffer
	rec := NewStreamRecorder(&buf)
	rec.OnShot(Shot{Tick: 1})
	rec.OnHit(Hit{Tick: 2})
	rec.OnSink(Sink{Tick: 3})
	rec.OnLoss(Loss{Tick: 4})
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}
	lines := 0
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var event StreamEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lines != 4 {
		t.Fatalf("lines = %d, want 4", lines)
	}
	counts := rec.Counts()
	for _, kind := range []string{"shot", "hit", "sink", "loss"} {
		if counts[kind] != 1 {
			t.Fatalf("counts[%s] = %d", kind, counts[kind])
		}
	}
	if rec.Err() != nil {
		t.Fatalf("Err = %v", rec.Err())
	}
}
