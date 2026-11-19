package observer

import "testing"

// TestRecorderCollectsAndResets 验证内存 Recorder 按调用顺序保存四类事件，并能清空复用。
// 场景：headless 采集器每个 tick 可能收到 Shot/Hit/Sink/Loss 混合事件；下一局或下一个 seed 要复用对象。
// 预期：四类切片长度各为 1 且字段不被改写；Reset 后长度全部为 0，仍可继续 append。
// 覆盖：依次调用 OnShot/OnHit/OnSink/OnLoss，断言切片内容，再 Reset 并追加一条 Shot。
func TestRecorderCollectsAndResets(t *testing.T) {
	r := &Recorder{}
	r.OnShot(Shot{WeaponName: "5in/38"})
	r.OnHit(Hit{TargetUID: "t"})
	r.OnSink(Sink{TargetUID: "s"})
	r.OnLoss(Loss{TargetUID: "p"})
	if len(r.Shots) != 1 || r.Shots[0].WeaponName != "5in/38" {
		t.Fatalf("shots = %+v", r.Shots)
	}
	if len(r.Hits) != 1 || r.Hits[0].TargetUID != "t" {
		t.Fatalf("hits = %+v", r.Hits)
	}
	if len(r.Sinks) != 1 || r.Sinks[0].TargetUID != "s" {
		t.Fatalf("sinks = %+v", r.Sinks)
	}
	if len(r.Losses) != 1 || r.Losses[0].TargetUID != "p" {
		t.Fatalf("losses = %+v", r.Losses)
	}
	r.Reset()
	if len(r.Shots)+len(r.Hits)+len(r.Sinks)+len(r.Losses) != 0 {
		t.Fatalf("Reset left events: %+v", r)
	}
	r.OnShot(Shot{WeaponName: "20mm"})
	if len(r.Shots) != 1 || r.Shots[0].WeaponName != "20mm" {
		t.Fatalf("append after Reset = %+v", r.Shots)
	}
}
