package observer

// Recorder is a simple in-memory observer used by tests and the headless runner.
// It is not safe for concurrent use; the simulation is single-threaded per tick.
type Recorder struct {
	Shots  []Shot
	Hits   []Hit
	Sinks  []Sink
	Losses []Loss
}

// OnShot appends a shot event.
func (r *Recorder) OnShot(event Shot) { r.Shots = append(r.Shots, event) }

// OnHit appends a hit event.
func (r *Recorder) OnHit(event Hit) { r.Hits = append(r.Hits, event) }

// OnSink appends a ship-loss event.
func (r *Recorder) OnSink(event Sink) { r.Sinks = append(r.Sinks, event) }

// OnLoss appends a plane-loss event.
func (r *Recorder) OnLoss(event Loss) { r.Losses = append(r.Losses, event) }

// Reset clears all recorded events.
func (r *Recorder) Reset() {
	r.Shots = r.Shots[:0]
	r.Hits = r.Hits[:0]
	r.Sinks = r.Sinks[:0]
	r.Losses = r.Losses[:0]
}

var _ Observer = (*Recorder)(nil)
