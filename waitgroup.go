package waitgroup

import (
	"sync"
)

type WaitGroup interface {
	Add(delta int)
	Done()
	Wait()
}

type LimitWaitGroup struct {
	wg    sync.WaitGroup
	limit chan struct{}
}

// NewWaitGroup creates a WaitGroup with an optional concurrency limit.
// If limit <= 0, it returns a standard sync.WaitGroup.
func NewLimitWaitGroup(limit int) (WaitGroup, error) {
	if limit <= 0 {
		return &sync.WaitGroup{}, nil
	}
	return &LimitWaitGroup{
		limit: make(chan struct{}, limit),
	}, nil
}

func (w *LimitWaitGroup) Add(delta int) {
	for i := 0; i < delta; i++ {
		w.limit <- struct{}{}
	}
	w.wg.Add(delta)
}

func (w *LimitWaitGroup) Done() {
	w.wg.Done()
	<-w.limit
}

func (w *LimitWaitGroup) Wait() {
	w.wg.Wait()
}
