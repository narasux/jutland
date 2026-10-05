package instruction

import (
	"sync"

	"github.com/narasux/jutland/pkg/utils/grid"
)

const pathSearchWorkers = 2

type pathSearchJob struct {
	grid        *grid.Grid
	start, goal grid.Point
	// costScale 贴岸代价缩放：大舰需要更多离岸余量，小艇可以贴着岸走
	costScale float64
	reply     chan []grid.Point
}

// pathSearchPool 同时只跑有限个寻路，结果经 channel 交回主线程。
type pathSearchPool struct {
	mu      sync.Mutex
	pending []pathSearchJob
	wake    chan struct{}
}

var pathSearches = newPathSearchPool()

func newPathSearchPool() *pathSearchPool {
	pool := &pathSearchPool{wake: make(chan struct{}, 1)}
	for range pathSearchWorkers {
		go pool.worker()
	}
	return pool
}

func (p *pathSearchPool) submit(job pathSearchJob) {
	p.mu.Lock()
	p.pending = append(p.pending, job)
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *pathSearchPool) worker() {
	for {
		<-p.wake
		for {
			p.mu.Lock()
			if len(p.pending) == 0 {
				p.mu.Unlock()
				break
			}
			job := p.pending[0]
			p.pending = p.pending[1:]
			p.mu.Unlock()
			job.reply <- job.grid.SearchWithCostScale(job.start, job.goal, job.costScale)
		}
	}
}
